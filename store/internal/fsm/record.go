package fsm

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// Largos de los deltas (formato en doc.go).
const (
	blockDeltaSize = 4 // un número de bloque
	vmDeltaSize    = 5 // bloque del fork principal (4) + bits (1)
)

// encodeBlock arma el delta de un número de bloque.
func encodeBlock(block page.BlockNumber) []byte {
	return le.AppendUint32(nil, uint32(block))
}

// encodeVM arma el delta de un cambio del mapa: el bloque del fork principal y los bits.
func encodeVM(block page.BlockNumber, bits VMBits) []byte {
	return append(le.AppendUint32(nil, uint32(block)), byte(bits))
}

/**
* badRecord arma el error de un registro con forma inválida.
* @param kind wal.Kind, why string
* @return error: envuelve ErrBadRecord
**/
func badRecord(kind wal.Kind, why string) error {
	return fmt.Errorf(MSG_BAD_RECORD, ErrBadRecord, kind, why)
}

/**
* checkShape valida la forma de un registro de fsm antes de tocar páginas: cantidad de bloques, forks, la meta en el
* bloque 0 y largos de los deltas.
* @param rec *wal.Record
* @return error: ErrBadRecord con el detalle
**/
func checkShape(rec *wal.Record) error {
	switch rec.Kind {
	case wal.KindFreePush, wal.KindFreePop:
		if len(rec.Blocks) != 2 {
			return badRecord(rec.Kind, "needs two blocks")
		}
		meta, other := rec.Blocks[0], rec.Blocks[1]
		if meta.Fork != smgr.ForkMain || meta.Block != metaBlock || len(meta.Delta) != blockDeltaSize {
			return badRecord(rec.Kind, "first block must be the meta page")
		}
		if other.Fork != smgr.ForkMain || other.File != meta.File || other.Block == metaBlock ||
			other.Flags&wal.BlockWillInit == 0 {
			return badRecord(rec.Kind, "second block must be a main-fork page of the same store with BlockWillInit")
		}
		if rec.Kind == wal.KindFreePush && len(other.Delta) != blockDeltaSize {
			return badRecord(rec.Kind, "free page delta must be a block number")
		}
		if rec.Kind == wal.KindFreePop && len(other.Delta) != 0 {
			return badRecord(rec.Kind, "taken page has no delta")
		}
	case wal.KindVMSet, wal.KindVMClear:
		if len(rec.Blocks) != 1 || rec.Blocks[0].Fork != smgr.ForkVM || len(rec.Blocks[0].Delta) != vmDeltaSize {
			return badRecord(rec.Kind, "needs one visibility map block")
		}
		block, bits := page.BlockNumber(le.Uint32(rec.Blocks[0].Delta)), VMBits(rec.Blocks[0].Delta[4])
		if vmBlockOf(block) != rec.Blocks[0].Block || bits == 0 || bits&^vmAllBits != 0 {
			return badRecord(rec.Kind, "visibility map delta does not match its block")
		}
	default:
		return badRecord(rec.Kind, "not a fsm kind")
	}
	return nil
}

/**
* apply aplica el delta del bloque i de un registro de fsm a su página, sin tocar pd_lsn. Es la misma función en el
* camino normal y al rehacer: así las dos llegan a la misma página. El registro ya pasó checkShape.
* @param kind wal.Kind, i int: el registro y el número de bloque dentro de él
* @param p page.Page: la página, bloqueada en exclusivo (restaurada de su imagen si la traía)
* @param delta []byte
* @return error: ErrNoMeta si la meta no es válida
**/
func apply(kind wal.Kind, i int, p page.Page, delta []byte) error {
	switch {
	case (kind == wal.KindFreePush || kind == wal.KindFreePop) && i == 0:
		if err := checkMeta(p); err != nil {
			return err
		}
		setFreeHead(p, page.BlockNumber(le.Uint32(delta)))
		return nil
	case kind == wal.KindFreePush:
		return initPage(p, page.Opaque{Next: page.BlockNumber(le.Uint32(delta)), Flags: page.OpaqueFree})
	case kind == wal.KindFreePop:
		return initPage(p, page.Opaque{})
	default: // KindVMSet, KindVMClear
		block, bits := page.BlockNumber(le.Uint32(delta)), VMBits(delta[4])
		if p.IsNew() {
			initVMPage(p)
		}
		cur := vmGet(p, block)
		if kind == wal.KindVMSet {
			vmPut(p, block, cur|bits)
		} else {
			vmPut(p, block, cur&^bits)
		}
		return nil
	}
}

/**
* initPage deja p como una página de store vacía (espacio especial de page.OpaqueSize) con el espacio especial o.
* @param p page.Page, o page.Opaque
* @return error: solo si p no mide page.Size
**/
func initPage(p page.Page, o page.Opaque) error {
	if err := p.Init(page.OpaqueSize); err != nil {
		return err
	}
	return p.SetOpaque(o)
}

/**
* logChange hace los pasos 6 a 8 del orden de un cambio (store/doc.go) con las páginas ya bloqueadas en exclusivo:
* arma el registro con las imágenes que hagan falta, lo reserva (repitiendo si un checkpoint empezó en medio y pide
* una imagen más), aplica los deltas, pone pd_lsn y marca sucias. Quien llama suelta las páginas y después escribe
* la reserva (paso 9). Las páginas ya se validaron: un error al aplicar tras reservar no debería pasar; si pasa, la
* reserva se escribe ahí mismo (toda reserva se escribe) y se devuelve solo el error.
* @param kind wal.Kind, xid page.XID
* @param blocks []wal.BlockRef: File, Fork, Block, Flags y Delta de cada bloque
* @param handles []buffer.Handle: las páginas, en el mismo orden
* @return wal.Reservation, error: con reserva no nula, quien llama debe llamar a Write
**/
func (m *Manager) logChange(kind wal.Kind, xid page.XID, blocks []wal.BlockRef, handles []buffer.Handle) (wal.Reservation, error) {
	rec := &wal.Record{XID: xid, Kind: kind, Blocks: blocks}
	for i := range blocks {
		blocks[i].PageLSN = handles[i].Page().LSN()
	}
	res, err := m.reserve(rec, handles)
	if err != nil {
		return nil, err
	}
	lsn := res.LSN()
	var applyErr error
	for i, h := range handles {
		p := h.Page()
		if err := apply(kind, i, p, blocks[i].Delta); err != nil {
			applyErr = errors.Join(applyErr, err)
			continue
		}
		setLSN(h, p, lsn)
	}
	if applyErr != nil {
		return nil, errors.Join(applyErr, res.Write())
	}
	return res, nil
}

/**
* reserve agrega la imagen de página entera a cada bloque cuyo pd_lsn es anterior al punto de rehacer vigente (salvo
* los que se inician enteros) y llama a wal.Log.Reserve; si un checkpoint empezó entre medio (ErrNeedImage), repite.
* @param rec *wal.Record, handles []buffer.Handle: las páginas antes del cambio
* @return wal.Reservation, error
**/
func (m *Manager) reserve(rec *wal.Record, handles []buffer.Handle) (wal.Reservation, error) {
	for {
		redo := m.log.RedoPoint().LSN
		for i := range rec.Blocks {
			b := &rec.Blocks[i]
			if b.Flags&(wal.BlockWillInit|wal.BlockHasImage) == 0 && b.PageLSN < redo {
				if err := b.SetImage(handles[i].Page()); err != nil {
					return nil, err
				}
			}
		}
		res, err := m.log.Reserve(rec, nil)
		if errors.Is(err, wal.ErrNeedImage) {
			continue
		}
		return res, err
	}
}
