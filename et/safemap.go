package et

import "sync"

type SafeMap struct {
	mu   sync.RWMutex
	Data Json
}

func (m *SafeMap) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.Data[key]
	return value, ok
}

func (m *SafeMap) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Data[key] = value
}

func (m *SafeMap) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.Data, key)
}

type SafeData struct {
	mu   sync.RWMutex
	Data []Json
}

func (s *SafeData) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.Data)
}

func (s *SafeData) Get(index int) (Json, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 || index >= len(s.Data) {
		return nil, false
	}

	return s.Data[index], true
}

func (s *SafeData) Set(index int, value Json) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Data[index] = value
}

func (s *SafeData) Add(value Json) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Data = append(s.Data, value)
}

func (s *SafeData) Delete(index int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.Data) {
		return
	}

	s.Data = append(s.Data[:index], s.Data[index+1:]...)
}
