package arg

import (
	"os"
	"strconv"
	"strings"
)

/**
* Get
* @param name, defaultVal string
* @return string, bool
**/
func Get(name, defaultVal string) (string, bool) {
	for i, arg := range os.Args[1:] {
		arg = strings.ReplaceAll(arg, "-", "")
		if arg == strings.ToLower(name) {
			value := os.Args[i+2]
			return value, true
		}
	}

	return defaultVal, false
}

/**
* GetInt
* @param name, defaultVal int
* @return int, bool
**/
func GetInt(name string, defaultVal int) (int, bool) {
	val, ok := Get(name, strconv.Itoa(defaultVal))
	if !ok {
		return defaultVal, false
	}

	result, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal, false
	}

	return result, true
}

/**
* GetInt64
* @param name, defaultVal int64
* @return int64, bool
**/
func GetInt64(name string, defaultVal int64) (int64, bool) {
	val, ok := Get(name, strconv.FormatInt(defaultVal, 10))
	if !ok {
		return defaultVal, false
	}

	result, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal, false
	}

	return result, true
}

/**
* GetFloat64
* @param name, defaultVal float64
* @return float64, bool
**/
func GetFloat64(name string, defaultVal float64) (float64, bool) {
	val, ok := Get(name, strconv.FormatFloat(defaultVal, 'f', -1, 64))
	if !ok {
		return defaultVal, false
	}

	result, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultVal, false
	}

	return result, true
}

/**
* GetBool
* @param name, defaultVal bool
* @return bool, bool
**/
func GetBool(name string, defaultVal bool) (bool, bool) {
	val, ok := Get(name, strconv.FormatBool(defaultVal))
	if !ok {
		return defaultVal, false
	}

	result, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal, false
	}

	return result, true
}
