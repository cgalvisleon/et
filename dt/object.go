package dt

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/cache"
	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

type Object struct {
	Value  any           `json:"value"`
	Ok     bool          `json:"ok"`
	Type   string        `json:"type"`
	Key    string        `json:"key"`
	Expire time.Duration `json:"duration"`
}

/**
* ToString
* @return string
**/
func (s *Object) ToString() string {
	bt, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(bt)
}

/**
* newObject
* @param key string
* @return *Object
**/
func newObject(key string) *Object {
	duration := time.Duration(envar.GetInt64("CACHE_DURATION", 5)) * time.Minute
	return &Object{
		Key:    key,
		Expire: duration,
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

	cache.Set(s.Key, s.ToString(), s.Expire)
	return true
}

/**
* up
* @param data any, save bool
**/
func (s *Object) up(data any, save bool) {
	switch v := data.(type) {
	case et.Item:
		if !v.Ok {
			s.drop()
			return
		}
	case et.Items:
		if !v.Ok {
			s.drop()
			return
		}
	case et.Json:
		if v.IsEmpty() {
			s.drop()
			return
		}
	case et.List:
		if v.Count == 0 {
			s.drop()
			return
		}
	}

	s.Value = data
	s.Ok = true
	s.Type = fmt.Sprintf("%T", data)
	if save {
		s.save()
	}
}

/**
* drop
* @return error
**/
func (s *Object) drop() error {
	_, err := cache.Delete(s.Key)
	if err != nil {
		return err
	}

	return nil
}

/**
* String
* @return string
**/
func (s *Object) String() (string, bool) {
	result, ok := s.Value.(string)
	return result, ok
}

/**
* Int
* @return int
**/
func (s *Object) Int() (int, bool) {
	result, ok := s.Value.(int)
	return result, ok
}

/**
* Int64
* @return int64
**/
func (s *Object) Int64() (int64, bool) {
	result, ok := s.Value.(int64)
	return result, ok
}

/**
* Float
* @return float64
**/
func (s *Object) Float() (float64, bool) {
	result, ok := s.Value.(float64)
	return result, ok
}

/**
* Bool
* @return bool
**/
func (s *Object) Bool() (bool, bool) {
	result, ok := s.Value.(bool)
	return result, ok
}

/**
* Time
* @return time.Time
**/
func (s *Object) Time() (time.Time, bool) {
	result, ok := s.Value.(time.Time)
	return result, ok
}

/**
* Duration
* @return time.Duration
**/
func (s *Object) Duration() (time.Duration, bool) {
	result, ok := s.Value.(time.Duration)
	return result, ok
}

/**
* Array
* @return []any
**/
func (s *Object) Array() ([]any, bool) {
	result, ok := s.Value.([]any)
	return result, ok
}

/**
* Json
* @return et.Json
**/
func (s *Object) Json() (et.Json, bool) {
	result, ok := s.Value.(et.Json)
	return result, ok
}

/**
* Item
* @return et.Item
**/
func (s *Object) Item() (et.Item, bool) {
	result, ok := s.Value.(et.Item)
	return result, ok
}

/**
* Items
* @return et.Items
**/
func (s *Object) Items() (et.Items, bool) {
	result, ok := s.Value.(et.Items)
	return result, ok
}

/**
* List
* @return et.List
**/
func (s *Object) List() (et.List, bool) {
	result, ok := s.Value.(et.List)
	return result, ok
}

/**
* ArrayStr
* @return []string
**/
func (s *Object) ArrayStr() ([]string, bool) {
	result, ok := s.Value.([]string)
	return result, ok
}

/**
* ArrayInt
* @return []int
**/
func (s *Object) ArrayInt() ([]int, bool) {
	result, ok := s.Value.([]int)
	return result, ok
}

/**
* ArrayInt64
* @return []int64
**/
func (s *Object) ArrayInt64() ([]int64, bool) {
	result, ok := s.Value.([]int64)
	return result, ok
}

/**
* ArrayFloat
* @return []float64
**/
func (s *Object) ArrayFloat() ([]float64, bool) {
	result, ok := s.Value.([]float64)
	return result, ok
}

/**
* ArrayTime
* @return []time.Time
**/
func (s *Object) ArrayTime() ([]time.Time, bool) {
	result, ok := s.Value.([]time.Time)
	return result, ok
}

/**
* ArrayDuration
* @return []time.Duration
**/
func (s *Object) ArrayDuration() ([]time.Duration, bool) {
	result, ok := s.Value.([]time.Duration)
	return result, ok
}

/**
* ArrayJson
* @return []et.Json
**/
func (s *Object) ArrayJson() ([]et.Json, bool) {
	result, ok := s.Value.([]et.Json)
	return result, ok
}
