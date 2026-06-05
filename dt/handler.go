package dt

import (
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
* Get
* @param key string
* @return Object
**/
func Get(key string) *Object {
	key = fmt.Sprintf("object:%s", key)
	var result *Object
	exists, err := cache.GetObject(key, result)
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
