package btree

import (
	"errors"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* logChange hace los pasos 6 a 8 del orden de un cambio (store/doc.go) con las páginas ya bloqueadas en exclusivo:
* arma el registro con las imágenes que hagan falta, lo reserva (repitiendo si un checkpoint empezó en medio), aplica
* los deltas, pone pd_lsn y marca sucias. Quien llama suelta las páginas y después escribe la reserva (paso 9): entre
* reservar y escribir no se espera a nada. Un error al aplicar tras reservar no debería pasar (las páginas ya se
* validaron); si pasa, la reserva se escribe ahí mismo (toda reserva se escribe) y se devuelve el error.
* @param kind wal.Kind, xid page.XID, flags wal.Flags
* @param refs []wal.BlockRef: Flags y Delta de cada bloque (File, Fork, Block y PageLSN se completan acá)
* @param hs []buffer.Handle: las páginas, en el mismo orden
* @param underLock func(lsn page.LSN): se pasa a wal.Log.Reserve
* @return wal.Reservation, error: con reserva no nula, quien llama debe llamar a Write
**/
func (t *BTree) logChange(kind wal.Kind, xid page.XID, flags wal.Flags, refs []wal.BlockRef, hs []buffer.Handle,
	underLock func(lsn page.LSN)) (wal.Reservation, error) {
	for i := range refs {
		refs[i].File, refs[i].Fork = t.file, smgr.ForkMain
		refs[i].Block, refs[i].PageLSN = hs[i].Tag().Block, hs[i].Page().LSN()
	}
	rec := &wal.Record{XID: xid, Kind: kind, Flags: flags, Blocks: refs}
	res, err := t.reserve(rec, hs, underLock)
	if err != nil {
		return nil, err
	}
	lsn := res.LSN()
	var applyErr error
	for i, h := range hs {
		p := h.Page()
		if err := applyBlock(kind, i, p, refs[i].Delta, lsn); err != nil {
			applyErr = errors.Join(applyErr, err)
			continue
		}
		p.SetLSN(lsn)
		h.MarkDirty(lsn)
	}
	if applyErr != nil {
		return nil, errors.Join(applyErr, res.Write())
	}
	return res, nil
}

/**
* reserve agrega la imagen de página entera (la página ANTES del cambio) a cada bloque cuyo pd_lsn es anterior al
* punto de rehacer vigente, salvo los que se inician enteros, y llama a wal.Log.Reserve; si un checkpoint empezó entre
* medio (ErrNeedImage), repite con las páginas todavía bloqueadas.
* @param rec *wal.Record, hs []buffer.Handle, underLock func(lsn page.LSN)
* @return wal.Reservation, error
**/
func (t *BTree) reserve(rec *wal.Record, hs []buffer.Handle, underLock func(lsn page.LSN)) (wal.Reservation, error) {
	for {
		redo := t.log.RedoPoint().LSN
		for i := range rec.Blocks {
			b := &rec.Blocks[i]
			if b.Flags&(wal.BlockWillInit|wal.BlockHasImage) == 0 && b.PageLSN < redo {
				if err := b.SetImage(hs[i].Page()); err != nil {
					return nil, err
				}
			}
		}
		res, err := t.log.Reserve(rec, underLock)
		if errors.Is(err, wal.ErrNeedImage) {
			continue
		}
		return res, err
	}
}
