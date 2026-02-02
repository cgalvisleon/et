package mem

import (
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
)

type Mem struct {
	items map[string]*Item
	mu    *sync.RWMutex
}

var conn *Mem

func Load() (*Mem, error) {
	result := &Mem{
		items: make(map[string]*Item),
		mu:    &sync.RWMutex{},
	}

	return result, nil
}

func init() {
	if conn != nil {
		return
	}

	var err error
	conn, err = Load()
	if err != nil {
		logs.Alert(err)
		return
	}
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
* @return interface{}
**/
func (s *Mem) Set(key string, value interface{}, expiration time.Duration) *Item {
	s.mu.RLock()
	item, ok := s.items[key]
	s.mu.RUnlock()
	if ok {
		item.Set(value, expiration)
	} else {
		item = New(key, value, expiration)
		s.mu.Lock()
		s.items[key] = item
		s.mu.Unlock()
	}

	clean := func() {
		s.Delete(key)
	}

	duration := expiration * time.Second
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	return item
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
* GetItem
* @param key string, dest *Item
* @return bool, error
**/
func (s *Mem) GetItem(key string) (*Item, bool) {
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
	item, exists := s.GetItem(key)
	if !exists {
		return nil, false
	}

	return item.Get(), true
}

/**
* GetStr
* @param key string
* @return string
**/
func (s *Mem) GetStr(key string) (string, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return "", false
	}

	return item.Str(), true
}

/**
* GetInt
* @param key string, def int
* @return int, bool
**/
func (s *Mem) GetInt(key string, def int) (int, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.Int(), true
}

/**
* GetInt64
* @param key string, def int
* @return int, error
**/
func (s *Mem) GetInt64(key string, def int64) (int64, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.Int64(), true
}

/**
* GetFloat
* @param key string, def float64
* @return float64, bool
**/
func (s *Mem) GetFloat(key string, def float64) (float64, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.Float(), true
}

/**
* GetBool
* @param key string, def bool
* @return bool, bool
**/
func (s *Mem) GetBool(key string, def bool) (bool, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.Bool(), true
}

/**
* GetTime
* @param key string, def time.Time
* @return time.Time, bool
**/
func (s *Mem) GetTime(key string, def time.Time) (time.Time, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.Time(), true
}

/**
* GetDuration
* @param key string, def time.Duration
* @return time.Duration, bool
**/
func (s *Mem) GetDuration(key string, def time.Duration) (time.Duration, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.Duration(), true
}

/**
* GetJson
* @param key string, def et.Json
* @return et.Json, error
**/
func (s *Mem) GetJson(key string, def et.Json) (et.Json, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.Json(), true
}

/**
* GetArrayStr
* @param key string, def []string
* @return []string, bool
**/
func (s *Mem) GetArrayStr(key string, def []string) ([]string, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.ArrayStr(), true
}

/**
* GetArrayInt
* @param key string, def []int
* @return []int, bool
**/
func (s *Mem) GetArrayInt(key string, def []int) ([]int, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.ArrayInt(), true
}

/**
* GetArrayFloat
* @param key string, def []float64
* @return []float64, bool
**/
func (s *Mem) GetArrayFloat(key string, def []float64) ([]float64, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.ArrayFloat(), true
}

/**
* GetArrayJson
* @param key string, def []et.Json
* @return []et.Json, bool
**/
func (s *Mem) GetArrayJson(key string, def []et.Json) ([]et.Json, bool) {
	item, exists := s.GetItem(key)
	if !exists {
		return def, false
	}

	return item.ArrayJson(), true
}

/**
* More
* @param key string
* @param expiration time.Duration
* @return int
**/
func (s *Mem) More(key string, expiration time.Duration) int64 {
	item, ok := s.items[key]
	if !ok {
		s.Set(key, "0", expiration)
		return 0
	} else {
		result := item.Int64() + 1
		str := strconv.FormatInt(result, 10)
		s.Set(key, str, expiration)
		return result
	}
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
