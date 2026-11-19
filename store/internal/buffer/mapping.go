package buffer

import "sync"

// partition es una parte del mapa Tag → búfer. Su lock es una hoja: con él tomado no se toma ningún otro, ni se
// espera a nada.
type partition struct {
	mu sync.RWMutex
	m  map[Tag]int32 // nunca más entradas que búferes: la memoria no crece con los datos
}

/**
* partitionOf elige la partición de un tag mezclando sus campos (multiplicación de Fibonacci).
* @param tag Tag
* @return *partition
**/
func (p *pool) partitionOf(tag Tag) *partition {
	h := uint64(tag.File)<<40 ^ uint64(tag.Fork)<<32 ^ uint64(tag.Block)
	h *= 0x9E3779B97F4A7C15
	return &p.parts[h>>57] // los 7 bits altos: NumPartitions = 128
}

/**
* lookup busca el búfer de tag.
* @param tag Tag
* @return int32: el búfer, bool: si estaba
**/
func (p *pool) lookup(tag Tag) (int32, bool) {
	part := p.partitionOf(tag)
	part.mu.RLock()
	id, ok := part.m[tag]
	part.mu.RUnlock()
	return id, ok
}

/**
* insert publica tag → id si tag no tenía búfer.
* @param tag Tag, id int32
* @return bool: false si ya tenía otro
**/
func (p *pool) insert(tag Tag, id int32) bool {
	part := p.partitionOf(tag)
	part.mu.Lock()
	defer part.mu.Unlock()
	if _, exists := part.m[tag]; exists {
		return false
	}
	part.m[tag] = id
	return true
}

/**
* remove quita tag del mapa si todavía apunta al búfer id (si no, ya es de otro).
* @param tag Tag, id int32
**/
func (p *pool) remove(tag Tag, id int32) {
	part := p.partitionOf(tag)
	part.mu.Lock()
	if cur, ok := part.m[tag]; ok && cur == id {
		delete(part.m, tag)
	}
	part.mu.Unlock()
}
