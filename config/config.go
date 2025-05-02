package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/cgalvisleon/et/arg"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/strs"
)

type Config struct {
	*et.Json
	file string `json:"-"`
}

/**
* Save
* @return error
**/
func (s *Config) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.file, data, 0644)
}

/**
* Load
* @return error
**/
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

/**
* Load
* @return error
**/
func Load() {
	config = &Config{
		Json: &et.Json{},
		file: ".config",
	}

	config.Load()
}

/**
* Set
* @param key string, value interface{}
* @return interface{}
**/
func Set(key string, value interface{}) interface{} {
	if config == nil {
		return envar.Set(key, value)
	}

	config.Set(key, value)
	config.Save()

	return value
}

/**
* SetStrByArg
* @param name string, defaultVal string
* @return string
**/
func SetStrByArg(name string, defaultVal string) string {
	val, ok := arg.Get(name, defaultVal)
	if ok {
		Set(name, val)
	}

	return val
}

/**
* SetIntByArg
* @param name string, defaultVal int
* @return int
**/
func SetIntByArg(name string, defaultVal int) int {
	val, ok := arg.GetInt(name, defaultVal)
	if ok {
		Set(name, val)
	}

	return val
}

/**
* SetInt64ByArg
* @param name string, defaultVal int64
* @return int64
**/
func SetInt64ByArg(name string, defaultVal int64) int64 {
	val, ok := arg.GetInt64(name, defaultVal)
	if ok {
		Set(name, val)
	}

	return val
}

/**
* SetBoolByArg
* @param name string, defaultVal bool
* @return bool
**/
func SetBoolByArg(name string, defaultVal bool) bool {
	val, ok := arg.GetBool(name, defaultVal)
	if ok {
		Set(name, val)
	}

	return val
}

/**
* Get
* @param key string, interface{} defaultValue
* @return interface{}
**/
func Get(key string, defaultValue interface{}) interface{} {
	if config == nil {
		return envar.Get(key, defaultValue)
	}

	return config.ValAny(defaultValue, key)
}

/**
* Int
* @param key string, defaultValue int
* @return int
**/
func Int(key string, defaultValue int) int {
	if config == nil {
		return envar.GetInt(key, defaultValue)
	}

	return config.ValInt(defaultValue, key)
}

/**
* Int64
* @param key string, defaultValue int64
* @return int64
**/
func Int64(key string, defaultValue int64) int64 {
	if config == nil {
		return envar.GetInt64(key, defaultValue)
	}

	return config.ValInt64(defaultValue, key)
}

/**
* String
* @param string key, defaultValue string
* @return string
**/
func String(key string, defaultValue string) string {
	if config == nil {
		return envar.GetStr(key, defaultValue)
	}

	return config.ValStr(defaultValue, key)
}

/**
* Str
* @param string key, defaultValue string
* @return string
**/
func Str(key string, defaultValue string) string {
	return String(key, defaultValue)
}

/**
* Bool
* @param string key, defaultValue bool
* @return bool
**/
func Bool(key string, defaultValue bool) bool {
	if config == nil {
		return envar.GetBool(key, defaultValue)
	}

	return config.ValBool(defaultValue, key)
}

/**
* Number
* @param string key, defaultValue float64
* @return float64
**/
func Number(key string, defaultValue float64) float64 {
	if config == nil {
		return envar.GetNumber(key, defaultValue)
	}

	return config.ValNum(defaultValue, key)
}

/**
* setEnvar
* @param key string, value interface{}
* @return error
**/
func setEnvar(key string, value interface{}) {
	upsetVal := func(k string, v interface{}) {
		switch val := v.(type) {
		case string:
			envar.SetStr(k, val)
		case int:
			envar.SetInt(k, val)
		case bool:
			envar.SetBool(k, val)
		case float64:
			envar.SetFloat(k, val)
		}
	}

	upsetArray := func(v map[string]interface{}, isPassword bool) {
		for kk, vv := range v {
			kk = strs.Uppcase(kk)
			if isPassword {
				vv = PasswordUnhash(vv.(string))
			}
			upsetVal(kk, vv)
		}
	}

	key = strs.Uppcase(key)

	switch val := value.(type) {
	case map[string]interface{}:
		upsetArray(val, slices.Contains([]string{"PASSWORD", "PASSWORDS", "PASS"}, key))
	case et.Json:
		upsetArray(val, slices.Contains([]string{"PASSWORD", "PASSWORDS", "PASS"}, key))
	default:
		upsetVal(key, val)
	}
}

/**
* SetToEnvar
* @param et.Json config
* @return error
**/
func SetToEnvar(config et.Json) error {
	for k, v := range config {
		setEnvar(k, v)
	}

	return nil
}

/**
* SetToConfig
* @param et.Json config
* @return error
**/
func SetToConfig(config et.Json) error {
	upsetVal := func(k string, v interface{}) {
		config.Set(k, v)
	}

	upsetArray := func(v map[string]interface{}, isPassword bool) {
		for kk, vv := range v {
			kk = strs.Uppcase(kk)
			if isPassword {
				vv = PasswordUnhash(vv.(string))
			}
			upsetVal(kk, vv)
		}
	}

	for key, value := range config {
		switch val := value.(type) {
		case map[string]interface{}:
			upsetArray(val, slices.Contains([]string{"PASSWORD", "PASSWORDS", "PASS"}, key))
		case et.Json:
			upsetArray(val, slices.Contains([]string{"PASSWORD", "PASSWORDS", "PASS"}, key))
		default:
			upsetVal(key, val)
		}
	}

	return nil
}

/**
* PasswordHash
* @param string password
* @return string
**/
func PasswordHash(password string) string {
	return base64.StdEncoding.EncodeToString([]byte(password))
}

/**
* PasswordUnhash
* @param string password
* @return string
**/
func PasswordUnhash(password string) string {
	result, err := base64.StdEncoding.DecodeString(password)
	if err != nil {
		return ""
	}

	return string(result)
}

var definedVars = make(map[string]bool)

/**
* Define
* @param vars []string
**/
func Define(vars []string) {
	for _, k := range vars {
		k = strs.Uppcase(k)
		definedVars[k] = true
	}
}

/**
* Valid
* @return error
**/
func Valid() error {
	for k := range definedVars {
		k = strs.Uppcase(k)
		val := Get(k, "")
		if val == "" {
			return fmt.Errorf(msg.ERR_ENV_REQUIRED, k)
		}
	}

	return nil
}

/**
* Validate
* @param vars []string
* @return error
**/
func Validate(vars []string) error {
	Define(vars)
	return Valid()
}

/**
* Reload
**/
func Reload() {
	App.Reload()
}
