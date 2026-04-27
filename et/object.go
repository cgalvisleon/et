package et

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/msg"
)

type Store interface {
	Set(key string, val interface{}, expiration time.Duration) interface{}
	GetItem(key string) (Item, error)
	Delete(key string) (int64, error)
}

var store Store
var isProduction = envar.GetBool("PRODUCTION", true)

/**
* SetStore
* @param s Store
**/
func SetStore(s Store) {
	store = s
}

type Object struct {
	Item
	key      string        `json:"-"`
	duration time.Duration `json:"-"`
	store    Store         `json:"-"`
}

/**
* newObject
* @param key string
* @return *Object
**/
func newObject(key string) *Object {
	return &Object{
		Item: Item{
			Ok:     false,
			Result: Json{},
		},
		key:      key,
		duration: 1 * time.Hour,
		store:    store,
	}
}

/**
* up
* @param data Json
* @return bool
**/
func (s *Object) up(data Item, save bool) {
	s.Ok = data.Ok
	s.Result = Json{}
	if !s.Ok {
		Drop(s.key)
		return
	}

	for key, val := range data.Result {
		s.Set(key, val)
	}

	if save {
		s.save()
	}
}

/**
* save
* @return bool
**/
func (s *Object) save() bool {
	if !isProduction {
		return false
	}

	if s.store == nil {
		return false
	}

	val := s.ToString()
	key := fmt.Sprintf("object:%s", s.key)
	s.store.Set(key, val, s.duration)

	return true
}

/**
* Load
* @return error
*
 */
func (s *Object) load() error {
	if s.store == nil {
		return fmt.Errorf(msg.MSG_STORE_NOT_INITIALIZED)
	}

	key := fmt.Sprintf("object:%s", s.key)
	item, err := s.store.GetItem(key)
	if err != nil {
		return err
	}

	s.up(item, false)

	return nil
}

/**
* Up
* @param key string, data Item
* @return Object
**/
func Up(key string, data Item) Item {
	obj := newObject(key)
	obj.up(data, data.Ok)

	return obj.Item
}

/**
* UpWithDuration
* @param key string, data Item, duration time.Duration
* @return Object
**/
func UpWithDuration(key string, data Item, duration time.Duration) Object {
	obj := newObject(key)
	obj.duration = duration
	obj.up(data, data.Ok)

	return *obj
}

/**
* Get
* @param key string
* @return Object
**/
func Get(key string) Item {
	obj := newObject(key)
	obj.load()

	return obj.Item
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
