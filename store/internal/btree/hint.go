package btree

import (
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// maxPendingHints acota las marcas diferidas de una operación: las que no entran se pondrán en otra lectura.
const maxPendingHints = 256

// hintItemSize es el largo de cada marca en el delta de KindSetHints: off(2) flags(4).
const hintItemSize = 2 + 4

// pendingHint es una marca que no se pudo poner sin WAL: su página es anterior al punto de rehacer vigente (sería la
// primera marca tras un checkpoint) y va al WAL con la página bloqueada en exclusivo.
type pendingHint struct {
	blk  page.BlockNumber
	k    key // copia
	flag EntryFlags
}

// hintItem es una marca dentro de una página: el número de la entrada y las marcas que se le agregan.
type hintItem struct {
	off  page.OffsetNumber
	flag EntryFlags
}

// hints es dónde una operación pone las marcas de visibilidad: la página bloqueada que está leyendo y las marcas
// diferidas. Un *hints nil, o sin página, no marca nada.
type hints struct {
	h       buffer.Handle
	pending []pendingHint
}

/**
* at cambia la página que se está leyendo (al seguir a la derecha).
* @param h buffer.Handle: bloqueada en compartido o exclusivo
**/
func (hs *hints) at(h buffer.Handle) {
	if hs != nil {
		hs.h = h
	}
}

/**
* hint marca una versión cuyo estado ya es definitivo. Si la página tiene su imagen en el WAL desde el último punto de
* rehacer, pone la marca enseguida con un OR atómico y MarkDirtyHint; si no, la deja diferida para flushHints, que la
* anota en el WAL (la primera marca tras un checkpoint, pregunta 36).
* @param hs *hints, e entry: de la página hs.h, flag EntryFlags: EntryXidCommitted o EntryXidAborted
**/
func (t *BTree) hint(hs *hints, e entry, flag EntryFlags) {
	if hs == nil || hs.h == nil {
		return
	}
	if t.hintUnlogged(hs.h, func() { e.orFlags(flag) }) {
		return
	}
	if len(hs.pending) < maxPendingHints {
		hs.pending = append(hs.pending, pendingHint{blk: hs.h.Tag().Block, k: cloneKey(e.key()), flag: flag})
	}
}

/**
* hintUnlogged pone marcas sin WAL si la página ya tiene su imagen desde el punto de rehacer vigente (pd_lsn ≥ punto
* de rehacer). Con HintGate, la decisión y la marca no se cruzan con el comienzo de un checkpoint. Sin ella, vuelve a
* mirar el punto de rehacer después de marcar: si un checkpoint empezó entre medio, la marca hay que anotarla en el WAL
* (devuelve false aunque ya la puso).
* @param h buffer.Handle: bloqueada (compartido o exclusivo), set func(): pone las marcas
* @return bool: true si las marcas quedaron puestas y no hace falta el WAL
**/
func (t *BTree) hintUnlogged(h buffer.Handle, set func()) bool {
	if t.gate != nil {
		t.gate.RLock()
		defer t.gate.RUnlock()
	}
	p := h.Page()
	if p.LSN() < t.log.RedoPoint().LSN {
		return false
	}
	set()
	h.MarkDirtyHint()
	return t.gate != nil || p.LSN() >= t.log.RedoPoint().LSN
}

/**
* flushHints anota las marcas diferidas, página por página, con la página bloqueada en exclusivo. Se llama sin páginas
* bloqueadas. Los errores se ignoran: una marca es solo una pista (un fallo del WAL lo verá la próxima escritura).
* @param hs *hints
**/
func (t *BTree) flushHints(hs *hints) {
	for len(hs.pending) > 0 {
		n := 1
		for n < len(hs.pending) && hs.pending[n].blk == hs.pending[0].blk {
			n++
		}
		_ = t.logHints(hs.pending[0].blk, hs.pending[:n])
		hs.pending = hs.pending[n:]
	}
	hs.pending = nil
}

/**
* logHints pone marcas en una hoja bloqueada en exclusivo: sin WAL si otra escritura ya anotó la página desde el punto
* de rehacer, o con un KindSetHints (que lleva la imagen de la página antes de las marcas). Una versión que ya no está
* en la hoja (se movió a la derecha en una partición) se salta.
* @param blk page.BlockNumber, list []pendingHint: todas de blk
* @return error
**/
func (t *BTree) logHints(blk page.BlockNumber, list []pendingHint) error {
	h, n, err := t.lockNode(blk, buffer.Exclusive)
	if err != nil {
		return err
	}
	var items []hintItem
	for _, ph := range list {
		if off, ok := findEntry(n, ph.k); ok {
			items = append(items, hintItem{off: off, flag: ph.flag})
		}
	}
	if len(items) == 0 || !n.leaf() || t.hintUnlogged(h, func() { _ = setHintItems(n.p, items) }) {
		h.Release()
		return nil
	}
	refs := []wal.BlockRef{{Delta: hintsDelta(items)}}
	res, err := t.logChange(wal.KindSetHints, 0, 0, refs, []buffer.Handle{h}, nil)
	h.Release()
	if err != nil {
		return err
	}
	return res.Write()
}

/**
* findEntry busca en una hoja la entrada de clave k.
* @param n node, k key
* @return page.OffsetNumber, bool: si está
**/
func findEntry(n node, k key) (page.OffsetNumber, bool) {
	off, err := n.lowerBound(k)
	if err != nil || off > n.maxOff() {
		return 0, false
	}
	e, err := n.entryAt(off)
	if err != nil || compareKeys(e.key(), k) != 0 {
		return 0, false
	}
	return off, true
}

/**
* hintsDelta arma el delta de KindSetHints: count(2) + count × [off(2) flags(4)].
* @param items []hintItem
* @return []byte
**/
func hintsDelta(items []hintItem) []byte {
	b := le.AppendUint16(nil, uint16(len(items)))
	for _, it := range items {
		b = le.AppendUint16(b, uint16(it.off))
		b = le.AppendUint32(b, uint32(it.flag))
	}
	return b
}

/**
* parseHints lee el delta de KindSetHints: al menos una marca, cada una con número de ítem y solo marcas de
* visibilidad (EntryXidCommitted, EntryXidAborted).
* @param delta []byte
* @return []hintItem, error: ErrBadRecord
**/
func parseHints(delta []byte) ([]hintItem, error) {
	if len(delta) < 2 {
		return nil, badRecord(wal.KindSetHints, "hints delta too short")
	}
	count := int(le.Uint16(delta))
	if count == 0 || len(delta) != 2+count*hintItemSize {
		return nil, badRecord(wal.KindSetHints, "bad hints delta length")
	}
	items := make([]hintItem, count)
	for i := range items {
		b := delta[2+i*hintItemSize:]
		items[i] = hintItem{off: page.OffsetNumber(le.Uint16(b)), flag: EntryFlags(le.Uint32(b[2:]))}
		if items[i].off == 0 || items[i].flag == 0 || items[i].flag&^hintFlags != 0 {
			return nil, badRecord(wal.KindSetHints, "bad hint")
		}
	}
	return items, nil
}

/**
* applySetHints aplica el delta de KindSetHints a la hoja (la misma función al marcar y al rehacer).
* @param p page.Page, delta []byte
* @return error: ErrBadRecord si una marca no cae en una entrada de la hoja
**/
func applySetHints(p page.Page, delta []byte) error {
	items, err := parseHints(delta)
	if err != nil {
		return err
	}
	return setHintsChecked(p, items)
}

/**
* setHintsChecked valida que cada marca caiga en una entrada de datos de una hoja y recién entonces las pone.
* @param p page.Page, items []hintItem
* @return error: ErrBadRecord
**/
func setHintsChecked(p page.Page, items []hintItem) error {
	o, err := p.Opaque()
	if err != nil || o.Flags&page.OpaqueLeaf == 0 {
		return badRecord(wal.KindSetHints, "not a leaf")
	}
	n := node{p: p, o: o}
	for _, it := range items {
		if it.off < n.firstData() || it.off > n.maxOff() {
			return badRecord(wal.KindSetHints, "hint outside the leaf's entries")
		}
		if _, err := n.entryAt(it.off); err != nil {
			return badRecord(wal.KindSetHints, err.Error())
		}
	}
	return setHintItems(p, items)
}

/**
* setHintItems pone las marcas con OR atómico en entradas ya validadas.
* @param p page.Page, items []hintItem
* @return error: ErrBadRecord si una entrada no está alineada (no pasa en una página de la caché)
**/
func setHintItems(p page.Page, items []hintItem) error {
	for _, it := range items {
		b, err := p.Item(it.off)
		if err != nil || !entry(b).orFlags(it.flag) {
			return badRecord(wal.KindSetHints, "cannot set a hint")
		}
	}
	return nil
}
