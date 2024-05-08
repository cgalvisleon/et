package cache

import (
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/et/redis"
)

type Cache interface {
	Type() string
	Set(key string, value interface{}, expiration time.Duration) interface{}
	Get(key string, def interface{}) interface{}
	Del(key string) bool
	Count(key string, expiration time.Duration) int
	Clear()
	Len() int
	Keys() []string
	Values() []interface{}
}

var conn Cache

const MSG_CACHE_NOT_FOUND = "Cache not found"

// Load a new cache connection
func Load(tp string) {
	switch tp {
	case "redis":
		res, err := redis.Load()
		if err != nil {
			logs.Fatal(err)
		}

		conn = res
	default:
		res, err := mem.Load()
		if err != nil {
			logs.Fatal(err)
		}

		conn = res
	}
}

// Return the type of cache
func Type() string {
	if conn == nil {
		logs.Fatal(MSG_CACHE_NOT_FOUND)
	}

	return conn.Type()
}

// Set a value in cache
func Set(key string, value interface{}, expiration time.Duration) interface{} {
	if conn == nil {
		logs.Fatal(MSG_CACHE_NOT_FOUND)
	}

	return conn.Set(key, value, expiration)
}

// Get a value from cache
func Get(key string, def interface{}) interface{} {
	if conn == nil {
		logs.Fatal(MSG_CACHE_NOT_FOUND)
	}

	return conn.Get(key, def)
}

// Delete a value from cache
func Del(key string) bool {
	if conn == nil {
		logs.Fatal(MSG_CACHE_NOT_FOUND)
	}

	return conn.Del(key)
}

// Count the number of keys in cache
func Count(key string, expiration time.Duration) int {
	if conn == nil {
		logs.Fatal(MSG_CACHE_NOT_FOUND)
	}

	return conn.Count(key, expiration)
}

// Clear all keys in cache
func Clear() {
	if conn == nil {
		logs.Fatal(MSG_CACHE_NOT_FOUND)
	}

	conn.Clear()
}

// Return the number of keys in cache
func Len() int {
	if conn == nil {
		logs.Fatal(MSG_CACHE_NOT_FOUND)
	}

	return conn.Len()
}

// Return all keys in cache
func Keys() []string {
	if conn == nil {
		logs.Fatal(MSG_CACHE_NOT_FOUND)
	}

	return conn.Keys()
}

// Return all values in cache
func Values() []interface{} {
	if conn == nil {
		logs.Fatal(MSG_CACHE_NOT_FOUND)
	}

	return conn.Values()
}
