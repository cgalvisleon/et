package store

import (
	"sync"
	"time"
)

// timers son los temporizadores vivos de los lotes y snapshots abiertos (TRANSACTION_TIMEOUT): uno por transacción
// abierta, así que crece con la concurrencia, no con las claves. Close los detiene todos: un temporizador sin
// detener retiene su lote o snapshot, y con él el motor entero (caché de páginas incluida), hasta que vence.
type timers struct {
	mu   sync.Mutex
	live map[*time.Timer]struct{}
	done bool // el motor cerró: no se arman más
}

/**
* after arma f para dentro de d y lo registra; al dispararse se borra solo. Con el motor cerrado no arma nada.
* @param d time.Duration, f func()
* @return *time.Timer: nil si el motor ya cerró
**/
func (ts *timers) after(d time.Duration, f func()) *time.Timer {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.done {
		return nil
	}
	if ts.live == nil {
		ts.live = map[*time.Timer]struct{}{}
	}
	var t *time.Timer
	t = time.AfterFunc(d, func() {
		ts.mu.Lock() // espera a que after termine: solo entonces t está asignado
		delete(ts.live, t)
		ts.mu.Unlock()
		f()
	})
	ts.live[t] = struct{}{}
	return t
}

/**
* stop detiene t y lo borra del registro. t nil no hace nada.
* @param t *time.Timer
**/
func (ts *timers) stop(t *time.Timer) {
	if t == nil {
		return
	}
	t.Stop()
	ts.forget(t)
}

/**
* forget borra t del registro sin detenerlo.
* @param t *time.Timer
**/
func (ts *timers) forget(t *time.Timer) {
	ts.mu.Lock()
	delete(ts.live, t)
	ts.mu.Unlock()
}

/**
* closeAll detiene todos los temporizadores vivos y no deja armar más (Close del motor).
**/
func (ts *timers) closeAll() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	for t := range ts.live {
		t.Stop()
	}
	ts.live, ts.done = nil, true
}
