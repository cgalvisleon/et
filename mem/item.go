package mem

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/et"
)

// Item struct to use in cache
type Item struct {
	Datemake   time.Time
	Dateupdate time.Time
	Key        string
	Value      interface{}
	mutex      sync.RWMutex
}

// Set method to use in item
func (i *Item) Set(value interface{}) interface{} {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	i.Dateupdate = time.Now()
	i.Value = value

	return value
}

// Get method to use in item
func (i *Item) Get() interface{} {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	return i.Value
}

// Return value string from item
func (i *Item) Str() string {
	result := i.Get()
	return result.(string)
}

// Return value int from item
func (i *Item) Int() int {
	result := i.Get()
	return result.(int)
}

// Return value float from item
func (i *Item) Float() float64 {
	result := i.Get()
	return result.(float64)
}

// Return value bool from item
func (i *Item) Bool() bool {
	result := i.Get()
	return result.(bool)
}

// Return value time from item
func (i *Item) Time() time.Time {
	result := i.Get()
	return result.(time.Time)
}

// Return value duration from item
func (i *Item) Duration() time.Duration {
	result := i.Get()
	return result.(time.Duration)
}

// Return value json from item
func (i *Item) Json() et.Json {
	result := i.Get()
	return result.(et.Json)
}

// Return value map from item
func (i *Item) Map() map[string]interface{} {
	result := i.Get()
	return result.(map[string]interface{})
}

// Return value array from item
func (i *Item) ArrayMap() []map[string]interface{} {
	result := i.Get()
	return result.([]map[string]interface{})
}

// Return value array from item
func (i *Item) ArrayStr() []string {
	result := i.Get()
	return result.([]string)
}

// Return value array from item
func (i *Item) ArrayInt() []int {
	result := i.Get()
	return result.([]int)
}

// Return value array from item
func (i *Item) ArrayFloat() []float64 {
	result := i.Get()
	return result.([]float64)
}

// Return value array from item
func (i *Item) ArrayBool() []bool {
	result := i.Get()
	return result.([]bool)
}

// Return value array from item
func (i *Item) ArrayTime() []time.Time {
	result := i.Get()
	return result.([]time.Time)
}

// Return value array from item
func (i *Item) ArrayDuration() []time.Duration {
	result := i.Get()
	return result.([]time.Duration)
}

// Return value array from item
func (i *Item) ArrayJson() []et.Json {
	result := i.Get()
	return result.([]et.Json)
}

// NewItem create new item
func NewItem(key string, value interface{}) *Item {
	now := time.Now()
	return &Item{
		Datemake:   now,
		Dateupdate: now,
		Key:        key,
		Value:      value,
		mutex:      sync.RWMutex{},
	}
}
