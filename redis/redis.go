package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Conn struct {
	ctx    context.Context
	host   string
	dbname int
	db     *redis.Client
}

var conn *Conn

// NewCache create new cache
func Load() (*Conn, error) {
	if conn != nil {
		return conn, nil
	}

	return connect()
}

// Close method to use in cache
func Close() error {
	if conn.db == nil {
		return nil
	}

	return conn.db.Close()
}

// Type method to use in cache
func (c *Conn) Type() string {
	return "redis"
}

// Set method to use in cache
func (c *Conn) Set(key string, value interface{}, expiration time.Duration) interface{} {
	c.db.Set(c.ctx, key, value, expiration)
	return value
}

// Get method to use in cache
func (c *Conn) Get(key string, def interface{}) interface{} {
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

// Del method to use in cache
func (c *Conn) Del(key string) bool {
	intCmd := c.db.Del(c.ctx, key)
	if intCmd.Err() != nil {
		return false
	}

	return intCmd.Val() > 0
}

func (c *Conn) Count(key string, expiration time.Duration) int {
	result := c.Get(key, 0)
	if result == 0 {
		c.Set(key, 1, expiration)
		return 1
	}

	val := result.(int) + 1
	c.Set(key, val, expiration)

	return val
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
func (c *Conn) Values() []interface{} {
	keys := c.Keys()
	values := make([]interface{}, len(keys))

	for i, key := range keys {
		values[i] = c.Get(key, nil)
	}

	return values
}
