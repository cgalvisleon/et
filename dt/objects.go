package dt

import (
	"encoding/json"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/et"
)

type Objects struct {
	Key     string            `json:"key"`
	Ok      bool              `json:"ok"`
	Count   int               `json:"count"`
	Objects map[string]Object `json:"objects"`
}

/**
* NewObjects
* @return *Objects
**/
func NewObjects(key string) *Objects {
	return &Objects{
		Key:     key,
		Objects: make(map[string]Object),
	}
}

/**
* Add
* @param obj Object
* @return bool
**/
func (s *Objects) Add(obj Object) bool {
	s.Objects[obj.Key] = obj
	s.Count = len(s.Objects)
	s.Ok = s.Count > 0

	return s.Ok
}

/**
* Up
* @param tag string
* @param pK string
* @param items et.Items
* @return bool
**/
func (s *Objects) Up(tag, pK string, items et.Items) bool {
	for _, item := range items.Result {
		id := item.Str(pK)
		key := GenId(tag, id)
		obj := NewObject(key)
		s.Add(*obj)
	}

	return false
}

/**
* Load
* @return error
**/
func (s *Objects) Load() bool {
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
func (s *Objects) Save() bool {
	val, err := et.Object(s)
	if err != nil {
		return false
	}

	cache.SetD(s.Key, val.ToString())

	return true
}

/**
* Drop
**/
func (s *Objects) Down() {
	cache.Delete(s.Key)
}
