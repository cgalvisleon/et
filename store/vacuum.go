package store

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/buffer"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/vfs"
)

/**
* vacuum limpia todos los stores del catálogo, uno a la vez, con el horizonte tomado al empezar (un snapshot o lote
* que empiece después tiene Xmin ≥ ese horizonte, así que no ve nada de lo que se quita). Un store borrado entre medio
* se salta. Si un store falla sigue con los demás y devuelve el primer error; con el motor cerrado o inutilizable se
* detiene enseguida.
* @return error
**/
func (e *Engine) vacuum() error {
	if err := e.enter(); err != nil {
		return err
	}
	horizon := e.txs.OldestActive()
	names := e.cat.Names()
	e.metrics.vacuumWrites.Store(e.writes.Load())
	e.metrics.vacuums.Add(1)
	e.leave()

	var first error
	for _, name := range names {
		err := e.vacuumStore(name, horizon)
		if errors.Is(err, ErrClosed) || errors.Is(err, ErrFailed) {
			return err
		}
		if first == nil {
			first = err
		}
	}
	return first
}

/**
* vacuumStore limpia el store name si sigue existiendo.
* @param name string, horizon page.XID
* @return error
**/
func (e *Engine) vacuumStore(name string, horizon page.XID) error {
	s, err := e.existing(name)
	if err != nil || s == nil {
		return err
	}
	err = s.vacuum(horizon)
	if errors.Is(err, ErrStoreDropped) {
		return nil
	}
	return err
}

/**
* vacuum limpia y congela el árbol del store (serializado por vacuumMu; ver freezeStore). Mientras dura tiene el store
* tomado en lectura: un DropStore espera a que termine, y uno que ya empezó deja el store en ErrStoreDropped y no se
* limpia.
* @param horizon page.XID: el xid vivo más viejo al empezar la limpieza del motor
* @return error: ErrStoreDropped, ErrClosed, ErrFailed, ErrDiskFull (no hay lugar para el WAL) o el error de una
* página dañada
**/
func (s *Store) vacuum(horizon page.XID) error {
	s.vacuumMu.Lock()
	defer s.vacuumMu.Unlock()
	if err := s.beginWrite(); err != nil {
		return err
	}
	defer s.end()
	if err := s.e.admit(); err != nil {
		return err
	}
	return s.e.vacuumError(s.freezeStore(horizon))
}

/**
* vacuumError trata el error de una limpieza: cada paso de la limpieza es un cambio completo y repetible, así que un
* disco lleno a mitad (también la falta de lugar del WAL) la corta con ErrDiskFull sin dejar nada a medias, si el WAL,
* la caché y los fsync siguen sanos. Lo demás, como una lectura (readError).
* @param err error
* @return error
**/
func (e *Engine) vacuumError(err error) error {
	if vfs.IsNoSpace(err) && !errors.Is(err, buffer.ErrFailed) && !errors.Is(err, ErrFailed) &&
		e.log.Failed() == nil && !e.guard.failed.Load() {
		return e.diskFull(err)
	}
	return e.readError(err)
}

/**
* freeDead libera lo que queda de la cadena de desborde de una versión muerta (desde el final, repetible tras un
* corte). Los registros de la limpieza van con xid 0.
* @param v btree.Version: con EntryOverflow
* @return error
**/
func (s *Store) freeDead(v btree.Version) error {
	_, err := s.e.chains.FreeTail(s.file, v.Overflow, v.XID, 0)
	return err
}

/**
* vacuumer es la limpieza de fondo (como autovacuum, con un criterio simple): cada VACUUM_INTERVAL, si hubo al menos
* VACUUM_THRESHOLD escrituras desde la última (0 = en cada vuelta) o se asignaron al menos vacuumXids xids (así el
* congelado y el truncado del archivo de estados siguen también con transacciones que no escriben), limpia todos los
* stores. Termina con Close o cuando el motor quedó inutilizable; Close la espera antes del checkpoint final.
**/
func (e *Engine) vacuumer() {
	defer e.bg.Done()
	ticker := time.NewTicker(e.cfg.vacuumEvery)
	defer ticker.Stop()
	lastXID := e.txs.NextXID()
	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			next := e.txs.NextXID()
			if e.writes.Load()-e.metrics.vacuumWrites.Load() < e.cfg.vacuumWrites && uint64(next-lastXID) < e.cfg.vacuumXids {
				continue
			}
			lastXID = next
			if err := e.vacuum(); errors.Is(err, ErrClosed) || errors.Is(err, ErrFailed) {
				return
			}
		}
	}
}

/**
* closeVacuum es la limpieza de Close: con la limpieza de fondo encendida (VACUUM_INTERVAL > 0) y al menos
* VACUUM_THRESHOLD escrituras desde la última limpieza, limpia todos los stores antes del checkpoint final, así un
* proceso que se reinicia más seguido que VACUUM_INTERVAL también limpia. Corre antes de marcar el motor cerrado y con
* los trabajos de fondo vivos (el checkpointer sigue soltando a los escritores frenados). Una réplica, un motor
* cerrado o inutilizable y un disco lleno limpio no limpian ni cortan el cierre.
* @return error: solo un error de la limpieza que no sea de esos (p. ej. una página dañada)
**/
func (e *Engine) closeVacuum() error {
	if e.cfg.vacuumEvery <= 0 || e.isReplica() ||
		e.writes.Load()-e.metrics.vacuumWrites.Load() < e.cfg.vacuumWrites {
		return nil
	}
	err := e.vacuum()
	if errors.Is(err, ErrClosed) || errors.Is(err, ErrReadOnly) || errors.Is(err, ErrFailed) || errors.Is(err, ErrDiskFull) {
		return nil
	}
	return err
}
