package dt

import (
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
)

var duration = 1 * time.Hour

type Object struct {
	et.Item
	Key string `json:"key"`
}

/**
* SetDuration
* @param d time.Duration
**/
func SetDuration(d time.Duration) {
	duration = d
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
		Key: key,
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
		cache.Delete(s.Key)
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
	production := config.App.Production
	if !production {
		return false
	}

	val := s.ToString()
	expiration := int(duration.Seconds())
	cache.Set(s.Key, val, expiration)

	return true
}

/**
* Load
* @return error
*
 */
func (s *Object) load() error {
	item, err := cache.GetItem(s.Key)
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
func Up(key string, data et.Item) Object {
	obj := newObject(key)
	obj.up(data, true)

	return *obj
}

/**
* Get
* @param key string
* @return Object
**/
func Get(key string) Object {
	obj := newObject(key)
	obj.load()

	return *obj
}

/**
* Drop
* @param key string
**/
func Drop(key string) {
	cache.Delete(key)
}
