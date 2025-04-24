package dt

import (
	"encoding/json"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type Object struct {
	et.Json
	Key string `json:"key"`
	Ok  bool   `json:"ok"`
}

func GenId(tag string, args ...interface{}) string {
	return cache.GenId(tag, ":", args)
}

/**
* NewObject
* @param key string
* @return *Object
**/
func NewObject(key string) *Object {
	return &Object{
		Json: et.Json{},
		Key:  key,
	}
}

/**
* Drop
* @param key string
**/
func Drop(key string) {
	cache.Delete(key)
}

/**
* Get
* @param key string
* @return Object
**/
func Get(key string) Object {
	obj := NewObject(key)
	obj.Load()

	return *obj
}

/**
* Up
* @param key string, data et.Json
* @return Object
**/
func Up(key string, data et.Json) Object {
	obj := NewObject(key)
	obj.Up(data)

	return *obj
}

/**
* UpItem
* @param key string, data et.Item
* @return Object
**/
func UpItem(key string, data et.Item) Object {
	obj := NewObject(key)
	if data.Ok {
		obj.Up(data.Result)
	} else {
		cache.Delete(key)
	}

	return *obj
}

/**
* Put
* @param id, key string, value interface{}
* @return Object
**/
func Put(id, key string, value interface{}) Object {
	obj := NewObject(key)
	if obj.Load() {
		obj.Set(id, value)
		obj.Save()
	}

	return *obj
}

/**
* Load
* @return error
*
 */
func (s *Object) Load() bool {
	val, err := cache.Get(s.Key, "")
	if err != nil {
		return false
	}

	if val == "" {
		return false
	}

	err = json.Unmarshal([]byte(val), &s)
	s.Ok = err != nil

	return s.Ok
}

/**
* Save
* @return bool
**/
func (s *Object) Save() bool {
	production := envar.GetBool(false, "PRODUCTION")
	if !production {
		return false
	}

	val := s.ToString()
	cache.SetD(s.Key, val, 1)

	return true
}

/**
* ToLoad
* @param data et.Json
* @return bool
**/
func (s *Object) ToLoad(data et.Json) {
	for key, val := range data {
		s.Set(key, val)
	}

	s.Ok = !s.IsEmpty()
}

/**
* ToData
* @param data et.Json
* @param source string
**/
func (s *Object) ToData(data et.Json, source string) {
	if data[source] == nil {
		s.ToLoad(data)
	} else {
		atribs := data.Json(source)
		s.ToLoad(atribs)
	}
	s.Delete([]string{source})
}

/**
* Up
* @param data et.Json
* @return bool
**/
func (s *Object) Up(data et.Json) bool {
	s.ToLoad(data)

	return s.Save()
}

/**
* Drop
**/
func (s *Object) Down() {
	cache.Delete(s.Key)
}

/**
* Item
* @return et.Json
**/
func (s *Object) Item() et.Item {
	return et.Item{
		Ok:     s.Ok,
		Result: s.Json,
	}
}
