package store

import (
	"sync"

	"github.com/cgalvisleon/et/store/internal/page"
	"github.com/cgalvisleon/et/store/internal/smgr"
	"github.com/cgalvisleon/et/store/internal/wal"
)

// counts son los contadores de Count vivos, por número de archivo de store: lo que confirmaron todas las
// escrituras hasta el último lsn reservado. Se suman bajo el writeMu del WAL (en el underLock de Reserve), así una
// copia tomada en BeginCheckpoint es exacta al punto de rehacer. mu es una hoja. Crece con los stores, no con las
// claves.
type counts struct {
	mu sync.Mutex
	m  map[smgr.FileNo]int64
}

/**
* newCounts crea los contadores.
* @return *counts
**/
func newCounts() *counts {
	return &counts{m: map[smgr.FileNo]int64{}}
}

/**
* set fija el contador de un store (al cargar el catálogo o al crearlo).
* @param file smgr.FileNo, n int64
**/
func (c *counts) set(file smgr.FileNo, n int64) {
	c.mu.Lock()
	c.m[file] = n
	c.mu.Unlock()
}

/**
* add suma delta al contador de un store; un store que no está (borrado) se ignora.
* @param file smgr.FileNo, delta int64
**/
func (c *counts) add(file smgr.FileNo, delta int64) {
	c.mu.Lock()
	if _, ok := c.m[file]; ok {
		c.m[file] += delta
	}
	c.mu.Unlock()
}

/**
* get devuelve el contador de un store.
* @param file smgr.FileNo
* @return int64
**/
func (c *counts) get(file smgr.FileNo) int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.m[file]
}

/**
* remove olvida el contador de un store borrado.
* @param file smgr.FileNo
**/
func (c *counts) remove(file smgr.FileNo) {
	c.mu.Lock()
	delete(c.m, file)
	c.mu.Unlock()
}

/**
* snapshot copia todos los contadores (el checkpoint, bajo el writeMu del WAL).
* @return map[smgr.FileNo]int64
**/
func (c *counts) snapshot() map[smgr.FileNo]int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	cp := make(map[smgr.FileNo]int64, len(c.m))
	for file, n := range c.m {
		cp[file] = n
	}
	return cp
}

// walProxy es el wal.Flusher que reciben la caché y el archivo de estados: se crean antes que el WAL, porque la
// recuperación los usa mientras wal.Open recorre los registros. Hasta que se conecta el WAL, todo cuenta como
// durable: lo que se rehace se leyó del WAL en disco (a confirmar: PostgreSQL hace lo mismo; un corte del sistema
// operativo durante la recuperación podría perder la cola del WAL leída sin fsync después de escribir una página).
type walProxy struct {
	mu  sync.RWMutex
	log wal.Flusher
}

/**
* connect conecta el WAL ya abierto.
* @param log wal.Flusher
**/
func (p *walProxy) connect(log wal.Flusher) {
	p.mu.Lock()
	p.log = log
	p.mu.Unlock()
}

/**
* target devuelve el WAL conectado, o nil durante la recuperación.
* @return wal.Flusher
**/
func (p *walProxy) target() wal.Flusher {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.log
}

/**
* Durable es el lsn durable del WAL; durante la recuperación, todos.
* @return page.LSN
**/
func (p *walProxy) Durable() page.LSN {
	if log := p.target(); log != nil {
		return log.Durable()
	}
	return ^page.LSN(0)
}

/**
* WaitDurable espera al WAL; durante la recuperación vuelve enseguida.
* @param lsn page.LSN
* @return error
**/
func (p *walProxy) WaitDurable(lsn page.LSN) error {
	if log := p.target(); log != nil {
		return log.WaitDurable(lsn)
	}
	return nil
}
