package store

import (
	"errors"
	"sync"

	"github.com/cgalvisleon/et/store/internal/btree"
	"github.com/cgalvisleon/et/store/internal/control"
	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
)

// frozenXIDs lleva el frozenXID de cada store que el motor ya leyó (de su página meta al abrirlo, o el que dejó una
// limpieza) y el del motor guardado en el control. Crece con los stores, no con las claves.
//
// Concurrencia: mu es una hoja.
type frozenXIDs struct {
	mu    sync.Mutex
	base  page.XID                 // el frozenXID del control al abrir: cota baja de todo store, leído o no
	files map[smgr.FileNo]page.XID // por número de archivo
}

/**
* set anota el frozenXID de un store, nunca por debajo del del motor al abrir (un store del formato anterior trae 0
* en su meta, pero ya estaba cubierto por ese frozenXID).
* @param file smgr.FileNo, frozen page.XID
* @return page.XID: lo anotado
**/
func (f *frozenXIDs) set(file smgr.FileNo, frozen page.XID) page.XID {
	f.mu.Lock()
	defer f.mu.Unlock()
	frozen = max(frozen, f.base)
	f.files[file] = frozen
	return frozen
}

/**
* forget olvida un store borrado o vaciado (su archivo viejo): deja de retener el frozenXID del motor.
* @param file smgr.FileNo
**/
func (f *frozenXIDs) forget(file smgr.FileNo) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.files, file)
}

/**
* atLeast es frozen acotado por debajo con el frozenXID del motor al abrir.
* @param frozen page.XID
* @return page.XID
**/
func (f *frozenXIDs) atLeast(frozen page.XID) page.XID {
	f.mu.Lock()
	defer f.mu.Unlock()
	return max(frozen, f.base)
}

/**
* min es el frozenXID del motor: el menor de los stores del catálogo (un store todavía no leído cuenta con el del
* control) y nunca más que oldest, el xid vivo más viejo.
* @param entries []control.Entry, oldest page.XID
* @return page.XID
**/
func (f *frozenXIDs) min(entries []control.Entry, oldest page.XID) page.XID {
	f.mu.Lock()
	defer f.mu.Unlock()
	frozen := oldest
	for _, entry := range entries {
		x, ok := f.files[entry.File]
		if !ok {
			x = f.base
		}
		frozen = min(frozen, x)
	}
	return frozen
}

/**
* engineFrozen calcula el frozenXID del motor para el checkpoint: con el valor de cada store anotado después de su
* KindSetFrozen, todo lo que lo justifica (los congelados de sus hojas y su meta) tiene lsn anterior al punto de
* rehacer que se fija después, así que el checkpoint lo deja en disco antes de guardar el control y truncar.
* @return page.XID
**/
func (e *Engine) engineFrozen() page.XID {
	return e.frozen.min(e.cat.Catalog().Entries, e.txs.OldestActive())
}

/**
* vacuumOptions arma la limpieza de un store: congela lo confirmado con xid < horizon − FREEZE_MIN_AGE y es agresiva
* (no salta las hojas solo todo-visibles) si el frozenXID del store quedó FREEZE_TABLE_AGE xids o más detrás del
* horizonte.
* @param horizon, frozen page.XID: el horizonte de la pasada y el frozenXID actual del store
* @return btree.VacuumOptions: sin Free
**/
func (e *Engine) vacuumOptions(horizon, frozen page.XID) btree.VacuumOptions {
	limit := page.XID(0)
	if age := page.XID(e.cfg.freezeMinAge); horizon > age {
		limit = horizon - age
	}
	aggressive := horizon > frozen && uint64(horizon-frozen) >= e.cfg.freezeTableAge
	return btree.VacuumOptions{Horizon: horizon, FreezeLimit: limit, Aggressive: aggressive}
}

/**
* freezeStore limpia el árbol del store y, si la pasada lo cubrió entero (Complete), sube su frozenXID en la meta;
* anota el resultado en el motor. Con vacuumMu y el store tomado en lectura. Un DDL que espera el store o Close cortan
* la pasada en la próxima hoja (btree.VacuumOptions.Stop): vuelve sin error y sin subir el frozenXID; la próxima
* limpieza sigue.
* @param horizon page.XID
* @return error
**/
func (s *Store) freezeStore(horizon page.XID) error {
	frozen, err := s.tree.FrozenXID()
	if err != nil {
		return err
	}
	frozen = s.e.frozen.atLeast(frozen)
	opts := s.e.vacuumOptions(horizon, frozen)
	opts.Free, opts.NextXID = s.freeDead, s.e.txs.NextXID
	opts.Stop = func() bool { return s.ddlWaiting.Load() > 0 || s.e.stopping() }
	st, err := s.tree.Vacuum(opts)
	s.e.metrics.pagesDeleted.Add(int64(st.Deleted))
	s.e.metrics.pagesRecycled.Add(int64(st.Recycled))
	if errors.Is(err, btree.ErrVacuumStopped) {
		return nil
	}
	if err != nil {
		return err
	}
	if st.Complete && st.FrozenXID > frozen {
		if err := s.tree.SetFrozenXID(st.FrozenXID); err != nil {
			return err
		}
		frozen = st.FrozenXID
	}
	s.e.frozen.set(s.file, frozen)
	return nil
}
