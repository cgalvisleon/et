package et

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

// unquote remove the quotes from a string
func unquote(str string) string {
	result, err := strconv.Unquote(str)
	if err != nil {
		result = str
	}

	return result
}

// quote add quotes to a string
func quote(str string) string {
	result := strconv.Quote(str)
	result = strs.Replace(result, `'`, ``)

	return result
}

// Unquote remove the quotes from a value
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

// Quote add quotes to a value
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

// ToJson convert a value to a Json
func ToJson(src interface{}) (Json, error) {
	var ba []byte
	switch v := src.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	case Json:
		return v, nil
	case map[string]interface{}:
		r := Json{}
		for k, v := range v {
			r[k] = v
		}
		return r, nil
	default:
		return nil, logs.Errorf(`ToJson value: %v type: %v`, src, reflect.TypeOf(v))
	}

	t := map[string]interface{}{}
	err := json.Unmarshal(ba, &t)
	if err != nil {
		return nil, err
	}

	return Json(t), nil
}

// ToString convert a value to a string
func ToString(val interface{}) string {
	s, err := json.Marshal(val)
	if err != nil {
		return ""
	}

	return string(s)
}

// ByteToJson convert a byte to a Json
func ByteToJson(scr interface{}) Json {
	var result Json
	result.Scan(scr)

	return result
}

// ArrayToJson convert a []interface{} to a Json
func ArrayToString(vals []Json) string {
	jsonData, err := json.Marshal(vals)
	if err != nil {
		return "[]"
	}

	return string(jsonData)
}

// Val get a value from a Json
func Val(data Json, _default any, atribs ...string) any {
	var result any
	var ok bool

	for i, atrib := range atribs {
		if i == 0 {
			result, ok = data[atrib]
			if !ok {
				return _default
			}
		} else {
			switch v := result.(type) {
			case Json:
				data, err := ToJson(v)
				if err != nil {
					return _default
				}

				result, ok = data[atrib]
				if !ok {
					return _default
				}
			case []interface{}:
				data, err := ToJson(v)
				if err != nil {
					return _default
				}

				result, ok = data[atrib]
				if !ok {
					return _default
				}
			case map[string]interface{}:
				data, err := ToJson(v)
				if err != nil {
					return _default
				}

				result, ok = data[atrib]
				if !ok {
					return _default
				}
			default:
				logs.Errorf("Val. Type (%v) value:%v", reflect.TypeOf(v), v)
				return _default
			}
		}
		if result == nil {
			return _default
		}
	}

	return result
}

// ApendJson add a Json to a Json
func ApendJson(m Json, n Json) Json {
	result := m
	for k, v := range n {
		result[k] = v
	}

	return result
}

// IsDiferent compare two Json and return true if they are different
func IsDiferent(a, b Json) bool {
	for k, new := range b {
		old := a[k]

		if old == nil {
			return true
		} else {
			switch v := old.(type) {
			case Json:
				_new, err := ToJson(new)
				if err != nil {
					return false
				}
				return IsDiferent(v, _new)
			case *Json:
				_new, err := ToJson(new)
				if err != nil {
					return false
				}
				return IsDiferent(*v, _new)
			case map[string]interface{}:
				_old, err := ToJson(old)
				if err != nil {
					return false
				}
				_new, err := ToJson(new)
				if err != nil {
					return false
				}
				return IsDiferent(_old, _new)
			default:
				if Quote(old) != Quote(new) {
					return true
				}
			}
		}
	}

	return false
}

// OkOrNotJson return a Json depending on a condition
func OkOrNotJson(condition bool, ok Json, not Json) Json {
	if condition {
		return ok
	} else {
		return not
	}
}
