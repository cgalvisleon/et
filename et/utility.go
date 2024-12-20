package et

import (
	"encoding/json"
)

/**
* Object convert a interface to a json
* @param src interface{}
* @return Json
* @return error
**/
func Object(src interface{}) (Json, error) {
	j, err := json.Marshal(src)
	if err != nil {
		return Json{}, err
	}

	result := Json{}
	err = json.Unmarshal(j, &result)
	if err != nil {
		return Json{}, err
	}

	return result, nil
}

/**
* Array convert a interface to a []Json
* @param src interface{}
* @return []Json
* @return error
**/
func Array(src interface{}) ([]interface{}, error) {
	var result []interface{} = []interface{}{}
	j, err := json.Marshal(src)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(j, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* ToItem convert a json to a item
* @param src interface{}
* @return Item
**/
func ToItem(src interface{}) (Item, error) {
	j, err := json.Marshal(src)
	if err != nil {
		return Item{}, err
	}

	result := Item{}
	err = json.Unmarshal(j, &result)
	if err != nil {
		return Item{}, err
	}

	return result, nil
}
