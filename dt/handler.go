package dt

import (
	"encoding/json"
	"fmt"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
)

/**
* Up
* @param key string, data et.Item
* @return Object
**/
func Up(key string, data any) Object {
	key = fmt.Sprintf("object:%s", key)
	obj := newObject(key)
	production := envar.GetBool("PRODUCTION", true)
	obj.up(data, production)

	return *obj
}

/**
* GetObject
* @param key string, dest any
* @return bool, error
**/
func GetObject(key string, dest any) (bool, error) {
	key = fmt.Sprintf("object:%s", key)
	var result *Object
	exists, err := cache.GetObject(key, &result)
	if err != nil {
		return false, err
	}

	bt, err := result.Byte()
	if err != nil {
		return false, err
	}

	err = json.Unmarshal(bt, dest)
	if err != nil {
		return false, err
	}

	return exists, nil
}

/**
* Get
* @param key string
* @return Object
**/
func Get(key string) *Object {
	key = fmt.Sprintf("object:%s", key)
	var result *Object
	exists, err := cache.GetObject(key, &result)
	if err != nil {
		return nil
	}

	if !exists {
		return nil
	}

	return result
}

/**
* Drop
* @param key string
**/
func Drop(key string) error {
	key = fmt.Sprintf("object:%s", key)
	_, err := cache.Delete(key)
	if err != nil {
		return err
	}

	return nil
}
