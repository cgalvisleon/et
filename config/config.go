package config

import (
	"encoding/json"
	"os"

	"github.com/cgalvisleon/et/et"
)

type Config struct {
	*et.Json
	file string `json:"-"`
}

func (s *Config) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.file, data, 0644)
}

func (s *Config) Load() error {
	data, err := os.ReadFile(s.file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, s)
	if err != nil {
		return err
	}

	return nil
}

var config *Config

func initConfig() {
	config = &Config{
		Json: &et.Json{},
		file: ".config",
	}
	config.Load()
}

/**
* Get
**/
func Set(key string, value interface{}) {
	if config == nil {
		initConfig()
	}

	config.Set(key, value)
	config.Save()
}

/**
* Get
* @param string key
* @return interface{}
**/
func Get(key string) interface{} {
	if config == nil {
		initConfig()
	}

	return config.Get(key)
}

/**
* Int
* @param string key
* @return int
**/
func Int(key string) int {
	return config.Int(key)
}

/**
* String
* @param string key
* @return string
**/
func String(key string) string {
	return config.String(key)
}

/**
* Bool
* @param string key
* @return bool
**/
func Bool(key string) bool {
	return config.Bool(key)
}

/**
* Number
* @param string key
* @return float64
**/
func Number(key string) float64 {
	return config.Num(key)
}
