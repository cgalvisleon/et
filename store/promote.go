package store

import (
	"github.com/cgalvisleon/et/store/internal/control"
)

/**
* promote vuelve primario a una réplica abierta sin cerrarla (tarea 85, como pg_promote): deja de aplicar WAL, termina
* la recuperación como Open (borra los stores de lotes del primario que no confirmaron, fija el horizonte del archivo
* de estados y el próximo xid; lo que seguía en curso en el primario queda abortado), arranca la limpieza de fondo y
* hace el checkpoint de fin de recuperación, que escribe su KindCheckpoint en el WAL. Los snapshots abiertos siguen
* andando. Los stores abiertos antes siguen sin poner marcas de visibilidad (solo es más lento leerlos) hasta reabrir.
* Bloqueos: ckptMu → applyMu → replayMu → ddlMu, como el restartpoint y el aplicador.
* @return error: ErrNotReplica, ErrClosed, ErrFailed
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
