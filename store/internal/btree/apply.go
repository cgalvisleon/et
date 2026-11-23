package btree

import (
	"bytes"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* applyBlock aplica el delta del bloque i de un registro de btree a su página, sin tocar pd_lsn. Es la misma función
* en el camino normal y al rehacer: así las dos llegan a la misma página. No confía en el delta (al rehacer viene del
* disco): un delta malo es ErrBadRecord, nunca un pánico.
* @param kind wal.Kind, i int: el registro y el papel del bloque dentro de él (blockRole)
* @param p page.Page: bloqueada en exclusivo (restaurada de su imagen si la traía)
* @param delta []byte, lsn page.LSN: el lsn del registro (lo lleva la entrada nueva de KindLeafWrite)
* @return error
**/
func applyBlock(kind wal.Kind, i int, p page.Page, delta []byte, lsn page.LSN) error {
	if err := checkDelta(kind, i, delta); err != nil {
		return err
	}
	switch {
	case kind == wal.KindInitStore && i == 0:
		root, frozen, flags := parseInitMeta(delta)
		return initMetaPage(p, root, frozen, flags)
	case kind == wal.KindSetFrozen:
		return applyMetaFrozen(p, delta)
	case kind == wal.KindInitStore:
		return initLeaf(p, page.Opaque{Flags: page.OpaqueLeaf | page.OpaqueRoot}, len(delta) > 0)
	case kind == wal.KindLeafWrite:
		return applyLeafWrite(p, delta, lsn)
	case kind == wal.KindLeafVacuum:
		return applyLeafVacuum(p, delta)
	case kind == wal.KindLeafRewrite:
		return applyLeafRewrite(p, delta)
	case kind == wal.KindSetHints:
		return applySetHints(p, delta)
	case kind == wal.KindSplit && i == 0:
		return applySplitLeft(p, delta)
	case kind == wal.KindSplit && i == 1:
		return applySplitRight(p, delta)
	case kind == wal.KindSplit:
		return applySetPrev(p, page.BlockNumber(le.Uint32(delta)))
	case kind == wal.KindInsertPivot && i == 0:
		off, item, _ := parseInsertPivot(delta)
		return insertAt(kind, p, off, item)
	case kind == wal.KindInsertPivot && i == 1:
		return clearFlags(p, page.OpaqueIncompleteSplit)
	case kind == wal.KindNewRoot && i == 0:
		return applyNewRoot(p, delta)
	case kind == wal.KindNewRoot && i == 1:
		return applyMetaRoot(p, delta)
	case kind == wal.KindNewRoot:
		return clearFlags(p, page.OpaqueIncompleteSplit|page.OpaqueRoot)
	}
	return applyDelete(kind, i, p, delta)
}

/**
* applyDelete aplica los bloques de los registros del borrado de páginas y la raíz rápida de KindInsertPivot.
* @param kind wal.Kind, i int: el papel del bloque (blockRole), p page.Page, delta []byte
* @return error
**/
func applyDelete(kind wal.Kind, i int, p page.Page, delta []byte) error {
	switch {
	case kind == wal.KindInsertPivot && i == 2:
		return applyMetaFast(p, delta)
	case kind == wal.KindMarkHalfDead && i == 0:
		return applyHalfDeadLeaf(p, delta)
	case kind == wal.KindMarkHalfDead:
		return applyHalfDeadParent(p, delta)
	case kind == wal.KindUnlinkPage:
		return applyUnlink(i, p, delta)
	case kind == wal.KindRecyclePage:
		return applyRecycle(i, p, delta)
	}
	return badRecord(kind, "not a btree kind")
}

/**
* blockRole es el papel del bloque i de un registro: su número, salvo en KindUnlinkPage, cuyos bloques opcionales no
* ocupan lugar si no vienen (unlinkRole).
* @param kind wal.Kind, blocks []wal.BlockRef, i int
* @return int
**/
func blockRole(kind wal.Kind, blocks []wal.BlockRef, i int) int {
	if kind == wal.KindUnlinkPage {
		return unlinkRole(blocks, i)
	}
	return i
}

/**
* applyMetaFast anota en la meta la raíz rápida nueva (el padre de la única página de su nivel que se partió).
* @param p page.Page, delta []byte: fastRoot(4) fastLevel(4)
* @return error
**/
func applyMetaFast(p page.Page, delta []byte) error {
	if _, err := readMetaPage(p); err != nil {
		return err
	}
	setMetaFast(p, page.BlockNumber(le.Uint32(delta)), le.Uint32(delta[4:]))
	return nil
}

/**
* initTreePage deja p como una página del árbol vacía con el espacio especial o.
* @param p page.Page, o page.Opaque
* @return error
**/
func initTreePage(p page.Page, o page.Opaque) error {
	if err := p.Init(page.OpaqueSize); err != nil {
		return err
	}
	return p.SetOpaque(o)
}

/**
* applyLeafWrite inserta la entrada del delta en su número y le pone el lsn del registro: la clave (id, lsn) recién
* existe acá, con la hoja bloqueada desde antes de reservar. Una entrada de la forma 1 va en una hoja de la forma 1 (un
* registro del formato 1); una de la forma 2 en una de la forma 2, sin el prefijo de la hoja (que el id tiene que
* tener). Apaga page.FlagAllVisible: la versión nueva no es visible para todos (su bit del mapa de visibilidad se borró
* antes, en un KindVMClear anterior).
* @param p page.Page, delta []byte, lsn page.LSN
* @return error
**/
func applyLeafWrite(p page.Page, delta []byte, lsn page.LSN) error {
	off, e, _ := parseLeafWrite(delta)
	o, err := p.Opaque()
	if err != nil || o.Flags&page.OpaqueLeaf == 0 {
		return badRecord(wal.KindLeafWrite, "not a leaf")
	}
	n, err := pageNode(p, o)
	if err != nil || n.v2 != e.v2 {
		return badRecord(wal.KindLeafWrite, "entry and leaf of different forms")
	}
	if off < n.firstData() {
		return badRecord(wal.KindLeafWrite, "item would go before the high key")
	}
	if !n.v2 {
		if err := p.InsertItem(e.b, off); err != nil {
			return badRecord(wal.KindLeafWrite, err.Error())
		}
		stored, _ := p.Item(off)
		le.PutUint64(stored[OffEntryLSN:], uint64(lsn))
	} else if err := insertForm2(p, off, e, n.pre, lsn); err != nil {
		return err
	}
	p.SetFlags(p.Flags() &^ page.FlagAllVisible)
	return nil
}

/**
* insertForm2 guarda en off la entrada de la forma 2 del delta (con el id entero) sin el prefijo de la hoja y con el
* lsn del registro, sin armarla antes en otro búfer.
* @param p page.Page, off page.OffsetNumber, e entry: la del delta, pre []byte: el prefijo de la hoja, lsn page.LSN
* @return error: ErrBadRecord
**/
func insertForm2(p page.Page, off page.OffsetNumber, e entry, pre []byte, lsn page.LSN) error {
	if !bytes.HasPrefix(e.suffix(), pre) {
		return badRecord(wal.KindLeafWrite, "id without the leaf prefix")
	}
	var head [entry2Header]byte
	copy(head[:], e.b[:entry2Header])
	sfx := len(e.suffix()) - len(pre)
	le.PutUint32(head[:], e.w0&^(word0LenMask<<word0SuffixShift)|uint32(sfx)<<word0SuffixShift)
	le.PutUint64(head[off2LSN:], uint64(lsn))
	if err := p.InsertItemParts(head[:], e.b[entry2Header+len(pre):], off); err != nil {
		return badRecord(wal.KindLeafWrite, err.Error())
	}
	return nil
}

/**
* insertAt inserta un ítem de datos en una página del árbol, comprobando que el número no pise la clave alta.
* @param kind wal.Kind, p page.Page, off page.OffsetNumber, item []byte
* @return error: ErrBadRecord si no va en la página
**/
func insertAt(kind wal.Kind, p page.Page, off page.OffsetNumber, item []byte) error {
	o, err := p.Opaque()
	if err != nil {
		return badRecord(kind, err.Error())
	}
	if o.Next != 0 && off <= page.FirstOffset {
		return badRecord(kind, "item would go before the high key")
	}
	if p.LayoutVersion() != page.LayoutVersion {
		return badRecord(kind, "a pivot never goes in a leaf of form 2")
	}
	if err := p.InsertItem(item, off); err != nil {
		return badRecord(kind, err.Error())
	}
	return nil
}

/**
* applySplitLeft rehace la izquierda de una partición: conserva sus primeros keep ítems de datos, pone la clave alta
* nueva, enlaza a la derecha nueva y se marca OpaqueIncompleteSplit (el enlace en el padre llega en otro registro).
* Arma la página de nuevo desde una copia: queda compacta. Una hoja de la forma 2 se arma con su prefijo nuevo
* (choosePrefix de las entradas que conserva: igual o más largo que el de antes) y sus entradas recortadas.
* @param p page.Page, delta []byte
* @return error
**/
func applySplitLeft(p page.Page, delta []byte) error {
	d, _ := parseSplitLeft(delta)
	old := append(page.Page(nil), p...)
	o, err := old.Opaque()
	if err != nil {
		return badRecord(wal.KindSplit, err.Error())
	}
	if old.LayoutVersion() == page.LayoutVersion2 {
		return splitLeft2(p, old, d)
	}
	first := page.FirstOffset
	if o.Next != 0 {
		first++
	}
	if int(first)+d.keep-1 > int(old.MaxOffset()) {
		return badRecord(wal.KindSplit, "left keeps more items than it has")
	}
	o.Next, o.Flags = d.right, o.Flags|page.OpaqueIncompleteSplit
	if err := initTreePage(p, o); err != nil {
		return err
	}
	if _, err := p.AddItem(d.highKey); err != nil {
		return err
	}
	for off := first; off < first+page.OffsetNumber(d.keep); off++ {
		it, err := old.Item(off)
		if err != nil {
			return badRecord(wal.KindSplit, err.Error())
		}
		if _, err := p.AddItem(it); err != nil {
			return err
		}
	}
	return nil
}

/**
* splitLeft2 es applySplitLeft de una hoja de la forma 2: sus primeras keep entradas con la clave alta nueva y el
* prefijo que eligen (choosePrefix), sin las marcas de la página ni pd_prune_xid (como la forma 1, que se arma con
* page.Init).
* @param p, old page.Page: la hoja y su copia de antes, d splitLeft
* @return error: ErrBadRecord
**/
func splitLeft2(p, old page.Page, d splitLeft) error {
	c, err := readLeafContent(old)
	if err == nil && d.keep > len(c.entries) {
		err = corrupt("left keeps more items than it has")
	}
	if err != nil {
		return badRecord(wal.KindSplit, err.Error())
	}
	c.entries, c.highKey, c.flags, c.prune = c.entries[:d.keep], d.highKey, 0, 0
	c.o.Next, c.o.Flags = d.right, c.o.Flags|page.OpaqueIncompleteSplit
	if err := c.write2(p, choosePrefix(c.firstKey(), c.lastKey())); err != nil {
		return badRecord(wal.KindSplit, err.Error())
	}
	return nil
}

/**
* applySplitRight inicia la página derecha nueva con el espacio especial y los ítems del delta.
* @param p page.Page, delta []byte
* @return error
**/
func applySplitRight(p page.Page, delta []byte) error {
	d, _ := parseSplitRight(delta)
	if err := initTreePage(p, d.opaque); err != nil {
		return err
	}
	if d.form2 {
		if err := p.SetLayoutVersion(page.LayoutVersion2); err != nil {
			return err
		}
	}
	for _, it := range d.items {
		if _, err := p.AddItem(it); err != nil {
			return badRecord(wal.KindSplit, err.Error())
		}
	}
	return nil
}

/**
* applySetPrev cambia el enlace a la izquierda de la antigua hermana derecha de una partición.
* @param p page.Page, prev page.BlockNumber
* @return error
**/
func applySetPrev(p page.Page, prev page.BlockNumber) error {
	o, err := p.Opaque()
	if err != nil {
		return badRecord(wal.KindSplit, err.Error())
	}
	o.Prev = prev
	return p.SetOpaque(o)
}

/**
* clearFlags apaga marcas del espacio especial (la hija cuyo enlace ya está en el padre).
* @param p page.Page, flags uint16
* @return error
**/
func clearFlags(p page.Page, flags uint16) error {
	o, err := p.Opaque()
	if err != nil {
		return corrupt("%v", err)
	}
	o.Flags &^= flags
	return p.SetOpaque(o)
}

/**
* applyNewRoot inicia la raíz nueva: nivel de la hija + 1, el enlace menos infinito a la hija izquierda y el pivote
* de la derecha.
* @param p page.Page, delta []byte
* @return error
**/
func applyNewRoot(p page.Page, delta []byte) error {
	d, _ := parseNewRoot(delta)
	if err := initTreePage(p, page.Opaque{Level: d.level, Flags: page.OpaqueRoot}); err != nil {
		return err
	}
	if _, err := p.AddItem(encodePivot(minusInf, d.left)); err != nil {
		return err
	}
	_, err := p.AddItem(d.rightPivot)
	return err
}

/**
* applyMetaRoot anota la raíz nueva en la meta.
* @param p page.Page, delta []byte
* @return error
**/
func applyMetaRoot(p page.Page, delta []byte) error {
	if _, err := readMetaPage(p); err != nil {
		return err
	}
	setMetaRoot(p, page.BlockNumber(le.Uint32(delta)), le.Uint32(delta[4:]))
	return nil
}
