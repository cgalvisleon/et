package store

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/fsm"
	"github.com/cgalvisleon/et/store/internal/overflow"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
	"github.com/cgalvisleon/et/store/internal/xact"
)

// Réplica física (tarea 46; ver «Replicación» en doc.go), como la replicación por streaming de PostgreSQL: una
// réplica empieza de una copia de Backup del primario, abierta con OpenReplica, y aplica los registros del WAL que le
// llegan (WalSince en el primario → ApplyWal en la réplica) en la misma posición, así su WAL es una copia byte a byte.
// Para el usuario es de solo lectura (ErrReadOnly); sus checkpoints son restartpoints: toman el punto de rehacer del
// último KindCheckpoint del primario ya rehecho y no escriben nada en el WAL.

// WalEntry es un registro del WAL tal como está en el segmento, con su posición (WalReader.Next lo da, ApplyWal lo
// recibe). El transporte entre primario y réplica es del llamador: los campos se copian tal cual.
type WalEntry struct {
	LSN     uint64 // lsn del registro
	Segment uint64 // segmento donde empieza
	Offset  int64  // desplazamiento dentro del segmento
	Data    []byte // el registro entero (encabezado, bloques y datos principales, con su crc)
}

// ckptMark es el último KindCheckpoint rehecho por la réplica: el punto de rehacer de su próximo restartpoint.
type ckptMark struct {
	info   checkpointInfo
	lsn    page.LSN // el del registro
	frozen page.XID // el frozenXID tomado al rehacer el checkpoint anterior (lo que lo justifica es anterior al punto de rehacer)
}

// replicaState es lo propio de una réplica abierta.
//
// Locks: applyMu → txGate (escritura, para los registros que cambian lo que ve un snapshot) → replayMu (escritura) →
// ddlMu → Store.mu. Las lecturas de páginas toman replayMu en lectura dentro de Store.begin (closeMu → replayMu →
// Store.mu). Verify y el restartpoint toman applyMu para ver un estado quieto. Tomar un snapshot: txGate (lectura) →
// xact → snapMu (openTxn); snapMu es una hoja.
type replicaState struct {
	applyMu  sync.Mutex   // un aplicador a la vez
	replayMu sync.RWMutex // aplicar un registro (escritura) contra las lecturas de páginas (lectura)

	redo    *redoState    // el de la recuperación de OpenReplica, que sigue rehaciendo (con applyMu)
	applied atomic.Uint64 // el último lsn aplicado
	done    page.LSN      // el checkpointLSN del último restartpoint (con ckptMu)

	snapMu    sync.Mutex
	snaps     map[*Snapshot]struct{} // snapshots abiertos: crece con la concurrencia
	conflicts atomic.Int64           // snapshots cancelados por un conflicto con el aplicador

	// promoted: Promote la volvió primario sin cerrarla (tarea 85). Desde ahí el motor no es una réplica
	// (isReplica); el puntero e.replica no cambia nunca, así nadie lo lee a medio escribir.
	promoted atomic.Bool
}

/**
* isReplica dice si el motor es una réplica: abierto con OpenReplica y sin promover.
* @return bool
**/
func (e *Engine) isReplica() bool {
	rs := e.replica
	return rs != nil && !rs.promoted.Load()
}

/**
* readLock toma replayMu en lectura si el motor es una réplica (r nil no hace nada).
**/
func (r *replicaState) readLock() {
	if r != nil {
		r.replayMu.RLock()
	}
}

/**
* readUnlock suelta lo que tomó readLock.
**/
func (r *replicaState) readUnlock() {
	if r != nil {
		r.replayMu.RUnlock()
	}
}

/**
* track anota un snapshot abierto de la réplica (para cancelarlo en un conflicto).
* @param v *Snapshot
**/
func (r *replicaState) track(v *Snapshot) {
	if r == nil {
		return
	}
	r.snapMu.Lock()
	r.snaps[v] = struct{}{}
	r.snapMu.Unlock()
}

/**
* untrack olvida un snapshot cerrado.
* @param v *Snapshot
**/
func (r *replicaState) untrack(v *Snapshot) {
	if r == nil {
		return
	}
	r.snapMu.Lock()
	delete(r.snaps, v)
	r.snapMu.Unlock()
}

// conflict dice si rehacer un registro cambia lo que ve un snapshot de la réplica (nil: no cambia nada para nadie).
type conflict func(snap *xact.Snapshot) bool

// everyone es el conflicto con todos los snapshots abiertos.
func everyone(*xact.Snapshot) bool { return true }

/**
* sees es el conflicto de una confirmación: la notan los snapshots que verían el xid confirmado (xid < Xmax). Uno
* tomado antes de que llegara el primer registro del xid no lo ve ni antes ni después.
* @param xid page.XID
* @return conflict
**/
func sees(xid page.XID) conflict {
	return func(snap *xact.Snapshot) bool { return snap.Visible(xid, xact.StatusCommitted) }
}

/**
* open dice si la réplica tiene snapshots abiertos.
* @return bool
**/
func (r *replicaState) open() bool {
	r.snapMu.Lock()
	defer r.snapMu.Unlock()
	return len(r.snaps) > 0
}

/**
* affected dice si algún snapshot abierto de la réplica nota c.
* @param c conflict
* @return bool
**/
func (r *replicaState) affected(c conflict) bool {
	r.snapMu.Lock()
	defer r.snapMu.Unlock()
	for v := range r.snaps {
		if c(&v.snap) {
			return true
		}
	}
	return false
}

/**
* cancel cierra con ErrRecoveryConflict los snapshots abiertos de la réplica que notan c.
* @param c conflict
**/
func (r *replicaState) cancel(c conflict) {
	r.snapMu.Lock()
	var list []*Snapshot
	for v := range r.snaps {
		if c(&v.snap) {
			list = append(list, v)
		}
	}
	r.snapMu.Unlock()
	for _, v := range list {
		v.release(ErrRecoveryConflict)
		r.conflicts.Add(1)
	}
}

/**
* startReplica termina la recuperación de OpenReplica: a diferencia de un primario no borra los stores de lotes sin
* confirmar ni fija el horizonte del archivo de estados (esas transacciones pueden confirmar más adelante en el
* primario), arma las piezas que leen y hace el primer restartpoint.
* @param r *redoState: el del recorrido del WAL, que sigue rehaciendo, end wal.Position: el final del WAL local
* @return error
**/
func (e *Engine) startReplica(r *redoState, end wal.Position) error {
	e.txs = xact.NewManager(max(r.nextXID, r.maxXID+1, 1))
	e.free = fsm.New(e.pool, e.sm, e.log)
	e.chains = overflow.New(e.pool, e.free, e.log)
	e.replayed = r.replayed
	e.replica = &replicaState{redo: r, snaps: map[*Snapshot]struct{}{}}
	e.replica.applied.Store(uint64(end.LSN - 1))
	return e.restartpoint(control.StateInProduction)
}

/**
* replayLSN es el último lsn aplicado por la réplica (0 en un primario).
* @return uint64
**/
func (e *Engine) replayLSN() uint64 {
	if !e.isReplica() {
		return 0
	}
	return e.replica.applied.Load()
}

/**
* applyWal aplica en la réplica los registros del WAL del primario, en orden: los ya aplicados (lsn ≤ ReplayLSN) se
* saltan, y el primero que no sigue da ErrWalGap sin aplicar nada de ahí en adelante. Cada uno se escribe en el WAL
* local en su posición y se rehace como en la recuperación. Si la tanda trae algo que puede cambiar lo que ve un
* snapshot (snapshotConflict) cierra la entrada de snapshots nuevos (txGate) y, antes de cada registro así, espera a
* que se cierren los snapshots abiertos que lo notarían (conflictOf, tarea 125), hasta replicaConflictDelay, y cancela
* los que queden; los demás siguen abiertos. No espera el fsync: WaitDurable (o el escritor de fondo del WAL) lo
* hace, y es lo que hay que esperar antes de confirmar la ranura en el primario.
* @param entries []WalEntry
* @return error: ErrNotReplica, ErrWalGap, ErrCorrupt (un registro inválido), ErrClosed, ErrFailed
**/
func (e *Engine) applyWal(entries []WalEntry) error {
	if !e.isReplica() {
		return ErrNotReplica
	}
	rs := e.replica
	if err := e.enterSlots(); err != nil {
		return err
	}
	defer e.leave()
	rs.applyMu.Lock()
	defer rs.applyMu.Unlock()
	if rs.promoted.Load() {
		return ErrNotReplica // la promovieron mientras esperaba
	}

	todo, recs, err := e.pendingEntries(entries)
	if err != nil || len(todo) == 0 {
		return err
	}
	gate := slices.ContainsFunc(recs, snapshotConflict)
	if gate { // ningún snapshot nuevo mientras se aplica algo que cambia lo que ven
		e.txGate.Lock()
		defer e.txGate.Unlock()
	}
	var deadline time.Time
	rs.replayMu.Lock()
	defer func() { rs.replayMu.Unlock() }()
	for i := range todo {
		if gate && rs.open() { // sin snapshots abiertos no hace falta mirar las páginas
			if c := e.conflictOf(recs[i]); c != nil && rs.affected(c) {
				rs.replayMu.Unlock() // las lecturas en curso de esos snapshots tienen que poder terminar
				deadline = e.waitConflict(c, deadline)
				rs.replayMu.Lock()
			}
		}
		if err := e.applyEntry(todo[i], recs[i]); err != nil {
			if errors.Is(err, wal.ErrOutOfOrder) {
				return fmt.Errorf(MSG_WAL_GAP, ErrWalGap, todo[i].LSN, rs.applied.Load()+1)
			}
			return e.fail(err)
		}
	}
	return nil
}

/**
* pendingEntries salta lo ya aplicado, comprueba que el resto siga en orden y decodifica cada registro (sin confiar
* en nada). Con applyMu.
* @param entries []WalEntry
* @return []WalEntry, []*wal.Record: los que faltan aplicar, error: ErrWalGap, ErrCorrupt
**/
func (e *Engine) pendingEntries(entries []WalEntry) ([]WalEntry, []*wal.Record, error) {
	next := e.replica.applied.Load() + 1
	for len(entries) > 0 && entries[0].LSN < next {
		entries = entries[1:]
	}
	recs := make([]*wal.Record, len(entries))
	for i, en := range entries {
		if en.LSN != next+uint64(i) {
			return nil, nil, fmt.Errorf(MSG_WAL_GAP, ErrWalGap, en.LSN, next+uint64(i))
		}
		rec, err := wal.Decode(en.Data)
		if err != nil || uint64(rec.LSN) != en.LSN {
			return nil, nil, fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, en.LSN, 0, "bad replicated record")
		}
		recs[i] = rec
	}
	return entries, recs, nil
}

/**
* snapshotConflict dice si rehacer rec puede cambiar lo que ve un snapshot abierto (sin mirar páginas): una
* confirmación (que hace visibles versiones), una limpieza o una liberación de páginas (que quitan versiones que el
* snapshot podría ver), el reciclado de una página borrada del árbol (un recorrido abierto puede tener todavía su
* enlace: como el safexid de XLOG_BTREE_REUSE_PAGE en PostgreSQL) o un DDL. A quiénes afecta lo dice conflictOf.
* @param rec *wal.Record
* @return bool
**/
func snapshotConflict(rec *wal.Record) bool {
	switch rec.Kind {
	case wal.KindCommit, wal.KindLeafVacuum, wal.KindFreePush, wal.KindRecyclePage, wal.KindCreateStore,
		wal.KindCreateStoreIn, wal.KindDropStore, wal.KindEmptyStore:
		return true
	case wal.KindLeafWrite:
		return rec.Flags&wal.FlagCommit != 0
	}
	return false
}

/**
* conflictOf dice a qué snapshots abiertos afecta rehacer rec, con la página todavía como antes del registro (tarea
* 125): una confirmación, a los que verían su xid; una limpieza de hoja, a los que ven una versión que quita sin ver
* una más nueva del mismo id, o no veían una que congela (btree.LeafVacuumConflict); una liberación o un reciclado de
* páginas y un DDL, a todos. Con applyMu, txGate en escritura y replayMu en escritura.
* @param rec *wal.Record
* @return conflict: nil si a ninguno
**/
func (e *Engine) conflictOf(rec *wal.Record) conflict {
	switch rec.Kind {
	case wal.KindCommit:
		if isDDL(rec) {
			return everyone
		}
		return sees(rec.XID)
	case wal.KindLeafWrite:
		if rec.Flags&wal.FlagCommit == 0 {
			return nil
		}
		if rec.XID == 0 { // nace visible para todos
			return everyone
		}
		return sees(rec.XID)
	case wal.KindLeafVacuum:
		return e.vacuumConflict(rec)
	}
	if snapshotConflict(rec) {
		return everyone
	}
	return nil
}

/**
* vacuumConflict lee la hoja de un KindLeafVacuum antes de rehacerlo y dice a quiénes afecta. Si el store ya no está
* el registro no se rehace (nil); si la hoja no se puede leer, a todos.
* @param rec *wal.Record: de btree, con la hoja en el bloque 0
* @return conflict
**/
func (e *Engine) vacuumConflict(rec *wal.Record) conflict {
	if len(rec.Blocks) == 0 {
		return everyone
	}
	b := rec.Blocks[0]
	if _, ok := e.cat.ByFile(b.File); !ok {
		return nil
	}
	h, err := e.pool.Pin(buffer.Tag{File: b.File, Fork: b.Fork, Block: b.Block}, buffer.Normal)
	if err != nil {
		return everyone
	}
	defer h.Release()
	h.Lock(buffer.Share)
	defer h.Unlock()
	c, err := btree.LeafVacuumConflict(h.Page(), b.Delta, e.xidStatus)
	if err != nil {
		return everyone
	}
	return c.Affects
}

/**
* xidStatus es el estado de xid en el archivo de estados.
* @param xid page.XID
* @return xact.Status, error
**/
func (e *Engine) xidStatus(xid page.XID) (xact.Status, error) {
	st, _, err := e.status.Status(xid)
	return st, err
}

/**
* waitConflict espera a que se cierren los snapshots que notan c y, pasado el plazo, los cancela
* (ErrRecoveryConflict). El plazo (replicaConflictDelay) corre desde la primera espera de un mismo applyWal. Con
* applyMu y txGate en escritura, sin replayMu.
* @param c conflict, deadline time.Time: el plazo ya corriendo (cero = empieza ahora)
* @return time.Time: el plazo
**/
func (e *Engine) waitConflict(c conflict, deadline time.Time) time.Time {
	if deadline.IsZero() {
		deadline = time.Now().Add(e.cfg.conflictDelay)
	}
	for e.replica.affected(c) {
		if time.Now().After(deadline) {
			e.replica.cancel(c)
			continue
		}
		time.Sleep(time.Millisecond)
	}
	return deadline
}

/**
* applyEntry escribe un registro en el WAL local y lo rehace. Un DDL espera antes su fsync: el catálogo y los
* archivos nunca se adelantan al WAL local. Con applyMu y replayMu en escritura.
* @param en WalEntry, rec *wal.Record: decodificado de en.Data
* @return error
**/
func (e *Engine) applyEntry(en WalEntry, rec *wal.Record) error {
	rs := e.replica

	at := wal.Position{Segment: en.Segment, Offset: en.Offset, LSN: rec.LSN}
	if err := e.log.Append(at, en.Data); err != nil {
		return err
	}
	ddl := isDDL(rec)
	if ddl {
		if err := e.log.WaitDurable(rec.LSN); err != nil {
			return err
		}
	}
	if err := rs.redo.apply(rec); err != nil {
		return err
	}
	if rec.XID != 0 {
		e.txs.Advance(rec.XID + 1)
	}
	if ddl {
		if err := e.syncHandles(); err != nil {
			return err
		}
	}
	if rec.Kind == wal.KindInitStore {
		e.openPendingTree(rec.Blocks[0].File)
	}
	rs.applied.Store(uint64(rec.LSN))
	return nil
}

/**
* isDDL dice si rec crea, vacía o borra stores (también el KindCommit de un lote con DDL).
* @param rec *wal.Record
* @return bool
**/
func isDDL(rec *wal.Record) bool {
	switch rec.Kind {
	case wal.KindCreateStore, wal.KindCreateStoreIn, wal.KindDropStore, wal.KindEmptyStore:
		return true
	case wal.KindCommit:
		info, err := decodeCommit(rec.Main, rec.Flags)
		return err == nil && len(info.creates)+len(info.drops) > 0
	}
	return false
}

/**
* syncHandles pone los *Store abiertos al día con el catálogo tras rehacer un DDL y guarda el catálogo (como el
* primario tras cada DDL): uno borrado queda inservible (ErrStoreDropped); uno vaciado pasa al archivo nuevo, sin
* árbol hasta que llegue su KindInitStore (ErrStorePending mientras tanto). Con replayMu en escritura: ninguna
* lectura está en curso.
* @return error
**/
func (e *Engine) syncHandles() error {
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	for name, s := range e.handles {
		entry, ok := e.cat.Get(name)
		if ok && entry.File == s.file {
			continue
		}
		s.mu.Lock()
		if ok {
			s.file, s.tree = entry.File, nil
		} else {
			s.dropped = true
			delete(e.handles, name)
		}
		s.mu.Unlock()
	}
	return e.storeCatalog()
}

/**
* openPendingTree da su árbol al *Store abierto que esperaba el KindInitStore del archivo file (tras un vaciado).
* Con replayMu en escritura.
* @param file smgr.FileNo
**/
func (e *Engine) openPendingTree(file smgr.FileNo) {
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	for _, s := range e.handles {
		if s.file != file || s.tree != nil {
			continue
		}
		tree, err := btree.Open(e.treeConfig(file))
		if err == nil {
			s.mu.Lock()
			s.tree = tree
			s.mu.Unlock()
		}
	}
}

/**
* restartpoint es el checkpoint de una réplica (como el restartpoint de PostgreSQL): no escribe en el WAL. Toma el
* último KindCheckpoint del primario ya rehecho, escribe todas las páginas sucias y el archivo de estados (lo
* aplicado hasta ahora, que incluye todo lo anterior a su punto de rehacer), guarda el catálogo con los contadores a
* lo aplicado, el control con ese punto de rehacer, recicla el WAL anterior y trunca el archivo de estados por debajo
* del frozenXID tomado al rehacer el checkpoint previo. Sin un checkpoint nuevo no hace nada (salvo el de Close).
* @param state control.State
* @return error
**/
func (e *Engine) restartpoint(state control.State) error {
	e.ckptMu.Lock()
	defer e.ckptMu.Unlock()
	if err := e.failed(); err != nil {
		return err
	}
	rs := e.replica
	start := time.Now()
	rs.applyMu.Lock()
	mark, applied := rs.redo.last, page.LSN(rs.applied.Load())
	counts, written := e.counts.snapshot(), e.log.Written()
	rs.applyMu.Unlock()
	if mark.lsn == 0 || mark.lsn == rs.done && state == control.StateInProduction {
		return nil
	}
	if err := e.log.WaitDurable(applied); err != nil {
		return e.fail(err)
	}
	if err := e.flushData(false, mark.info.redo, start); err != nil {
		return e.writeError(err)
	}
	if err := e.saveCatalog(counts, applied+1); err != nil {
		return e.writeError(err)
	}
	info := mark.info
	info.time, info.frozen = start.UnixNano(), mark.frozen
	if err := e.writeControl(state, info, mark.lsn); err != nil {
		return e.fail(err)
	}
	e.log.SetRedo(mark.info.redo)
	if err := e.checkpointSlots(); err != nil && !isCleanDiskFull(err) {
		return err
	}
	if err := e.log.Recycle(mark.info.redo); err != nil {
		return e.fail(err)
	}
	if err := e.status.Truncate(mark.frozen); err != nil {
		return e.fail(err)
	}
	rs.done = mark.lsn
	e.lastCheckpoint.Store(start.UnixNano())
	e.metrics.checkpointTook.Store(int64(time.Since(start)))
	e.metrics.checkpoints.Add(1)
	e.checkpointFinished(written)
	return nil
}

/**
* noteCheckpoint anota en la réplica el KindCheckpoint que acaba de rehacer (el punto de rehacer de su próximo
* restartpoint) y toma el frozenXID del motor ahora, que usará el restartpoint del checkpoint siguiente: todo lo que
* lo justifica queda antes del punto de rehacer de ese siguiente.
* @param info checkpointInfo, lsn page.LSN
**/
func (r *redoState) noteCheckpoint(info checkpointInfo, lsn page.LSN) {
	r.last = ckptMark{info: info, lsn: lsn, frozen: r.sample}
	oldest := info.oldest
	if oldest == 0 {
		oldest = info.nextXID
	}
	r.sample = r.e.frozen.min(r.e.cat.Catalog().Entries, oldest)
}

/**
* replicaWaitDurable hace durable el WAL local de la réplica hasta lo aplicado: lo que hay que esperar antes de
* confirmar la ranura en el primario (AdvanceSlot con ReplayLSN).
* @return error
**/
func (e *Engine) replicaWaitDurable() error {
	if err := e.enterSlots(); err != nil {
		return err
	}
	defer e.leave()
	if err := e.log.WaitDurable(page.LSN(e.replica.applied.Load())); err != nil {
		return e.fail(err)
	}
	return nil
}
