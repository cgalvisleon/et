package btree

import (
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Largos fijos de los deltas (formatos en doc.go, «Registros del WAL»).
const (
	blockDeltaSize       = 4     // un número de bloque
	offsetDeltaSize      = 2     // un número de ítem
	splitLeftHeaderSize  = 4 + 2 // right(4) keep(2)
	splitRightHeaderSize = 16    // prev(4) next(4) level(4) flags(2) count(2)
	newRootHeaderSize    = 4 + 4 // level(4) left(4)
	metaRootDeltaSize    = 4 + 4 // root(4) level(4)
	initMetaDeltaSize    = 4 + 8 // root(4) frozenXID(8); el formato viejo trae solo root(4)
	frozenDeltaSize      = 8     // frozenXID(8)
	itemLenSize          = 2     // largo de cada ítem en el delta de la derecha
)

// blockDelta arma el delta de un número de bloque.
func blockDelta(b page.BlockNumber) []byte {
	return le.AppendUint32(nil, uint32(b))
}

// initMetaDelta arma el delta de la meta de un store nuevo: la raíz y el frozenXID.
func initMetaDelta(root page.BlockNumber, frozen page.XID) []byte {
	return le.AppendUint64(blockDelta(root), uint64(frozen))
}

/**
* parseInitMeta lee el delta de la meta de KindInitStore: root(4) y frozenXID(8), o solo root(4) (formato anterior al
* congelado: frozenXID 0, el store retiene el truncado hasta su primera limpieza completa).
* @param delta []byte: ya validado por checkFixedDelta
* @return page.BlockNumber, page.XID
**/
func parseInitMeta(delta []byte) (page.BlockNumber, page.XID) {
	root := page.BlockNumber(le.Uint32(delta))
	if len(delta) < initMetaDeltaSize {
		return root, 0
	}
	return root, page.XID(le.Uint64(delta[blockDeltaSize:]))
}

// itemDelta arma el delta de un ítem nuevo: el número donde va y el ítem.
func itemDelta(off page.OffsetNumber, item []byte) []byte {
	return append(le.AppendUint16(nil, uint16(off)), item...)
}

// splitLeftDelta arma el delta de la izquierda de una partición: su hermana derecha nueva, cuántos ítems de datos
// conserva y su clave alta nueva.
func splitLeftDelta(right page.BlockNumber, keep int, highKey []byte) []byte {
	b := le.AppendUint32(nil, uint32(right))
	b = le.AppendUint16(b, uint16(keep))
	return append(b, highKey...)
}

// splitRightDelta arma el delta de la página derecha nueva: su espacio especial y todos sus ítems, en orden.
func splitRightDelta(o page.Opaque, items [][]byte) []byte {
	b := le.AppendUint32(nil, uint32(o.Prev))
	b = le.AppendUint32(b, uint32(o.Next))
	b = le.AppendUint32(b, o.Level)
	b = le.AppendUint16(b, o.Flags)
	b = le.AppendUint16(b, uint16(len(items)))
	for _, it := range items {
		b = le.AppendUint16(b, uint16(len(it)))
		b = append(b, it...)
	}
	return b
}

// newRootDelta arma el delta de una raíz nueva: su nivel, la hija izquierda y el pivote de la derecha.
func newRootDelta(level uint32, left page.BlockNumber, rightPivot []byte) []byte {
	b := le.AppendUint32(nil, level)
	b = le.AppendUint32(b, uint32(left))
	return append(b, rightPivot...)
}

// metaRootDelta arma el delta de la meta con la raíz nueva.
func metaRootDelta(root page.BlockNumber, level uint32) []byte {
	return le.AppendUint32(le.AppendUint32(nil, uint32(root)), level)
}

// splitLeft es el delta de la izquierda de una partición, ya leído.
type splitLeft struct {
	right   page.BlockNumber
	keep    int
	highKey []byte
}

// splitRight es el delta de la derecha nueva, ya leído.
type splitRight struct {
	opaque page.Opaque
	items  [][]byte
}

// newRootData es el delta de la raíz nueva, ya leído.
type newRootData struct {
	level      uint32
	left       page.BlockNumber
	rightPivot []byte
}

/**
* parseLeafWrite lee el delta de KindLeafWrite: el número del ítem y la entrada.
* @param delta []byte
* @return page.OffsetNumber, []byte, error: ErrBadRecord
**/
func parseLeafWrite(delta []byte) (page.OffsetNumber, []byte, error) {
	off, item, err := parseItemDelta(wal.KindLeafWrite, delta)
	if err == nil {
		if _, perr := parseEntry(item); perr != nil {
			err = badRecord(wal.KindLeafWrite, perr.Error())
		}
	}
	return off, item, err
}

/**
* parseInsertPivot lee el delta del padre en KindInsertPivot: el número del ítem y el pivote.
* @param delta []byte
* @return page.OffsetNumber, []byte, error: ErrBadRecord
**/
func parseInsertPivot(delta []byte) (page.OffsetNumber, []byte, error) {
	off, item, err := parseItemDelta(wal.KindInsertPivot, delta)
	if err == nil {
		if v, perr := parsePivot(item); perr != nil || v.minusInf() || v.downlink() == 0 {
			err = badRecord(wal.KindInsertPivot, "bad pivot")
		}
	}
	return off, item, err
}

/**
* parseItemDelta separa el delta de un ítem nuevo en su número y el ítem.
* @param kind wal.Kind, delta []byte
* @return page.OffsetNumber, []byte, error
**/
func parseItemDelta(kind wal.Kind, delta []byte) (page.OffsetNumber, []byte, error) {
	if len(delta) <= offsetDeltaSize || le.Uint16(delta) == 0 {
		return 0, nil, badRecord(kind, "bad item delta")
	}
	return page.OffsetNumber(le.Uint16(delta)), delta[offsetDeltaSize:], nil
}

/**
* parseSplitLeft lee el delta de la izquierda de una partición.
* @param delta []byte
* @return splitLeft, error: ErrBadRecord
**/
func parseSplitLeft(delta []byte) (splitLeft, error) {
	if len(delta) <= splitLeftHeaderSize {
		return splitLeft{}, badRecord(wal.KindSplit, "left delta too short")
	}
	d := splitLeft{right: page.BlockNumber(le.Uint32(delta)), keep: int(le.Uint16(delta[4:])), highKey: delta[splitLeftHeaderSize:]}
	if v, err := parsePivot(d.highKey); err != nil || v.minusInf() || v.downlink() != 0 || d.right <= MetaBlock || d.keep < 1 {
		return splitLeft{}, badRecord(wal.KindSplit, "bad left high key, right link or kept items")
	}
	return d, nil
}

/**
* parseSplitRight lee el delta de la derecha nueva de una partición, ítem por ítem.
* @param delta []byte
* @return splitRight, error: ErrBadRecord
**/
func parseSplitRight(delta []byte) (splitRight, error) {
	if len(delta) < splitRightHeaderSize {
		return splitRight{}, badRecord(wal.KindSplit, "right delta too short")
	}
	d := splitRight{opaque: page.Opaque{
		Prev:  page.BlockNumber(le.Uint32(delta)),
		Next:  page.BlockNumber(le.Uint32(delta[4:])),
		Level: le.Uint32(delta[8:]),
		Flags: le.Uint16(delta[12:]),
	}}
	count, rest := int(le.Uint16(delta[14:])), delta[splitRightHeaderSize:]
	for range count {
		if len(rest) < itemLenSize {
			return splitRight{}, badRecord(wal.KindSplit, "right delta cut")
		}
		n := int(le.Uint16(rest))
		if n == 0 || len(rest) < itemLenSize+n {
			return splitRight{}, badRecord(wal.KindSplit, "right delta item cut")
		}
		d.items, rest = append(d.items, rest[itemLenSize:itemLenSize+n]), rest[itemLenSize+n:]
	}
	if len(rest) != 0 || count == 0 || d.opaque.Prev <= MetaBlock || d.opaque.Flags&(notTreePage|ignoredPage) != 0 {
		return splitRight{}, badRecord(wal.KindSplit, "bad right delta")
	}
	return d, nil
}

/**
* parseNewRoot lee el delta de la raíz nueva.
* @param delta []byte
* @return newRootData, error: ErrBadRecord
**/
func parseNewRoot(delta []byte) (newRootData, error) {
	if len(delta) <= newRootHeaderSize {
		return newRootData{}, badRecord(wal.KindNewRoot, "root delta too short")
	}
	d := newRootData{level: le.Uint32(delta), left: page.BlockNumber(le.Uint32(delta[4:])), rightPivot: delta[newRootHeaderSize:]}
	if v, err := parsePivot(d.rightPivot); err != nil || v.minusInf() || v.downlink() <= MetaBlock || d.level == 0 || d.left <= MetaBlock {
		return newRootData{}, badRecord(wal.KindNewRoot, "bad root delta")
	}
	return d, nil
}

/**
* checkDelta valida el delta del bloque i antes de tocar ninguna página: al rehacer, un bloque con BlockWillInit se
* pone en ceros al fijarlo, y un delta malo lo dejaría así.
* @param kind wal.Kind, i int, delta []byte
* @return error: ErrBadRecord
**/
func checkDelta(kind wal.Kind, i int, delta []byte) error {
	var err error
	switch {
	case kind == wal.KindLeafWrite:
		_, _, err = parseLeafWrite(delta)
	case kind == wal.KindInsertPivot && i == 0:
		_, _, err = parseInsertPivot(delta)
	case kind == wal.KindSplit && i == 0:
		_, err = parseSplitLeft(delta)
	case kind == wal.KindSplit && i == 1:
		_, err = parseSplitRight(delta)
	case kind == wal.KindNewRoot && i == 0:
		_, err = parseNewRoot(delta)
	case kind == wal.KindLeafVacuum:
		_, err = parseVacuum(delta)
	case kind == wal.KindSetHints:
		_, err = parseHints(delta)
	case kind == wal.KindMarkHalfDead:
		err = checkHalfDeadDelta(i, delta)
	case kind == wal.KindUnlinkPage && i == roleTarget:
		_, err = parseUnlinkTarget(delta)
	case kind == wal.KindUnlinkPage && i == roleMeta:
		if len(delta) != blockDeltaSize && len(delta) != blockDeltaSize+fastDeltaSize {
			err = badRecord(kind, "bad meta delta length")
		}
	case kind == wal.KindUnlinkPage:
		if len(delta) != blockDeltaSize || i < 0 || i > roleLeaf {
			err = badRecord(kind, "bad delta length")
		}
	case kind == wal.KindRecyclePage:
		if (i == 0 && len(delta) != recycleMetaSize) || (i != 0 && len(delta) != blockDeltaSize) {
			err = badRecord(kind, "bad recycle delta length")
		}
	case kind == wal.KindInsertPivot && i == 2:
		if len(delta) != fastDeltaSize || page.BlockNumber(le.Uint32(delta)) <= MetaBlock {
			err = badRecord(kind, "bad fast root delta")
		}
	default:
		err = checkFixedDelta(kind, i, delta)
	}
	return err
}

/**
* checkFixedDelta valida los deltas de largo fijo: un bloque (con el frozenXID en la meta de KindInitStore), la raíz
* de la meta, el frozenXID de KindSetFrozen o ninguno.
* @param kind wal.Kind, i int, delta []byte
* @return error
**/
func checkFixedDelta(kind wal.Kind, i int, delta []byte) error {
	want := 0
	switch {
	case kind == wal.KindInitStore && i == 0 && len(delta) == initMetaDeltaSize:
		want = initMetaDeltaSize
	case kind == wal.KindInitStore && i == 0, kind == wal.KindSplit:
		want = blockDeltaSize
	case kind == wal.KindSetFrozen:
		if len(delta) != frozenDeltaSize {
			return badRecord(kind, "bad delta length")
		}
		return nil
	case kind == wal.KindNewRoot && i == 1:
		want = metaRootDeltaSize
	}
	if len(delta) != want {
		return badRecord(kind, "bad delta length")
	}
	if want > 0 && page.BlockNumber(le.Uint32(delta)) <= MetaBlock {
		return badRecord(kind, "delta points to the meta page")
	}
	return nil
}
