package dt

import (
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type Object struct {
	et.Item
	key      string        `json:"-"`
	duration time.Duration `json:"-"`
}

/**
* newObject
* @param key string
* @return *Object
**/
func newObject(key string) *Object {
	return &Object{
		Item: et.Item{
			Ok:     false,
			Result: et.Json{},
		},
		key:      key,
		duration: 1 * time.Hour,
	}
}

/**
* up
* @param data et.Json
* @return bool
**/
func (s *Object) up(data et.Item, save bool) {
	s.Ok = data.Ok
	s.Result = et.Json{}
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
	production := envar.GetBool("PRODUCTION", true)
	if !production {
		return false
	}

	val := s.ToString()
	key := fmt.Sprintf("object:%s", s.key)
	cache.Set(key, val, s.duration)

	return true
}

/**
* Load
* @return error
*
 */
func (s *Object) load() error {
	key := fmt.Sprintf("object:%s", s.key)
	item, err := cache.GetItem(key)
	if err != nil {
		return err
	}

	s.up(item, false)

	return nil
}

/**
* Up
* @param key string, data et.Item
* @return Object
**/
func Up(key string, data et.Item) et.Item {
	obj := newObject(key)
	obj.up(data, data.Ok)

	return obj.Item
}

/**
* UpWithDuration
* @param key string, data et.Item, duration time.Duration
* @return Object
**/
func UpWithDuration(key string, data et.Item, duration time.Duration) Object {
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
func Get(key string) et.Item {
	obj := newObject(key)
	obj.load()

	return obj.Item
}

/**
* Drop
* @param key string
**/
func Drop(key string) {
	key = fmt.Sprintf("object:%s", key)
	cache.Delete(key)
}
