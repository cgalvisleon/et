package envar

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

/**
* Get
* @param name string, def interface{}
* @return interface{}
**/
func Get(name string, def interface{}) interface{} {
	if _store != nil {
		result := _store.Get(name, def)
		_config[name] = result
		return result
	}

	name = strings.ToUpper(name)
	result := os.Getenv(name)
	if result == "" {
		_config[name] = def
		return def
	}

	_config[name] = result
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
	result := GetStr(name, strconv.FormatInt(int64(def), 10))
	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return def
	}
	return time.Duration(val)
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
