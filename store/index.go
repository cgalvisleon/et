package store

import (
	"hash/maphash"
	"maps"
	"slices"
	"sync"
)

const indexShards = 64

/**
* indexShard: Fragmento del índice con su propio lock.
**/
type indexShard struct {
	mu sync.RWMutex
	m  map[string]*recordRef
}

/**
* shardedIndex: Índice en memoria repartido en fragmentos por hash del id, para que una
* escritura bloquee solo su fragmento y no a todos los lectores. Las operaciones sobre un id
* toman el lock de su fragmento; las que recorren todo toman todos, en orden.
**/
type shardedIndex struct {
	seed   maphash.Seed
	shards [indexShards]indexShard
}

/**
* newShardedIndex: Crea un índice vacío, o con el contenido de from.
* @param from map[string]*recordRef
* @return *shardedIndex
**/
func newShardedIndex(from map[string]*recordRef) *shardedIndex {
	result := &shardedIndex{seed: maphash.MakeSeed()}
	for i := range result.shards {
		result.shards[i].m = make(map[string]*recordRef, len(from)/indexShards)
	}
	for id, ref := range from {
		result.shard(id).m[id] = ref
	}
	return result
}

/**
* shard: Retorna el fragmento de id.
* @param id string
* @return *indexShard
**/
func (s *shardedIndex) shard(id string) *indexShard {
	return &s.shards[maphash.String(s.seed, id)%indexShards]
}

/**
* get: Retorna la referencia de id.
* @param id string
* @return *recordRef, bool
**/
func (s *shardedIndex) get(id string) (*recordRef, bool) {
	sh := s.shard(id)
	sh.mu.RLock()
	defer sh.mu.RUnlock()
	ref, exists := sh.m[id]
	return ref, exists
}

/**
* set: Guarda la referencia de id; retorna true si ya existía.
* @param id string, ref *recordRef
* @return bool
**/
func (s *shardedIndex) set(id string, ref *recordRef) bool {
	sh := s.shard(id)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	_, exists := sh.m[id]
	sh.m[id] = ref
	return exists
}

/**
* remove: Quita id; retorna true si existía.
* @param id string
* @return bool
**/
func (s *shardedIndex) remove(id string) bool {
	sh := s.shard(id)
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if _, exists := sh.m[id]; !exists {
		return false
	}
	delete(sh.m, id)
	return true
}

/**
* rlockAll: Toma todos los fragmentos para lectura, en orden: el índice queda fijo.
**/
func (s *shardedIndex) rlockAll() {
	for i := range s.shards {
		s.shards[i].mu.RLock()
	}
}

/**
* runlockAll: Suelta lo tomado por rlockAll.
**/
func (s *shardedIndex) runlockAll() {
	for i := range s.shards {
		s.shards[i].mu.RUnlock()
	}
}

/**
* len: Retorna el número de claves.
* @return int
**/
func (s *shardedIndex) len() int {
	s.rlockAll()
	defer s.runlockAll()
	result := 0
	for i := range s.shards {
		result += len(s.shards[i].m)
	}
	return result
}

/**
* clone: Retorna una copia del índice completo, tomada en un solo instante.
* @return map[string]*recordRef
**/
func (s *shardedIndex) clone() map[string]*recordRef {
	s.rlockAll()
	defer s.runlockAll()
	result := make(map[string]*recordRef)
	for i := range s.shards {
		maps.Copy(result, s.shards[i].m)
	}
	return result
}

/**
* sorted: Retorna las claves ordenadas asc/desc con offset y limit (limit <= 0 sin límite)
* y sus referencias, tomadas en un solo instante.
* @param asc bool, offset int, limit int
* @return []string, []*recordRef
**/
func (s *shardedIndex) sorted(asc bool, offset, limit int) ([]string, []*recordRef) {
	s.rlockAll()
	defer s.runlockAll()
	keys := []string{}
	for i := range s.shards {
		keys = slices.AppendSeq(keys, maps.Keys(s.shards[i].m))
	}

	offset = max(offset, 0)
	if offset >= len(keys) {
		return []string{}, []*recordRef{}
	}

	slices.Sort(keys)
	if !asc {
		slices.Reverse(keys)
	}

	end := len(keys)
	if limit > 0 {
		end = min(offset+limit, end)
	}

	keys = keys[offset:end]
	refs := make([]*recordRef, len(keys))
	for i, id := range keys {
		refs[i] = s.shard(id).m[id]
	}
	return keys, refs
}

/**
* each: Llama a fn por cada clave con el índice fijo (todos los fragmentos tomados).
* fn no debe usar el índice.
* @param fn func(id string, ref *recordRef)
**/
func (s *shardedIndex) each(fn func(id string, ref *recordRef)) {
	s.rlockAll()
	defer s.runlockAll()
	for i := range s.shards {
		for id, ref := range s.shards[i].m {
			fn(id, ref)
		}
	}
}
