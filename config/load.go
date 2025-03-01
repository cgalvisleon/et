package config

import (
	"slices"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/utility"
)

/**
* Load
* @param et.Json config
* @return error
**/
func Load(config et.Json) error {
	upsetVal := func(k string, v interface{}) {
		switch val := v.(type) {
		case string:
			envar.UpSetStr(k, val)
		case int:
			envar.UpSetInt(k, val)
		case bool:
			envar.UpSetBool(k, val)
		case float64:
			envar.UpSetFloat(k, val)
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

	for k, v := range config {
		k = strs.Uppcase(k)

		switch val := v.(type) {
		case map[string]interface{}:
			upsetArray(val, slices.Contains([]string{"PASSWORD", "PASSWORDS", "PASS"}, k))
		case et.Json:
			upsetArray(val, slices.Contains([]string{"PASSWORD", "PASSWORDS", "PASS"}, k))
		default:
			upsetVal(k, val)
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
	return utility.ToBase64(password)
}

/**
* PasswordUnhash
* @param string password
* @return string
**/
func PasswordUnhash(password string) string {
	return utility.FromBase64(password)
}
