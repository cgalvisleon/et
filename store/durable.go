package store

import (
	"time"

	"github.com/cgalvisleon/et/store/internal/page"
)

// Deferred son las escrituras sueltas de un store que confirman sin esperar el fsync del WAL (como
// synchronous_commit = off de PostgreSQL): lo escrito se ve enseguida, y es durable cuando vuelve Engine.WaitDurable,
// cuando una confirmación durable posterior hace su fsync o cuando pasa el escritor de fondo del WAL. Es un valor
// sin estado propio: se puede copiar y usar desde varias goroutines.
type Deferred struct {
	s *Store
}

/**
* syncCommit es la espera de una confirmación: con SYNC_ON_WRITE y si no es diferida, el WAL durable hasta lsn (group
* commit); si no, solo el freno de las diferidas (deferredBrake). Siempre anota lsn como confirmado (noteCommit) antes
* de volver, para que WaitDurable lo cubra.
* @param lsn page.LSN: el del registro que confirma, deferred bool
* @return error
**/
func (e *Engine) syncCommit(lsn page.LSN, deferred bool) error {
	e.noteCommit(lsn)
	if deferred || !e.cfg.syncOnWrite {
		return e.deferredBrake(lsn)
	}
	return e.log.WaitDurable(lsn)
}

/**
* noteCommit sube lastCommit hasta lsn (nunca lo baja: las confirmaciones terminan en cualquier orden).
* @param lsn page.LSN
**/
func (e *Engine) noteCommit(lsn page.LSN) {
	for {
		cur := e.lastCommit.Load()
		if uint64(lsn) <= cur || e.lastCommit.CompareAndSwap(cur, uint64(lsn)) {
			return
		}
	}
}

/**
* waitDurable espera a que el WAL sea durable hasta la última confirmación anotada: todo lo confirmado (diferido o
* no, en cualquier store) antes de llamar queda en disco. Un fsync fallido deja el motor inutilizable.
* @return error: ErrClosed, ErrFailed
**/
func (e *Engine) waitDurable() error {
	if e.replica != nil {
		return e.replicaWaitDurable()
	}
	if err := e.enter(); err != nil {
		return err
	}
	defer e.leave()
	lsn := page.LSN(e.lastCommit.Load())
	if lsn <= e.log.Durable() {
		return nil
	}
	if err := e.log.WaitDurable(lsn); err != nil {
		return e.fail(err)
	}
	return nil
}

/**
* walWriter es el escritor de fondo del WAL (como el walwriter de PostgreSQL): cada walWriterDelay hace durables las
* confirmaciones que no esperaron su fsync, así lo que puede perder un corte queda acotado a esa pausa más un fsync.
* Termina con Close o cuando el motor queda inutilizable.
**/
func (e *Engine) walWriter() {
	defer e.bg.Done()
	ticker := time.NewTicker(e.cfg.walWriterDelay)
	defer ticker.Stop()
	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			if err := e.waitDurable(); err != nil {
				return
			}
		}
	}
}
