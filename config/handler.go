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
	if CNF == nil {
		return errors.New(MSG_CONFIG_NOT_LOADED)
	}
	return CNF.Set(param, userId)
}

/**
* Get
* @param key string, def interface{}
* @return interface{}
**/
func Get(key string, def interface{}) interface{} {
	if CNF == nil {
		return envar.Get(key, def)
	}
	return CNF.Get(key, def)
}

/**
* GetStr
* @param key string, def string
* @return string
**/
func GetStr(key string, def string) string {
	if CNF == nil {
		return envar.GetStr(key, def)
	}
	return CNF.GetStr(key, def)
}

/**
* GetInt
* @param key string, def int
* @return int
**/
func GetInt(key string, def int) int {
	if CNF == nil {
		return envar.GetInt(key, def)
	}
	return CNF.GetInt(key, def)
}

/**
* GetInt64
* @param key string, def int64
* @return int64
**/
func GetInt64(key string, def int64) int64 {
	if CNF == nil {
		return envar.GetInt64(key, def)
	}
	return CNF.GetInt64(key, def)
}

/**
* GetFloat
* @param key string, def float64
* @return float64
**/
func GetFloat(key string, def float64) float64 {
	if CNF == nil {
		return envar.GetFloat(key, def)
	}
	return CNF.GetFloat(key, def)
}

/**
* GetBool
* @param key string, def bool
* @return bool
**/
func GetBool(key string, def bool) bool {
	if CNF == nil {
		return envar.GetBool(key, def)
	}
	return CNF.GetBool(key, def)
}

/**
* GetTime
* @param key string, def time.Time
* @return time.Time
**/
func GetTime(key string, def time.Time) time.Time {
	if CNF == nil {
		result := envar.GetStr(key, timezone.Format(def, timezone.RFC3339))
		val, err := timezone.Parse(timezone.RFC3339, result)
		if err != nil {
			return def
		}
		return val
	}
	return CNF.GetTime(key, def)
}

/**
* GetJson
* @param key string, def et.Json
* @return et.Json
**/
func GetJson(key string, def et.Json) et.Json {
	if CNF == nil {
		result := envar.GetStr(key, def.ToString())
		var resultJson et.Json
		err := json.Unmarshal([]byte(result), &resultJson)
		if err != nil {
			return def
		}
		return resultJson
	}
	return CNF.GetJson(key, def)
}
