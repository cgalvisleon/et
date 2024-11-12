package race

import (
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/utility"
)

type Value struct {
	value interface{}
	mutex sync.RWMutex
}

func NewValue(value interface{}) *Value {
	return &Value{
		value: value,
		mutex: sync.RWMutex{},
	}
}

/**
* Set: New value
* @param interface{}
**/
func (s *Value) Set(value interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.value = value
}

/**
* Delete: Delete value
**/
func (s *Value) Delete() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.value = nil
}

/**
* Get: Get value
* @return interface{}
**/
func (s *Value) Get() interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.value
}

/**
* String: Get value as string
* @return string
**/
func (s *Value) String() string {
	return fmt.Sprintf("%v", s.Get())
}

/**
* Int: Get value as int
* @return int
**/
func (s *Value) Int() int {
	result, ok := s.Get().(int)
	if !ok {
		return 0
	}

	return result
}

/**
* Float64: Get value as float64
* @return float64
**/
func (s *Value) Float64() float64 {
	result, ok := s.Get().(float64)
	if !ok {
		return 0
	}

	return result
}

/**
* Bool: Get value as bool
* @return bool
**/
func (s *Value) Bool() bool {
	result, ok := s.Get().(bool)
	if !ok {
		return false
	}

	return result
}

/**
* Time: Get value as time.Time
* @return time.Time
**/
func (s *Value) Time() time.Time {
	result, ok := s.Get().(time.Time)
	if !ok {
		return utility.NowTime()
	}

	return result
}

/**
* Array: Get value as []interface{}
* @return []interface{}
**/
func (s *Value) Array() []interface{} {
	result, ok := s.Get().([]interface{})
	if !ok {
		return []interface{}{}
	}

	return result
}

/**
* Map: Get value as map[string]interface{}
* @return map[string]interface{}
**/
func (s *Value) Map() map[string]interface{} {
	result, ok := s.Get().(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}

	return result
}

/**
* StringArray: Get value as []string
* @return []string
**/
func (s *Value) StringArray() []string {
	result, ok := s.Get().([]string)
	if !ok {
		return []string{}
	}

	return result
}

/**
* IntArray: Get value as []int
* @return []int
**/
func (s *Value) IntArray() []int {
	result, ok := s.Get().([]int)
	if !ok {
		return []int{}
	}

	return result
}

/**
* Float64Array: Get value as []float64
* @return []float64
**/
func (s *Value) Float64Array() []float64 {
	result, ok := s.Get().([]float64)
	if !ok {
		return []float64{}
	}

	return result
}

/**
* BoolArray: Get value as []bool
* @return []bool
**/
func (s *Value) IsNil() bool {
	return s.Get() == nil
}

/**
* IsNil: Check if value is nil
* @return bool
**/
func (s *Value) MapRange(f func(key, value interface{}) bool) {
	if s.IsNil() {
		return
	}

	for key, value := range s.Map() {
		if !f(key, value) {
			break
		}
	}
}

/**
* MapRange: Iterate over map
* @param func (key, value interface{}) bool
**/
func (s *Value) ArrayRange(f func(key, value interface{}) bool) {
	if s.IsNil() {
		return
	}

	for key, value := range s.Array() {
		if !f(key, value) {
			break
		}
	}
}

/**
* ArrayRange: Iterate over array
* @param func (key, value interface{}) bool
**/
func (s *Value) Range(f func(key, value interface{}) bool) {
	if s.IsNil() {
		return
	}

	switch s.Get().(type) {
	case map[string]interface{}:
		s.MapRange(f)
	case []interface{}:
		s.ArrayRange(f)
	}
}

func (s *Value) Increase(n int) {
	switch v := s.Get().(type) {
	case int:
		s.Set(v + n)
	case float64:
		s.Set(v + float64(n))
	}
}
