package redis

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/redis/go-redis/v9"
)

type Conn struct {
	ctx    context.Context
	locks  map[string]*sync.RWMutex
	host   string
	dbname int
	db     *redis.Client
}

var conn *Conn

/**
* Load redis connection
* @return (*Conn, error)
**/
func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	return connect()
}

/**
* Close redis connection
* @return error
**/
func Close() error {
	if conn.db == nil {
		return nil
	}

	return conn.db.Close()
}

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
func (c *Conn) Set(key string, value interface{}, expiration time.Duration) interface{} {
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
**/
func (c *Conn) Get(key string, def string) string {
	result, err := c.db.Get(c.ctx, key).Result()
	switch {
	case err == redis.Nil:
		return def
	case err != nil:
		return def
	case result == "":
		return result
	default:
		return result
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
* Count method to use in cache
* @param key string
* @param expiration time.Duration
* @return int
**/
func (c *Conn) Count(key string, expiration time.Duration) int {
	lock := c.lock(key)
	lock.RLock()

	def := "0"
	val := c.Get(key, def)
	lock.RUnlock()

	lock.Lock()
	defer lock.Unlock()

	if val == def {
		c.Set(key, "1", expiration)
		return 1
	}

	num, err := strconv.Atoi(val)
	if logs.Alert(err) != nil {
		return 0
	}

	num++
	c.Set(key, val, expiration)

	return num
}

// Clear method to use in cache
func (c *Conn) Clear() {
	c.db.FlushDB(c.ctx)
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
		values[i] = c.Get(key, "")
	}

	return values
}
