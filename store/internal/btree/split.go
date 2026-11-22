package btree

import (
	"bytes"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Llenado de la izquierda al partir la página de más a la derecha cuando la inserción va al final (claves
// crecientes): casi llena, como el fillfactor de nbtree (90 hojas, 70 internas). Si no, mitad y mitad por bytes, salvo
// en una hoja con inserción ascendente local (ver ascendingAt).
const (
	leafFillAtEnd     = 0.9
	internalFillAtEnd = 0.7
	balancedFill      = 0.5

	// ascendingRun es el largo de la racha de lsn creciente antes del punto de inserción que hace partir ahí
	// (ascendingAt).
	ascendingRun = 4
)

// splitPlan es cómo se parte una página: cuántos ítems de datos quedan a la izquierda, su clave alta nueva (que es
// también el separador que va al padre) y los ítems de la derecha nueva, en orden (su clave alta primero, si tiene).
type splitPlan struct {
	keep    int
	highKey []byte
	right   [][]byte
}

/**
* splitAndFinish parte la página (primer paso, KindSplit) y termina la partición (segundo paso, el enlace en el padre
* o la raíz nueva). Consume h.
* @param h buffer.Handle, n node: bloqueada en exclusivo, sin partición incompleta
* @param pos page.OffsetNumber: dónde iba la inserción que no cupo
* @param xid page.XID: la transacción que pide la página nueva
* @return error
**/
func (t *BTree) splitAndFinish(h buffer.Handle, n node, pos page.OffsetNumber, xid page.XID) error {
	if err := t.split(h, n, pos, xid); err != nil {
		return err
	}
	if t.afterSplit != nil {
		if err := t.afterSplit(n.blk); err != nil {
			return err
		}
	}
	return t.finishSplit(n.blk)
}

/**
* split hace el primer paso de una partición (como _bt_split): pide una página nueva, pasa a ella la parte derecha de
* los ítems, deja en la izquierda su clave alta nueva y OpaqueIncompleteSplit, y enlaza la antigua hermana derecha
* con la nueva, todo en un KindSplit. El árbol queda válido: la derecha se alcanza por el enlace de la izquierda
* hasta que su enlace llegue al padre. Pide páginas con la izquierda bloqueada, como nbtree (a confirmar en
* store/doc.go: la regla de no fijar con una página bloqueada); la página nueva no la ve nadie más, y la antigua
* hermana se bloquea de izquierda a derecha. Consume h.
* @param h buffer.Handle, n node: la página, bloqueada en exclusivo, sin partición incompleta
* @param pos page.OffsetNumber, xid page.XID: ver splitAndFinish
* @return error
**/
func (t *BTree) split(h buffer.Handle, n node, pos page.OffsetNumber, xid page.XID) error {
	plan, err := planSplit(n, pos)
	if err == nil {
		err = t.clearVisible(n) // la izquierda se arma de nuevo, sin page.FlagAllVisible
	}
	if err != nil {
		h.Release()
		return err
	}
	newH, err := t.free.Allocate(t.file, xid)
	if err != nil {
		h.Release()
		return err
	}
	var oldRight buffer.Handle
	if !n.rightmost() {
		var on node
		if oldRight, on, err = t.lockNode(n.o.Next, buffer.Exclusive); err != nil {
			releaseAll(h, newH)
			return err
		}
		if on.o.Prev != n.blk {
			releaseAll(h, newH, oldRight)
			return corruptAt(on.blk, "left link %d, expected %d", on.o.Prev, n.blk)
		}
	}

	rightBlk := newH.Tag().Block
	rightOpaque := page.Opaque{
		Prev:  n.blk,
		Next:  n.o.Next,
		Level: n.o.Level,
		Flags: n.o.Flags &^ (page.OpaqueRoot | page.OpaqueIncompleteSplit | page.OpaqueSplitEnd),
	}
	refs := []wal.BlockRef{
		{Delta: splitLeftDelta(rightBlk, plan.keep, plan.highKey)},
		{Flags: wal.BlockWillInit, Delta: splitRightDelta(rightOpaque, plan.right)},
	}
	hs := []buffer.Handle{h, newH}
	if oldRight != nil {
		refs = append(refs, wal.BlockRef{Delta: blockDelta(rightBlk)})
		hs = append(hs, oldRight)
	}
	res, err := t.logChange(wal.KindSplit, xid, 0, refs, hs, nil)
	releaseAll(hs...)
	if err != nil {
		return err
	}
	return res.Write()
}

/**
* planSplit elige dónde partir una página: el punto que deja la izquierda más cerca del llenado buscado, por bytes
* (las claves tienen largo variable), entre los que dejan las dos mitades dentro de la página con sus claves altas.
* En una hoja la clave alta nueva es el separador truncado entre la última clave de la izquierda y la primera de la
* derecha; en una interna, la clave del primer enlace de la derecha, que pasa a ser menos infinito ahí.
* @param n node: la página, bloqueada en exclusivo
* @param pos page.OffsetNumber: dónde iba la inserción que no cupo (decide el llenado buscado, splitTarget)
* @return splitPlan, error: ErrCorrupt si no hay punto válido (no pasa con MaxEntrySize)
**/
func planSplit(n node, pos page.OffsetNumber) (splitPlan, error) {
	first, count := n.firstData(), n.dataCount()
	if count < 2 {
		return splitPlan{}, corruptAt(n.blk, "page with %d items cannot be split", count)
	}
	items, keys := make([][]byte, count), make([]key, count)
	total := 0
	for i := range count {
		off := first + page.OffsetNumber(i)
		it, err := n.item(off)
		if err != nil {
			return splitPlan{}, err
		}
		if keys[i], err = n.keyAt(off); err != nil {
			return splitPlan{}, err
		}
		items[i] = it
		total += page.SpaceFor(len(it))
	}
	rightBase := 0
	if !n.rightmost() {
		hk, err := n.highKey()
		if err != nil {
			return splitPlan{}, err
		}
		rightBase = page.SpaceFor(len(hk))
	}

	target := splitTarget(n, pos, items, keys, total)
	best, bestDiff, left := -1, 0, 0
	for s := 1; s < count; s++ {
		left += page.SpaceFor(len(items[s-1]))
		right := rightBase + total - left
		if !n.leaf() {
			right += page.SpaceFor(PivotHeaderSize) - page.SpaceFor(len(items[s]))
		}
		hk := splitKey(n.leaf(), keys[s-1], keys[s])
		if left+page.SpaceFor(PivotHeaderSize+len(hk.id)) > capacity || right > capacity {
			continue
		}
		if diff := abs(left - target); best < 0 || diff < bestDiff {
			best, bestDiff = s, diff
		}
	}
	if best < 0 {
		return splitPlan{}, corruptAt(n.blk, "no valid split point")
	}
	return buildPlan(n, items, keys, best)
}

/**
* buildPlan arma el plan de partir antes del ítem de datos s: copia los ítems de la derecha (los de la página son
* vistas que la partición pisa).
* @param n node, items [][]byte, keys []key, s int
* @return splitPlan, error
**/
func buildPlan(n node, items [][]byte, keys []key, s int) (splitPlan, error) {
	plan := splitPlan{keep: s, highKey: encodePivot(splitKey(n.leaf(), keys[s-1], keys[s]), 0)}
	if !n.rightmost() {
		hk, err := n.highKey()
		if err != nil {
			return splitPlan{}, err
		}
		plan.right = append(plan.right, bytes.Clone(hk))
	}
	for i := s; i < len(items); i++ {
		it := bytes.Clone(items[i])
		if !n.leaf() && i == s {
			v, err := parsePivot(it)
			if err != nil {
				return splitPlan{}, corruptAt(n.blk, "%v", err)
			}
			it = encodePivot(minusInf, v.downlink())
		}
		plan.right = append(plan.right, it)
	}
	return plan, nil
}

/**
* splitKey es la clave alta nueva de la izquierda: en una hoja el separador truncado; en una interna, la clave del
* primer enlace que pasa a la derecha.
* @param leaf bool, last, first key
* @return key: copia
**/
func splitKey(leaf bool, last, first key) key {
	if leaf {
		return separator(last, first)
	}
	return cloneKey(first)
}

/**
* splitTarget es cuántos bytes de datos buscan quedar a la izquierda. Al final de la página de más a la derecha, el
* llenado de claves crecientes (fillFor). En una hoja con inserción ascendente local (ascendingAt), el punto de
* inserción: lo de la izquierda no vuelve a recibir inserciones, así que queda llena; acotado arriba al 90 % (la
* inserción al final de una hoja que no es la de más a la derecha deja un 10 % a la derecha, como el usemult de
* _bt_findsplitloc) y solo si deja la izquierda al menos a la mitad (si no, mitad y mitad: partir antes de la mitad
* dejaría la izquierda medio vacía para siempre). Si no, mitad y mitad.
* Con grupos (v01|… seguido de v02|…) el separador del corte en pos sale truncado («v02») y la entrada nueva, menor,
* cae a la izquierda: queda la última de la izquierda, como «partir después del ítem nuevo» de _bt_afternewitemoff, y
* las siguientes del grupo también van ahí hasta que la izquierda se llena y se parte al final (al 90 %). No es una
* partición de más: exigir que la entrada nueva caiga a la derecha (si no, mitad y mitad) bajó el llenado de los
* índices de josefina del 83 al 65 % (TestIndexHeaderShare) y el de TestSplitAscendingGroups del 86 al 55 % con 5
* grupos (revisión de la tarea 129 de F-23).
* @param n node, pos page.OffsetNumber, items [][]byte, keys []key: los ítems de datos, total int: sus bytes
* @return int
**/
func splitTarget(n node, pos page.OffsetNumber, items [][]byte, keys []key, total int) int {
	atEnd := n.rightmost() && pos > n.maxOff()
	if atEnd || !n.leaf() || !ascendingAt(n, pos, keys) {
		return int(float64(total) * fillFor(n.leaf(), atEnd))
	}
	left := 0
	for i := range int(pos - n.firstData()) {
		left += page.SpaceFor(len(items[i]))
	}
	if left < int(float64(total)*balancedFill) {
		return int(float64(total) * balancedFill)
	}
	return min(left, int(float64(total)*leafFillAtEnd))
}

/**
* ascendingAt dice si la inserción en pos sigue una racha ascendente local (como _bt_afternewitemoff de nbtree): las
* ascendingRun versiones justo antes de pos se escribieron en ese orden (lsn creciente hacia pos). Es lo que deja un
* grupo de duplicados con sufijo creciente que crece por el final (un índice valor + jid de josefina: cada documento
* nuevo va al final del grupo de su valor), aunque varios grupos terminen en la misma hoja, y cualquier secuencia
* creciente que cae en medio del árbol. No se guarda nada en memoria: el lsn de cada versión ya dice el orden. Con
* inserciones al azar la racha sale una vez cada ascendingRun! (24) y además tiene que caer pasada la mitad de la
* hoja; si no, se parte mitad y mitad. Las versiones de un mismo id van de la más nueva a la más vieja: cortan la
* racha.
* @param n node: una hoja, pos page.OffsetNumber, keys []key: las claves de sus ítems de datos
* @return bool
**/
func ascendingAt(n node, pos page.OffsetNumber, keys []key) bool {
	end := int(pos - n.firstData()) // la racha son keys[end-ascendingRun : end]
	if end < ascendingRun || end > len(keys) {
		return false
	}
	for i := end - ascendingRun + 1; i < end; i++ {
		if keys[i].lsn <= keys[i-1].lsn {
			return false
		}
	}
	return true
}

/**
* fillFor es el llenado buscado de la izquierda.
* @param leaf, atEnd bool
* @return float64
**/
func fillFor(leaf, atEnd bool) float64 {
	switch {
	case !atEnd:
		return balancedFill
	case leaf:
		return leafFillAtEnd
	}
	return internalFillAtEnd
}

// abs es el valor absoluto.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
