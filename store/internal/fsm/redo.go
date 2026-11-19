package fsm

import (
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* Redo rehace un registro de fsm (KindFreePush, KindFreePop, KindVMSet, KindVMClear) durante la recuperación. Por
* cada bloque: si el fork es más corto, lo extiende; con imagen, la restaura sin condiciones y aplica el delta; con
* BlockWillInit, inicia la página entera; si no, aplica el delta solo si pd_lsn < lsn del registro. Así es
* idempotente: rehacer dos veces el mismo WAL deja las mismas páginas. Es una función del paquete (no de Manager)
* porque corre dentro de wal.Open, antes de que exista el WAL abierto.
* @param pool buffer.Pool, sm smgr.Manager: la caché y los archivos de datos del motor en recuperación
* @param rec *wal.Record: el registro leído (válido solo durante la llamada)
* @return error: ErrBadRecord si el registro no tiene la forma de su Kind, o el error de la caché
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

/**
* redoBlock rehace el bloque i de rec (ver Redo).
* @param pool buffer.Pool, sm smgr.Manager, rec *wal.Record, i int
* @return error
**/
func redoBlock(pool buffer.Pool, sm smgr.Manager, rec *wal.Record, i int) error {
	b := &rec.Blocks[i]
	if err := ensureBlock(pool, sm, b.File, b.Fork, b.Block); err != nil {
		return err
	}
	h, err := pinForRedo(pool, b)
	if err != nil {
		return err
	}
	defer release(h)

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
	if err := apply(rec.Kind, i, p, b.Delta); err != nil {
		return blockErr(err, b.File, b.Block)
	}
	setLSN(h, p, rec.LSN)
	return nil
}

// setLSN pone pd_lsn y marca la página sucia con ese lsn.
func setLSN(h buffer.Handle, p page.Page, lsn page.LSN) {
	p.SetLSN(lsn)
	h.MarkDirty(lsn)
}

// pinForRedo fija y bloquea la página de un bloque del registro. Si el registro la pisa entera (imagen o
// BlockWillInit) no la lee: la página en disco puede estar rota por un corte y su suma fallaría.
func pinForRedo(pool buffer.Pool, b *wal.BlockRef) (buffer.Handle, error) {
	if b.Flags&(wal.BlockHasImage|wal.BlockWillInit) != 0 {
		return pool.PinZero(buffer.Tag{File: b.File, Fork: b.Fork, Block: b.Block})
	}
	return pinLocked(pool, b.File, b.Fork, b.Block)
}
