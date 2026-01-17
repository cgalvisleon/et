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
* @param name, varName, defaultVal string
* @return string
**/
func SetStrByArg(name, varName, defaultVal string) string {
	val, ok := ArgStr(name, defaultVal)
	if ok {
		setEnvar(varName, val)
	}

	return val
}

/**
* SetIntByArg
* @param name, varName string, defaultVal int
* @return int
**/
func SetIntByArg(name, varName string, defaultVal int) int {
	val, ok := ArgInt(name, defaultVal)
	if ok {
		setEnvar(varName, strconv.Itoa(val))
	}

	return val
}

/**
* SetInt64ByArg
* @param name, varName string, defaultVal int64
* @return int64
**/
func SetInt64ByArg(name, varName string, defaultVal int64) int64 {
	val, ok := ArgInt64(name, defaultVal)
	if ok {
		setEnvar(varName, strconv.FormatInt(val, 10))
	}

	return val
}

/**
* SetBoolByArg
* @param name, varName string, defaultVal bool
* @return bool
**/
func SetBoolByArg(name, varName string, defaultVal bool) bool {
	val, ok := ArgBool(name, defaultVal)
	if ok {
		setEnvar(varName, strconv.FormatBool(val))
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
* @param name string, defaultVal interface{}
* @return interface{}
**/
func Get(name string, defaultVal interface{}) interface{} {
	name = strings.ToUpper(name)
	result := os.Getenv(name)

	if result == "" {
		return defaultVal
	}

	return result
}

/**
* GetStr
* @param varName, defaultVal string
* @return string
**/
func GetStr(varName, defaultVal string) string {
	result := Get(varName, defaultVal)
	if result == "" {
		return defaultVal
	}

	return fmt.Sprintf("%v", result)
}

/**
* GetInt
* @param varName string, defaultVal int
* @return int
**/
func GetInt(varName string, defaultVal int) int {
	result := GetStr(varName, strconv.Itoa(defaultVal))
	val, err := strconv.Atoi(result)
	if err != nil {
		return defaultVal
	}

	return val
}

/**
* GetInt64
* @param varName string, defaultVal int64
* @return int64
**/
func GetInt64(varName string, defaultVal int64) int64 {
	result := GetStr(varName, strconv.FormatInt(defaultVal, 10))
	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return defaultVal
	}

	return val
}

/**
* GetNumber
* @param varName string, defaultVal float64
* @return float64
**/
func GetNumber(varName string, defaultVal float64) float64 {
	result := GetStr(varName, strconv.FormatFloat(defaultVal, 'f', -1, 64))
	val, err := strconv.ParseFloat(result, 64)
	if err != nil {
		return defaultVal
	}

	return val
}

/**
* GetBool
* @param varName string, defaultVal bool
* @return bool
**/
func GetBool(varName string, defaultVal bool) bool {
	result := GetStr(varName, strconv.FormatBool(defaultVal))
	val, err := strconv.ParseBool(result)
	if err != nil {
		return defaultVal
	}

	return val
}

/**
* Str
* @param varName string
* @return string
**/
func Str(varName string) string {
	return GetStr(varName, "")
}

/**
* Int
* @param varName string
* @return int
**/
func Int(varName string) int {
	return GetInt(varName, 0)
}

/**
* Int64
* @param varName string
* @return int64
**/
func Int64(varName string) int64 {
	return GetInt64(varName, 0)
}

/**
* Number
* @param varName string
* @return float64
**/
func Number(varName string) float64 {
	return GetNumber(varName, 0)
}

/**
* Bool
* @param varName string
* @return bool
**/
func Bool(varName string) bool {
	return GetBool(varName, false)
}
