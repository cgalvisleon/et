package store

import (
	"time"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* checkpoint hace un checkpoint (pasos 1–7 de «Checkpoint» en doc.go; el 7 trunca el archivo de estados por debajo
* del frozenXID que guardó el control). Cualquier fallo deja el motor inutilizable.
* @param state control.State: StateInProduction, o StateShutdown en el de Close
* @param spread bool: repartir la escritura de páginas en el 90 % del intervalo (el checkpoint de fondo)
* @return error
**/
func (e *Engine) checkpoint(state control.State, spread bool) error {
	e.ckptMu.Lock()
	defer e.ckptMu.Unlock()
	if err := e.failed(); err != nil {
		return err
	}
	start := time.Now()
	info, counts, err := e.beginCheckpoint(start)
	if err != nil {
		return e.fail(err)
	}
	if err := e.waitChanges(info.redo.LSN); err != nil {
		return e.fail(err)
	}
	e.stepDone(1)
	e.waitCommits()
	if err := e.flushData(spread, info.redo, start); err != nil {
		return e.fail(err)
	}
	e.stepDone(2)
	if err := e.saveCatalog(counts, info.redo.LSN); err != nil {
		return e.fail(err)
	}
	e.stepDone(3)
	lsn, err := e.logRecord(wal.KindCheckpoint, encodeCheckpoint(info))
	if err != nil {
		return e.fail(err)
	}
	e.stepDone(4)
	if err := e.writeControl(state, info, lsn); err != nil {
		return e.fail(err)
	}
	e.stepDone(5)
	if err := e.log.Recycle(info.redo); err != nil {
		return e.fail(err)
	}
	e.stepDone(6)
	if err := e.status.Truncate(info.frozen); err != nil {
		return e.fail(err)
	}
	e.stepDone(7)
	e.lastCheckpoint.Store(start.UnixNano())
	return nil
}

/**
* stepDone avisa a las pruebas que terminó un paso del checkpoint (afterStep).
* @param step int
**/
func (e *Engine) stepDone(step int) {
	if e.afterStep != nil {
		e.afterStep(step)
	}
}

/**
* beginCheckpoint es el paso 1: toma el frozenXID del motor (antes del punto de rehacer: lo que lo justifica ya
* está en el WAL) y, bajo el writeMu del WAL, fija el punto de rehacer y, en ese mismo instante, toma el siguiente
* xid, el xid vivo más viejo y los contadores. Lo hace con hintGate en exclusivo: ninguna lectura está decidiendo o
* poniendo una marca de visibilidad sin WAL mientras se mueve el punto de rehacer (DELAY_CHKPT_START).
* @param now time.Time
* @return checkpointInfo, map[smgr.FileNo]int64: los contadores al punto de rehacer, error
**/
func (e *Engine) beginCheckpoint(now time.Time) (checkpointInfo, map[smgr.FileNo]int64, error) {
	info := checkpointInfo{time: now.UnixNano(), frozen: e.engineFrozen()}
	var counts map[smgr.FileNo]int64
	e.hintGate.Lock()
	defer e.hintGate.Unlock()
	redo, err := e.log.BeginCheckpoint(func(wal.Position) {
		counts = e.counts.snapshot()
		info.nextXID = e.txs.NextXID()
		if oldest := e.txs.OldestActive(); oldest < info.nextXID {
			info.oldest = oldest
		}
	})
	info.redo = redo
	return info, counts, err
}

/**
* waitChanges espera a que estén escritos todos los registros anteriores al punto de rehacer. Cada cambio de página
* reserva su registro, aplica el cambio, marca la página sucia y recién después escribe el registro: un cambio
* reservado antes del punto de rehacer pero marcado sucio después de que FlushAll juntó las sucias quedaría fuera del
* checkpoint y no se rehace (PostgreSQL lo evita marcando sucia antes de insertar el registro). Esperar el WAL durable
* hasta el último lsn anterior al punto de rehacer garantiza que todas esas páginas ya están sucias.
* @param redo page.LSN: el lsn del punto de rehacer
* @return error: el del WAL
**/
func (e *Engine) waitChanges(redo page.LSN) error {
	if redo <= 1 {
		return nil
	}
	return e.log.WaitDurable(redo - 1)
}

/**
* waitCommits espera a las operaciones que reservaron su registro antes del punto de rehacer y todavía no anotaron
* su estado: después de esto el archivo de estados que se guarda las incluye.
**/
func (e *Engine) waitCommits() {
	e.commitMu.Lock()
	e.commitMu.Unlock()
}

/**
* flushData es el paso 2: escribe las páginas sucias (repartidas si spread), las del archivo de estados y hace
* fsync de los archivos de datos.
* @param spread bool, redo wal.Position, start time.Time
* @return error
**/
func (e *Engine) flushData(spread bool, redo wal.Position, start time.Time) error {
	var pace func(written, total int)
	if spread {
		pace = e.pacer(redo, start)
	}
	if err := e.pool.FlushAll(pace); err != nil {
		return err
	}
	if err := e.status.Flush(); err != nil {
		return err
	}
	return e.sm.SyncAll()
}

/**
* saveCatalog es el paso 3: guarda el catálogo con los contadores tomados en el paso 1 y countLSN = el punto de
* rehacer. Un store creado después del punto de rehacer no está en counts y queda en 0: todas sus confirmaciones
* tienen lsn ≥ countLSN y la recuperación las suma.
* @param counts map[smgr.FileNo]int64, redoLSN page.LSN
* @return error
**/
func (e *Engine) saveCatalog(counts map[smgr.FileNo]int64, redoLSN page.LSN) error {
	e.ddlMu.Lock()
	defer e.ddlMu.Unlock()
	for _, entry := range e.cat.Catalog().Entries {
		if _, err := e.cat.AddCount(entry.File, counts[entry.File]-entry.Count); err != nil {
			return err
		}
	}
	e.cat.SetCountLSN(redoLSN)
	return e.catFile.Save(e.cat.Catalog())
}

/**
* writeControl es el paso 5: escribe el control con el punto de rehacer y el lsn del registro de checkpoint (en un
* motor nuevo, lo crea).
* @param state control.State, info checkpointInfo, lsn page.LSN
* @return error
**/
func (e *Engine) writeControl(state control.State, info checkpointInfo, lsn page.LSN) error {
	c := control.Control{
		State:           state,
		CheckpointTime:  info.time,
		Redo:            info.redo,
		CheckpointLSN:   lsn,
		NextXID:         info.nextXID,
		OldestActiveXID: info.oldest,
		FrozenXID:       info.frozen,
	}
	if e.hasControl {
		return e.ctl.Write(c)
	}
	if err := e.ctl.Create(c); err != nil {
		return err
	}
	e.hasControl = true
	return nil
}

/**
* pacer reparte la escritura de páginas del checkpoint de fondo: tras cada página, si va adelantado respecto del
* tiempo (90 % de CHECKPOINT_TIMEOUT) y del WAL escrito desde el punto de rehacer (90 % del volumen que dispara el
* siguiente), duerme hasta emparejar, de a 100 ms. Close lo apura (stop).
* @param redo wal.Position, start time.Time
* @return func(written, total int)
**/
func (e *Engine) pacer(redo wal.Position, start time.Time) func(written, total int) {
	budget := float64(e.cfg.checkpointTimeout) * completionTarget
	trigger := float64(e.cfg.walTrigger())
	return func(written, total int) {
		for {
			done := float64(written) / float64(total)
			wait := time.Duration(done*budget) - time.Since(start)
			if wait <= 0 || float64(e.log.SizeSince(redo))/trigger >= done {
				return
			}
			select {
			case <-e.stop:
				return
			case <-time.After(min(wait, 100*time.Millisecond)):
			}
		}
	}
}

/**
* checkpointer es el proceso de fondo: cada checkEvery mira si toca un checkpoint (por tiempo o por volumen de WAL) y
* lo hace repartido. Termina con Close o cuando un checkpoint falla (el motor quedó inutilizable).
**/
func (e *Engine) checkpointer() {
	defer e.bg.Done()
	ticker := time.NewTicker(e.cfg.checkEvery)
	defer ticker.Stop()
	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			if !e.checkpointDue() {
				continue
			}
			if err := e.checkpoint(control.StateInProduction, true); err != nil {
				return
			}
		}
	}
}

/**
* checkpointDue dice si pasó CHECKPOINT_TIMEOUT desde el último checkpoint o si el WAL desde el punto de rehacer pasó
* MAX_WAL_SIZE / 1,9.
* @return bool
**/
func (e *Engine) checkpointDue() bool {
	since := time.Since(time.Unix(0, e.lastCheckpoint.Load()))
	return since >= e.cfg.checkpointTimeout || e.log.SizeSince(e.log.RedoPoint()) >= e.cfg.walTrigger()
}
