package mem

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"sync"
	"time"

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
func (c *Mem) lock(tag string) *sync.RWMutex {
	if c.locks[tag] == nil {
		c.locks[tag] = &sync.RWMutex{}
	}

	return c.locks[tag]
}

/**
* Type
* @return string
**/
func (c *Mem) Type() string {
	return "mem"
}

/**
* Set
* @param key string
* @param value string
* @param expiration time.Duration
* @return string
**/
func (c *Mem) Set(key string, value string, expiration time.Duration) string {
	lock := c.lock(key)
	lock.Lock()
	defer lock.Unlock()

	item, ok := c.items[key]
	if ok {
		item.Set(value)
	} else {
		item = New(key, value)
		c.items[key] = item
	}

	clean := func() {
		c.Del(key)
	}

	duration := expiration * time.Second
	if duration != 0 {
		go time.AfterFunc(duration, clean)
	}

	return value
}

/**
* Get
* @param key string
* @param def string
* @return string
* @return error
**/
func (c *Mem) Get(key string, def string) (string, error) {
	lock := c.lock(key)
	lock.RLock()
	defer lock.RUnlock()

	if item, ok := c.items[key]; ok {
		return item.Str(), nil
	}

	return def, errors.New("IsNil")
}

/**
* Del
* @param key string
* @return bool
**/
func (c *Mem) Del(key string) bool {
	lock := c.lock(key)
	lock.Lock()
	defer lock.Unlock()

	if _, ok := c.items[key]; !ok {
		return false
	}

	delete(c.items, key)

	return true
}

/**
* More
* @param key string
* @param expiration time.Duration
* @return int
**/
func (c *Mem) More(key string, expiration time.Duration) int64 {
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

/**
* Clear
* @param match string
**/
func (c *Mem) Clear(match string) {
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
	c.Clear("")
}

/**
* Len
* @return int
**/
func (c *Mem) Len() int {
	return len(c.items)
}

/**
* Keys
* @return []string
**/
func (c *Mem) Keys() []string {
	keys := make([]string, 0, len(c.items))

	for key := range c.items {
		keys = append(keys, key)
	}

	return keys
}

/**
* Values
* @return []string
**/
func (c *Mem) Values() []string {
	values := make([]string, 0, len(c.items))

	for _, item := range c.items {
		str := item.Str()
		values = append(values, str)
	}

	return values
}
