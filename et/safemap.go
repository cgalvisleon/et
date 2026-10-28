package et

import (
	"encoding/json"
	"sync"
)

/**
* SafeMap: A Json guarded by a RWMutex, for concurrent Get/Set/Delete.
**/
type SafeMap struct {
	mu   sync.RWMutex
	Data Json
}

/**
* NewSafeMap: A SafeMap ready to use
* @return *SafeMap
**/
func NewSafeMap() *SafeMap {
	return &SafeMap{Data: Json{}}
}

/**
* Get: The value of key and whether it exists.
* @param key string
* @return interface{}, bool
**/
func (m *SafeMap) Get(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.Data[key]
	return value, ok
}

/**
* Set: Stores value under key (a nil map is created).
* @param key string, value interface{}
**/
func (m *SafeMap) Set(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Data == nil {
		m.Data = Json{}
	}
	m.Data[key] = value
}

/**
* Delete: Removes key.
* @param key string
**/
func (m *SafeMap) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.Data, key)
}

/**
* SafeData: A list of Json guarded by a RWMutex (a log: Add, Trim); serialized as the list itself.
**/
type SafeData struct {
	Data []Json       `json:"data"`
	mu   sync.RWMutex `json:"-"`
}

/**
* NewSafeData: An empty SafeData ready to use.
* @return *SafeData
**/
func NewSafeData() *SafeData {
	return &SafeData{
		Data: make([]Json, 0),
		mu:   sync.RWMutex{},
	}
}

/**
* Len: How many items the list has.
* @return int
**/
func (s *SafeData) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.Data)
}

/**
* Get: The item at index (0-based) and whether it exists.
* @param index int
* @return Json, bool
**/
func (s *SafeData) Get(index int) (Json, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if index < 0 || index >= len(s.Data) {
		return nil, false
	}

	return s.Data[index], true
}

/**
* Set: Replaces the item at index; an index out of range is ignored (false)
* @param index int, value Json
* @return bool
**/
func (s *SafeData) Set(index int, value Json) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.Data) {
		return false
	}

	s.Data[index] = value
	return true
}

/**
* Add: Appends value to the list.
* @param value Json
**/
func (s *SafeData) Add(value Json) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Data = append(s.Data, value)
}

/**
* Delete: Removes the item at index; out of range is ignored.
* @param index int
**/
func (s *SafeData) Delete(index int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if index < 0 || index >= len(s.Data) {
		return
	}

	s.Data = append(s.Data[:index], s.Data[index+1:]...)
}

/**
* Trim: Keeps only the last max items (the newest ones, for a log); max <= 0 empties the list
* @param max int
* @return int: how many were removed
**/
func (s *SafeData) Trim(max int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if max < 0 {
		max = 0
	}
	removed := len(s.Data) - max
	if removed <= 0 {
		return 0
	}

	s.Data = append(s.Data[:0:0], s.Data[removed:]...)
	return removed
}

/**
* All: A copy of the list (safe to read while others add); empty on a nil SafeData
* @return []Json
**/
func (s *SafeData) All() []Json {
	if s == nil {
		return []Json{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Json, len(s.Data))
	copy(result, s.Data)
	return result
}

/**
* MarshalJSON: The list itself ([...]), not {data: [...]}
* @return []byte, error
**/
func (s *SafeData) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.All())
}

/**
* UnmarshalJSON: A list, or {data: [...]} as it was saved before
* @param data []byte
* @return error
**/
func (s *SafeData) UnmarshalJSON(data []byte) error {
	list := []Json{}
	if err := json.Unmarshal(data, &list); err != nil {
		old := struct {
			Data []Json `json:"data"`
		}{}
		if err := json.Unmarshal(data, &old); err != nil {
			return err
		}
		list = old.Data
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data = list
	return nil
}
