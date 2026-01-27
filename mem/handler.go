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
* GetItem
* @param key string
* @return *Item, error
**/
func GetItem(key string) (*Item, error) {
	if conn == nil {
		return nil, nil
	}

	return conn.GetItem(key)
}

/**
* Get
* @param key, def string
* @return string, error
**/
func Get(key string) (interface{}, error) {
	if conn == nil {
		return nil, nil
	}

	return conn.Get(key)
}

/**
* GetStr
* @param key string
* @return string, error
**/
func GetStr(key string) (string, error) {
	if conn == nil {
		return "", nil
	}

	return conn.GetStr(key)
}

/**
* GetInt
* @param key string
* @return int, error
**/
func GetInt(key string) (int, error) {
	if conn == nil {
		return 0, nil
	}

	return conn.GetInt(key, 0)
}

/**
* GetInt64
* @param key string
* @return int64, error
**/
func GetInt64(key string) (int64, error) {
	if conn == nil {
		return 0, nil
	}

	return conn.GetInt64(key, 0)
}

/**
* GetFloat64
* @param key string
* @return float64, error
**/
func GetFloat64(key string) (float64, error) {
	if conn == nil {
		return 0, nil
	}

	return conn.GetFloat(key, 0)
}

func GetBool(key string) (bool, error) {
	if conn == nil {
		return false, nil
	}

	return conn.GetBool(key, false)
}

/**
* GetTime
* @param key string
* @return time.Time, error
**/
func GetTime(key string) (time.Time, error) {
	if conn == nil {
		return time.Time{}, nil
	}

	return conn.GetTime(key, time.Time{})
}

/**
* GetDuration
* @param key string
* @return time.Duration, error
**/
func GetDuration(key string) (time.Duration, error) {
	if conn == nil {
		return 0, nil
	}

	return conn.GetDuration(key, 0)
}

func GetJson(key string) (et.Json, error) {
	if conn == nil {
		return et.Json{}, nil
	}

	return conn.GetJson(key, et.Json{})
}

/**
* GetArrayJson
* @param key string
* @return []et.Json, error
**/
func GetArrayJson(key string) ([]et.Json, error) {
	if conn == nil {
		return []et.Json{}, nil
	}

	return conn.GetArrayJson(key, []et.Json{})
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

	return conn.Del(key)
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
