package utility

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type Config interface {
	GetParams() et.Json
	Set(key string, value interface{}) error
	Exists(key string) bool
	Remove(key string) error
	Get(key string, def interface{}) interface{}
	GetStr(key string, def string) string
	GetInt(key string, def int) int
	GetFloat(key string, def float64) float64
	GetBool(key string, def bool) bool
}

type JConfig struct {
	params et.Json
}

/**
* GetParams
* @return et.Json
**/
func (s *JConfig) GetParams() et.Json {
	return s.params
}

/**
* Set
* @param string key, interface{} value
* @return error
**/
func (s *JConfig) Set(key string, value interface{}) error {
	s.params[key] = value
	return nil
}

/**
* Exists
* @param string key
* @return bool
**/
func (s *JConfig) Exists(key string) bool {
	_, ok := s.params[key]
	return ok
}

/**
* Remove
* @param string key
* @return error
**/
func (s *JConfig) Remove(key string) error {
	delete(s.params, key)
	return nil
}

/**
* Get
* @param string key, interface{} def
* @return (interface{}, error)
**/
func (s *JConfig) Get(key string, def interface{}) interface{} {
	result, ok := s.params[key]
	if !ok {
		return def
	}
	return result
}

/**
* GetStr
* @param string key, string def
* @return string
**/
func (s *JConfig) GetStr(key string, def string) string {
	result := s.Get(key, def)
	resultStr, ok := result.(string)
	if !ok {
		return def
	}
	if resultStr == "" {
		return envar.GetStr(key, def)
	}
	return resultStr
}

/**
* GetInt
* @param string key, int def
* @return int
**/
func (s *JConfig) GetInt(key string, def int) int {
	result := s.Get(key, def)
	resultInt, ok := result.(int)
	if !ok {
		return def
	}
	if resultInt == 0 {
		return envar.GetInt(key, def)
	}
	return resultInt
}

/**
* GetFloat
* @param string key, float64 def
* @return float64
**/
func (s *JConfig) GetFloat(key string, def float64) float64 {
	result := s.Get(key, def)
	resultFloat, ok := result.(float64)
	if !ok {
		return def
	}
	if resultFloat == 0 {
		return envar.GetNumber(key, def)
	}
	return resultFloat
}

/**
* GetBool
* @param string key, bool def
* @return bool
**/
func (s *JConfig) GetBool(key string, def bool) bool {
	result := s.Get(key, def)
	resultBool, ok := result.(bool)
	if !ok {
		return def
	}
	if !resultBool {
		return envar.GetBool(key, def)
	}
	return resultBool
}

/**
* NewJConfig
* @param et.Json params
* @return JConfig
**/
func NewJConfig(params et.Json) *JConfig {
	return &JConfig{
		params: params,
	}
}
