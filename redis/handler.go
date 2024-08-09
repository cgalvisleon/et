package redis

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/redis/go-redis/v9"
)

/**
* lock return a lock
* @param tag string
* @return *sync.RWMutex
**/
func (c *Conn) lock(tag string) *sync.RWMutex {
	if c.locks[tag] == nil {
		c.locks[tag] = &sync.RWMutex{}
	}

	return c.locks[tag]
}

/**
* Type return the type of connection
* @return string
**/
func (c *Conn) Type() string {
	return "redis"
}

/**
* Set method to use in cache
* @param key string
* @param value string
* @param expiration time.Duration
* @return string
**/
func (c *Conn) Set(key string, value string, expiration time.Duration) string {
	duration := expiration * time.Second

	err := c.db.Set(c.ctx, key, value, duration).Err()
	if logs.Alert(err) != nil {
		return value
	}

	return value
}

/**
* Get method to use in cache
* @param key string
* @param def string
* @return string
* @return error
**/
func (c *Conn) Get(key string, def string) (string, error) {
	result, err := c.db.Get(c.ctx, key).Result()
	switch {
	case err == redis.Nil:
		return def, errors.New("IsNil")
	case err != nil:
		return def, err
	default:
		return result, nil
	}
}

/**
* Del method to use in cache
* @param key string
* @return bool
**/
func (c *Conn) Del(key string) bool {
	intCmd := c.db.Del(c.ctx, key)
	if intCmd.Err() != nil {
		return false
	}

	return intCmd.Val() > 0
}

/**
* More method to use in cache
* @param key string
* @param expiration time.Duration
* @return int
**/
func (c *Conn) More(key string, expiration time.Duration) int64 {
	lock := c.lock(key)
	lock.RLock()

	def := "0"
	val, err := c.Get(key, def)
	if err != nil {
		c.Set(key, "0", expiration)
		lock.RUnlock()
		return 0
	} else {
		lock.RUnlock()
	}

	lock.Lock()
	defer lock.Unlock()

	num, err := strconv.ParseInt(val, 10, 64)
	if logs.Alert(err) != nil {
		return 0
	}

	num++
	c.Set(key, val, expiration)

	return num
}

// Clear method to use in cache
func (c *Conn) Clear(match string) {
	keys, err := c.db.Keys(c.ctx, match).Result()
	if logs.Alert(err) != nil {
		return
	}

	for _, key := range keys {
		c.Del(key)
	}
}

// Empty method to use in cache
func (c *Conn) Empty() {
	conn.db.FlushAll(c.ctx)
}

// Len method to use in cache
func (c *Conn) Len() int {
	return len(c.Keys())
}

// Keys method to use in cache
func (c *Conn) Keys() []string {
	keys := c.db.Keys(c.ctx, "*").Val()

	return keys
}

// Values method to use in cache
func (c *Conn) Values() []string {
	keys := c.Keys()
	values := make([]string, len(keys))

	for i, key := range keys {
		values[i], _ = c.Get(key, "")
	}

	return values
}
