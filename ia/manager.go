package ia

import (
	"errors"
	"sync"
	"time"

	"github.com/cgalvisleon/et/strs"
)

// DefaultIdleTTL is the time a KnowledgeBase can stay in memory without being
// accessed before the Manager evicts it automatically.
const DefaultIdleTTL = time.Hour

type kbEntry struct {
	kb    *KnowledgeBase
	timer *time.Timer
}

// Manager keeps a bounded set of KnowledgeBase instances in memory, keyed by id
// (typically a conversation/context id). Each entry carries its own idle timer so
// unused knowledge bases are unloaded and persisted (when a Store is set) after
// idleTTL of inactivity, without affecting the expiry of any other loaded entry.
// It also supports unloading a specific knowledge base on demand.
// @param store Store, idleTTL time.Duration
type Manager struct {
	mu      sync.RWMutex
	entries map[string]*kbEntry
	idleTTL time.Duration
	store   Store
}

/**
* NewManager: builds a Manager backed by store (may be nil for in-memory-only use)
* that evicts idle knowledge bases after idleTTL (DefaultIdleTTL when <= 0).
* @param store Store, idleTTL time.Duration
* @return *Manager
**/
func NewManager(store Store, idleTTL time.Duration) *Manager {
	if idleTTL <= 0 {
		idleTTL = DefaultIdleTTL
	}

	return &Manager{
		entries: make(map[string]*kbEntry),
		idleTTL: idleTTL,
		store:   store,
	}
}

/**
* Load: returns the knowledge base for id, loading it from memory if already present
* (refreshing its idle timer) or from the Store/creating it otherwise.
* @param id string
* @return *KnowledgeBase, error
**/
func (s *Manager) Load(id string) (*KnowledgeBase, error) {
	if strs.IsEmpty(id) {
		return nil, errors.New(MSG_KB_ID_REQUIRED)
	}

	s.mu.Lock()
	if entry, ok := s.entries[id]; ok {
		s.touchLocked(entry)
		s.mu.Unlock()
		return entry.kb, nil
	}
	s.mu.Unlock()

	kb, err := s.fetch(id)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if entry, ok := s.entries[id]; ok {
		s.touchLocked(entry)
		return entry.kb, nil
	}

	entry := &kbEntry{kb: kb}
	entry.timer = time.AfterFunc(s.idleTTL, func() { s.evict(id) })
	s.entries[id] = entry

	return kb, nil
}

/**
* fetch: loads a knowledge base from the Store, or creates an empty one when it does
* not exist yet or no Store is configured.
* @param id string
* @return *KnowledgeBase, error
**/
func (s *Manager) fetch(id string) (*KnowledgeBase, error) {
	if s.store == nil {
		return NewKnowledgeBase(id), nil
	}

	kb := &KnowledgeBase{}
	exists, err := s.store.Get(collectionKnowledgeBases, id, kb)
	if err != nil {
		return nil, err
	}

	if !exists {
		return NewKnowledgeBase(id), nil
	}

	kb.tokenIndex = make(map[string]map[string]bool)
	for _, fact := range kb.Facts {
		if fact.Status == FactActive {
			kb.indexFact(fact)
		}
	}

	return kb, nil
}

/**
* touchLocked: restarts an entry's idle timer. Caller must hold s.mu.
* @param entry *kbEntry
**/
func (s *Manager) touchLocked(entry *kbEntry) {
	entry.timer.Stop()
	id := entry.kb.ID
	entry.timer = time.AfterFunc(s.idleTTL, func() { s.evict(id) })
}

/**
* evict: removes id from memory and persists it through the Store, if any. It is the
* shared path for both automatic (idle timer) and manual (Unload) eviction.
* @param id string
**/
func (s *Manager) evict(id string) {
	s.mu.Lock()
	entry, ok := s.entries[id]
	if ok {
		entry.timer.Stop()
		delete(s.entries, id)
	}
	s.mu.Unlock()

	if !ok {
		return
	}

	if s.store != nil {
		_ = s.store.Set(collectionKnowledgeBases, id, id, entry.kb)
	}
}

/**
* Unload: evicts id from memory immediately (manual selection). It is a no-op if id
* is not loaded.
* @param id string
**/
func (s *Manager) Unload(id string) {
	s.evict(id)
}

/**
* IsLoaded: reports whether id currently has a knowledge base held in memory.
* @param id string
* @return bool
**/
func (s *Manager) IsLoaded(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.entries[id]
	return ok
}

/**
* Loaded: returns the ids of every knowledge base currently held in memory.
* @return []string
**/
func (s *Manager) Loaded() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]string, 0, len(s.entries))
	for id := range s.entries {
		result = append(result, id)
	}

	return result
}

/**
* Len: returns the number of knowledge bases currently held in memory.
* @return int
**/
func (s *Manager) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.entries)
}
