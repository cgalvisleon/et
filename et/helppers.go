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
* ToJson convert a string to a json
* @param src string
* @return Json, error
**/
func ToJson(src string) (Json, error) {
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
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(vals); err != nil {
		panic(err)
	}

	return buf.String()
}
