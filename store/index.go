package store

import (
	"sync"

	"github.com/google/btree"
)

// btreeDegree: cuántos elementos agrupa cada nodo del árbol B. 32 es el valor habitual: nodos que caben en pocas
// líneas de caché y un árbol de pocos niveles (10 M de ids son unos 5).
const btreeDegree = 32

/**
* ref: Dónde está el último registro de un id: segmento, posición y largo en disco, y su lsn (para que gane el más
* nuevo aunque dos escrituras publiquen en otro orden). deleted marca una lápida: el último registro es un borrado.
**/
type ref struct {
	segment int
	offset  int64
	size    uint32
	deleted bool // cabe en el relleno tras size: la ref sigue ocupando 32 bytes
	lsn     uint64
}

/**
* entry: Un elemento del índice.
**/
type entry struct {
	id  string
	ref ref
}

/**
* lessEntry: El orden del índice: por id, byte a byte.
* @param a, b entry
* @return bool
**/
func lessEntry(a, b entry) bool {
	return a.id < b.id
}

/**
* index: El índice ordenado en memoria (un árbol B), seguro entre goroutines. Puede tener lápidas (ids cuyo último
* registro es un borrado) mientras queden escrituras pendientes de ese id: ver doc.go, «Borrados». live cuenta los ids
* que existen (sin lápidas).
**/
type index struct {
	mu   sync.RWMutex
	tree *btree.BTreeG[entry]
	live int
}

/**
* newIndex: Un índice vacío.
* @return *index
**/
func newIndex() *index {
	return &index{tree: btree.NewG(btreeDegree, lessEntry)}
}

/**
* set: Guarda la referencia de id (un valor o una lápida) si es más nueva (mayor lsn) que la que había.
* @param id string, r ref
**/
func (x *index) set(id string, r ref) {
	x.mu.Lock()
	defer x.mu.Unlock()
	old, had := x.tree.Get(entry{id: id})
	if had && old.ref.lsn > r.lsn {
		return
	}
	x.tree.ReplaceOrInsert(entry{id: id, ref: r})
	x.live += liveDelta(had && !old.ref.deleted, !r.deleted)
}

/**
* liveDelta: Cuánto cambia la cuenta de ids vivos cuando uno pasa de existir (before) a existir (after).
* @param before, after bool
* @return int
**/
func liveDelta(before, after bool) int {
	switch {
	case before && !after:
		return -1
	case !before && after:
		return 1
	}
	return 0
}

/**
* remove: Quita id del índice, sea valor o lápida. Al recuperar: un borrado leído del log.
* @param id string
**/
func (x *index) remove(id string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if old, had := x.tree.Delete(entry{id: id}); had && !old.ref.deleted {
		x.live--
	}
}

/**
* dropTombstone: Quita id si lo que hay es una lápida (cuando ya no quedan escrituras pendientes de ese id).
* @param id string
**/
func (x *index) dropTombstone(id string) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if e, ok := x.tree.Get(entry{id: id}); ok && e.ref.deleted {
		x.tree.Delete(e)
	}
}

/**
* get: La referencia del valor de id; una lápida cuenta como que no existe.
* @param id string
* @return ref, bool
**/
func (x *index) get(id string) (ref, bool) {
	r, ok := x.lookup(id)
	if !ok || r.deleted {
		return ref{}, false
	}
	return r, true
}

/**
* lookup: Lo que el índice tiene para id, lápida incluida.
* @param id string
* @return ref, bool
**/
func (x *index) lookup(id string) (ref, bool) {
	x.mu.RLock()
	defer x.mu.RUnlock()
	e, ok := x.tree.Get(entry{id: id})
	return e.ref, ok
}

/**
* count: Cuántos ids existen (sin contar lápidas).
* @return int
**/
func (x *index) count() int {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.live
}

/**
* entries: Cuántos elementos tiene el árbol, lápidas incluidas (para las pruebas).
* @return int
**/
func (x *index) entries() int {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.tree.Len()
}
