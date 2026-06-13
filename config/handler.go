package config

import (
	"fmt"

	"github.com/cgalvisleon/et/envar"
)

/**
* IsLoad
* @return bool
**/
func IsLoad() bool {
	return cnf == nil
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

/**
* Set
* @param param map[string]interface{}, userId string
* @return error
**/
func Set(param map[string]interface{}) *Config {
	if cnf == nil || cnf.store == nil {
		for key, value := range param {
			envar.Set(key, value)
		}
		return nil
	}
	return cnf.Set(param)
}

/**
* Get
* @param key string, def interface{}
* @return interface{}
**/
func Get(key string, def interface{}) interface{} {
	if cnf == nil || cnf.store == nil {
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
	if cnf == nil || cnf.store == nil {
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
	if cnf == nil || cnf.store == nil {
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
	if cnf == nil || cnf.store == nil {
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
	if cnf == nil || cnf.store == nil {
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
	if cnf == nil || cnf.store == nil {
		return envar.GetBool(key, def)
	}
	return cnf.GetBool(key, def)
}
