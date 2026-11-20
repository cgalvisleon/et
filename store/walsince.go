package store

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// defaultWalBatch es lo que entrega WalReader.Next si se pide 0 o menos: 1 MiB de registros.
const defaultWalBatch = 1 << 20

// WalReader recorre el WAL de un motor desde un lsn (WalSince): lo que se envía a una réplica. Lee del disco, no de
// la memoria, de a tandas acotadas, y solo lo que ya es durable (una réplica nunca tiene algo que el primario puede
// perder en un corte). Guarda solo la posición siguiente: se puede tener abierto todo lo que dure la conexión.
type WalReader struct {
	e   *Engine
	mu  sync.Mutex
	pos wal.Position // dónde está el próximo registro; LSN es lo único que se sabe hasta resolverla
	set bool         // pos ya se resolvió en el WAL (Segment y Offset valen)
}

/**
* walSince abre un lector del WAL que empieza en el registro siguiente a lsn (lsn = lo que el otro lado ya tiene;
* ReplayLSN de una réplica). No retiene nada: para que el WAL no se recicle antes de leerlo, una ranura (CreateSlot,
* AdvanceSlot).
* @param lsn uint64
* @return *WalReader, error: ErrClosed, ErrFailed
**/
func (e *Engine) walSince(lsn uint64) (*WalReader, error) {
	if err := e.enterRead(); err != nil {
		return nil, err
	}
	defer e.leave()
	return &WalReader{e: e, pos: wal.Position{LSN: page.LSN(lsn + 1)}}, nil
}

/**
* next devuelve los registros siguientes ya durables, hasta maxBytes (al menos uno si hay); ninguno si todavía no
* hay nada nuevo. Lo que devuelve es del llamador.
* @param maxBytes int: ≤ 0 = defaultWalBatch
* @return []WalEntry, error: ErrWalRemoved si el WAL desde ahí ya se recicló, ErrClosed, ErrFailed
**/
func (r *WalReader) next(maxBytes int) ([]WalEntry, error) {
	e := r.e
	if err := e.enterRead(); err != nil {
		return nil, err
	}
	defer e.leave()
	r.mu.Lock()
	defer r.mu.Unlock()
	if maxBytes <= 0 {
		maxBytes = defaultWalBatch
	}
	stop := e.log.Durable()
	if r.pos.LSN > stop {
		return nil, nil
	}
	if !r.set {
		pos, err := e.log.Locate(r.pos.LSN)
		if err != nil {
			return nil, removedError(r.pos.LSN, err)
		}
		r.pos, r.set = pos, true
	}
	var out []WalEntry
	next, err := e.log.Read(r.pos, stop, maxBytes, func(at wal.Position, raw []byte, _ *wal.Record) error {
		out = append(out, WalEntry{LSN: uint64(at.LSN), Segment: at.Segment, Offset: at.Offset, Data: bytes.Clone(raw)})
		return nil
	})
	if err != nil {
		return nil, removedError(r.pos.LSN, err)
	}
	if len(out) == 0 { // había algo durable y no se leyó: el segmento se recicló mientras tanto
		return nil, fmt.Errorf(MSG_WAL_REMOVED, ErrWalRemoved, r.pos.LSN, wal.ErrRemoved)
	}
	r.pos = next
	return out, nil
}

/**
* removedError traduce el error de leer el WAL: un segmento que ya no está es ErrWalRemoved, y uno dañado ErrCorrupt.
* @param lsn page.LSN, err error
* @return error
**/
func removedError(lsn page.LSN, err error) error {
	if errors.Is(err, wal.ErrRemoved) {
		return fmt.Errorf(MSG_WAL_REMOVED, ErrWalRemoved, lsn, err)
	}
	return corruptError(err)
}

/**
* corruptError traduce el daño que encuentra el paquete wal (wal.ErrCorrupted: un segmento sellado cortado, un hueco)
* a ErrCorrupt, así errors.Is(err, ErrCorrupt) vale para todo daño del WAL (tarea 101).
* @param err error
* @return error
**/
func corruptError(err error) error {
	if errors.Is(err, wal.ErrCorrupted) && !errors.Is(err, ErrCorrupt) {
		return fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	return err
}
