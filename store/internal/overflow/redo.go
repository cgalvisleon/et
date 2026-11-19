package overflow

import (
	"errors"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* Redo rehace un KindOverflowWrite durante la recuperación: valida su forma, extiende el fork principal si el archivo
* quedó más corto que el WAL y pisa la página entera con su trozo (BlockWillInit: se fija con PinZero, sin leerla del
* disco, que puede estar rota por un corte). Es idempotente: rehacer dos veces deja la misma página. Es una función
* del paquete (no de Manager) porque corre dentro de wal.Open, antes de que exista el WAL abierto.
* @param pool buffer.Pool, sm smgr.Manager: la caché y los archivos de datos del motor en recuperación
* @param rec *wal.Record: el registro leído (válido solo durante la llamada)
* @return error: ErrBadRecord si el registro no tiene la forma de su Kind, o el error de la caché
**/
func Redo(pool buffer.Pool, sm smgr.Manager, rec *wal.Record) error {
	if err := checkShape(rec); err != nil {
		return err
	}
	b := &rec.Blocks[0]
	if err := ensureBlock(pool, sm, b.File, b.Block); err != nil {
		return err
	}
	h, err := pool.PinZero(buffer.Tag{File: b.File, Fork: smgr.ForkMain, Block: b.Block})
	if err != nil {
		return err
	}
	defer release(h)

	next, seq := decodeMain(rec.Main)
	p := h.Page()
	if err := apply(p, rec.XID, next, seq, b.Delta); err != nil {
		return err
	}
	p.SetLSN(rec.LSN)
	h.MarkDirty(rec.LSN)
	return nil
}

/**
* ensureBlock extiende el fork principal de file hasta que tenga block (con páginas en ceros), y lo crea si no
* existe: tras un corte el archivo puede ser más corto que lo que dice el WAL.
* @param pool buffer.Pool, sm smgr.Manager, file smgr.FileNo, block page.BlockNumber
* @return error
**/
func ensureBlock(pool buffer.Pool, sm smgr.Manager, file smgr.FileNo, block page.BlockNumber) error {
	n, err := sm.NBlocks(file, smgr.ForkMain)
	if errors.Is(err, smgr.ErrNoFork) {
		err = sm.Create(file, smgr.ForkMain)
	}
	if err != nil {
		return err
	}
	for ; n <= block; n++ {
		h, err := pool.PinNew(file, smgr.ForkMain)
		if err != nil {
			return err
		}
		release(h)
	}
	return nil
}
