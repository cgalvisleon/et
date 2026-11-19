package store

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/store/internal/xact"
)

// Snapshot es una foto fija de todos los stores del motor para leer, sin bloquear a los escritores. Mientras siga
// abierto cuenta en el horizonte de la limpieza (xact.Manager.OldestActive), que no quita las versiones que podría
// ver. Se puede usar desde varias goroutines a la vez.
type Snapshot struct {
	e     *Engine
	snap  xact.Snapshot
	id    uint64      // su número en xact (ReleaseSnapshot)
	timer *time.Timer // lo cierra al pasar su tiempo máximo (nil sin tiempo máximo)

	// mu: las lecturas lo toman en lectura; cerrarlo, en escritura (una lectura en curso termina antes de soltar
	// el horizonte).
	mu  sync.RWMutex
	err error // nil mientras está abierto; ErrTxDone tras Close, ErrTimeout si pasó su tiempo
}

/**
* snapshotFor toma un snapshot de solo lectura y lo registra en el horizonte. Si el motor está cerrado o
* inutilizable, el snapshot nace cerrado con ese error.
* @param timeout time.Duration: tiempo máximo; 0 o menos = sin tiempo máximo
* @return *Snapshot
**/
func (e *Engine) snapshotFor(timeout time.Duration) *Snapshot {
	v := &Snapshot{e: e}
	if err := e.enter(); err != nil {
		v.err = err
		return v
	}
	defer e.leave()
	v.snap, v.id = e.txs.TakeSnapshot()
	if timeout > 0 {
		v.mu.Lock() // el temporizador puede disparar antes de que se guarde
		v.timer = time.AfterFunc(timeout, func() { v.release(ErrTimeout) })
		v.mu.Unlock()
	}
	return v
}

/**
* release cierra el snapshot con why (si seguía abierto) y lo quita del horizonte.
* @param why error: ErrTxDone (Close) o ErrTimeout
**/
func (v *Snapshot) release(why error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err != nil {
		return
	}
	v.err = why
	if v.timer != nil {
		v.timer.Stop()
	}
	v.e.txs.ReleaseSnapshot(v.id)
}

/**
* close cierra el snapshot. Cerrarlo dos veces, o después de que pasó su tiempo, no hace nada.
* @return error
**/
func (v *Snapshot) close() error {
	v.release(ErrTxDone)
	return nil
}

/**
* readAt devuelve la versión de id que ve el snapshot v.
* @param v *Snapshot, id string
* @return []byte: copia del valor, bool: si existe, error: ErrOtherEngine, ErrTxDone, ErrTimeout…
**/
func (s *Store) readAt(v *Snapshot, id string) ([]byte, bool, error) {
	if v.e != s.e {
		return nil, false, ErrOtherEngine
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.err != nil {
		return nil, false, v.err
	}
	cur, found, err := s.readVersion(id, &v.snap)
	return cur.value, found, err
}
