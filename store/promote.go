package store

import (
	"errors"

	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/wal"
)

/**
* promote vuelve primario a una réplica abierta sin cerrarla (tarea 85, como pg_promote): deja de aplicar WAL, termina
* la recuperación como Open (antes de nada rota el WAL y, con WAL_ARCHIVE=on, arranca el archivo: startPrimaryWAL;
* borra los stores de lotes del primario que no confirmaron, fija el horizonte del archivo
* de estados y el próximo xid; lo que seguía en curso en el primario queda abortado), arranca la limpieza de fondo y
* hace el checkpoint de fin de recuperación, que escribe su KindCheckpoint en el WAL. Los snapshots abiertos siguen
* andando. Los stores abiertos antes siguen sin poner marcas de visibilidad (solo es más lento leerlos) hasta reabrir.
* Bloqueos: ckptMu → applyMu → replayMu → ddlMu, como el restartpoint y el aplicador.
* @return error: ErrNotReplica, ErrDiskFull (sin promover: se puede reintentar), ErrClosed, ErrFailed
**/
func (e *Engine) promote() error {
	if !e.isReplica() {
		return ErrNotReplica
	}
	if err := e.enterSlots(); err != nil {
		return err
	}
	defer e.leave()
	e.ckptMu.Lock()
	defer e.ckptMu.Unlock()
	rs := e.replica
	rs.applyMu.Lock()
	if rs.promoted.Load() {
		rs.applyMu.Unlock()
		return ErrNotReplica
	}
	if err := e.startPrimaryWAL(); err != nil {
		rs.applyMu.Unlock()
		return err
	}
	err := e.finishPromotion()
	rs.applyMu.Unlock()
	if err != nil {
		return e.fail(err)
	}
	if e.cfg.vacuumEvery > 0 {
		e.bg.Add(1)
		go e.vacuumer()
	}
	_, _, err = e.checkpointLocked(control.StateInProduction, false)
	return err
}

/**
* finishPromotion es lo que finishRecovery hace en un primario, sobre la réplica quieta: con replayMu en escritura
* ninguna lectura de páginas está en curso. Con ckptMu y applyMu.
* @return error
**/
func (e *Engine) finishPromotion() error {
	rs := e.replica
	rs.replayMu.Lock()
	defer rs.replayMu.Unlock()
	r := rs.redo
	if err := r.dropPending(); err != nil {
		return err
	}
	if err := e.syncHandles(); err != nil {
		return err
	}
	next := max(r.nextXID, r.maxXID+1, 1, e.txs.NextXID())
	if err := e.status.Recover(next); err != nil {
		return err
	}
	e.txs.Advance(next)
	rs.promoted.Store(true)
	return nil
}

/**
* startPrimaryWAL rota el WAL de la réplica al promoverla (tarea 138) y, con WAL_ARCHIVE=on, arranca el archivo
* continuo como Open (una réplica no archiva). Con applyMu nadie más escribe en el WAL, así que el primer registro
* como primario cae en un segmento nuevo con encabezado de wal.FormatVersion (una réplica de una copia del formato 1
* seguía en el segmento del 1 hasta el próximo Open) y el sellado queda retenido para archivar. Es una rotación
* normal (encabezado del siguiente con fsync antes de sellar), segura ante un corte. Va antes de tocar nada más: con
* el disco lleno la réplica sigue siendo réplica y Promote se puede reintentar.
* @return error: ErrDiskFull, ErrFailed
**/
func (e *Engine) startPrimaryWAL() error {
	_, err := e.log.StartSegment()
	switch {
	case errors.Is(err, wal.ErrNoRoom):
		return e.diskFull(err)
	case err != nil:
		return e.fail(err)
	}
	if !e.cfg.archive {
		return nil
	}
	if err := e.startArchive(); err != nil {
		return e.writeError(err)
	}
	return nil
}
