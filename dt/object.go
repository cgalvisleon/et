package dt

import (
	"encoding/json"
	"errors"
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
	duration := envar.GetDuration("CACHE_DURATION", 5*time.Minute)
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
* byte
* @return []byte
**/
func (s *Object) Byte() ([]byte, error) {
	result, ok := s.Value.([]byte)
	if !ok {
		return nil, errors.New(MSG_VALUE_IS_NOT_BYTE)
	}
	return result, nil
}

/**
* String
* @return string
**/
func (s *Object) String() (string, error) {
	result, ok := s.Value.(string)
	if !ok {
		return "", errors.New(MSG_VALUE_IS_NOT_STRING)
	}
	return result, nil
}

/**
* Int
* @return int
**/
func (s *Object) Int() (int, error) {
	result, ok := s.Value.(int)
	if !ok {
		return 0, errors.New(MSG_VALUE_IS_NOT_INT)
	}
	return result, nil
}

/**
* Int64
* @return int64
**/
func (s *Object) Int64() (int64, error) {
	result, ok := s.Value.(int64)
	if !ok {
		return 0, errors.New(MSG_VALUE_IS_NOT_INT64)
	}
	return result, nil
}

/**
* Float
* @return float64
**/
func (s *Object) Float() (float64, error) {
	result, ok := s.Value.(float64)
	if !ok {
		return 0, errors.New(MSG_VALUE_IS_NOT_FLOAT)
	}
	return result, nil
}

/**
* Bool
* @return bool
**/
func (s *Object) Bool() (bool, error) {
	result, ok := s.Value.(bool)
	if !ok {
		return false, errors.New(MSG_VALUE_IS_NOT_BOOL)
	}
	return result, nil
}

/**
* Time
* @return time.Time
**/
func (s *Object) Time() (time.Time, error) {
	result, ok := s.Value.(time.Time)
	if !ok {
		return time.Time{}, errors.New(MSG_VALUE_IS_NOT_TIME)
	}
	return result, nil
}

/**
* Duration
* @return time.Duration
**/
func (s *Object) Duration() (time.Duration, error) {
	result, ok := s.Value.(time.Duration)
	if !ok {
		return 0, errors.New(MSG_VALUE_IS_NOT_DURATION)
	}
	return result, nil
}

/**
* Array
* @return []any
**/
func (s *Object) Array() ([]any, error) {
	result, ok := s.Value.([]any)
	if !ok {
		return nil, errors.New(MSG_VALUE_IS_NOT_ARRAY)
	}
	return result, nil
}

/**
* Json
* @return et.Json
**/
func (s *Object) Json() (et.Json, error) {
	result, ok := s.Value.(et.Json)
	if !ok {
		return et.Json{}, errors.New(MSG_VALUE_IS_NOT_JSON)
	}
	return result, nil
}

/**
* Item
* @return et.Item
**/
func (s *Object) Item() (et.Item, error) {
	result, ok := s.Value.(et.Item)
	if !ok {
		return et.Item{}, errors.New(MSG_VALUE_IS_NOT_ITEM)
	}
	return result, nil
}

/**
* Items
* @return et.Items
**/
func (s *Object) Items() (et.Items, error) {
	result, ok := s.Value.(et.Items)
	if !ok {
		return et.Items{}, errors.New(MSG_VALUE_IS_NOT_ITEMS)
	}
	return result, nil
}

/**
* List
* @return et.List
**/
func (s *Object) List() (et.List, error) {
	result, ok := s.Value.(et.List)
	if !ok {
		return et.List{}, errors.New(MSG_VALUE_IS_NOT_LIST)
	}
	return result, nil
}

/**
* ArrayStr
* @return []string
**/
func (s *Object) ArrayStr() ([]string, error) {
	result, ok := s.Value.([]string)
	if !ok {
		return nil, errors.New(MSG_VALUE_IS_NOT_ARRAY_STR)
	}
	return result, nil
}

/**
* ArrayInt
* @return []int
**/
func (s *Object) ArrayInt() ([]int, error) {
	result, ok := s.Value.([]int)
	if !ok {
		return nil, errors.New(MSG_VALUE_IS_NOT_ARRAY_INT)
	}
	return result, nil
}

/**
* ArrayInt64
* @return []int64
**/
func (s *Object) ArrayInt64() ([]int64, error) {
	result, ok := s.Value.([]int64)
	if !ok {
		return nil, errors.New(MSG_VALUE_IS_NOT_ARRAY_INT64)
	}
	return result, nil
}

/**
* ArrayFloat
* @return []float64
**/
func (s *Object) ArrayFloat() ([]float64, error) {
	result, ok := s.Value.([]float64)
	if !ok {
		return nil, errors.New(MSG_VALUE_IS_NOT_ARRAY_FLOAT)
	}
	return result, nil
}

/**
* ArrayTime
* @return []time.Time
**/
func (s *Object) ArrayTime() ([]time.Time, error) {
	result, ok := s.Value.([]time.Time)
	if !ok {
		return nil, errors.New(MSG_VALUE_IS_NOT_ARRAY_TIME)
	}
	return result, nil
}

/**
* ArrayDuration
* @return []time.Duration
**/
func (s *Object) ArrayDuration() ([]time.Duration, error) {
	result, ok := s.Value.([]time.Duration)
	if !ok {
		return nil, errors.New(MSG_VALUE_IS_NOT_ARRAY_DURATION)
	}
	return result, nil
}

/**
* ArrayJson
* @return []et.Json
**/
func (s *Object) ArrayJson() ([]et.Json, error) {
	result, ok := s.Value.([]et.Json)
	if !ok {
		return nil, errors.New(MSG_VALUE_IS_NOT_ARRAY_JSON)
	}
	return result, nil
}
