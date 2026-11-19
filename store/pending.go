package store

import "sync"

/**
* pendingOp: Las escrituras de un id reservadas que aún no terminan: cuántas son y qué hace la última (la de mayor
* lsn), que es el estado del id para quien decide una escritura condicional.
**/
type pendingOp struct {
	lsn     uint64
	deleted bool
	count   int
}

/**
* pending: Las escrituras reservadas y sin terminar, por id. Solo tiene los ids que se están escribiendo: es pequeño.
* Su lock va después de writeMu y antes del lock del índice (ver doc.go).
**/
type pending struct {
	mu  sync.Mutex
	ops map[string]pendingOp
}

/**
* current: El estado lógico de un id: el que dejó su última escritura reservada, termine o no.
**/
type current struct {
	lsn    uint64 // lsn de la última escritura del id; 0 si no hay rastro de él
	exists bool
}

/**
* newPending: Un registro vacío.
* @return *pending
**/
func newPending() *pending {
	return &pending{ops: map[string]pendingOp{}}
}

/**
* add: Anota una escritura reservada de id. Se llama con writeMu tomado, así la última anotada es la de mayor lsn.
* @param id string, lsn uint64, deleted bool
**/
func (p *pending) add(id string, lsn uint64, deleted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	op := p.ops[id]
	p.ops[id] = pendingOp{lsn: lsn, deleted: deleted, count: op.count + 1}
}

/**
* state: El estado lógico de id: la última escritura pendiente si la hay; si no, lo que dice el índice. Las dos
* fuentes se leen bajo p.mu, así una escritura que termina (pasa del registro al índice) no queda en medio.
* @param id string, x *index
* @return current, bool (si hay escrituras pendientes de id)
**/
func (p *pending) state(id string, x *index) (current, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if op, busy := p.ops[id]; busy {
		return current{lsn: op.lsn, exists: !op.deleted}, true
	}
	r, ok := x.lookup(id)
	return current{lsn: r.lsn, exists: ok && !r.deleted}, false
}

/**
* finish: Una escritura de id terminó: pasa su ref al índice (si se escribió) y deja de estar pendiente. Si era la
* última pendiente del id y lo que queda en el índice es una lápida, la lápida ya no hace falta y se quita.
* @param id string, r ref, written bool (false si WriteAt falló: el índice no cambia)
* @param x *index
**/
func (p *pending) finish(id string, r ref, written bool, x *index) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if written {
		x.set(id, r)
	}
	op := p.ops[id]
	op.count--
	if op.count > 0 {
		p.ops[id] = op
		return
	}
	delete(p.ops, id)
	x.dropTombstone(id)
}

/**
* size: Cuántos ids tienen escrituras pendientes (para las pruebas).
* @return int
**/
func (p *pending) size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.ops)
}
