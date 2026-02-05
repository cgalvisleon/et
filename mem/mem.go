package mem

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
)

type Mem struct {
	items map[string]*Entry
	mu    *sync.RWMutex
}

var (
	conn *Mem
)

func Load() *Mem {
	result := &Mem{
		items: make(map[string]*Entry),
		mu:    &sync.RWMutex{},
	}

	return result
}

func init() {
	conn = Load()
}

/**
* Type
* @return string
**/
func (s *Mem) Type() string {
	return "mem"
}

/**
* Set
* @param key string, value interface{}, expiration time.Duration
* @return *Entry, error
**/
func (s *Mem) Set(key string, value interface{}, expiration time.Duration) (*Entry, error) {
	s.mu.RLock()
	item, ok := s.items[key]
	s.mu.RUnlock()
	if ok {
		_, err := item.Set(value, expiration)
		if err != nil {
			return nil, err
		}

		return item, nil
	}

	var err error
	item, err = New(key, value, expiration)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.items[key] = item
	s.mu.Unlock()

	clean := func() {
		s.Delete(key)
	}

	duration := expiration * time.Second
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	return item, nil
}

/**
* Delete
* @param key string
* @return bool
**/
func (s *Mem) Delete(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.items[key]
	if !ok {
		return false
	}

	delete(s.items, key)

	return true
}

/**
* Exists
* @param key string
* @return bool
**/
func (s *Mem) Exists(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.items[key]
	return ok
}

/**
* GetEntry
* @param key string, dest *Entry
* @return bool, error
**/
func (s *Mem) GetEntry(key string) (*Entry, bool) {
	s.mu.RLock()
	item, ok := s.items[key]
	s.mu.RUnlock()
	if ok {
		return item, true
	}

	return nil, false
}

/**
* Get
* @param key string
* @return interfase{}, error
**/
func (s *Mem) Get(key string) (interface{}, bool) {
	item, exists := s.GetEntry(key)
	if !exists {
		return nil, false
	}

	return item.Get(), true
}

/**
* GetStr
* @param key string
* @return string, bool
**/
func (s *Mem) GetStr(key string) (string, bool) {
	item, exists := s.GetEntry(key)
	if !exists {
		return "", false
	}

	return item.Str(), true
}

/**
* GetInt
* @param key string, def int
* @return int, bool, error
**/
func (s *Mem) GetInt(key string, def int) (int, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.Int()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetInt64
* @param key string, def int
* @return int, error, bool
**/
func (s *Mem) GetInt64(key string, def int64) (int64, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.Int64()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetFloat
* @param key string, def float64
* @return float64, bool, error
**/
func (s *Mem) GetFloat(key string, def float64) (float64, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.Float()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetBool
* @param key string, def bool
* @return bool, bool, error
**/
func (s *Mem) GetBool(key string, def bool) (bool, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.Bool()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetTime
* @param key string, def time.Time
* @return time.Time, bool, error
**/
func (s *Mem) GetTime(key string, def time.Time) (time.Time, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.Time()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetDuration
* @param key string, def time.Duration
* @return time.Duration, bool, error
**/
func (s *Mem) GetDuration(key string, def time.Duration) (time.Duration, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.Duration()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetJson
* @param key string, def et.Json
* @return et.Json, bool, error
**/
func (s *Mem) GetJson(key string, def et.Json) (et.Json, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.Json()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetArrayStr
* @param key string, def []string
* @return []string, bool, error
**/
func (s *Mem) GetArrayStr(key string, def []string) ([]string, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.ArrayStr()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetArrayInt
* @param key string, def []int
* @return []int, bool, error
**/
func (s *Mem) GetArrayInt(key string, def []int) ([]int, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.ArrayInt()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetArrayFloat
* @param key string, def []float64
* @return []float64, bool, error
**/
func (s *Mem) GetArrayFloat(key string, def []float64) ([]float64, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.ArrayFloat()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* GetArrayJson
* @param key string, def []et.Json
* @return []et.Json, bool, error
**/
func (s *Mem) GetArrayJson(key string, def []et.Json) ([]et.Json, bool, error) {
	item, exists := s.GetEntry(key)
	if !exists {
		return def, false, nil
	}

	result, err := item.ArrayJson()
	if err != nil {
		return def, false, err
	}

	return result, true, nil
}

/**
* More
* @param key string
* @param expiration time.Duration
* @return int
**/
func (s *Mem) More(key string, expiration time.Duration) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var err error
	var result int64
	item, ok := s.items[key]
	if !ok {
		result = 1
	} else {
		result, err = item.Int64()
		if err != nil {
			return 0, err
		}
		result++
	}

	s.Set(key, result, expiration)
	return result, nil
}

/**
* Clear
* @param match string
**/
func (s *Mem) Clear(match string) {
	matchPattern := func(substring, str string) bool {
		pattern := fmt.Sprintf(".*%s.*", regexp.QuoteMeta(substring))
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Println("Error compilando la expresión regular:", err)
			return false
		}
		return re.MatchString(str)
	}

	for key := range s.items {
		if matchPattern(match, key) {
			delete(s.items, key)
		}
	}
}

func (s *Mem) Empty() {
	s.Clear("")
}

/**
* Len
* @return int
**/
func (s *Mem) Len() int {
	return len(s.items)
}

/**
* Keys
* @return []string
**/
func (s *Mem) Keys() []string {
	keys := make([]string, 0, len(s.items))

	for key := range s.items {
		keys = append(keys, key)
	}

	return keys
}

/**
* Values
* @return []string
**/
func (s *Mem) Values() []string {
	values := make([]string, 0, len(s.items))

	for _, item := range s.items {
		str := item.Str()
		values = append(values, str)
	}

	return values
}
