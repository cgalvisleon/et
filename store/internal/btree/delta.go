package btree

import (
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Largos fijos de los deltas (formatos en doc.go, «Registros del WAL»).
const (
	blockDeltaSize       = 4         // un número de bloque
	offsetDeltaSize      = 2         // un número de ítem
	splitLeftHeaderSize  = 4 + 2     // right(4) keep(2)
	splitRightHeaderSize = 16        // prev(4) next(4) level(4) flags(2) count(2)
	newRootHeaderSize    = 4 + 4     // level(4) left(4)
	metaRootDeltaSize    = 4 + 4     // root(4) level(4)
	initMetaDeltaSize    = 4 + 8     // root(4) frozenXID(8); el formato viejo trae solo root(4)
	initMetaFlagsSize    = 4 + 8 + 4 // root(4) frozenXID(8) flags(4): formato 2
	frozenDeltaSize      = 8         // frozenXID(8)
	frozenFlagsDeltaSize = 8 + 4     // frozenXID(8) flags(4): formato 2
	itemLenSize          = 2         // largo de cada ítem en el delta de la derecha
	layoutDeltaSize      = 2         // layout(2): la hoja raíz de KindInitStore es de la forma 2 (formato 2)
)

// layoutDelta arma el delta de la hoja raíz de un store nuevo del formato 2: su versión de página.
func layoutDelta(v int) []byte {
	return le.AppendUint16(nil, uint16(v))
}

// blockDelta arma el delta de un número de bloque.
func blockDelta(b page.BlockNumber) []byte {
	return le.AppendUint32(nil, uint32(b))
}

// initMetaDelta arma el delta de la meta de un store nuevo: la raíz, el frozenXID y sus marcas (formato 2).
func initMetaDelta(root page.BlockNumber, frozen page.XID, flags MetaFlags) []byte {
	return le.AppendUint32(le.AppendUint64(blockDelta(root), uint64(frozen)), uint32(flags))
}

/**
* parseInitMeta lee el delta de la meta de KindInitStore: root(4), frozenXID(8) y flags(4) (formato 2); root(4) y
* frozenXID(8) (formato 1: sin marcas, sus hojas son de la forma 1); o solo root(4) (anterior al congelado:
* frozenXID 0, el store retiene el truncado hasta su primera limpieza completa).
* @param delta []byte: ya validado por checkFixedDelta
* @return page.BlockNumber, page.XID, MetaFlags
**/
func parseInitMeta(delta []byte) (page.BlockNumber, page.XID, MetaFlags) {
	root := page.BlockNumber(le.Uint32(delta))
	if len(delta) < initMetaDeltaSize {
		return root, 0, 0
	}
	frozen := page.XID(le.Uint64(delta[blockDeltaSize:]))
	if len(delta) < initMetaFlagsSize {
		return root, frozen, 0
	}
	return root, frozen, MetaFlags(le.Uint32(delta[initMetaDeltaSize:]))
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

// splitRightForm2 es el bit 15 de count en el delta de la derecha nueva: es una hoja de la forma 2 (formato 2).
const splitRightForm2 = 0x8000

// splitRightDelta arma el delta de la página derecha nueva: su espacio especial y todos sus ítems, en orden; form2
// marca una hoja de la forma 2.
func splitRightDelta(o page.Opaque, items [][]byte, form2 bool) []byte {
	count := uint16(len(items))
	if form2 {
		count |= splitRightForm2
	}
	b := le.AppendUint32(nil, uint32(o.Prev))
	b = le.AppendUint32(b, uint32(o.Next))
	b = le.AppendUint32(b, o.Level)
	b = le.AppendUint16(b, o.Flags)
	b = le.AppendUint16(b, count)
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
	form2  bool // hoja de la forma 2
}

// newRootData es el delta de la raíz nueva, ya leído.
type newRootData struct {
	level      uint32
	left       page.BlockNumber
	rightPivot []byte
}

// leafWriteForm2 es el bit 15 del número de ítem del delta de KindLeafWrite: la entrada es de la forma 2, con el id
// entero (formato 2; sin él, la entrada es de la forma 1 y va en una hoja de la forma 1).
const leafWriteForm2 = 0x8000

// writeFlags son las marcas que puede traer la entrada de la forma 2 de una escritura.
const writeFlags = EntryTombstone | EntryOverflow | EntryCompressed

// leafWriteDelta arma el delta de KindLeafWrite de una hoja de la forma 2: el número con leafWriteForm2 y la entrada.
func leafWriteDelta(off page.OffsetNumber, item []byte) []byte {
	return append(le.AppendUint16(nil, uint16(off)|leafWriteForm2), item...)
}

/**
* parseLeafWrite lee el delta de KindLeafWrite: el número del ítem, la entrada y su forma. La de la forma 2 lleva el
* id entero (el flujo lógico la lee sin la página) y solo marcas de escritura.
* @param delta []byte
* @return page.OffsetNumber, entry: la entrada del delta (en la forma 2, con pre vacío y el id entero), error:
* ErrBadRecord
**/
func parseLeafWrite(delta []byte) (page.OffsetNumber, entry, error) {
	if len(delta) <= offsetDeltaSize || le.Uint16(delta)&^leafWriteForm2 == 0 {
		return 0, entry{}, badRecord(wal.KindLeafWrite, "bad item delta")
	}
	raw, item := le.Uint16(delta), delta[offsetDeltaSize:]
	var e entry
	var err error
	if raw&leafWriteForm2 != 0 {
		e, err = parseEntry2(item, nil)
		if err == nil && (e.flags()&^writeFlags != 0 || e.hdrLen() != entry2Header) {
			err = corrupt("leaf write with flags %#x", e.flags())
		}
	} else {
		e, err = parseEntry(item)
	}
	if err != nil {
		return 0, entry{}, badRecord(wal.KindLeafWrite, err.Error())
	}
	return page.OffsetNumber(raw &^ leafWriteForm2), e, nil
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
	raw, rest := le.Uint16(delta[14:]), delta[splitRightHeaderSize:]
	count := int(raw &^ splitRightForm2)
	d.form2 = raw&splitRightForm2 != 0
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
	if d.form2 && !rightPrefixOk(d) {
		return splitRight{}, badRecord(wal.KindSplit, "bad right leaf of form 2")
	}
	return d, nil
}

/**
* rightPrefixOk dice si la derecha nueva de la forma 2 es una hoja con su ítem de prefijo en su lugar (después de la
* clave alta, si tiene) y entradas de la forma 2 sanas.
* @param d splitRight
* @return bool
**/
func rightPrefixOk(d splitRight) bool {
	at := 0
	if d.opaque.Next != 0 {
		at = 1
	}
	if d.opaque.Flags&page.OpaqueLeaf == 0 || d.opaque.Level != 0 || len(d.items) <= at {
		return false
	}
	v, err := parsePivot(d.items[at])
	if err != nil || v.downlink() != 0 || le.Uint16(v[offPivotFlags:]) != 0 || len(v)-PivotHeaderSize > MaxPrefixLen {
		return false
	}
	for _, it := range d.items[at+1:] { // sus entradas, de la forma 2 con ese prefijo
		if _, err := parseEntry2(it, v[PivotHeaderSize:]); err != nil {
			return false
		}
	}
	return true
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
	case kind == wal.KindLeafRewrite:
		if len(delta) != rewriteDeltaSize || le.Uint16(delta) > MaxPrefixLen {
			err = badRecord(kind, "bad rewrite delta")
		}
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
		} else if i == roleLeft && le.Uint32(delta) == 0 {
			// La borrada nunca es la de más a la derecha: su izquierda no puede quedar sin hermana derecha (con una
			// hoja de la forma 2 sería además leer su clave alta como ítem de prefijo)
			err = badRecord(kind, "left sibling without a right link")
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
	case kind == wal.KindInitStore && i == 0 && len(delta) == initMetaFlagsSize:
		if MetaFlags(le.Uint32(delta[initMetaDeltaSize:]))&^knownMetaFlags != 0 {
			return badRecord(kind, "unknown meta flags")
		}
		want = initMetaFlagsSize
	case kind == wal.KindInitStore && i == 0, kind == wal.KindSplit:
		want = blockDeltaSize
	case kind == wal.KindInitStore && i == 1 && len(delta) > 0:
		if len(delta) != layoutDeltaSize || le.Uint16(delta) != page.LayoutVersion2 {
			return badRecord(kind, "bad root layout")
		}
		return nil
	case kind == wal.KindSetFrozen:
		if len(delta) != frozenDeltaSize && (len(delta) != frozenFlagsDeltaSize ||
			MetaFlags(le.Uint32(delta[frozenDeltaSize:]))&^knownMetaFlags != 0) {
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
