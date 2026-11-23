package store

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"

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

/**
* recover hace la recuperación de Open (pasos 1–5 de «Recuperación» en doc.go): lee el control y el catálogo (o
* empieza un motor nuevo), abre las piezas, reproduce el WAL desde el punto de rehacer (hasta el objetivo de OpenAt,
* si lo hay), activa el archivo continuo (WAL_ARCHIVE), fija el horizonte de xid y termina con un checkpoint.
* @return error
**/
func (e *Engine) recover() error {
	if err := e.checkFormat(); err != nil { // antes de leer o cambiar nada
		return err
	}
	label, restoring, err := e.readBackupLabel()
	if err != nil {
		return err
	}
	ctl, fresh, err := e.readControl()
	if err != nil {
		return err
	}
	if e.cfg.replica && fresh {
		return fmt.Errorf(MSG_REPLICA_FRESH, ErrNotReplica, e.dir)
	}
	if restoring {
		if fresh {
			return fmt.Errorf(MSG_BACKUP_UNFINISHED, ErrBackupIncomplete)
		}
		if err := e.checkBackupWAL(ctl.Redo, label); err != nil {
			return err
		}
	}
	if !e.cfg.stopTime.IsZero() {
		if e.cfg.stopLSN, err = e.timeTarget(ctl, label, restoring); err != nil {
			return err
		}
	}
	if e.cfg.stopLSN > 0 { // antes de cambiar nada: un objetivo inalcanzable deja la copia como estaba
		if err := e.checkTarget(ctl, label, restoring); err != nil {
			return err
		}
	}
	redo := wal.Start
	e.frozen.base = ctl.FrozenXID
	if fresh {
		e.cat = control.NewStores()
	} else if err := e.loadCatalog(); err != nil {
		return err
	} else {
		redo = ctl.Redo
	}
	for _, entry := range e.cat.Catalog().Entries {
		e.counts.set(entry.File, entry.Count)
	}
	if err := e.openPieces(); err != nil {
		return err
	}
	if err := e.status.Truncate(ctl.FrozenXID); err != nil { // termina un truncado cortado y fija el límite
		return err
	}

	r := &redoState{e: e, countLSN: e.cat.CountLSN(), checkpointLSN: ctl.CheckpointLSN, nextXID: ctl.NextXID,
		replica: e.cfg.replica, sample: ctl.FrozenXID}
	opts := e.cfg.wal
	opts.StopLSN = e.cfg.stopLSN
	opts.Upgrade = !e.cfg.replica // un directorio del formato 1 sigue en un segmento del 2 (formato 2, doc.go)
	clean := !fresh && !restoring && !e.cfg.replica && e.cfg.stopLSN == 0 && ctl.State == control.StateShutdown
	if clean {
		opts.CleanEnd = ctl.CheckpointLSN + 1 // el WAL termina en el checkpoint de Close (tarea 109)
	}
	log, end, err := wal.Open(e.fs, e.dir, opts, redo, r.apply)
	if err != nil {
		return err
	}
	e.log = log
	e.proxy.connect(log)
	if clean {
		if err := e.markInProduction(ctl); err != nil {
			return err
		}
	}
	if !fresh && !r.sawCheckpoint {
		return fmt.Errorf(MSG_NO_CHECKPOINT, ErrCorrupt, ctl.CheckpointLSN)
	}
	if err := e.loadSlots(); err != nil {
		return err
	}
	if !e.cfg.replica {
		if err := e.startArchive(); err != nil { // una réplica no archiva: lo hace el primario
			return err
		}
	}
	if err := e.finishRecovery(r, end); err != nil {
		return err
	}
	if restoring {
		return e.removeBackupLabel() // la copia ya es un motor más: su checkpoint pasó el fin del respaldo
	}
	return nil
}

/**
* markInProduction anota en el control, con fsync, que el motor está abierto antes de escribir nada en el WAL: tras un
* cierre limpio wal.Open no corta la cola (Options.CleanEnd), y si un corte dejara registros nuevos después del final
* con el control todavía en StateShutdown, el próximo Open tampoco la cortaría (un registro sin confirmar podría quedar
* justo donde termina uno nuevo). Así el control vuelve a StateShutdown solo con el checkpoint de un Close.
* @param ctl control.Control: el leído al abrir
* @return error
**/
func (e *Engine) markInProduction(ctl control.Control) error {
	ctl.State = control.StateInProduction
	return e.ctl.Write(ctl)
}

/**
* readControl lee el control. Sin archivo de control, o con un control sin copias válidas y un catálogo sin stores
* (un Create cortado), el motor es nuevo.
* @return control.Control, bool: si el motor es nuevo, error
**/
func (e *Engine) readControl() (control.Control, bool, error) {
	names, err := e.fs.ReadDir(e.dir)
	if err != nil {
		return control.Control{}, false, err
	}
	if !slices.Contains(names, control.FileName) {
		return control.Control{}, true, nil
	}
	ctl, err := e.ctl.Read()
	if err == nil {
		e.hasControl = true
		e.ctlRedo.Store(uint64(ctl.Redo.LSN))
		return ctl, false, nil
	}
	if !errors.Is(err, control.ErrNoValidCopy) {
		return control.Control{}, false, err
	}
	if !slices.Contains(names, control.CatalogFileName) {
		return control.Control{}, true, nil
	}
	cat, cerr := e.catFile.Load()
	if cerr != nil || len(cat.Entries) > 0 {
		return control.Control{}, false, fmt.Errorf(MSG_BAD_CONTROL_NEW, ErrCorrupt, errors.Join(err, cerr))
	}
	return control.Control{}, true, nil
}

/**
* loadCatalog lee el catálogo guardado.
* @return error
**/
func (e *Engine) loadCatalog() error {
	cat, err := e.catFile.Load()
	if err != nil {
		return err
	}
	e.cat, err = control.LoadStores(cat)
	return err
}

/**
* openPieces abre los archivos de datos, la caché y el archivo de estados, que la recuperación necesita antes que el
* WAL (por eso usan walProxy).
* @return error
**/
func (e *Engine) openPieces() error {
	var err error
	if e.sm, err = smgr.Open(e.fs, e.dir, smgr.DefaultMaxFiles); err != nil {
		return err
	}
	cfg := buffer.Config{Smgr: e.sm, Flusher: e.proxy, Buffers: e.cfg.buffers, BgWriterDelay: e.cfg.bgWriterDelay}
	if e.pool, err = buffer.Open(cfg); err != nil {
		return err
	}
	e.status, err = xact.Open(xact.Config{FS: e.fs, Dir: e.dir, Flusher: e.proxy, FilePages: e.cfg.xactFilePages})
	return err
}

/**
* finishRecovery borra los stores de lotes que no confirmaron, fija el siguiente xid (el mayor entre el del control y el mayor visto + 1), el horizonte del
* archivo de estados, arma las piezas que escriben y hace el checkpoint de fin de recuperación.
* En una réplica sigue startReplica.
* @param r *redoState: lo que dejó el recorrido del WAL, end wal.Position: el final del WAL
* @return error
**/
func (e *Engine) finishRecovery(r *redoState, end wal.Position) error {
	if e.cfg.replica {
		return e.startReplica(r, end)
	}
	if err := r.dropPending(); err != nil {
		return err
	}
	if err := r.dropUninitialized(); err != nil {
		return err
	}
	if err := e.removeUnallocated(); err != nil {
		return err
	}
	next := max(r.nextXID, r.maxXID+1, 1, e.cfg.firstXID)
	if err := e.status.Recover(next); err != nil {
		return err
	}
	e.txs = xact.NewManager(next)
	e.free = fsm.New(e.pool, e.sm, e.log)
	e.chains = overflow.New(e.pool, e.free, e.log)
	e.replayed = r.replayed
	return e.checkpoint(control.StateInProduction, false)
}

// redoState es lo que lleva la recuperación mientras wal.Open recorre el WAL.
type redoState struct {
	e             *Engine
	countLSN      page.LSN // las confirmaciones con lsn ≥ countLSN suman a los contadores
	checkpointLSN page.LSN // el registro de checkpoint que tiene que aparecer
	sawCheckpoint bool
	nextXID       page.XID      // el mayor nextXID de los checkpoints vistos (y del control)
	maxXID        page.XID      // el mayor xid visto en un registro
	replayed      int           // registros rehechos, sin contar los checkpoints
	created       []smgr.FileNo // altas sueltas rehechas (no en una réplica): dropUninitialized las revisa

	// Solo en una réplica (OpenReplica), que sigue rehaciendo después de abrir (replica.go).
	replica bool
	last    ckptMark // el último KindCheckpoint rehecho: el punto de rehacer del próximo restartpoint
	sample  page.XID // el frozenXID del motor al rehacer el último KindCheckpoint
}

/**
* apply rehace un registro del WAL (la función que recibe wal.Open).
* @param rec *wal.Record: válido solo durante la llamada
* @return error
**/
func (r *redoState) apply(rec *wal.Record) error {
	r.maxXID = max(r.maxXID, rec.XID)
	if frozen, ok := btree.RecordFrozenXID(rec); ok {
		r.nextXID = max(r.nextXID, frozen) // ningún xid nuevo por debajo del frozenXID de un store
		if r.replica {
			r.e.frozen.set(rec.Blocks[0].File, frozen)
		}
	}
	if rec.Kind != wal.KindCheckpoint {
		r.replayed++
	}
	if err := r.dispatch(rec); err != nil {
		return fmt.Errorf(MSG_REDO, rec.LSN, rec.Kind, err)
	}
	return nil
}

/**
* dispatch reparte el registro a su dueño: los Kind del motor (1–15) se rehacen acá; los de btree (16–31), overflow
* (32–39) y fsm (40–47), con la función Redo de su paquete.
* @param rec *wal.Record
* @return error
**/
func (r *redoState) dispatch(rec *wal.Record) error {
	switch k := rec.Kind; {
	case k == wal.KindCheckpoint:
		return r.checkpoint(rec)
	case k == wal.KindCommit:
		return r.commit(rec)
	case k == wal.KindAbort:
		return r.e.status.SetStatus(rec.XID, xact.StatusAborted, rec.LSN) // truncado: ErrTruncated, corrupción
	case k == wal.KindCreateStore, k == wal.KindCreateStoreIn:
		return r.createStore(rec)
	case k == wal.KindDropStore:
		return r.dropStore(rec)
	case k == wal.KindEmptyStore:
		return r.emptyStore(rec)
	case k == wal.KindBackupEnd:
		return r.backupEnd(rec)
	case k >= wal.KindLeafWrite && k < wal.KindOverflowWrite:
		return r.blocks(rec, btree.Redo)
	case k >= wal.KindOverflowWrite && k < wal.KindFreePush:
		return r.blocks(rec, overflow.Redo)
	case k >= wal.KindFreePush && k < wal.KindFreePush+8:
		return r.blocks(rec, fsm.Redo)
	}
	return fmt.Errorf(MSG_NOT_SUPPORTED, ErrCorrupt, rec.LSN, rec.Kind)
}

/**
* checkpoint anota que apareció el registro del checkpoint del control y toma su nextXID.
* @param rec *wal.Record
* @return error
**/
func (r *redoState) checkpoint(rec *wal.Record) error {
	info, ok := decodeCheckpoint(rec.Main)
	if !ok {
		return fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, "bad checkpoint data")
	}
	if rec.LSN == r.checkpointLSN {
		r.sawCheckpoint = true
	}
	r.nextXID = max(r.nextXID, info.nextXID)
	if r.replica {
		r.noteCheckpoint(info, rec.LSN)
	}
	return nil
}

/**
* backupEnd rehace un KindBackupEnd: no cambia nada (lo que comprueba la restauración lo mira checkBackupWAL antes de
* rehacer); solo se valida su forma.
* @param rec *wal.Record
* @return error
**/
func (r *redoState) backupEnd(rec *wal.Record) error {
	if _, ok := decodeBackupEnd(rec.Main); !ok {
		return fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, "bad backup end data")
	}
	return nil
}

/**
* commit rehace un KindCommit (fase 1B): estado confirmada y, si su lsn ≥ countLSN, sus sumas a los contadores.
* @param rec *wal.Record
* @return error
**/
func (r *redoState) commit(rec *wal.Record) error {
	info, err := decodeCommit(rec.Main, rec.Flags)
	if err != nil {
		return fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, err.Error())
	}
	if rec.LSN >= r.countLSN {
		for file, delta := range info.deltas {
			r.e.counts.add(file, delta)
		}
	}
	for _, file := range info.creates {
		r.e.cat.ClearFlags(file, control.FlagPendingCreate)
	}
	if err := r.setCommitted(rec); err != nil {
		return err
	}
	for _, file := range info.drops {
		if err := r.dropFile(file); err != nil {
			return err
		}
	}
	return nil
}

/**
* dropFile rehace el borrado de un store por su número de archivo (el KindCommit de un lote que lo borró): lo saca
* del catálogo si todavía lo tiene y borra sus archivos.
* @param file smgr.FileNo
* @return error
**/
func (r *redoState) dropFile(file smgr.FileNo) error {
	if entry, ok := r.e.cat.ByFile(file); ok {
		r.e.cat.Drop(entry.Name, file)
	}
	return r.unlinkUnused(file)
}

/**
* removeUnallocated borra, al terminar de rehacer el WAL, los archivos de datos con número ≥ NextFileNo: son de un
* alta cuyo registro un corte perdió (createStore no espera su fsync para crear los archivos), así que ningún
* registro durable los nombra. El número se vuelve a dar y openTree igual borraría lo que encontrara.
* @return error
**/
func (e *Engine) removeUnallocated() error {
	names, err := e.fs.ReadDir(filepath.Join(e.dir, smgr.DataDir))
	if err != nil {
		return err
	}
	next := e.cat.NextFileNo()
	seen := map[smgr.FileNo]bool{}
	for _, name := range names {
		if file, ok := parseDataName(name); ok && file >= next && !seen[file] {
			seen[file] = true
			e.pool.DropFile(file)
			if err := e.sm.Unlink(file); err != nil {
				return err
			}
		}
	}
	return nil
}

/**
* dropPending borra, al terminar de rehacer el WAL, los stores que siguen con FlagPendingCreate: los creó un lote
* cuyo KindCommit no llegó a ser durable (ninguna transacción sobrevive a reabrir). No escribe en el WAL: el
* checkpoint de fin de recuperación guarda el catálogo sin ellos y, si se corta antes, la próxima recuperación los
* vuelve a borrar.
* @return error
**/
func (r *redoState) dropPending() error {
	for _, file := range pendingFiles(r.e.cat.Catalog().Entries) {
		if err := r.dropFile(file); err != nil {
			return err
		}
	}
	return nil
}

/**
* setCommitted anota confirmada la transacción de un registro. Un xid ya truncado se lee confirmado (y por debajo del
* frozenXID ningún registro puede seguir en el WAL a rehacer): no hay nada que anotar.
* @param rec *wal.Record
* @return error
**/
func (r *redoState) setCommitted(rec *wal.Record) error {
	err := r.e.status.SetStatus(rec.XID, xact.StatusCommitted, rec.LSN)
	if errors.Is(err, xact.ErrTruncated) {
		return nil
	}
	return err
}

/**
* createStore rehace el alta de un store en el catálogo (idempotente: no hace nada si el nombre ya existe). La de
* un lote (KindCreateStoreIn) queda con FlagPendingCreate hasta rehacer su KindCommit.
* @param rec *wal.Record
* @return error
**/
func (r *redoState) createStore(rec *wal.Record) error {
	file, name, ok := decodeStoreMain(rec.Main)
	if !ok {
		return fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, "bad store data")
	}
	created, err := r.e.cat.Create(name, file)
	if created {
		r.e.counts.set(file, 0)
		if rec.Kind == wal.KindCreateStoreIn {
			r.e.cat.SetFlags(file, control.FlagPendingCreate)
		} else if !r.replica {
			r.created = append(r.created, file)
		}
	}
	return err
}

/**
* dropUninitialized borra, al terminar de rehacer el WAL, los stores sueltos dados de alta en lo rehecho cuyo
* KindInitStore no llegó a ser durable (sin página meta): su Store no volvió bien (handle espera los dos registros),
* así que nadie lo vio. Como dropPending, sin WAL: el checkpoint de fin de recuperación guarda el catálogo sin ellos.
* Crece con las altas desde el último checkpoint, no con las claves.
* @return error
**/
func (r *redoState) dropUninitialized() error {
	for _, file := range r.created {
		if _, alive := r.e.cat.ByFile(file); !alive {
			continue
		}
		ready, err := r.e.initialized(file)
		if err != nil {
			return err
		}
		if !ready {
			if err := r.dropFile(file); err != nil {
				return err
			}
		}
	}
	return nil
}

/**
* dropStore rehace la baja de un store: lo saca del catálogo si todavía tiene ese número de archivo y, si el número
* ya no es de ningún store (los números no se reutilizan), borra sus archivos, también los que un corte dejó.
* @param rec *wal.Record
* @return error
**/
func (r *redoState) dropStore(rec *wal.Record) error {
	file, name, ok := decodeStoreMain(rec.Main)
	if !ok {
		return fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, "bad store data")
	}
	r.e.cat.Drop(name, file)
	return r.unlinkUnused(file)
}

/**
* emptyStore rehace el vaciado de un store: si todavía tiene el archivo viejo pasa al nuevo con el contador en 0 (los
* bloques del nuevo, empezando por su KindInitStore, vienen después en el WAL) y, si el viejo ya no es de ningún
* store, borra sus archivos, también los que un corte dejó.
* @param rec *wal.Record
* @return error
**/
func (r *redoState) emptyStore(rec *wal.Record) error {
	old, file, name, ok := decodeEmptyMain(rec.Main)
	if !ok {
		return fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, "bad empty store data")
	}
	emptied, err := r.e.cat.Empty(name, old, file)
	if err != nil {
		return err
	}
	if emptied {
		r.e.counts.set(file, 0)
	}
	return r.unlinkUnused(old)
}

/**
* unlinkUnused borra los archivos y las páginas en caché de un número que ya no es de ningún store (los números no
* se reutilizan); si sigue en uso no hace nada.
* @param file smgr.FileNo
* @return error
**/
func (r *redoState) unlinkUnused(file smgr.FileNo) error {
	if _, alive := r.e.cat.ByFile(file); alive {
		return nil
	}
	r.e.counts.remove(file)
	r.e.frozen.forget(file)
	r.e.pool.DropFile(file)
	return r.e.sm.Unlink(file)
}

/**
* blocks rehace un registro con bloques con la función de su dueño; los de un store que ya no está en el catálogo
* (borrado más adelante) se saltan. Una escritura suelta (KindLeafWrite con FlagCommit) además confirma su xid y
* suma a los contadores si su lsn ≥ countLSN.
* @param rec *wal.Record, redo func: btree.Redo, overflow.Redo o fsm.Redo
* @return error
**/
func (r *redoState) blocks(rec *wal.Record, redo func(buffer.Pool, smgr.Manager, *wal.Record) error) error {
	if len(rec.Blocks) == 0 {
		return fmt.Errorf(MSG_BAD_RECORD, ErrCorrupt, rec.LSN, rec.Kind, "no blocks")
	}
	file := rec.Blocks[0].File
	if _, ok := r.e.cat.ByFile(file); ok {
		if err := redo(r.e.pool, r.e.sm, rec); err != nil {
			return err
		}
	}
	if rec.Kind != wal.KindLeafWrite || rec.Flags&wal.FlagCommit == 0 || rec.XID == 0 {
		return nil
	}
	if rec.LSN >= r.countLSN {
		r.e.counts.add(file, leafCountDelta(rec.Flags))
	}
	return r.setCommitted(rec)
}
