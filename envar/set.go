package envar

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

/**
* setEnvar
* @param name string, value interface{}
* @return error
**/
func setEnvar(name string, value interface{}) {
	name = strings.ToUpper(name)
	val := fmt.Sprintf("%v", value)
	os.Setenv(name, val)
	_config[name] = value
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
