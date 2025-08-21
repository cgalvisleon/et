package ephemeral

import (
	"sync"
	"time"
)

type Instance struct {
	models     map[string]interface{}
	expiration time.Duration
	timer      *time.Timer
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
		expiration: expiration,
		timer:      nil,
	}
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

	go func() {
		s.timer = time.AfterFunc(s.expiration, func() {
			s.Del(key)
		})
	}()
}

/**
* Del
* @param key string
**/
func (s *Instance) Del(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
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

	if s.timer != nil {
		s.timer.Stop()
	}

	go func() {
		s.timer = time.AfterFunc(s.expiration, func() {
			s.Del(key)
		})
	}()

	return result, true
}
