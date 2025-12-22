package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

/**
* setEnvar
* @param key string, value interface{}
* @return error
**/
func setEnvar(key string, value interface{}) {
	upsetVal := func(k string, v interface{}) {
		envar.Set(k, v)
		lower := strings.ToLower(k)
		switch lower {
		case "name":
			App.Name = GetStr(k, "")
		case "version":
			App.Version = GetStr(k, "")
		case "company":
			App.Company = GetStr(k, "")
		case "web":
			App.Web = GetStr(k, "")
		case "help":
			App.Help = GetStr(k, "")
		case "host":
			App.Host = GetStr(k, "")
		case "path_api":
			App.PathApi = GetStr(k, "")
		case "path_app":
			App.PathApp = GetStr(k, "")
		case "production":
			App.Production = GetBool(k, false)
		case "port":
			App.Port = GetInt(k, 3300)
		case "stage":
			App.Stage = GetStr(k, "local")
		case "debug":
			App.Debug = GetBool(k, false)
		}
	}

	key = strs.Uppcase(key)
	if map[string]bool{
		"SECRETS":   true,
		"PASSWORDS": true,
		"PASS":      true,
	}[key] {
		str := fmt.Sprintf("%v", value)
		encoded := base64.StdEncoding.EncodeToString([]byte(str))
		upsetVal(key, encoded)
	} else {
		upsetVal(key, value)
	}
}

/**
* Set
* @param key string, value interface{}
* @return interface{}
**/
func Set(key string, value interface{}) interface{} {
	setEnvar(key, value)
	return value
}

/**
* Get
* @param key string, defaultValue interface{}
* @return interface{}
**/
func Get(key string, defaultValue interface{}) interface{} {
	key = strs.Uppcase(key)
	result := envar.Get(key, defaultValue)
	return result
}

/**
* GetStr
* @param key string, defaultValue string
* @return string
**/
func GetStr(key string, defaultValue string) string {
	val := Get(key, defaultValue)
	return fmt.Sprintf("%v", val)
}

/**
* GetPassword
* @param key string, defaultValue string
* @return string
**/
func GetPassword(key string, defaultValue string) (string, error) {
	result := GetStr(key, defaultValue)
	str := fmt.Sprintf("%v", result)
	result, err := utility.FromBase64(str)
	if err != nil {
		return "", err
	}

	return result, nil
}

/**
* GetInt
* @param key string, defaultValue int
* @return int
**/
func GetInt(key string, defaultValue int) int {
	v := Get(key, defaultValue)
	scr := fmt.Sprintf("%v", v)
	if val, err := strconv.Atoi(scr); err == nil {
		return val
	}
	return defaultValue
}

/**
* GetInt64
* @param key string, defaultValuve int64
* @return int64
**/
func GetInt64(key string, defaultValue int64) int64 {
	v := Get(key, defaultValue)
	scr := fmt.Sprintf("%v", v)
	if val, err := strconv.ParseInt(scr, 10, 64); err == nil {
		return val
	}
	return defaultValue
}

/**
* GetBool
* @param key string, defaultValue bool
* @return bool
**/
func GetBool(key string, defaultValue bool) bool {
	v := Get(key, defaultValue)
	str := fmt.Sprintf("%v", v)
	if val, err := strconv.ParseBool(str); err == nil {
		return val
	}
	return defaultValue
}

/**
* GetFloat
* @param key string, defaultValue float64
* @return float64
**/
func GetFloat(key string, defaultValue float64) float64 {
	v := Get(key, defaultValue)
	scr := fmt.Sprintf("%v", v)
	if val, err := strconv.ParseFloat(scr, 64); err == nil {
		return val
	}
	return defaultValue
}

/**
* GetTime
* @param key string, defaultValue time.Time
* @return time.Time
**/
func GetTime(key string, defaultValue time.Time) time.Time {
	v := Get(key, defaultValue)
	str := fmt.Sprintf("%v", v)
	if val, err := time.Parse(time.RFC3339, str); err == nil {
		return val
	}
	return defaultValue
}

/**
* Validate
* @param keys []string
* @return error
**/
func Validate(keys []string) error {
	for _, key := range keys {
		if GetStr(key, "") == "" {
			return fmt.Errorf(msg.MSG_ATRIB_REQUIRED, key)
		}
	}
	return nil
}

/**
* SetEnvar
* @param values et.Json
* @return error
**/
func SetEnvar(values et.Json) {
	for k, v := range values {
		setEnvar(k, v)
	}
}

/**
* param
* @param key string, defaultValue interface{}
* @return interface{}
**/
func param(key string, defaultValue interface{}) interface{} {
	for i, arg := range os.Args[1:] {
		upc := strs.Uppcase(key)
		if strs.Uppcase(arg) == strs.Format("-%s", upc) {
			val := os.Args[i+2]
			setEnvar(key, val)
			return val
		} else if strs.Uppcase(arg) == strs.Format("--%s", upc) {
			val := os.Args[i+2]
			setEnvar(key, val)
			return val
		}
	}

	return defaultValue
}

/**
* ParamStr
* @param key string, defaultValue string
* @return string
**/
func ParamStr(key string, defaultValue string) string {
	val := param(key, defaultValue)
	return fmt.Sprintf("%v", val)
}

/**
* ParamInt
* @param key string, defaultValue int
* @return int
**/
func ParamInt(key string, defaultValue int) int {
	val := param(key, defaultValue)
	if val, ok := val.(int); ok {
		return val
	}
	return defaultValue
}

/**
* ParamInt64
* @param key string, defaultValue int64
* @return int64
**/
func ParamInt64(key string, defaultValue int64) int64 {
	val := param(key, defaultValue)
	if val, ok := val.(int64); ok {
		return val
	}
	return defaultValue
}

/**
* ParamBool
* @param key string, defaultValue bool
* @return bool
**/
func ParamBool(key string, defaultValue bool) bool {
	val := param(key, defaultValue)
	if val, ok := val.(bool); ok {
		return val
	}
	return defaultValue
}

/**
* ParamFloat
* @param key string, defaultValue float64
* @return float64
**/
func ParamFloat(key string, defaultValue float64) float64 {
	val := param(key, defaultValue)
	if val, ok := val.(float64); ok {
		return val
	}
	return defaultValue
}

/**
* ParamTime
* @param key string, defaultValue time.Time
* @return time.Time
**/
func ParamTime(key string, defaultValue time.Time) time.Time {
	val := param(key, defaultValue)
	if val, ok := val.(time.Time); ok {
		return val
	}
	return defaultValue
}
