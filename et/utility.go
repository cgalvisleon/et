package et

import (
	"encoding/json"
	"fmt"
)

/**
* valAny
* @param data Json, defaultVal interface{}, atribs ...string
* @return any
**/
func valAny(data Json, defaultVal interface{}, atribs ...string) interface{} {
	current := data.Clone()
	n := len(atribs)
	for i := 0; i < n; i++ {
		val := current[atribs[i]]
		if val == nil {
			return defaultVal
		}

		if i == n-1 {
			return val
		}

		switch v := val.(type) {
		case Json:
			current = v
		case map[string]interface{}:
			current = Json(v)
		default:
			return defaultVal
		}
	}

	return defaultVal
}

/**
* valJson
* @param data Json, defaultVal Json, atribs ...string
* @return Json
**/
func valJson(data Json, defaultVal Json, atribs ...string) (Json, error) {
	val := valAny(data, defaultVal, atribs...)

	switch v := val.(type) {
	case Json:
		return v, nil
	case map[string]interface{}:
		return Json(v), nil
	case string:
		var result map[string]interface{}
		err := json.Unmarshal([]byte(v), &result)
		if err != nil {
			return defaultVal, fmt.Errorf(`error:%v type:%T`, err.Error(), val)
		}

		return result, nil
	default:
		src, err := Serialize(val)
		if err != nil {
			return defaultVal, fmt.Errorf(`error:%v type:%T`, err.Error(), val)
		}

		result, err := Object(string(src))
		if err != nil {
			return defaultVal, fmt.Errorf(`error:%v type:%T`, err.Error(), val)
		}

		return result, nil
	}
}

/**
* Serialize convert a interface to a []byte
* @param src any
* @return []byte, error
**/
func Serialize(src any) ([]byte, error) {
	result, err := json.Marshal(src)
	if err != nil {
		return []byte{}, err
	}

	return result, nil
}

/**
* Object convert a interface to a json
* @param src interface{}
* @return Json, error
**/
func Object(src string) (Json, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(src), &result); err != nil {
		return result, err
	}

	return result, nil
}
