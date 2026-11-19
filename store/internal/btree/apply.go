package btree

import (
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* applyBlock aplica el delta del bloque i de un registro de btree a su página, sin tocar pd_lsn. Es la misma función
* en el camino normal y al rehacer: así las dos llegan a la misma página. No confía en el delta (al rehacer viene del
* disco): un delta malo es ErrBadRecord, nunca un pánico.
* @param kind wal.Kind, i int: el registro y el número de bloque dentro de él
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
		return initMetaPage(p, page.BlockNumber(le.Uint32(delta)))
	case kind == wal.KindInitStore:
		return initTreePage(p, page.Opaque{Flags: page.OpaqueLeaf | page.OpaqueRoot})
	case kind == wal.KindLeafWrite:
		return applyLeafWrite(p, delta, lsn)
	case kind == wal.KindSplit && i == 0:
		return applySplitLeft(p, delta)
	case kind == wal.KindSplit && i == 1:
		return applySplitRight(p, delta)
	case kind == wal.KindSplit:
		return applySetPrev(p, page.BlockNumber(le.Uint32(delta)))
	case kind == wal.KindInsertPivot && i == 0:
		off, item, _ := parseInsertPivot(delta)
		return insertAt(kind, p, off, item)
	case kind == wal.KindInsertPivot:
		return clearFlags(p, page.OpaqueIncompleteSplit)
	case kind == wal.KindNewRoot && i == 0:
		return applyNewRoot(p, delta)
	case kind == wal.KindNewRoot && i == 1:
		return applyMetaRoot(p, delta)
	case kind == wal.KindNewRoot:
		return clearFlags(p, page.OpaqueIncompleteSplit|page.OpaqueRoot)
	}
	return badRecord(kind, "not a btree kind")
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
* existe acá, con la hoja bloqueada desde antes de reservar.
* @param p page.Page, delta []byte, lsn page.LSN
* @return error
**/
func applyLeafWrite(p page.Page, delta []byte, lsn page.LSN) error {
	off, item, _ := parseLeafWrite(delta)
	if err := insertAt(wal.KindLeafWrite, p, off, item); err != nil {
		return err
	}
	stored, err := p.Item(off)
	if err != nil {
		return err
	}
	le.PutUint64(stored[OffEntryLSN:], uint64(lsn))
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
	if err := p.InsertItem(item, off); err != nil {
		return badRecord(kind, err.Error())
	}
	return nil
}

/**
* applySplitLeft rehace la izquierda de una partición: conserva sus primeros keep ítems de datos, pone la clave alta
* nueva, enlaza a la derecha nueva y se marca OpaqueIncompleteSplit (el enlace en el padre llega en otro registro).
* Arma la página de nuevo desde una copia: queda compacta.
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
* applySplitRight inicia la página derecha nueva con el espacio especial y los ítems del delta.
* @param p page.Page, delta []byte
* @return error
**/
func applySplitRight(p page.Page, delta []byte) error {
	d, _ := parseSplitRight(delta)
	if err := initTreePage(p, d.opaque); err != nil {
		return err
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
