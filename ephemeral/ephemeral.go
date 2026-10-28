// Package ephemeral is an in-memory map whose keys expire when they are not read for a while (a Get renews them).
// Use it for short-lived state that must vanish when unused; mem keeps a value for a fixed TTL and dt caches in Redis.
package ephemeral

import (
	"sync"
	"time"
)

// Instance: an in-memory map whose keys expire after expiration without being read; a Get renews the key.
type Instance struct {
	models     map[string]interface{}
	timers     map[string]*time.Timer
	expiration time.Duration
	mu         sync.Mutex
}

/**
* NewInstance
* @param expiration time.Duration
* @return *Instance
**/
func NewInstance(expiration time.Duration) *Instance {
	return &Instance{
		models:     make(map[string]interface{}),
		timers:     make(map[string]*time.Timer),
		expiration: expiration,
	}
}

/**
* arm: Starts (or restarts) the expiration timer of key; the caller holds mu.
* @param key string
**/
func (s *Instance) arm(key string) {
	if timer, ok := s.timers[key]; ok {
		timer.Stop()
	}
	s.timers[key] = time.AfterFunc(s.expiration, func() {
		s.Del(key)
	})
}

/**
* Set
* @param key string
* @param value interface{}
**/
func (s *Instance) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models[key] = value
	s.arm(key)
}

/**
* Del
* @param key string
**/
func (s *Instance) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if timer, ok := s.timers[key]; ok {
		timer.Stop()
		delete(s.timers, key)
	}
	delete(s.models, key)
}

/**
* Get
* @param key string
* @return interface{}
* @return bool
**/
func (s *Instance) Get(key string) (interface{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result, ok := s.models[key]
	if !ok {
		return nil, false
	}

	s.arm(key)
	return result, true
}
