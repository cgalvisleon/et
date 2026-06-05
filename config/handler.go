package config

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/timezone"
)

/**
* Set
* @param param et.Json, userId string
* @return error
**/
func Set(param et.Json, userId string) error {
	if cnf == nil {
		return errors.New(MSG_CONFIG_NOT_LOADED)
	}
	return cnf.Set(param, userId)
}

/**
* Get
* @param key string, def interface{}
* @return interface{}
**/
func Get(key string, def interface{}) interface{} {
	if cnf == nil {
		return envar.Get(key, def)
	}
	return cnf.Get(key, def)
}

/**
* GetStr
* @param key string, def string
* @return string
**/
func GetStr(key string, def string) string {
	if cnf == nil {
		return envar.GetStr(key, def)
	}
	return cnf.GetStr(key, def)
}

/**
* GetInt
* @param key string, def int
* @return int
**/
func GetInt(key string, def int) int {
	if cnf == nil {
		return envar.GetInt(key, def)
	}
	return cnf.GetInt(key, def)
}

/**
* GetInt64
* @param key string, def int64
* @return int64
**/
func GetInt64(key string, def int64) int64 {
	if cnf == nil {
		return envar.GetInt64(key, def)
	}
	return cnf.GetInt64(key, def)
}

/**
* GetFloat
* @param key string, def float64
* @return float64
**/
func GetFloat(key string, def float64) float64 {
	if cnf == nil {
		return envar.GetFloat(key, def)
	}
	return cnf.GetFloat(key, def)
}

/**
* GetBool
* @param key string, def bool
* @return bool
**/
func GetBool(key string, def bool) bool {
	if cnf == nil {
		return envar.GetBool(key, def)
	}
	return cnf.GetBool(key, def)
}

/**
* GetTime
* @param key string, def time.Time
* @return time.Time
**/
func GetTime(key string, def time.Time) time.Time {
	if cnf == nil {
		result := envar.GetStr(key, timezone.Format(def, timezone.RFC3339))
		val, err := timezone.Parse(timezone.RFC3339, result)
		if err != nil {
			return def
		}
		return val
	}
	return cnf.GetTime(key, def)
}

/**
* GetJson
* @param key string, def et.Json
* @return et.Json
**/
func GetJson(key string, def et.Json) et.Json {
	if cnf == nil {
		result := envar.GetStr(key, def.ToString())
		var resultJson et.Json
		err := json.Unmarshal([]byte(result), &resultJson)
		if err != nil {
			return def
		}
		return resultJson
	}
	return cnf.GetJson(key, def)
}
