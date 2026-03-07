package et

import (
	"bytes"
	"encoding/json"
)

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

/**
* ToString
* @param vals []Json
* @return string
**/
func ToString(vals interface{}) string {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false) // 🔥 clave

	if err := encoder.Encode(vals); err != nil {
		panic(err)
	}

	return buf.String()
}

/**
* SetNested
* @param data Json, keys []string, value interface{}
* @return Json
**/
func SetNested(data Json, keys []string, value interface{}) Json {
	result := data.Clone()
	for i, key := range keys {
		_, ok := result[key]
		if !ok {
			return Json{}
		}

		if i == len(keys)-1 {
			result[key] = value
		} else {
			result = result.Json(key)
		}
	}

	return result
}
