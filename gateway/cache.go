package gateway

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
)

type Item struct {
	Datemake   time.Time
	Dateupdate time.Time
	Key        string
	Value      interface{}
	mutex      sync.RWMutex
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

// Cache struct to use in gateway
type Cache struct {
	items map[string]*Item
}

// NewCache create new cache
func NewCache() *Cache {
	return &Cache{
		items: make(map[string]*Item),
	}
}

// Set method to use in cache
func (c *Cache) Set(key string, value interface{}, expiration time.Duration) {
	val, ok := c.items[key]
	if ok {
		val.mutex.Lock()
		val.Dateupdate = time.Now()
		val.Value = value
		val.mutex.Unlock()
	} else {
		val = NewItem(key, value)
	}

	duration := expiration * time.Second

	clean := func() {
		c.Del(key)
	}

	time.AfterFunc(duration, clean)
}

// Get method to use in cache
func (c *Cache) Get(key string, _default interface{}) interface{} {
	if val, ok := c.items[key]; ok {
		return val.Value
	}

	return _default
}

// Del method to use in cache
func (c *Cache) Del(key string) bool {
	if _, ok := c.items[key]; !ok {
		return false
	}

	logs.Log("Cache deleted", key)
	delete(c.items, key)

	return true
}

func (c *Cache) Count(key string, expiration time.Duration) int {
	var val int = 1
	res, ok := c.items[key]
	if ok {
		res.mutex.Lock()
		val = res.Value.(int) + 1
		res.Value = val
		res.mutex.Unlock()
	} else {
		res = NewItem(key, val)
		c.items[key] = res
	}

	return val
}

// Clear method to use in cache
func (c *Cache) Clear() {
	c.items = make(map[string]*Item)
}

// Len method to use in cache
func (c *Cache) Len() int {
	return len(c.items)
}

// Keys method to use in cache
func (c *Cache) Keys() []string {
	keys := make([]string, 0, len(c.items))

	for key := range c.items {
		keys = append(keys, key)
	}

	return keys
}

// Values method to use in cache
func (c *Cache) Values() []interface{} {
	values := make([]interface{}, 0, len(c.items))

	for _, value := range c.items {
		values = append(values, value)
	}

	return values
}

// Items method to use in cache
func (c *Cache) Items() map[string]*Item {
	return c.items
}
