package envar

import (
	"os"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/strs"
	_ "github.com/joho/godotenv/autoload"
)

/**
* metaSet
* @param name string
* @param def string
* @param description string
* @param _var string
* @return string
**/
func metaSet(name, def, description, _var string) string {
	for i, arg := range os.Args[1:] {
		if arg == strs.Format("-%s", name) {
			val := os.Args[i+2]
			os.Setenv(_var, val)
			return val
		}
	}

	return def
}

/**
* SetStr, set string environment variable
* @param name string
* @param def string
* @param description string
* @param _var string
* @return string
**/
func SetStr(name, def, description, _var string) string {
	return metaSet(name, def, description, _var)
}

/**
* SetInt, set integer environment variable
* @param name string
* @param def int
* @param description string
* @param _var string
* @return int
**/
func SetInt(name string, def int, description, _var string) int {
	result := metaSet(name, strconv.Itoa(def), description, _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return def
	}

	return val
}

/**
* SetInt64, set integer64 environment variable
* @param name string
* @param def int64
* @param description string
* @param _var string
* @return int64
**/
func SetBool(name string, def bool, description, _var string) bool {
	result := metaSet(name, strconv.FormatBool(def), description, _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return def
	}

	return val
}

/**
* SetTime, set time environment variable
* @param name string
* @param def time.Time
* @param description string
* @param _var string
* @return time.Time
**/
func SetTime(name string, def time.Time, description, _var string) time.Time {
	result := metaSet(name, def.Format(time.RFC3339), description, _var)

	val, err := time.Parse(time.RFC3339, result)
	if err != nil {
		return def
	}

	return val
}

/**
* GetStr, get string environment variable
* @param def string
* @param _var string
* @return string
**/
func GetStr(def string, _var string) string {
	result := os.Getenv(_var)

	if result == "" {
		return def
	}

	return result
}

/**
* GetInt, get integer environment variable
* @param def int
* @param _var string
* @return int
**/
func GetInt(def int, _var string) int {
	result := GetStr(strconv.Itoa(def), _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return def
	}

	return val
}

/**
* GetInt64, get integer64 environment variable
* @param def int64
* @param _var string
* @return int64
**/
func GetInt64(def int64, _var string) int64 {
	result := GetStr(strconv.FormatInt(def, 10), _var)

	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return def
	}

	return val
}

/**
* GetBool, get boolean environment variable
* @param def bool
* @param _var string
* @return bool
**/
func GetBool(def bool, _var string) bool {
	result := GetStr(strconv.FormatBool(def), _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return def
	}

	return val
}

/**
* GetTime, get time environment variable
* @param def time.Time
* @param _var string
* @return time.Time
**/
func GetTime(def time.Time, _var string) time.Time {
	result := GetStr(def.Format(time.RFC3339), _var)

	val, err := time.Parse(time.RFC3339, result)
	if err != nil {
		return def
	}

	return val
}
