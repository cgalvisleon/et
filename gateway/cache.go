package gateway

import (
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
)

// Cache struct to use in gateway
type Cache struct {
	items map[string]interface{}
	mutex *sync.RWMutex
}

// NewCache create new cache
func NewCache() *Cache {
	return &Cache{
		items: make(map[string]interface{}),
		mutex: &sync.RWMutex{},
	}
}

// Set method to use in cache
func (c *Cache) Set(key string, value interface{}, expiration time.Duration) {
	c.mutex.Lock()
	c.items[key] = value
	c.mutex.Unlock()

	duration := expiration * time.Second

	clean := func() {
		c.mutex.Lock()
		logs.Log("Cache expired: " + key)
		delete(c.items, key)
		c.mutex.Unlock()
	}

	time.AfterFunc(duration, clean)
}

// Get method to use in cache
func (c *Cache) Get(key string, _default interface{}) interface{} {
	if val, ok := c.items[key]; ok {
		return val
	}

	return _default
}

// Del method to use in cache
func (c *Cache) Del(key string) bool {
	if _, ok := c.items[key]; !ok {
		return false
	}

	c.mutex.Lock()
	delete(c.items, key)
	c.mutex.Unlock()

	return true
}

func (c *Cache) Count(key string, expiration time.Duration) int {
	var val int
	var res interface{}
	var ok bool
	if res, ok = c.items[key]; !ok {
		val = 1
	} else {
		val = res.(int) + 1
	}

	c.Set(key, val, expiration)

	return val
}

// Clear method to use in cache
func (c *Cache) Clear() {
	c.items = make(map[string]interface{})
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
func (c *Cache) Items() map[string]interface{} {
	return c.items
}
