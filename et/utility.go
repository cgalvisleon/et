package et

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

/**
* Object convert a interface to a json
* @param src interface{}
* @return Json
* @return error
**/
func Object(src interface{}) (Json, error) {
	j, err := json.Marshal(src)
	if err != nil {
		return Json{}, err
	}

	result := Json{}
	err = json.Unmarshal(j, &result)
	if err != nil {
		return Json{}, err
	}

	return result, nil
}

/**
* Array convert a interface to a []Json
* @param src interface{}
* @return []Json
* @return error
**/
func Array(src interface{}) ([]interface{}, error) {
	var result []interface{} = []interface{}{}
	j, err := json.Marshal(src)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal(j, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

/**
* ToItem convert a json to a item
* @param src interface{}
* @return Item
**/
func ToItem(src interface{}) (Item, error) {
	j, err := json.Marshal(src)
	if err != nil {
		return Item{}, err
	}

	result := Item{}
	err = json.Unmarshal(j, &result)
	if err != nil {
		return Item{}, err
	}

	return result, nil
}

/**
* unquote
* @param str string
* @return string
**/
func unquote(str string) string {
	result, err := strconv.Unquote(str)
	if err != nil {
		result = str
	}

	return result
}

/**
* quote
* @param str string
* @return string
**/
func quote(str string) string {
	result := strconv.Quote(str)
	result = strs.Replace(result, `'`, ``)

	return result
}

/**
* Unquote
* @param val interface{}
* @return any
**/
func Unquote(val interface{}) any {
	switch v := val.(type) {
	case string:
		return strs.Format(`'%s'`, unquote(v))
	case int:
		return v
	case float64:
		return v
	case float32:
		return v
	case int16:
		return v
	case int32:
		return v
	case int64:
		return v
	case bool:
		return v
	case time.Time:
		return strs.Format(`'%s'`, v.Format("2006-01-02 15:04:05"))
	case Json:
		return strs.Format(`%s`, v.ToUnquote())
	case []string:
		var r string
		for i, _v := range v {
			if i == 0 {
				r = strs.Format(`'%s'`, unquote(_v))
			} else {
				r = strs.Format(`%s, '%s'`, r, unquote(_v))
			}
		}
		return strs.Format(`'(%s)'`, r)
	case []Json:
		var r string
		for i, _v := range v {
			if i == 0 {
				r = strs.Format(`%s`, _v.ToUnquote())
			} else {
				r = strs.Format(`%s, %s`, r, _v.ToUnquote())
			}
		}
		return strs.Format(`'[%s]'`, r)
	case []interface{}:
		var r string
		for i, _v := range v {
			q := Unquote(_v)
			if i == 0 {
				r = strs.Format(`%v`, q)
			} else {
				r = strs.Format(`%s, %v`, r, q)
			}
		}
		return strs.Format(`'[%s]'`, r)
	case map[string]interface{}:
		j := Json(v)
		return strs.Format(`%s`, j.ToUnquote())
	case []map[string]interface{}:
		var r string
		for i, _v := range v {
			j := Json(_v)
			if i == 0 {
				r = strs.Format(`%s`, j.ToUnquote())
			} else {
				r = strs.Format(`%s, %s`, r, j.ToUnquote())
			}
		}
		return strs.Format(`'[%s]'`, r)
	case nil:
		return strs.Format(`%s`, "NULL")
	default:
		logs.Errorf("Not quoted type:%v value:%v", reflect.TypeOf(v), v)
		return val
	}
}

/**
* Quote
* @param val interface{}
* @return any
**/
func Quote(val interface{}) any {
	switch v := val.(type) {
	case string:
		return strs.Format(`%s`, quote(v))
	case int:
		return v
	case float64:
		return v
	case float32:
		return v
	case int16:
		return v
	case int32:
		return v
	case int64:
		return v
	case bool:
		return v
	case time.Time:
		return strs.Format(`"%s"`, v.Format("2006-01-02 15:04:05"))
	case Json:
		j := Json(v)
		return strs.Format(`%s`, j.ToQuote())
	case []string:
		var r string
		for i, _v := range v {
			if i == 0 {
				r = strs.Format(`'%s'`, unquote(_v))
			} else {
				r = strs.Format(`%s, '%s'`, r, unquote(_v))
			}
		}
		return strs.Format(`(%s)`, r)
	case []Json:
		var r string
		for i, _v := range v {
			j := Json(_v)
			if i == 0 {
				r = strs.Format(`%s`, j.ToQuote())
			} else {
				r = strs.Format(`%s, %s`, r, j.ToQuote())
			}
		}
		return strs.Format(`[%s]`, r)
	case []interface{}:
		var r string
		for i, _v := range v {
			q := Quote(_v)
			if i == 0 {
				r = strs.Format(`%v`, q)
			} else {
				r = strs.Format(`%s, %v`, r, q)
			}
		}
		return strs.Format(`[%s]`, r)
	case map[string]interface{}:
		j := Json(v)
		return strs.Format(`%s`, j.ToQuote())
	case []map[string]interface{}:
		var r string
		for i, _v := range v {
			j := Json(_v)
			if i == 0 {
				r = strs.Format(`%s`, j.ToQuote())
			} else {
				r = strs.Format(`%s, %s`, r, j.ToQuote())
			}
		}
		return strs.Format(`[%s]`, r)
	case nil:
		return strs.Format(`%s`, "NULL")
	default:
		logs.Errorf("Not double quoted type:%v value:%v", reflect.TypeOf(v), v)
		return val
	}
}
