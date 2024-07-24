package cache

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/js"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/mem"
	"github.com/cgalvisleon/et/redis"
)

type Cache interface {
	Type() string
	Set(key string, value string, expiration time.Duration) string
	Get(key string, def string) string
	Del(key string) bool
	Count(key string, expiration time.Duration) int
	Clear()
	Len() int
	Keys() []string
	Values() []string
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

/**
* Load cache
* @return error
**/
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

/**
* Type return the type of cache
* @return string
**/
func Type() string {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Type()
}

/**
* Set a value in cache
* @param key string
* @param value string
* @param expiration time.Duration
* @return string
**/
func Set(key string, value interface{}, expiration time.Duration) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	switch v := value.(type) {
	case js.Json:
		return conn.Set(key, v.ToString(), expiration)
	case js.Items:
		return conn.Set(key, v.ToString(), expiration)
	case js.Item:
		return conn.Set(key, v.ToString(), expiration)
	default:
		val, ok := value.(string)
		if ok {
			return conn.Set(key, val, expiration)
		}

		return val
	}
}

/**
* SetH a value in cache for one hour
* @param key string
* @param value interface{}
* @return interface{}
**/
func SetH(key string, value interface{}) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	expiration := time.Hour * 1

	return Set(key, value, expiration)
}

/**
* SetD a value in cache for one day
* @param key string
* @param value interface{}
* @return interface{}
**/
func SetD(key string, value interface{}) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	expiration := time.Hour * 24

	return Set(key, value, expiration)
}

/**
* SetW a value in cache for one week
* @param key string
* @param value interface{}
* @return interface{}
**/
func SetW(key string, value interface{}) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	expiration := time.Hour * 24 * 7

	return Set(key, value, expiration)
}

/**
* SetM a value in cache for one month
* @param key string
* @param value interface{}
* @return interface{}
**/
func SetM(key string, value interface{}) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	expiration := time.Hour * 24 * 30

	return Set(key, value, expiration)
}

/**
* SetY a value in cache for one year
* @param key string
* @param value interface{}
* @return interface{}
**/
func Get(key string, def interface{}) interface{} {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	if def == nil {
		result := conn.Get(key, "<nil>")
		if result == "<nil>" {
			return nil
		}

		return result
	}

	switch v := def.(type) {
	case js.Json:
		return conn.Get(key, v.ToString())
	case js.Items:
		return conn.Get(key, v.ToString())
	case js.Item:
		return conn.Get(key, v.ToString())
	case string:
		return conn.Get(key, v)
	default:
		val, ok := v.(string)
		if ok {
			return conn.Get(key, val)
		}

		return val
	}
}

/**
* Del a value in cache
* @param key string
* @return bool
**/
func Del(key string) bool {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Del(key)
}

/**
* Count the number of keys in cache
* @param key string
* @param expiration time.Duration
* @return int
**/
func Count(key string, expiration time.Duration) int {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Count(key, expiration)
}

/**
* Clear all keys in cache
**/
func Clear() {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	conn.Clear()
}

/**
* Len return the number of keys in cache
* @return int
**/
func Len() int {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Len()
}

/**
* Keys return all keys in cache
* @return []string
**/
func Keys() []string {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Keys()
}

/**
* Values return all values in cache
* @return []string
**/
func Values() []string {
	if conn == nil {
		logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	return conn.Values()
}

/**
* Json return a json object from cache
* @param key string
* @return js.Json
* @return error
**/
func Json(key string) (js.Json, error) {
	if conn == nil {
		return js.Json{}, logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	val := Get(key, nil)
	if val == nil {
		return js.Json{}, nil
	}

	result := js.Json{}
	err := result.Scan(val)
	if err != nil {
		return js.Json{}, logs.Alert(err)
	}

	return result, nil
}

/**
* Items return a items object from cache
* @param key string
* @return js.Items
* @return error
**/
func Items(key string) (js.Items, error) {
	if conn == nil {
		return js.Items{}, logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	val := Get(key, nil)
	if val == nil {
		return js.Items{}, nil
	}

	jsonString := val.(string)
	var result []js.Json
	err := json.Unmarshal([]byte(jsonString), &result)
	if err != nil {
		return js.Items{}, err
	}

	return js.Items{
		Ok:     true,
		Result: result,
	}, nil
}

/**
* Item return a item object from cache
* @param key string
* @return js.Item
* @return error
**/
func Item(key string) (js.Item, error) {
	if conn == nil {
		return js.Item{}, logs.Alertm(MSG_CACHE_NOT_FOUND)
	}

	val := Get(key, nil)
	if val == nil {
		return js.Item{}, nil
	}

	jsonString := val.(string)
	var result js.Json
	err := json.Unmarshal([]byte(jsonString), &result)
	if err != nil {
		return js.Item{}, err
	}

	return js.Item{
		Ok:     true,
		Result: result,
	}, nil
}
