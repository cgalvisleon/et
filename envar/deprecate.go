package envar

import (
	"os"
	"strconv"
)

/**
* This function is deprecated, use SetStr instead
**/
func SetvarStr(name string, _default string, usage, _var string) string {
	return MetaSet(name, _default, usage, _var)
}

/**
* This function is deprecated, use SetInt instead
**/
func SetvarInt(name string, _default int, usage, _var string) int {
	result := MetaSet(name, strconv.Itoa(_default), usage, _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return _default
	}

	return val
}

/**
* This function is deprecated, use SetBool instead
**/
func SetvarBool(name string, _default bool, usage, _var string) bool {
	result := MetaSet(name, strconv.FormatBool(_default), usage, _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return _default
	}

	return val
}

/**
* This function is deprecated, use GetStr instead
**/
func EnvarStr(_default string, _var string) string {
	result := os.Getenv(_var)

	if result == "" {
		return _default
	}

	return result
}

/**
* This function is deprecated, use GetInt instead
**/
func EnvarInt(_default int, _var string) int {
	result := EnvarStr(strconv.Itoa(_default), _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return _default
	}

	return val
}

/**
* This function is deprecated, use GetInt64 instead
**/
func EnvarInt64(_default int64, _var string) int64 {
	result := EnvarStr(strconv.FormatInt(_default, 10), _var)

	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return _default
	}

	return val
}

/**
* This function is deprecated, use GetBool instead
**/
func EnvarBool(_default bool, _var string) bool {
	result := EnvarStr(strconv.FormatBool(_default), _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return _default
	}

	return val
}
