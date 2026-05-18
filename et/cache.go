package et

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/msg"
)

type CacheStore interface {
	Set(key string, val interface{}, duration time.Duration) interface{}
	GetItem(key string) (Item, error)
	Delete(key string) (int64, error)
}

var store CacheStore

/**
* SetCacheStore
* @param s CacheStore
**/
func SetCacheStore(s CacheStore) {
	store = s
}

/**
* Up
* @param key string, data Item
* @return Object
**/
func Up(key string, data Item, duration ...time.Duration) Item {
	if store == nil {
		return data
	}

	if len(duration) == 0 {
		expiration := envar.GetInt64("OBJECT_DURATION", 15)
		duration = append(duration, time.Duration(expiration)*time.Minute)
	}

	key = fmt.Sprintf("object:%s", key)
	store.Set(key, data, duration[0])
	return data
}

/**
* Get
* @param key string
* @return (Item, error)
**/
func Get(key string) Item {
	if store == nil {
		return Item{Result: Json{}}
	}

	key = fmt.Sprintf("object:%s", key)
	item, err := store.GetItem(key)
	if err != nil {
		return Item{Result: Json{}}
	}

	return item
}

/**
* Drop
* @param key string
* @return error
**/
func Drop(key string) error {
	if store == nil {
		return fmt.Errorf(msg.MSG_STORE_NOT_INITIALIZED)
	}

	key = fmt.Sprintf("object:%s", key)
	_, err := store.Delete(key)
	return err
}
