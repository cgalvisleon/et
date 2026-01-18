package envar

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	_ "github.com/joho/godotenv/autoload"
)

var Envar = map[string]interface{}{}

/**
* setEnvar
* @param name string, value interface{}
* @return error
**/
func setEnvar(name string, value interface{}) {
	name = strings.ToUpper(name)
	val := fmt.Sprintf("%v", value)
	Envar[name] = val
	os.Setenv(name, val)
}

/**
* SetStrByArg
* @param arg, name, def string
* @return string
**/
func SetStrByArg(arg, name, def string) string {
	val, ok := ArgStr(arg, def)
	if ok {
		setEnvar(name, val)
	}

	return val
}

/**
* SetIntByArg
* @param arg, name string, def int
* @return int
**/
func SetIntByArg(arg, name string, def int) int {
	val, ok := ArgInt(arg, def)
	if ok {
		setEnvar(name, strconv.Itoa(val))
	}

	return val
}

/**
* SetInt64ByArg
* @param arg, name string, def int64
* @return int64
**/
func SetInt64ByArg(arg, name string, def int64) int64 {
	val, ok := ArgInt64(arg, def)
	if ok {
		setEnvar(name, strconv.FormatInt(val, 10))
	}

	return val
}

/**
* SetBoolByArg
* @param arg, name string, def bool
* @return bool
**/
func SetBoolByArg(arg, name string, def bool) bool {
	val, ok := ArgBool(arg, def)
	if ok {
		setEnvar(name, strconv.FormatBool(val))
	}

	return val
}

/**
* Set
* @param name string, value interface{}
* @return interface{}
**/
func Set(name string, value interface{}) interface{} {
	s := fmt.Sprintf("%v", value)
	setEnvar(name, s)
	return value
}

/**
* SetStr
* @param name, value string
* @return string
**/
func SetStr(name string, value string) string {
	Set(name, value)
	return value
}

/**
* SetInt
* @param name string, value int
* @return int
**/
func SetInt(name string, value int) int {
	Set(name, value)
	return value
}

/**
* SetInt64
* @param name string, value int64
* @return int64
**/
func SetInt64(name string, value int64) int64 {
	Set(name, value)
	return value
}

/**
* SetNumber
* @param name string, value float64
* @return float64
**/
func SetNumber(name string, value float64) float64 {
	Set(name, value)
	return value
}

/**
* SetBool
* @param name string, value bool
* @return bool
**/
func SetBool(name string, value bool) bool {
	Set(name, value)
	return value
}

/**
* Get
* @param name string, def interface{}
* @return interface{}
**/
func Get(name string, def interface{}) interface{} {
	name = strings.ToUpper(name)
	result := os.Getenv(name)

	if result == "" {
		return def
	}

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
* GetNumber
* @param name string, def float64
* @return float64
**/
func GetNumber(name string, def float64) float64 {
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
func Number(name string) float64 {
	return GetNumber(name, 0)
}

/**
* Bool
* @param name string
* @return bool
**/
func Bool(name string) bool {
	return GetBool(name, false)
}

/**
* Validate
* @param keys []string
* @return error
**/
func Validate(keys []string) error {
	for _, key := range keys {
		val := Get(key, "")
		if val == "" {
			return fmt.Errorf(MSG_ATRIB_REQUIRED, key)
		}
	}
	return nil
}
