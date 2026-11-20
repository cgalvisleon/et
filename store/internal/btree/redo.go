package btree

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* Redo rehace un registro de btree (KindInitStore, KindLeafWrite, KindSplit, KindInsertPivot, KindNewRoot,
* KindLeafVacuum, KindSetHints, KindSetFrozen, KindMarkHalfDead, KindUnlinkPage, KindRecyclePage) durante la
* recuperación. Por cada bloque: si el fork es más corto, lo extiende; con imagen, la restaura sin condiciones y
* aplica el delta; con BlockWillInit, inicia la página entera; si no, aplica el delta solo si pd_lsn < lsn del
* registro. Así es idempotente. Los bloques con imagen o BlockWillInit se fijan con buffer.Pool.PinZero: la página en
* disco puede estar rota por el corte. Es una función del paquete porque corre dentro de wal.Open.
* @param pool buffer.Pool, sm smgr.Manager: la caché y los archivos de datos del motor en recuperación
* @param rec *wal.Record: el registro leído (válido solo durante la llamada)
* @return error: ErrBadRecord si no tiene la forma de su Kind, o el error de la caché
**/
func Redo(pool buffer.Pool, sm smgr.Manager, rec *wal.Record) error {
	if err := checkShape(rec); err != nil {
		return err
	}
	for i := range rec.Blocks {
		if err := redoBlock(pool, sm, rec, i); err != nil {
			return err
		}
	}
	return nil
}

// blockRule dice qué exige un registro de un bloque suyo.
type blockRule struct {
	meta     bool // es la página meta (si no, no puede serlo)
	willInit bool // lleva BlockWillInit
}

// shapes son los bloques de cada Kind de btree, en orden. KindSplit admite el tercero (la antigua hermana derecha)
// o no: la izquierda puede ser la de más a la derecha; KindInsertPivot, el tercero (la meta con la raíz rápida) o no.
// KindUnlinkPage y KindRecyclePage tienen su propia validación (checkUnlinkShape, checkRecycleShape).
var shapes = map[wal.Kind][]blockRule{
	wal.KindInitStore:    {{meta: true, willInit: true}, {willInit: true}},
	wal.KindLeafWrite:    {{}},
	wal.KindSplit:        {{}, {willInit: true}, {}},
	wal.KindInsertPivot:  {{}, {}, {meta: true}},
	wal.KindNewRoot:      {{willInit: true}, {meta: true}, {}},
	wal.KindLeafVacuum:   {{}},
	wal.KindSetHints:     {{}},
	wal.KindSetFrozen:    {{meta: true}},
	wal.KindMarkHalfDead: {{}, {}},
}

// optionalLast son los Kind cuyo último bloque de shapes puede faltar.
var optionalLast = map[wal.Kind]bool{wal.KindSplit: true, wal.KindInsertPivot: true}

/**
* checkShape valida la forma de un registro antes de tocar páginas: cantidad de bloques, fork principal de un mismo
* store, la meta donde va, BlockWillInit donde se exige y cada delta (checkDelta).
* @param rec *wal.Record
* @return error: ErrBadRecord
**/
func checkShape(rec *wal.Record) error {
	switch rec.Kind {
	case wal.KindUnlinkPage:
		return checkBlocksOfOneStore(rec, checkUnlinkShape)
	case wal.KindRecyclePage:
		return checkBlocksOfOneStore(rec, checkRecycleShape)
	}
	rules, ok := shapes[rec.Kind]
	if !ok {
		return badRecord(rec.Kind, "not a btree kind")
	}
	n := len(rec.Blocks)
	if n != len(rules) && !(optionalLast[rec.Kind] && n == len(rules)-1) {
		return badRecord(rec.Kind, fmt.Sprintf("has %d blocks", n))
	}
	for i := range rec.Blocks {
		b, rule := &rec.Blocks[i], rules[i]
		switch {
		case b.Fork != smgr.ForkMain || b.File != rec.Blocks[0].File || b.File == 0:
			return badRecord(rec.Kind, "blocks must be of one store's main fork")
		case (b.Block == MetaBlock) != rule.meta:
			return badRecord(rec.Kind, fmt.Sprintf("block %d is in the wrong place", b.Block))
		case rule.willInit && b.Flags&wal.BlockWillInit == 0:
			return badRecord(rec.Kind, fmt.Sprintf("block %d needs BlockWillInit", i))
		}
		if err := checkDelta(rec.Kind, i, b.Delta); err != nil {
			return err
		}
	}
	return nil
}

/**
* checkBlocksOfOneStore valida que todos los bloques sean del fork principal de un mismo store y después la forma
* propia del Kind.
* @param rec *wal.Record, check func(rec *wal.Record) error
* @return error: ErrBadRecord
**/
func checkBlocksOfOneStore(rec *wal.Record, check func(rec *wal.Record) error) error {
	for i := range rec.Blocks {
		b := &rec.Blocks[i]
		if b.Fork != smgr.ForkMain || b.File != rec.Blocks[0].File || b.File == 0 {
			return badRecord(rec.Kind, "blocks must be of one store's main fork")
		}
	}
	return check(rec)
}

/**
* redoBlock rehace el bloque i de rec (ver Redo).
* @param pool buffer.Pool, sm smgr.Manager, rec *wal.Record, i int
* @return error
**/
func redoBlock(pool buffer.Pool, sm smgr.Manager, rec *wal.Record, i int) error {
	b := &rec.Blocks[i]
	if err := ensureBlock(pool, sm, b.File, b.Block); err != nil {
		return err
	}
	h, err := pinForRedo(pool, b)
	if err != nil {
		return err
	}
	defer h.Release()

	p := h.Page()
	switch {
	case b.Flags&wal.BlockHasImage != 0:
		if err := b.RestoreImage(p); err != nil {
			return err
		}
	case b.Flags&wal.BlockWillInit != 0:
		// El delta inicia la página entera: no importa lo que había.
	case p.LSN() >= rec.LSN:
		return nil // el cambio ya estaba en la página
	}
	if err := applyBlock(rec.Kind, blockRole(rec.Kind, rec.Blocks, i), p, b.Delta, rec.LSN); err != nil {
		return fmt.Errorf("btree: redo lsn %d block %d: %w", rec.LSN, b.Block, err)
	}
	p.SetLSN(rec.LSN)
	h.MarkDirty(rec.LSN)
	return nil
}

/**
* ensureBlock extiende el fork principal hasta que block exista (creándolo si falta), con páginas en ceros: el
* archivo puede haber quedado más corto que el WAL.
* @param pool buffer.Pool, sm smgr.Manager, file smgr.FileNo, block page.BlockNumber
* @return error
**/
func ensureBlock(pool buffer.Pool, sm smgr.Manager, file smgr.FileNo, block page.BlockNumber) error {
	n, err := sm.NBlocks(file, smgr.ForkMain)
	if errors.Is(err, smgr.ErrNoFork) {
		n, err = 0, sm.Create(file, smgr.ForkMain)
	}
	if err != nil {
		return err
	}
	for ; n <= block; n++ {
		h, err := pool.PinNew(file, smgr.ForkMain)
		if err != nil {
			return err
		}
		h.Release()
	}
	return nil
}

/**
* pinForRedo fija y bloquea en exclusivo la página de un bloque. Si el registro la pisa entera (imagen o
* BlockWillInit) no la lee del disco (PinZero).
* @param pool buffer.Pool, b *wal.BlockRef
* @return buffer.Handle, error
**/
func pinForRedo(pool buffer.Pool, b *wal.BlockRef) (buffer.Handle, error) {
	tag := buffer.Tag{File: b.File, Fork: b.Fork, Block: b.Block}
	if b.Flags&(wal.BlockHasImage|wal.BlockWillInit) != 0 {
		return pool.PinZero(tag)
	}
	h, err := pool.Pin(tag, buffer.Normal)
	if err != nil {
		return nil, err
	}
	h.Lock(buffer.Exclusive)
	return h, nil
}
