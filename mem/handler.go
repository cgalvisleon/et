package mem

import (
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* Set
* @param key string, value interface{}, expiration time.Duration
* @return *Item
**/
func Set(key string, value interface{}, expiration time.Duration) *Item {
	if conn == nil {
		return nil
	}

	return conn.Set(key, value, expiration)
}

/**
* Delete
* @param key string
* @return bool
**/
func Delete(key string) bool {
	if conn == nil {
		return false
	}

	return conn.Delete(key)
}

/**
* Exists
* @param key string
* @return bool
**/
func Exists(key string) bool {
	if conn == nil {
		return false
	}

	return conn.Exists(key)
}

/**
* GetItem
* @param key string
* @return *Item, bool
**/
func GetItem(key string) (*Item, bool) {
	if conn == nil {
		return nil, false
	}

	return conn.GetItem(key)
}

/**
* Get
* @param key, def string
* @return string, bool
**/
func Get(key string) (interface{}, bool) {
	if conn == nil {
		return nil, false
	}

	return conn.Get(key)
}

/**
* GetStr
* @param key string
* @return string, bool
**/
func GetStr(key string) (string, bool) {
	if conn == nil {
		return "", false
	}

	return conn.GetStr(key)
}

/**
* GetInt
* @param key string
* @return int, bool
**/
func GetInt(key string) (int, bool) {
	if conn == nil {
		return 0, false
	}

	return conn.GetInt(key, 0)
}

/**
* GetInt64
* @param key string
* @return int64, bool
**/
func GetInt64(key string) (int64, bool) {
	if conn == nil {
		return 0, false
	}

	return conn.GetInt64(key, 0)
}

/**
* GetFloat64
* @param key string
* @return float64, bool
**/
func GetFloat64(key string) (float64, bool) {
	if conn == nil {
		return 0, false
	}

	return conn.GetFloat(key, 0)
}

/**
* GetBool
* @param key string
* @return bool, bool
**/
func GetBool(key string) (bool, bool) {
	if conn == nil {
		return false, false
	}

	return conn.GetBool(key, false)
}

/**
* GetTime
* @param key string
* @return time.Time, bool
**/
func GetTime(key string) (time.Time, bool) {
	if conn == nil {
		return time.Time{}, false
	}

	return conn.GetTime(key, time.Time{})
}

/**
* GetDuration
* @param key string
* @return time.Duration, bool
**/
func GetDuration(key string) (time.Duration, bool) {
	if conn == nil {
		return 0, false
	}

	return conn.GetDuration(key, 0)
}

/**
* GetJson
* @param key string
* @return et.Json, bool
**/
func GetJson(key string) (et.Json, bool) {
	if conn == nil {
		return et.Json{}, false
	}

	return conn.GetJson(key, et.Json{})
}

/**
* GetArrayJson
* @param key string
* @return []et.Json, bool
**/
func GetArrayJson(key string) ([]et.Json, bool) {
	if conn == nil {
		return []et.Json{}, false
	}

	return conn.GetArrayJson(key, []et.Json{})
}

/**
* More
* @param key string
* @param expiration time.Duration
**/
func More(key string, expiration time.Duration) {
	if conn == nil {
		return
	}

	conn.More(key, expiration)
}

/**
* Clear
* @param match string
**/
func Clear(match string) {
	if conn == nil {
		return
	}

	conn.Clear(match)
}

/**
* Empty
**/
func Empty() {
	if conn == nil {
		return
	}

	conn.Empty()
}

/**
* Len
* @return int
**/
func Len() int {
	if conn == nil {
		return 0
	}

	return conn.Len()
}

/**
* Keys
* @return []string
**/
func Keys() []string {
	if conn == nil {
		return []string{}
	}

	return conn.Keys()
}

/**
* Values
* @return []string
**/
func Values() []string {
	if conn == nil {
		return []string{}
	}

	return conn.Values()
}
