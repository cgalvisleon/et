package utility

import (
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type Config struct {
	params et.Json
}

/**
* GetParams
* @return et.Json
**/
func (s *Config) GetParams() et.Json {
	return s.params
}

/**
* Set
* @param string key, interface{} value
* @return error
**/
func (s *Config) Set(key string, value interface{}) error {
	s.params[key] = value
	return nil
}

/**
* Exists
* @param string key
* @return bool
**/
func (s *Config) Exists(key string) bool {
	_, ok := s.params[key]
	return ok
}

/**
* Remove
* @param string key
* @return error
**/
func (s *Config) Remove(key string) error {
	delete(s.params, key)
	return nil
}

/**
* Get
* @param string key, interface{} def
* @return (interface{}, error)
**/
func (s *Config) Get(key string, def interface{}) interface{} {
	result, ok := s.params[key]
	if !ok {
		return envar.Get(key, def)
	}
	return result
}

/**
* GetStr
* @param string key, string def
* @return string
**/
func (s *Config) GetStr(key string, def string) string {
	result := s.Get(key, def)
	resultStr, ok := result.(string)
	if !ok {
		return def
	}
	return resultStr
}

/**
* GetInt
* @param string key, int def
* @return int
**/
func (s *Config) GetInt(key string, def int) int {
	result := s.Get(key, def)
	resultInt, ok := result.(int)
	if !ok {
		return def
	}
	return resultInt
}

/**
* GetFloat
* @param string key, float64 def
* @return float64
**/
func (s *Config) GetFloat(key string, def float64) float64 {
	result := s.Get(key, def)
	resultFloat, ok := result.(float64)
	if !ok {
		return def
	}
	return resultFloat
}

/**
* GetBool
* @param string key, bool def
* @return bool
**/
func (s *Config) GetBool(key string, def bool) bool {
	result := s.Get(key, def)
	resultBool, ok := result.(bool)
	if !ok {
		return def
	}
	return resultBool
}

/**
* NewConfig
* @param et.Json params
* @return Config
**/
func NewConfig(params et.Json) Config {
	return Config{
		params: params,
	}
}
