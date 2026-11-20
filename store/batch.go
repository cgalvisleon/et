package store

import (
	"bytes"
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// batchState es el momento de la vida de un lote.
type batchState int

const (
	batchActive  batchState = iota // admite operaciones
	batchAborted                   // abortó solo (error de serialización, conflicto, tiempo, motor cerrado): espera Rollback
	batchDone                      // confirmado o deshecho
)

// Batch es un lote: una transacción con aislamiento por snapshot (Repeatable Read) que puede abarcar varios stores
// del motor. Ve la base como al crearlo más sus propios cambios, y Commit es todo o nada. Sus métodos se pueden
// llamar desde varias goroutines, pero corren de a uno.
//
// Sus versiones van al árbol desde la primera escritura, como las de cualquier transacción en curso (nadie más las
// ve hasta que confirma): en RAM solo guarda su snapshot y la suma de Count por store.
type Batch struct {
	e    *Engine
	xid  page.XID
	snap xact.Snapshot

	deadline time.Time   // cero = sin tiempo máximo
	timer    *time.Timer // aborta el lote al pasar deadline (nil sin tiempo máximo)

	// mu serializa las operaciones del lote, Commit, Rollback y el aborto por tiempo.
	mu     sync.Mutex
	state  batchState
	err    error                 // por qué abortó solo (con batchAborted)
	wrote  bool                  // escribió al menos una versión o hizo DDL: Commit y Rollback dejan registro en el WAL
	deltas map[smgr.FileNo]int64 // lo que suma a Count de cada store (por número de archivo)

	// DDL en lote (ddlbatch.go): los stores que creó (pendientes hasta Commit) y los que borra al confirmar, y los
	// stores en los que intentó escribir (cuentan en Store.batchWriters hasta que el lote termina).
	creates []*Store
	drops   []*Store
	written map[*Store]struct{}
}

/**
* newBatch empieza un lote: su xid, su snapshot y su tiempo máximo (TRANSACTION_TIMEOUT). Si el motor está cerrado
* o inutilizable, el lote nace abortado con ese error.
* @return *Batch
**/
func (e *Engine) newBatch() *Batch {
	b := &Batch{e: e, deltas: map[smgr.FileNo]int64{}, written: map[*Store]struct{}{}}
	if err := e.enter(); err != nil {
		b.state, b.err = batchAborted, err
		return b
	}
	defer e.leave()
	e.openTxn(&e.metrics.openBatches)
	b.xid = e.txs.Begin()
	b.snap = e.txs.Snapshot(b.xid)
	if e.cfg.txTimeout > 0 {
		b.mu.Lock() // el temporizador puede disparar antes de que se guarde
		b.deadline = time.Now().Add(e.cfg.txTimeout)
		b.timer = e.timers.after(e.cfg.txTimeout, b.expire)
		b.mu.Unlock()
	}
	return b
}

/**
* expire es el aborto por tiempo: si el lote sigue activo al pasar TRANSACTION_TIMEOUT, lo aborta con ErrTimeout
* (así no retiene los ids que escribió ni el horizonte de la limpieza).
**/
func (b *Batch) expire() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == batchActive {
		b.abortLocked(ErrTimeout)
	}
}

/**
* acquire toma el lote para una operación sobre un store de e: comprueba que sea del mismo motor, que siga activo y
* que no haya pasado su tiempo. Quien lo toma llama a b.mu.Unlock al terminar.
* @param e *Engine: el motor del store
* @return error: ErrOtherEngine, ErrTxDone, ErrTimeout o por qué abortó
**/
func (b *Batch) acquire(e *Engine) error {
	if b.e != e {
		return ErrOtherEngine
	}
	b.mu.Lock()
	if err := b.usableLocked(); err != nil {
		b.mu.Unlock()
		return err
	}
	return nil
}

/**
* usableLocked dice si el lote admite operaciones; si pasó su tiempo, lo aborta. Con b.mu tomado.
* @return error
**/
func (b *Batch) usableLocked() error {
	switch b.state {
	case batchDone:
		return ErrTxDone
	case batchAborted:
		return b.err
	}
	if b.expired() {
		b.abortLocked(ErrTimeout)
		return ErrTimeout
	}
	return nil
}

/**
* expired dice si el lote pasó su tiempo máximo.
* @return bool
**/
func (b *Batch) expired() bool {
	return !b.deadline.IsZero() && !time.Now().Before(b.deadline)
}

/**
* waitLimit es cuánto puede esperar una escritura del lote a otra transacción: LOCK_TIMEOUT, sin pasar el tiempo
* máximo del lote (≤ 0 = sin límite, como xact.Manager.Wait).
* @return time.Duration
**/
func (b *Batch) waitLimit() time.Duration {
	limit := b.e.cfg.lockTimeout
	if b.deadline.IsZero() {
		return limit
	}
	left := max(time.Until(b.deadline), time.Nanosecond)
	if limit <= 0 {
		return left
	}
	return min(limit, left)
}

/**
* commit confirma el lote (paso 10 del orden de un cambio): un KindCommit con lo que suma a Count de cada store
* (sumado a los contadores bajo writeMu), un solo fsync con SYNC_ON_WRITE salvo diferido, confirmada en el archivo
* de estados y End. Un lote abortado solo se deshace y devuelve por qué abortó.
* @param deferred bool: confirmar sin esperar el fsync (CommitDeferred)
* @return error: ErrTxDone, ErrSerialization, ErrConflict, ErrTimeout, ErrClosed o ErrFailed
**/
func (b *Batch) commit(deferred bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err := b.usableLocked(); err != nil {
		if b.state == batchAborted {
			b.state = batchDone
		}
		return err
	}
	return b.finishLocked(true, deferred)
}

/**
* rollback deshace el lote: un KindAbort si escribió, abortada en el archivo de estados y End. Deshacer un lote que
* abortó solo no hace nada más; deshacer uno terminado devuelve ErrTxDone.
* @return error
**/
func (b *Batch) rollback() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case batchDone:
		return ErrTxDone
	case batchAborted:
		b.state = batchDone
		return nil
	}
	return b.finishLocked(false, false)
}

/**
* abortLocked aborta el lote por err (queda esperando Rollback): sus versiones dejan de contar y los que esperan a
* su xid siguen. Si el lote ya no está activo no hace nada. Con b.mu tomado.
* @param err error: por qué
**/
func (b *Batch) abortLocked(err error) {
	if b.state != batchActive {
		return // ya abortó o terminó (p. ej. su temporizador durante una espera): queda el primer motivo
	}
	if ferr := b.finishLocked(false, false); ferr != nil && !errors.Is(ferr, ErrClosed) {
		err = errors.Join(err, ferr)
	}
	b.state, b.err = batchAborted, err
}

/**
* finishLocked termina la transacción del lote: confirma (commit) o aborta. Un lote sin escrituras no escribe nada
* en el WAL (su xid queda «en curso» en el archivo de estados y se lee abortado tras reabrir). Con b.mu tomado.
* @param commit bool, deferred bool: confirmar sin esperar el fsync
* @return error
**/
func (b *Batch) finishLocked(commit, deferred bool) error {
	b.state = batchDone
	b.e.timers.stop(b.timer)
	defer b.e.closeTxn(&b.e.metrics.openBatches)
	defer b.e.txs.End(b.xid) // después de resolver el DDL: quien espera al lote ya ve el catálogo final
	defer b.forgetWrites()
	if !b.wrote {
		return nil
	}
	if err := b.e.enter(); err != nil {
		return err // DDL sin resolver: el motor está cerrado o inutilizable y la recuperación lo deshace
	}
	defer b.e.leave()
	if b.hasDDL() {
		return b.finishDDL(commit)
	}
	if err := b.logEnd(commit, modeOf(deferred)); err != nil {
		return b.e.fail(err)
	}
	return nil
}

/**
* logEnd escribe el registro de confirmación o aborto y anota el estado, con commitMu en lectura desde antes de
* reservar hasta anotarlo (el checkpoint no puede dejar el registro antes de su punto de rehacer y el estado fuera
* del archivo de estados que guarda). Un aborto nunca espera el fsync: sin registro durable, cuenta como abortado.
* @param commit bool, mode syncMode: cuánto espera el fsync de la confirmación
* @return error
**/
func (b *Batch) logEnd(commit bool, mode syncMode) error {
	e := b.e
	start := time.Now()
	e.commitMu.RLock()
	defer e.commitMu.RUnlock()
	rec := &wal.Record{XID: b.xid, Kind: wal.KindAbort}
	var underLock func(page.LSN)
	status := xact.StatusAborted
	if commit {
		rec.Kind, rec.Main, status = wal.KindCommit, encodeCommit(time.Now().UnixNano(), b.commitInfo()), xact.StatusCommitted
		underLock = func(page.LSN) {
			for file, delta := range b.deltas {
				e.counts.add(file, delta)
			}
		}
	}
	res, err := e.log.Reserve(rec, underLock)
	if err != nil {
		return err
	}
	lsn := res.LSN()
	if err := res.Write(); err != nil {
		return err
	}
	if commit {
		if err := e.syncCommit(lsn, mode); err != nil {
			return err
		}
	}
	if err := e.status.SetStatus(b.xid, status, lsn); err != nil {
		return err
	}
	if commit {
		e.observeCommit(start)
	}
	return nil
}

/**
* txError dice si err aborta el lote (error de serialización, conflicto, tiempo, freno o disco lleno sin tocar el
* motor).
* @param err error
* @return bool
**/
func txError(err error) bool {
	return errors.Is(err, ErrSerialization) || errors.Is(err, ErrConflict) || errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrThrottled) || isCleanDiskFull(err)
}

/**
* writeIn hace una escritura dentro del lote b: como la suelta, pero con el xid y el snapshot del lote y sin
* confirmar (Commit = false); lo que suma a Count queda en el lote hasta Commit. Si otra transacción en curso tiene el
* id, la espera sin b.mu, Store.mu, closeMu ni páginas (waitOther) y repite. Un error de serialización, conflicto,
* tiempo o disco lleno aborta el lote. Si no escribió, la cadena de desborde que había preparado se libera.
* @param b *Batch, id string, c change
* @return bool: si escribió, error
**/
func (s *Store) writeIn(b *Batch, id string, c change) (bool, error) {
	if err := checkWrite(id, c.data); err != nil {
		return false, err
	}
	if err := s.e.throttle(); err != nil {
		return false, err
	}
	if err := b.acquire(s.e); err != nil {
		return false, err
	}
	defer b.mu.Unlock()
	for {
		applied, wait, err := s.writeInRound(b, id, c)
		if wait != 0 {
			if err = b.waitOther(wait); err == nil {
				continue
			}
		}
		if txError(err) {
			b.abortLocked(err)
		}
		return applied, err
	}
}

/**
* writeInRound es una vuelta de writeIn con el lote tomado: entra al store, prepara la versión, la escribe y libera
* el desborde sin usar (también si hay que esperar: el store puede vaciarse o borrarse durante la espera).
* @param b *Batch, id string, c change
* @return bool: si escribió, page.XID: la transacción a esperar (0 = ninguna), error
**/
func (s *Store) writeInRound(b *Batch, id string, c change) (bool, page.XID, error) {
	if err := s.beginWrite(); err != nil {
		return false, 0, err
	}
	defer s.end()
	if err := s.pendingFor(b.xid); err != nil {
		return false, 0, err
	}
	b.noteWrite(s)
	if err := s.e.admit(); err != nil {
		return false, 0, err
	}
	w, err := s.prepare(b.xid, id, c)
	if err != nil {
		return false, 0, s.e.writeError(err)
	}
	w.Commit, w.UnderLock, w.Snapshot = false, nil, &b.snap
	applied, wait, err := s.batchLeaf(b, id, w)
	unused := wait != 0 || err == nil || txError(err)
	if !applied && unused && w.Overflow != nil {
		if ferr := s.e.chains.Free(s.file, *w.Overflow, b.xid); ferr != nil {
			return false, 0, s.e.fail(ferr)
		}
	}
	return applied, wait, err
}

/**
* batchLeaf intenta una vez escribir la versión del lote. No toma commitMu: las versiones de un lote no confirman
* nada hasta su KindCommit.
* @param b *Batch, id string, w btree.Write
* @return bool: si escribió, page.XID: la transacción en curso que tiene el id (0 = ninguna), error:
* ErrSerialization, ErrDiskFull o ErrFailed con la causa
**/
func (s *Store) batchLeaf(b *Batch, id string, w btree.Write) (bool, page.XID, error) {
	res, err := s.tree.Write(id, w)
	var wait *btree.WaitError
	switch {
	case errors.As(err, &wait):
		return false, wait.XID, nil
	case errors.Is(err, btree.ErrSerialization):
		return false, 0, ErrSerialization
	case err != nil:
		return false, 0, s.e.writeError(err)
	}
	if res.Applied {
		s.e.writes.Add(1)
		b.wrote = true
		b.deltas[s.file] += int64(res.CountDelta)
	}
	return res.Applied, 0, nil
}

/**
* waitOther espera a la transacción xid (hasta LOCK_TIMEOUT sin pasar el tiempo del lote) con b.mu suelto, como
* XactLockTableWait de PostgreSQL: así el temporizador del lote, Rollback y las otras operaciones del lote no quedan
* detrás de la espera. Al volver a tomar b.mu, el lote pudo haber terminado o abortado. Con b.mu tomado (al entrar y
* al salir) y sin Store.mu, closeMu ni páginas.
* @param xid page.XID
* @return error: nil (repetir), ErrConflict, ErrTimeout, por qué terminó el lote o el fallo del motor
**/
func (b *Batch) waitOther(xid page.XID) error {
	limit := b.waitLimit()
	b.mu.Unlock()
	werr := b.e.txs.Wait(xid, limit)
	b.mu.Lock()
	if err := b.usableLocked(); err != nil {
		return err
	}
	if werr != nil {
		if b.expired() {
			return ErrTimeout
		}
		return ErrConflict
	}
	return b.e.failed()
}

/**
* readIn devuelve la versión de id que ve el lote: la propia si la escribió, si no la de su snapshot.
* @param b *Batch, id string
* @return current, bool, error
**/
func (s *Store) readIn(b *Batch, id string) (current, bool, error) {
	if err := checkID(id); err != nil {
		return current{}, false, err
	}
	if err := b.acquire(s.e); err != nil {
		return current{}, false, err
	}
	defer b.mu.Unlock()
	return s.readVersion(id, &b.snap)
}

/**
* putIn guarda id con data dentro del lote.
* @param b *Batch, id string, data []byte
* @return error
**/
func (s *Store) putIn(b *Batch, id string, data []byte) error {
	_, err := s.writeIn(b, id, change{data: data})
	return err
}

/**
* removeIn borra id dentro del lote sin preguntar; si no tiene versión viva no escribe nada.
* @param b *Batch, id string
* @return error
**/
func (s *Store) removeIn(b *Batch, id string) error {
	_, err := s.writeIn(b, id, change{tombstone: true})
	return err
}

/**
* insertIn guarda id dentro del lote solo si no existe para él. A diferencia de insert, no mira antes con una
* lectura si existe (con un valor grande): esa lectura usa el snapshot y no vería un cambio confirmado después, que
* tiene que dar ErrSerialization como en cualquier escritura del lote; si no escribe, el desborde se libera.
* @param b *Batch, id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) insertIn(b *Batch, id string, data []byte) (bool, error) {
	return s.writeIn(b, id, change{data: data, check: func(latest btree.Latest) btree.Action {
		if latest.Live {
			return btree.Skip
		}
		return btree.Apply
	}})
}

/**
* deleteIn borra id dentro del lote solo si existe para él.
* @param b *Batch, id string
* @return bool: si borró, error
**/
func (s *Store) deleteIn(b *Batch, id string) (bool, error) {
	return s.writeIn(b, id, change{tombstone: true})
}

/**
* updateIn reemplaza id dentro del lote solo si existe para él y el valor cambió; compara como update (antes de
* bloquear la hoja, y con la hoja bloqueada comprueba por su lsn que la versión comparada siga siendo la última).
* @param b *Batch, id string, data []byte
* @return bool: si escribió, error
**/
func (s *Store) updateIn(b *Batch, id string, data []byte) (bool, error) {
	if err := checkWrite(id, data); err != nil {
		return false, err
	}
	for {
		cur, found, err := s.readIn(b, id)
		if err != nil || !found {
			return false, err
		}
		if bytes.Equal(cur.value, data) {
			return false, nil
		}
		moved := false
		applied, err := s.writeIn(b, id, change{data: data, check: func(latest btree.Latest) btree.Action {
			moved = !latest.Live || latest.Version.LSN != cur.lsn
			if moved {
				return btree.Skip
			}
			return btree.Apply
		}})
		if err != nil || applied || !moved {
			return applied, err
		}
	}
}
