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
* nuevo aunque dos escrituras publiquen en otro orden).
**/
type ref struct {
	segment int
	offset  int64
	size    uint32
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
* index: El índice ordenado en memoria (un árbol B), seguro entre goroutines.
**/
type index struct {
	mu   sync.RWMutex
	tree *btree.BTreeG[entry]
}

/**
* newIndex: Un índice vacío.
* @return *index
**/
func newIndex() *index {
	return &index{tree: btree.NewG(btreeDegree, lessEntry)}
}

/**
* set: Guarda la referencia de id si es más nueva (mayor lsn) que la que había.
* @param id string, r ref
**/
func (x *index) set(id string, r ref) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if old, ok := x.tree.Get(entry{id: id}); ok && old.ref.lsn > r.lsn {
		return
	}
	x.tree.ReplaceOrInsert(entry{id: id, ref: r})
}

/**
* get: La referencia de id.
* @param id string
* @return ref, bool
**/
func (x *index) get(id string) (ref, bool) {
	x.mu.RLock()
	defer x.mu.RUnlock()
	e, ok := x.tree.Get(entry{id: id})
	return e.ref, ok
}

/**
* count: Cuántos ids hay.
* @return int
**/
func (x *index) count() int {
	x.mu.RLock()
	defer x.mu.RUnlock()
	return x.tree.Len()
}
