package ia

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

type memStore struct {
	mu    sync.Mutex
	saved map[string][]byte
}

func newMemStore() *memStore {
	return &memStore{saved: make(map[string][]byte)}
}

func (s *memStore) Set(collection, id, ownerId string, obj any) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved[collection+"/"+id] = data

	return nil
}

func (s *memStore) Get(collection, id string, dest any) (bool, error) {
	s.mu.Lock()
	data, ok := s.saved[collection+"/"+id]
	s.mu.Unlock()

	if !ok {
		return false, nil
	}

	return true, json.Unmarshal(data, dest)
}

func (s *memStore) Delete(collection, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.saved, collection+"/"+id)

	return nil
}

func (s *memStore) Query(query et.Json) (et.Items, error) {
	return et.Items{}, nil
}

func TestManagerIdleEviction(t *testing.T) {
	store := newMemStore()
	mgr := NewManager(store, 30*time.Millisecond)

	kb, err := mgr.Load("conv-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err := kb.AddFact("el cielo es azul", 1, nil); err != nil {
		t.Fatalf("AddFact: %v", err)
	}
	if !mgr.IsLoaded("conv-1") {
		t.Fatalf("expected conv-1 to be loaded")
	}

	time.Sleep(80 * time.Millisecond)

	if mgr.IsLoaded("conv-1") {
		t.Fatalf("expected conv-1 to be evicted after idle TTL")
	}

	reloaded, err := mgr.Load("conv-1")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Len() != 1 {
		t.Fatalf("expected reloaded kb to keep its fact, got %d facts", reloaded.Len())
	}
}

func TestManagerTouchResetsTimer(t *testing.T) {
	store := newMemStore()
	mgr := NewManager(store, 60*time.Millisecond)

	if _, err := mgr.Load("conv-2"); err != nil {
		t.Fatalf("Load: %v", err)
	}

	for range 3 {
		time.Sleep(30 * time.Millisecond)
		if _, err := mgr.Load("conv-2"); err != nil {
			t.Fatalf("Load: %v", err)
		}
	}

	if !mgr.IsLoaded("conv-2") {
		t.Fatalf("expected conv-2 to still be loaded due to repeated access")
	}
}

func TestManagerUnload(t *testing.T) {
	store := newMemStore()
	mgr := NewManager(store, time.Hour)

	if _, err := mgr.Load("conv-3"); err != nil {
		t.Fatalf("Load: %v", err)
	}

	mgr.Unload("conv-3")

	if mgr.IsLoaded("conv-3") {
		t.Fatalf("expected conv-3 to be unloaded")
	}

	if _, ok := store.saved[collectionKnowledgeBases+"/conv-3"]; !ok {
		t.Fatalf("expected conv-3 to be persisted on unload")
	}
}
