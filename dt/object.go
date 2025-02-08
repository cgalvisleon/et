package dt

import (
	"encoding/json"

	"github.com/cgalvisleon/et/cache"
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
		Key: key,
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
* Load
* @return error
**/
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
	val := s.ToString()
	cache.SetD(s.Key, val)

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
