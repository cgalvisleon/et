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
	locks map[string]*sync.RWMutex
}

var conn *Mem

func Load() (*Mem, error) {
	result := &Mem{
		items: make(map[string]*Item),
		locks: make(map[string]*sync.RWMutex),
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
* lock return a lock
* @param tag string
* @return *sync.RWMutex
**/
func (s *Mem) lock(tag string) *sync.RWMutex {
	if s.locks[tag] == nil {
		s.locks[tag] = &sync.RWMutex{}
	}

	return s.locks[tag]
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
func (s *Mem) Set(key string, value interface{}, expiration time.Duration) interface{} {
	lock := s.lock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := s.items[key]
	if ok {
		item.Set(value)
	} else {
		item = New(key, value)
		s.items[key] = item
	}

	clean := func() {
		s.Del(key)
	}

	duration := expiration * time.Second
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	return value
}

/**
* Get
* @param key string, def string
* @return string, error
**/
func (s *Mem) Get(key string, def string) (string, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.Str(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetInt
* @param key string, def int
* @return int, error
**/
func (s *Mem) GetInt(key string, def int) (int, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.Int(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetInt64
* @param key string, def int
* @return int, error
**/
func (s *Mem) GetInt64(key string, def int64) (int64, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.Int64(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetFloat
* @param key string, def float64
* @return float64, error
**/
func (s *Mem) GetFloat(key string, def float64) (float64, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.Float(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetBool
* @param key string, def bool
* @return bool, error
**/
func (s *Mem) GetBool(key string, def bool) (bool, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.Bool(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetTime
* @param key string, def time.Time
* @return time.Time, error
**/
func (s *Mem) GetTime(key string, def time.Time) (time.Time, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.Time(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetDuration
* @param key string, def time.Duration
* @return time.Duration, error
**/
func (s *Mem) GetDuration(key string, def time.Duration) (time.Duration, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.Duration(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetJson
* @param key string, def et.Json
* @return et.Json, error
**/
func (s *Mem) GetJson(key string, def et.Json) (et.Json, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.Json(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetMap
* @param key string, def map[string]interface{}
* @return map[string]interface{}, error
**/
func (s *Mem) GetMap(key string, def map[string]interface{}) (map[string]interface{}, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.Map(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetArrayMap
* @param key string, def []map[string]interface{}
* @return []map[string]interface{}, error
**/
func (s *Mem) GetArrayMap(key string, def []map[string]interface{}) ([]map[string]interface{}, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.ArrayMap(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetArrayStr
* @param key string, def []string
* @return []string, error
**/
func (s *Mem) GetArrayStr(key string, def []string) ([]string, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.ArrayStr(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetArrayInt
* @param key string, def []int
* @return []int, error
**/
func (s *Mem) GetArrayInt(key string, def []int) ([]int, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.ArrayInt(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetArrayFloat
* @param key string, def []float64
* @return []float64, error
**/
func (s *Mem) GetArrayFloat(key string, def []float64) ([]float64, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.ArrayFloat(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetArrayTime
* @param key string, def []time.Time
* @return []time.Time, error
**/
func (s *Mem) GetArrayTime(key string, def []time.Time) ([]time.Time, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.ArrayTime(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetArrayDuration
* @param key string, def []time.Duration
* @return []time.Duration, error
**/
func (s *Mem) GetArrayDuration(key string, def []time.Duration) ([]time.Duration, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.ArrayDuration(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* GetArrayJson
* @param key string, def []et.Json
* @return []et.Json, error
**/
func (s *Mem) GetArrayJson(key string, def []et.Json) ([]et.Json, error) {
	lock := s.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	item, ok := s.items[key]
	if ok {
		return item.ArrayJson(), nil
	}

	return def, fmt.Errorf("IsNil")
}

/**
* Del
* @param key string
* @return bool
**/
func (s *Mem) Del(key string) bool {
	lock := s.lock(key)
	lock.Lock()
	defer lock.Unlock()

	if _, ok := s.items[key]; !ok {
		return false
	}

	delete(s.items, key)

	return true
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
