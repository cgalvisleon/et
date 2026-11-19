package store

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/page"
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
* vacuum limpia el árbol del store (serializado por vacuumMu). Mientras dura tiene el store tomado en lectura: un
* DropStore espera a que termine, y uno que ya empezó deja el store en ErrStoreDropped y no se limpia.
* @param horizon page.XID: el xid vivo más viejo al empezar la limpieza del motor
* @return error: ErrStoreDropped, ErrClosed, ErrFailed o el error de una página dañada
**/
func (s *Store) vacuum(horizon page.XID) error {
	s.vacuumMu.Lock()
	defer s.vacuumMu.Unlock()
	if err := s.begin(); err != nil {
		return err
	}
	defer s.end()
	_, err := s.tree.Vacuum(btree.VacuumOptions{Horizon: horizon, Free: s.freeDead})
	return s.e.readError(err)
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
* VACUUM_THRESHOLD escrituras desde la última (0 = en cada vuelta), limpia todos los stores. Termina con Close o
* cuando el motor quedó inutilizable; Close la espera antes del checkpoint final.
**/
func (e *Engine) vacuumer() {
	defer e.bg.Done()
	ticker := time.NewTicker(e.cfg.vacuumEvery)
	defer ticker.Stop()
	var last int64
	for {
		select {
		case <-e.stop:
			return
		case <-ticker.C:
			n := e.writes.Load()
			if n-last < e.cfg.vacuumWrites {
				continue
			}
			last = n
			if err := e.vacuum(); errors.Is(err, ErrClosed) || errors.Is(err, ErrFailed) {
				return
			}
		}
	}
}
