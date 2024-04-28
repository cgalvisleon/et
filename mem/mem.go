package mem

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
)

// Mem struct to use in gateway
type Mem struct {
	items map[string]*Item
	mutex sync.RWMutex
}

// NewCache create new cache
func Load() (Mem, error) {
	result := Mem{
		items: make(map[string]*Item),
		mutex: sync.RWMutex{},
	}

	return result, nil
}

// Type method to use in cache
func (c *Mem) Type() string {
	return "mem"
}

// Set method to use in cache
func (c *Mem) Set(key string, value interface{}, expiration time.Duration) interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, ok := c.items[key]
	if ok {
		item.Set(value)
	} else {
		item = NewItem(key, value)
		c.items[key] = item
	}

	clean := func() {
		logs.Log("Mem expired", key)
		c.Del(key)
	}

	duration := expiration * time.Second
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	return value
}

// Get method to use in cache
func (c *Mem) Get(key string, _default interface{}) interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if item, ok := c.items[key]; ok {
		return item.Get()
	}

	return _default
}

// Del method to use in cache
func (c *Mem) Del(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.items[key]; !ok {
		return false
	}

	delete(c.items, key)

	return true
}

func (c *Mem) Count(key string, expiration time.Duration) int {
	item, ok := c.items[key]
	if !ok {
		c.Set(key, 1, expiration)
		return 1
	} else {
		result := item.Int() + 1
		c.Set(key, result, expiration)
		return result
	}
}

// Clear method to use in cache
func (c *Mem) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]*Item)
}

// Len method to use in cache
func (c *Mem) Len() int {
	return len(c.items)
}

// Keys method to use in cache
func (c *Mem) Keys() []string {
	keys := make([]string, 0, len(c.items))

	for key := range c.items {
		keys = append(keys, key)
	}

	return keys
}

// Values method to use in cache
func (c *Mem) Values() []interface{} {
	values := make([]interface{}, 0, len(c.items))

	for _, value := range c.items {
		values = append(values, value)
	}

	return values
}
