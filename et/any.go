package et

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/timezone"
)

type Any struct {
	value interface{}
}

/**
* New, create new Any
* @param val interface{}
* @return *Any
**/
func NewAny(val interface{}) *Any {
	return &Any{
		value: val,
	}
}

/**
* Set, set value to Any
* @param val interface{}
* @return interface{}
**/
func (a *Any) Set(val interface{}) interface{} {
	a.value = val
	return a.value
}

/**
* Val, get value from Any
* @return interface{}
**/
func (a *Any) Val() interface{} {
	return a.value
}

/**
* Str, get value from Any as string
* @return string
**/
func (a *Any) Str() string {
	result, ok := a.value.(string)
	if !ok {
		result = fmt.Sprintf(`%v`, a.value)
	}

	return result
}

/**
* Int, get value from Any as int
* @return int
**/
func (a *Any) Int() int {
	switch v := a.value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case float32:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	default:
		result, ok := a.value.(int)
		if !ok {
			log.Println("Any value int not conver:", reflect.TypeOf(v), "value:", v)
			return 0
		}
		return result
	}
}

/**
* Num, get value from Any as float64
* @return float64
**/
func (a *Any) Num() float64 {
	switch v := a.value.(type) {
	case int:
		return float64(v)
	case float64:
		return v
	case float32:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	default:
		log.Println("Any value number not conver:", reflect.TypeOf(v), "value:", v)
		return 0
	}
}

/**
* Bool, get value from Any as boolean
* @return bool
**/
func (a *Any) Bool() bool {
	switch v := a.value.(type) {
	case bool:
		return v
	case int:
		if v == 1 {
			return true
		}

		return false
	case string:
		if strings.ToLower(v) == "true" {
			return true
		}

		if strings.ToLower(v) == "si" {
			return true
		}

		return false
	default:
		log.Println("Any value boolean not conver:", reflect.TypeOf(v), "value:", v)
		return false
	}
}

/**
* Time, get value from Any as time.Time
* @return time.Time
**/
func (a *Any) Time() time.Time {
	_default := timezone.NowTime()
	switch v := a.value.(type) {
	case int:
		return _default
	case string:
		layout := "2006-01-02T15:04:05.000Z"
		result, err := time.Parse(layout, v)
		if err != nil {
			return _default
		}
		return result
	case time.Time:
		return v
	default:
		log.Println("Any value time not conver:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}
