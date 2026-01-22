package envar

import (
	"os"
	"strconv"
	"strings"
)

/**
* ArgStr
* @param name, defaultVal string
* @return string, bool
**/
func ArgStr(name, defaultVal string) (string, bool) {
	for i, arg := range os.Args[1:] {
		if arg == strings.ToLower(name) {
			n := len(os.Args)
			if n < i+2 {
				return defaultVal, false
			}
			value := os.Args[i+2]
			return value, true
		}
	}

	return defaultVal, false
}

/**
* ArgInt
* @param name, defaultVal int
* @return int, bool
**/
func ArgInt(name string, defaultVal int) (int, bool) {
	val, ok := ArgStr(name, strconv.Itoa(defaultVal))
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
* ArgInt64
* @param name, defaultVal int64
* @return int64, bool
**/
func ArgInt64(name string, defaultVal int64) (int64, bool) {
	s := strconv.FormatInt(defaultVal, 10)
	val, ok := ArgStr(name, s)
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
* ArgFloat64
* @param name, defaultVal float64
* @return float64, bool
**/
func ArgFloat64(name string, defaultVal float64) (float64, bool) {
	s := strconv.FormatFloat(defaultVal, 'f', -1, 64)
	val, ok := ArgStr(name, s)
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
* ArgBool
* @param name, defaultVal bool
* @return bool, bool
**/
func ArgBool(name string, defaultVal bool) (bool, bool) {
	s := strconv.FormatBool(defaultVal)
	val, ok := ArgStr(name, s)
	if !ok {
		return defaultVal, false
	}

	result, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal, false
	}

	return result, true
}
