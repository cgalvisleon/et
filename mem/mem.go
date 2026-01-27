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
	ch := make(chan *Item)
	go func() {
		s.mu.RLock()
		item, ok := s.items[key]
		s.mu.RUnlock()
		if ok {
			item.Set(value)
		} else {
			item = New(key, value)
			s.mu.Lock()
			s.items[key] = item
			s.mu.Unlock()
		}

		ch <- item
	}()

	clean := func() {
		s.Delete(key)
	}

	duration := expiration * time.Second
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	result := <-ch
	return result
}

/**
* Delete
* @param key string
* @return bool
**/
func (s *Mem) Delete(key string) bool {
	ch := make(chan bool)
	go func() {
		s.mu.Lock()
		_, ok := s.items[key]
		s.mu.Unlock()
		if !ok {
			ch <- false
			return
		}

		delete(s.items, key)
		ch <- true
	}()

	return <-ch
}

/**
* GetItem
* @param key string
* @return *Item, error
**/
func (s *Mem) GetItem(key string) (*Item, error) {
	type Result struct {
		result *Item
		err    error
	}

	ch := make(chan Result)
	go func() {
		s.mu.RLock()
		item, ok := s.items[key]
		s.mu.RUnlock()
		if ok {
			ch <- Result{
				result: item,
				err:    nil,
			}
			return
		}

		ch <- Result{
			result: nil,
			err:    fmt.Errorf("NotExists"),
		}
	}()

	result := <-ch
	return result.result, result.err
}

/**
* Get
* @param key string
* @return interfase{}, error
**/
func (s *Mem) Get(key string) (interface{}, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return nil, err
	}

	return item.Get(), nil
}

/**
* GetStr
* @param key string
* @return string
**/
func (s *Mem) GetStr(key string) (string, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return "", err
	}

	return item.Str(), nil
}

/**
* GetInt
* @param key string, def int
* @return int, error
**/
func (s *Mem) GetInt(key string, def int) (int, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.Int(), nil
}

/**
* GetInt64
* @param key string, def int
* @return int, error
**/
func (s *Mem) GetInt64(key string, def int64) (int64, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.Int64(), nil
}

/**
* GetFloat
* @param key string, def float64
* @return float64, error
**/
func (s *Mem) GetFloat(key string, def float64) (float64, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.Float(), nil
}

/**
* GetBool
* @param key string, def bool
* @return bool, error
**/
func (s *Mem) GetBool(key string, def bool) (bool, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.Bool(), nil
}

/**
* GetTime
* @param key string, def time.Time
* @return time.Time, error
**/
func (s *Mem) GetTime(key string, def time.Time) (time.Time, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.Time(), nil
}

/**
* GetDuration
* @param key string, def time.Duration
* @return time.Duration, error
**/
func (s *Mem) GetDuration(key string, def time.Duration) (time.Duration, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.Duration(), nil
}

/**
* GetJson
* @param key string, def et.Json
* @return et.Json, error
**/
func (s *Mem) GetJson(key string, def et.Json) (et.Json, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.Json(), nil
}

/**
* GetArrayStr
* @param key string, def []string
* @return []string, error
**/
func (s *Mem) GetArrayStr(key string, def []string) ([]string, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.ArrayStr(), nil
}

/**
* GetArrayInt
* @param key string, def []int
* @return []int, error
**/
func (s *Mem) GetArrayInt(key string, def []int) ([]int, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.ArrayInt(), nil
}

/**
* GetArrayFloat
* @param key string, def []float64
* @return []float64, error
**/
func (s *Mem) GetArrayFloat(key string, def []float64) ([]float64, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.ArrayFloat(), nil
}

/**
* GetArrayJson
* @param key string, def []et.Json
* @return []et.Json, error
**/
func (s *Mem) GetArrayJson(key string, def []et.Json) ([]et.Json, error) {
	item, err := s.GetItem(key)
	if err != nil {
		return def, err
	}

	return item.ArrayJson(), nil
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
