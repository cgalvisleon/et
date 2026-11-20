package btree

import (
	"sync"
	"sync/atomic"
	"time"
)

// opGuard cuenta las operaciones del árbol en vuelo (Lookup, Exists, Write, cada Next de un Cursor, Verify) por época,
// para que el reciclado de páginas borradas espere a las que pudieron leer un enlace a una de ellas antes de que se
// desenganchara (ver «Borrado de páginas» en doc.go). Es de tamaño fijo: dos contadores y la época. Entrar cuesta dos
// operaciones atómicas y nunca espera.
type opGuard struct {
	epoch atomic.Uint64
	count [2]atomic.Int64
	mu    sync.Mutex // un drain a la vez
}

// guardPoll es cada cuánto mira drain si terminaron las operaciones de la época anterior.
const guardPoll = 50 * time.Microsecond

/**
* enter anota una operación en la época vigente. Si la época cambió entre leerla y contarse, se descuenta y repite:
* así drain nunca deja pasar una operación que entró antes de cambiarla sin esperarla.
* @return uint64: la época, para leave
**/
func (g *opGuard) enter() uint64 {
	for {
		e := g.epoch.Load()
		g.count[e&1].Add(1)
		if g.epoch.Load() == e {
			return e
		}
		g.count[e&1].Add(-1)
	}
}

/**
* leave termina una operación que entró en la época e.
* @param e uint64
**/
func (g *opGuard) leave(e uint64) {
	g.count[e&1].Add(-1)
}

/**
* drain espera, hasta timeout, a que terminen todas las operaciones que entraron antes de llamarlo: primero las de la
* época anterior (si un drain previo se cortó por tiempo), después cambia de época y espera a las de la vigente. Quien
* llama no tiene páginas bloqueadas.
* @param timeout time.Duration
* @return bool: false si se acabó el tiempo (no se puede reciclar todavía)
**/
func (g *opGuard) drain(timeout time.Duration) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	deadline := time.Now().Add(timeout)
	e := g.epoch.Load()
	if !g.wait(&g.count[(e+1)&1], deadline) {
		return false
	}
	g.epoch.Store(e + 1)
	return g.wait(&g.count[e&1], deadline)
}

/**
* wait espera hasta deadline a que el contador llegue a cero.
* @param c *atomic.Int64, deadline time.Time
* @return bool: si llegó
**/
func (g *opGuard) wait(c *atomic.Int64, deadline time.Time) bool {
	for c.Load() != 0 {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(guardPoll)
	}
	return true
}
