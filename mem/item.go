package mem

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

type Item struct {
	Datemake   time.Time
	Dateupdate time.Time
	Key        string
	Value      interface{}
	lock       sync.RWMutex
}

/**
* New create new item
* @param key string
* @param value interface{}
* @return *Item
**/
func New(key string, value interface{}) *Item {
	now := timezone.Now()
	return &Item{
		Datemake:   now,
		Dateupdate: now,
		Key:        key,
		Value:      value,
		lock:       sync.RWMutex{},
	}
}

/**
* Set a value in item
* @param value interface{}
* @return interface{}
**/
func (i *Item) Set(value interface{}) interface{} {
	i.lock.Lock()
	defer i.lock.Unlock()

	i.Dateupdate = timezone.Now()
	i.Value = value

	return value
}

/**
* Get a value from item
* @return interface{}
**/
func (i *Item) Get() interface{} {
	i.lock.RLock()
	defer i.lock.RUnlock()

	return i.Value
}

/**
* Str return the value of item
* @return string
**/
func (i *Item) Str() string {
	result := i.Get()
	val, ok := result.(string)
	if !ok {
		return ""
	}

	return val
}

/**
* Int return the value of item
* @return int
**/
func (i *Item) Int() int {
	result := i.Get()
	val, ok := result.(int)
	if !ok {
		return 0
	}

	return val
}

/**
* Int64 return the value of item
* @return int64
**/
func (i *Item) Int64() int64 {
	result := i.Get()
	val, ok := result.(int64)
	if !ok {
		return 0
	}

	return val
}

/**
* Float return the value of item
* @return float64
**/
func (i *Item) Float() float64 {
	result := i.Get()
	val, ok := result.(float64)
	if !ok {
		return 0
	}

	return val
}

/**
* Bool return the value of item
* @return bool
**/
func (i *Item) Bool() bool {
	result := i.Get()
	val, ok := result.(bool)
	if !ok {
		return false
	}

	return val
}

/**
* Time return the value of item
* @return time.Time
**/
func (i *Item) Time() time.Time {
	result := i.Get()
	val, ok := result.(time.Time)
	if !ok {
		return time.Time{}
	}

	return val
}

/**
* Duration return the value of item
* @return time.Duration
**/
func (i *Item) Duration() time.Duration {
	result := i.Get()
	val, ok := result.(time.Duration)
	if !ok {
		return time.Duration(0)
	}

	return val
}

/**
* Json return the value of item
* @return et.Json
**/
func (i *Item) Json() et.Json {
	result := i.Get()
	val, ok := result.(et.Json)
	if !ok {
		return et.Json{}
	}

	return val
}

/**
* Map return the value of item
* @return map[string]interface{}
**/
func (i *Item) Map() map[string]interface{} {
	result := i.Get()
	val, ok := result.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	return val
}

/**
* ArrayMap return the value of item
* @return []interface{}
**/
func (i *Item) ArrayMap() []map[string]interface{} {
	result := i.Get()
	val, ok := result.([]map[string]interface{})
	if !ok {
		return []map[string]interface{}{}
	}

	return val
}

/**
* ArrayStr return the value of item
* @return []string
**/
func (i *Item) ArrayStr() []string {
	result := i.Get()
	val, ok := result.([]string)
	if !ok {
		return []string{}
	}

	return val
}

/**
* ArrayInt return the value of item
* @return []int
**/
func (i *Item) ArrayInt() []int {
	result := i.Get()
	val, ok := result.([]int)
	if !ok {
		return []int{}
	}

	return val
}

/**
* ArrayFloat return the value of item
* @return []float64
**/
func (i *Item) ArrayFloat() []float64 {
	result := i.Get()
	val, ok := result.([]float64)
	if !ok {
		return []float64{}
	}

	return val
}

/**
* ArrayTime return the value of item
* @return []time.Time
**/
func (i *Item) ArrayTime() []time.Time {
	result := i.Get()
	val, ok := result.([]time.Time)
	if !ok {
		return []time.Time{}
	}

	return val
}

/**
* ArrayDuration return the value of item
* @return []time.Duration
**/
func (i *Item) ArrayDuration() []time.Duration {
	result := i.Get()
	val, ok := result.([]time.Duration)
	if !ok {
		return []time.Duration{}
	}

	return val
}

/**
* ArrayJson return the value of item
* @return []et.Json
**/
func (i *Item) ArrayJson() []et.Json {
	result := i.Get()
	val, ok := result.([]et.Json)
	if !ok {
		return []et.Json{}
	}

	return val
}
