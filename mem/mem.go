package mem

import (
	"fmt"
	"regexp"
	"strconv"
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
func Load() (*Mem, error) {
	result := &Mem{
		items: make(map[string]*Item),
		mutex: sync.RWMutex{},
	}

	logs.Logf("Mem", "Load memory cache")

	return result, nil
}

// Type method to use in cache
func (c *Mem) Type() string {
	return "mem"
}

// Set method to use in cache
func (c *Mem) Set(key string, value string, expiration time.Duration) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, ok := c.items[key]
	if ok {
		item.Set(value)
	} else {
		item = New(key, value)
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
func (c *Mem) Get(key string, def string) string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if item, ok := c.items[key]; ok {
		return item.Str()
	}

	return def
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

func (c *Mem) Count(key string, expiration time.Duration) int64 {
	item, ok := c.items[key]
	if !ok {
		c.Set(key, "0", expiration)
		return 0
	} else {
		result := item.Int64() + 1
		str := strconv.FormatInt(result, 10)
		c.Set(key, str, expiration)
		return result
	}
}

// Clear method to use in cache
func (c *Mem) Clear(match string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	matchPattern := func(substring, str string) bool {
		pattern := fmt.Sprintf(".*%s.*", regexp.QuoteMeta(substring))
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Println("Error compilando la expresión regular:", err)
			return false
		}
		return re.MatchString(str)
	}

	for key := range c.items {
		if matchPattern(match, key) {
			delete(c.items, key)
		}
	}
}

func (c *Mem) Empty() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.Clear("")
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
func (c *Mem) Values() []string {
	values := make([]string, 0, len(c.items))

	for _, item := range c.items {
		str := item.Str()
		values = append(values, str)
	}

	return values
}
