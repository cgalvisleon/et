package cache

import (
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
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

// Type adatapter
type TpCahe int

const (
	TpRedis TpCahe = iota
	TpMem
)

func (tp TpCahe) String() string {
	switch tp {
	case TpRedis:
		return "redis"
	case TpMem:
		return "memory"
	default:
		return ""
	}
}

var conn Cache

const MSG_CACHE_NOT_FOUND = "Cache not found"

// Load a new cache connection
func Load() error {
	if conn != nil {
		return nil
	}

	tp := envar.GetStr(TpMem.String(), "CACHE_TYPE")

	switch tp {
	case TpRedis.String():
		res, err := redis.Load()
		if err != nil {
			return err
		}

		conn = res

		return nil
	default:
		res, err := mem.Load()
		if err != nil {
			return err
		}

		conn = res

		return nil
	}
}

// Return the type of cache
func Type() string {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Type()
}

// Set a value in cache
func Set(key string, value interface{}, expiration time.Duration) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Set(key, value, expiration)
}

// Set a value in cache for one day
func SetD(key string, value interface{}) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	expiration := time.Hour * 24

	return conn.Set(key, value, expiration)
}

// Set a value in cache for one week
func SetW(key string, value interface{}) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	expiration := time.Hour * 24 * 7

	return conn.Set(key, value, expiration)
}

// Set a value in cache for one month
func SetM(key string, value interface{}) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	expiration := time.Hour * 24 * 30

	return conn.Set(key, value, expiration)
}

// Get a value from cache
func Get(key string, def interface{}) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Get(key, def)
}

// Delete a value from cache
func Del(key string) bool {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Del(key)
}

// Count the number of keys in cache
func Count(key string, expiration time.Duration) int {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Count(key, expiration)
}

// Clear all keys in cache
func Clear() {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	conn.Clear()
}

// Return the number of keys in cache
func Len() int {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Len()
}

// Return all keys in cache
func Keys() []string {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Keys()
}

// Return all values in cache
func Values() []interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Values()
}

// Json return a json object from cache
func Json(key string) (et.Json, error) {
	if conn == nil {
		return et.Json{}, logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	result := et.Json{}
	val := Get(key, result)
	err := result.Scan(val)
	if err != nil {
		return et.Json{}, logs.Alert(err)
	}

	return result, nil
}

// Items return a items object from cache
func Items(key string) (et.Items, error) {
	if conn == nil {
		return et.Items{}, logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	result := et.Items{}
	val := Get(key, result)
	err := result.Scan(val)
	if err != nil {
		return et.Items{}, logs.Alert(err)
	}

	return result, nil
}

func Item(key string) (et.Item, error) {
	if conn == nil {
		return et.Item{}, logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	result := et.Item{}
	val := Get(key, result)
	err := result.Scan(val)
	if err != nil {
		return et.Item{}, logs.Alert(err)
	}

	return result, nil
}
