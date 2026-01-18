package et

import (
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
	result, err := json.Marshal(vals)
	if err != nil {
		return ""
	}

	return string(result)
}
