package mem

import "time"

/**
* Set
* @param key, value string, expiration time.Duration
* @return string
**/
func Set(key, value string, expiration time.Duration) string {
	if conn == nil {
		return value
	}

	return conn.Set(key, value, expiration)
}

/**
* Get
* @param key, def string
* @return string, error
**/
func Get(key, def string) (string, error) {
	if conn == nil {
		return def, nil
	}

	return conn.Get(key, def)
}

/**
* Del
* @param key string
* @return bool
**/
func Del(key string) bool {
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
