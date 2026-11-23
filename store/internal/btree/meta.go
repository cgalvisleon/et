package btree

import (
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Posiciones de los campos de la página meta (doc.go). La magia, el bloque y la cabeza de la lista de libres los
// define page (page/meta.go), compartidos con fsm.
const (
	offMetaMagic     = page.MetaOffMagic
	offMetaVersion   = page.HeaderSize + 4
	offMetaRoot      = page.HeaderSize + 8
	offMetaLevel     = page.HeaderSize + 12
	offMetaFastRoot  = page.HeaderSize + 16
	offMetaFastLevel = page.HeaderSize + 20
	offMetaFreeHead  = page.MetaOffFreeHead
	offMetaDeleted   = page.HeaderSize + 28
	offMetaFrozen    = page.HeaderSize + 32
	offMetaFlags     = page.HeaderSize + 40

	// metaVersion es la versión del formato de la meta.
	metaVersion = 1
)

// MetaFlags son las marcas de la meta (formato 2; en una meta anterior, el campo reservado en cero: ninguna).
type MetaFlags uint32

const (
	// MetaLeavesForm2: todas las hojas del store son de la forma 2 (LayoutVersion 2). La pone Create en un store
	// nuevo y la limpieza al terminar de reescribir las hojas de la forma 1 de un store del formato 1 (migración
	// perezosa, doc.go «Hojas de la forma 2»).
	MetaLeavesForm2 MetaFlags = 1 << 0

	knownMetaFlags = MetaLeavesForm2
)

// meta es lo que el árbol lee de la página meta.
type meta struct {
	root      page.BlockNumber
	level     uint32
	fastRoot  page.BlockNumber // la raíz rápida: la página de más a la izquierda de fastLevel (btm_fastroot)
	fastLevel uint32
	deleted   page.BlockNumber // la cabeza de la pila de páginas borradas que esperan reciclarse (0 = vacía)
	freeHead  page.BlockNumber // la cabeza de la lista de libres (de fsm)
	flags     MetaFlags
}

/**
* readMetaPage lee la raíz, la raíz rápida y las pilas de la página meta, comprobando que lo sea (iniciada, magia, versión, page.OpaqueMeta).
* @param p page.Page
* @return meta, error: ErrCorrupt
**/
func readMetaPage(p page.Page) (meta, error) {
	if p.IsNew() || le.Uint32(p[offMetaMagic:]) != MetaMagic || le.Uint32(p[offMetaVersion:]) != metaVersion ||
		p.LayoutVersion() != page.LayoutVersion {
		return meta{}, corruptAt(MetaBlock, "not a meta page")
	}
	if o, err := p.Opaque(); err != nil || o.Flags&page.OpaqueMeta == 0 {
		return meta{}, corruptAt(MetaBlock, "meta page without OpaqueMeta")
	}
	m := meta{
		root:      page.BlockNumber(le.Uint32(p[offMetaRoot:])),
		level:     le.Uint32(p[offMetaLevel:]),
		fastRoot:  page.BlockNumber(le.Uint32(p[offMetaFastRoot:])),
		fastLevel: le.Uint32(p[offMetaFastLevel:]),
		deleted:   page.BlockNumber(le.Uint32(p[offMetaDeleted:])),
		freeHead:  page.BlockNumber(le.Uint32(p[offMetaFreeHead:])),
		flags:     MetaFlags(le.Uint32(p[offMetaFlags:])),
	}
	if m.root == MetaBlock || m.fastRoot == MetaBlock {
		return meta{}, corruptAt(MetaBlock, "root or fast root is the meta page")
	}
	if m.flags&^knownMetaFlags != 0 {
		return meta{}, corruptAt(MetaBlock, "unknown meta flags %#x", m.flags)
	}
	return m, nil
}

/**
* setMetaFast anota la raíz rápida: la página de más a la izquierda de su nivel por la que pueden empezar las bajadas
* (la única de su nivel, salvo una partición todavía sin terminar).
* @param p page.Page, fast page.BlockNumber, level uint32
**/
func setMetaFast(p page.Page, fast page.BlockNumber, level uint32) {
	le.PutUint32(p[offMetaFastRoot:], uint32(fast))
	le.PutUint32(p[offMetaFastLevel:], level)
}

/**
* initMetaPage deja p como la meta de un store nuevo: raíz root de nivel 0, lista de libres vacía, su frozenXID y sus
* marcas. pd_lower cubre el contenido para que la imagen de página entera del WAL no lo deje fuera.
* @param p page.Page, root page.BlockNumber, frozen page.XID, flags MetaFlags
* @return error
**/
func initMetaPage(p page.Page, root page.BlockNumber, frozen page.XID, flags MetaFlags) error {
	if err := p.Init(page.OpaqueSize); err != nil {
		return err
	}
	if err := p.SetOpaque(page.Opaque{Flags: page.OpaqueMeta}); err != nil {
		return err
	}
	le.PutUint32(p[offMetaMagic:], MetaMagic)
	le.PutUint32(p[offMetaVersion:], metaVersion)
	setMetaRoot(p, root, 0)
	le.PutUint64(p[offMetaFrozen:], uint64(frozen))
	le.PutUint32(p[offMetaFlags:], uint32(flags))
	return p.SetLower(page.HeaderSize + MetaSize)
}

/**
* applyMetaFrozen anota el frozenXID nuevo en la meta (KindSetFrozen): solo sube; con flags(4) al final del delta
* (formato 2), además suma esas marcas a las de la meta.
* @param p page.Page, delta []byte: frozenXID(8) [flags(4)]
* @return error
**/
func applyMetaFrozen(p page.Page, delta []byte) error {
	m, err := readMetaPage(p)
	if err != nil {
		return err
	}
	if frozen := page.XID(le.Uint64(delta)); frozen > metaFrozen(p) {
		le.PutUint64(p[offMetaFrozen:], uint64(frozen))
	}
	if len(delta) == frozenFlagsDeltaSize {
		le.PutUint32(p[offMetaFlags:], uint32(m.flags|MetaFlags(le.Uint32(delta[frozenDeltaSize:]))))
	}
	return nil
}

/**
* LeavesForm2 dice si todas las hojas del store son de la forma 2 (MetaLeavesForm2): si no, quedan hojas del
* formato 1 que la limpieza todavía no reescribió.
* @return bool, error
**/
func (t *BTree) LeavesForm2() (bool, error) {
	m, err := t.readMeta()
	return m.flags&MetaLeavesForm2 != 0, err
}

/**
* setMetaFlags suma marcas a la meta con un KindSetFrozen (frozenXID sin cambiar + flags) sobre la meta bloqueada en
* exclusivo. Si ya las tiene, no escribe nada.
* @param flags MetaFlags
* @return error
**/
func (t *BTree) setMetaFlags(flags MetaFlags) error {
	h, err := t.pool.Pin(t.tag(MetaBlock), buffer.Normal)
	if err != nil {
		return err
	}
	h.Lock(buffer.Exclusive)
	m, err := readMetaPage(h.Page())
	if err != nil || m.flags&flags == flags {
		h.Release()
		return err
	}
	delta := le.AppendUint32(le.AppendUint64(nil, uint64(metaFrozen(h.Page()))), uint32(flags))
	res, err := t.logChange(wal.KindSetFrozen, 0, 0, []wal.BlockRef{{Delta: delta}}, []buffer.Handle{h}, nil)
	h.Release()
	if err != nil {
		return err
	}
	return res.Write()
}

/**
* metaFrozen lee el frozenXID de una meta ya validada.
* @param p page.Page
* @return page.XID
**/
func metaFrozen(p page.Page) page.XID {
	return page.XID(le.Uint64(p[offMetaFrozen:]))
}

/**
* FrozenXID lee el frozenXID del store: ninguna versión con un xid menor queda sin congelar (fase 2, «Congelado» en
* store/doc.go).
* @return page.XID, error
**/
func (t *BTree) FrozenXID() (page.XID, error) {
	h, err := t.pool.Pin(t.tag(MetaBlock), buffer.Normal)
	if err != nil {
		return 0, err
	}
	defer h.Release()
	h.Lock(buffer.Share)
	if _, err := readMetaPage(h.Page()); err != nil {
		return 0, err
	}
	return metaFrozen(h.Page()), nil
}

/**
* SetFrozenXID sube el frozenXID del store con un KindSetFrozen sobre la meta bloqueada en exclusivo. Si no sube, no
* escribe nada. Quien llama garantiza que toda versión con xid < frozen ya está congelada o quitada.
* @param frozen page.XID
* @return error
**/
func (t *BTree) SetFrozenXID(frozen page.XID) error {
	h, err := t.pool.Pin(t.tag(MetaBlock), buffer.Normal)
	if err != nil {
		return err
	}
	h.Lock(buffer.Exclusive)
	if _, err := readMetaPage(h.Page()); err != nil || frozen <= metaFrozen(h.Page()) {
		h.Release()
		return err
	}
	refs := []wal.BlockRef{{Delta: le.AppendUint64(nil, uint64(frozen))}}
	res, err := t.logChange(wal.KindSetFrozen, 0, 0, refs, []buffer.Handle{h}, nil)
	h.Release()
	if err != nil {
		return err
	}
	return res.Write()
}

/**
* setMetaRoot anota la raíz y su nivel, y la raíz rápida en la misma (una raíz nueva es la única página de su nivel,
* como en _bt_newroot).
* @param p page.Page, root page.BlockNumber, level uint32
**/
func setMetaRoot(p page.Page, root page.BlockNumber, level uint32) {
	le.PutUint32(p[offMetaRoot:], uint32(root))
	le.PutUint32(p[offMetaLevel:], level)
	setMetaFast(p, root, level)
}

/**
* RecordFrozenXID da el frozenXID que un registro de btree pone en una página meta (KindInitStore con frozenXID,
* KindSetFrozen). La recuperación lo usa para que el siguiente xid no quede por debajo: un corte puede perder los xid
* asignados después del registro, y un xid reutilizado menor que el frozenXID de un store rompería su promesa.
* @param rec *wal.Record
* @return page.XID, bool: si el registro pone uno
**/
func RecordFrozenXID(rec *wal.Record) (page.XID, bool) {
	if len(rec.Blocks) == 0 {
		return 0, false
	}
	delta := rec.Blocks[0].Delta
	switch {
	case rec.Kind == wal.KindSetFrozen && len(delta) >= 8:
		return page.XID(le.Uint64(delta)), true
	case rec.Kind == wal.KindInitStore && len(delta) >= 12:
		_, frozen, _ := parseInitMeta(delta)
		return frozen, true
	}
	return 0, false
}

/**
* DecodeLeafWrite lee el cambio lógico de un KindLeafWrite (el decodificado lógico de la réplica): la versión que
* escribe, con su lsn, su xid, sus marcas, el valor en línea copiado o la referencia a desborde.
* @param rec *wal.Record: un KindLeafWrite leído del WAL
* @return Version, error: ErrBadRecord si no tiene la forma del formato
**/
func DecodeLeafWrite(rec *wal.Record) (Version, error) {
	if rec.Kind != wal.KindLeafWrite || len(rec.Blocks) != 1 || len(rec.Blocks[0].Delta) <= offsetDeltaSize {
		return Version{}, badRecord(rec.Kind, "not a leaf write")
	}
	_, e, err := parseLeafWrite(rec.Blocks[0].Delta)
	if err != nil {
		return Version{}, err
	}
	v := e.version()
	v.LSN = rec.LSN
	return v, nil
}
