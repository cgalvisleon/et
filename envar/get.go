package envar

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

/**
* getValueConfig
* @param name string, def interface{}
* @return interface{}, bool
**/
func getValueConfig(name string, def interface{}) (interface{}, bool) {
	_mu.RLock()
	result, exists := _config[name]
	_mu.RUnlock()
	return result, exists
}

/**
* setValueConfig
* @param name string, value interface{}
* @return void
**/
func setValueConfig(name string, value interface{}) {
	_mu.Lock()
	_config[name] = value
	_mu.Unlock()
}

/**
* Get
* @param name string, def interface{}
* @return interface{}
**/
func Get(name string, def interface{}) interface{} {
	result, exists := getValueConfig(name, def)
	if exists {
		return result
	}

	name = strings.ToUpper(name)
	result = os.Getenv(name)
	if _store != nil {
		result, exists := _store.Get(name, def)
		if !exists {
			setValueConfig(name, result)
			_store.Set(name, result)
		}
	}

	setValueConfig(name, result)
	return result
}

/**
* GetStr
* @param name, def string
* @return string
**/
func GetStr(name, def string) string {
	result := Get(name, def)
	if result == "" {
		return def
	}

	return fmt.Sprintf("%v", result)
}

/**
* GetInt
* @param name string, def int
* @return int
**/
func GetInt(name string, def int) int {
	result := GetStr(name, strconv.Itoa(def))
	val, err := strconv.Atoi(result)
	if err != nil {
		return def
	}

	return val
}

/**
* GetInt64
* @param name string, def int64
* @return int64
**/
func GetInt64(name string, def int64) int64 {
	result := GetStr(name, strconv.FormatInt(def, 10))
	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return def
	}

	return val
}

/**
* GetFloat
* @param name string, def float64
* @return float64
**/
func GetFloat(name string, def float64) float64 {
	result := GetStr(name, strconv.FormatFloat(def, 'f', -1, 64))
	val, err := strconv.ParseFloat(result, 64)
	if err != nil {
		return def
	}

	return val
}

/**
* GetBool
* @param name string, def bool
* @return bool
**/
func GetBool(name string, def bool) bool {
	result := GetStr(name, strconv.FormatBool(def))
	val, err := strconv.ParseBool(result)
	if err != nil {
		return def
	}

	return val
}

/**
* GetDuration
* @param name string, def time.Duration
* @return time.Duration
**/
func GetDuration(name string, def time.Duration) time.Duration {
	result := GetStr(name, def.String())
	val, err := time.ParseDuration(result)
	if err != nil {
		return def
	}
	return val
}

/**
* Str
* @param name string
* @return string
**/
func Str(name string) string {
	return GetStr(name, "")
}

/**
* Int
* @param name string
* @return int
**/
func Int(name string) int {
	return GetInt(name, 0)
}

/**
* Int64
* @param name string
* @return int64
**/
func Int64(name string) int64 {
	return GetInt64(name, 0)
}

/**
* Number
* @param name string
* @return float64
**/
func Float(name string) float64 {
	return GetFloat(name, 0)
}

/**
* Bool
* @param name string
* @return bool
**/
func Bool(name string) bool {
	return GetBool(name, false)
}
