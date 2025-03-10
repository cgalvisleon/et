package envar

import (
	"os"
	"strconv"

	"github.com/cgalvisleon/et/strs"
	_ "github.com/joho/godotenv/autoload"
)

/**
* MetaSet
* @param string name
* @param string _default
* @param string description
* @param string _var
* @return string
**/
func MetaSet(name string, _default string, description, _var string) string {
	for i, arg := range os.Args[1:] {
		if arg == strs.Format("-%s", name) {
			val := os.Args[i+2]
			os.Setenv(_var, val)
			return val
		}
	}

	return _default
}

/**
* SetStr
* @param string name
* @param string _default
* @param string usage
* @param string _var
* @return string
**/
func SetStr(name string, _default string, usage, _var string) string {
	return MetaSet(name, _default, usage, _var)
}

/**
* SetInt
* @param string name
* @param int _default
* @param string usage
* @param string _var
* @return int
**/
func SetInt(name string, _default int, usage, _var string) int {
	result := MetaSet(name, strconv.Itoa(_default), usage, _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return _default
	}

	return val
}

/**
* SetInt64
* @param string name
* @param int64 _default
* @param string usage
* @param string _var
* @return int64
**/
func SetIn64(name string, _default int64, usage, _var string) int64 {
	result := MetaSet(name, strconv.FormatInt(_default, 10), usage, _var)

	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return _default
	}

	return val
}

/**
* SetBool
* @param string name
* @param bool _default
* @param string usage
* @param string _var
* @return bool
**/
func SetBool(name string, _default bool, usage, _var string) bool {
	result := MetaSet(name, strconv.FormatBool(_default), usage, _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return _default
	}

	return val
}

/**
* UpSetStr
* @param string name
* @param string value
* @return string
**/
func UpSetStr(name string, value string) string {
	os.Setenv(name, value)
	return value
}

/**
* SetInt
* @param string name
* @param int value
* @return int
**/
func UpSetInt(name string, value int) int {
	os.Setenv(name, strconv.Itoa(value))
	return value
}

/**
* UpSetFloat
* @param string name
* @param int64 value
* @return int64
**/
func UpSetFloat(name string, value float64) float64 {
	os.Setenv(name, strconv.FormatFloat(float64(value), 'f', -1, 64))
	return value
}

/**
* UpSetBool
* @param string name
* @param bool value
* @return bool
**/
func UpSetBool(name string, value bool) bool {
	os.Setenv(name, strconv.FormatBool(value))
	return value
}

/**
* GetStr
* @param string _default
* @param string _var
* @return string
**/
func GetStr(_default string, _var string) string {
	result := os.Getenv(_var)

	if result == "" {
		return _default
	}

	return result
}

/**
* GetInt
* @param int _default
* @param string _var
* @return int
**/
func GetInt(_default int, _var string) int {
	result := GetStr(strconv.Itoa(_default), _var)

	val, err := strconv.Atoi(result)
	if err != nil {
		return _default
	}

	return val
}

/**
* GetInt64
* @param int64 _default
* @param string _var
* @return int64
**/
func GetInt64(_default int64, _var string) int64 {
	result := GetStr(strconv.FormatInt(_default, 10), _var)

	val, err := strconv.ParseInt(result, 10, 64)
	if err != nil {
		return _default
	}

	return val
}

/**
* GetBool
* @param bool _default
* @param string _var
* @return bool
**/
func GetBool(_default bool, _var string) bool {
	result := GetStr(strconv.FormatBool(_default), _var)

	val, err := strconv.ParseBool(result)
	if err != nil {
		return _default
	}

	return val
}

/**
* Int
* @param string _var
* @return int
**/
func Int(_var string) int {
	return GetInt(0, _var)
}

/**
* Int64
* @param string _var
* @return int64
**/
func Int64(_var string) int64 {
	return GetInt64(0, _var)
}

/**
* Bool
* @param string _var
* @return bool
**/
func Bool(_var string) bool {
	return GetBool(false, _var)
}

/**
* Str
* @param string _var
* @return string
**/
func Str(_var string) string {
	return GetStr("", _var)
}
