package mem

import (
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* Set
* @param key string, value interface{}, expiration time.Duration
* @return *Entry
**/
func Set(key string, value interface{}, expiration time.Duration) (*Entry, error) {
	return conn.Set(key, value, expiration)
}

/**
* Delete
* @param key string
* @return bool
**/
func Delete(key string) bool {
	return conn.Delete(key)
}

/**
* Exists
* @param key string
* @return bool
**/
func Exists(key string) bool {
	return conn.Exists(key)
}

/**
* GetEntry
* @param key string
* @return *Entry, bool
**/
func GetEntry(key string) (*Entry, bool) {
	return conn.GetEntry(key)
}

/**
* Get
* @param key, def string
* @return string, bool
**/
func Get(key string) (interface{}, bool) {
	return conn.Get(key)
}

/**
* GetStr
* @param key string
* @return string, bool, error
**/
func GetStr(key string) (string, bool, error) {
	return conn.GetStr(key)
}

/**
* GetInt
* @param key string
* @return int, bool, error
**/
func GetInt(key string) (int, bool, error) {
	return conn.GetInt(key, 0)
}

/**
* GetInt64
* @param key string
* @return int64, bool, error
**/
func GetInt64(key string) (int64, bool, error) {
	return conn.GetInt64(key, 0)
}

/**
* GetFloat64
* @param key string
* @return float64, bool, error
**/
func GetFloat64(key string) (float64, bool, error) {
	return conn.GetFloat(key, 0)
}

/**
* GetBool
* @param key string
* @return bool, bool, error
**/
func GetBool(key string) (bool, bool, error) {
	return conn.GetBool(key, false)
}

/**
* GetTime
* @param key string
* @return time.Time, bool, error
**/
func GetTime(key string) (time.Time, bool, error) {
	return conn.GetTime(key, time.Time{})
}

/**
* GetDuration
* @param key string
* @return time.Duration, bool, error
**/
func GetDuration(key string) (time.Duration, bool, error) {
	return conn.GetDuration(key, 0)
}

/**
* GetJson
* @param key string
* @return et.Json, bool, error
**/
func GetJson(key string) (et.Json, bool, error) {
	return conn.GetJson(key, et.Json{})
}

/**
* GetArrayJson
* @param key string
* @return []et.Json, bool, error
**/
func GetArrayJson(key string) ([]et.Json, bool, error) {
	return conn.GetArrayJson(key, []et.Json{})
}

/**
* More
* @param key string
* @param expiration time.Duration
**/
func More(key string, expiration time.Duration) (int64, error) {
	return conn.More(key, expiration)
}

/**
* Clear
* @param match string
**/
func Clear(match string) {
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
