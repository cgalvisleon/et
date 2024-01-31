package json

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/cgalvisleon/elvis/logs"
)

/**
*
**/
func Quoted(val interface{}) any {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf(`'%s'`, v)
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
		return fmt.Sprintf(`'%s'`, v.Format("2006-01-02 15:04:05"))
	case Json:
		j := Json(v)
		return fmt.Sprintf(`'%s'`, j.ToString())
	case []Json:
		var r string
		for i, _v := range v {
			j := Json(_v)
			if i == 0 {
				r = fmt.Sprintf(`'%s'`, j.ToString())
			} else {
				r = fmt.Sprintf(`%s, '%s'`, r, j.ToString())
			}
		}
		return fmt.Sprintf(`'[%s]'`, r)
	case []interface{}:
		var r string
		var j Json
		for i, _v := range v {
			bt, err := json.Marshal(_v)
			if err != nil {
				logs.Errorf("Not quoted type:%v value:%v", reflect.TypeOf(v), v)
				return val
			}
			j.Scan(bt)
			if i == 0 {
				r = fmt.Sprintf(`'%s'`, j.ToString())
			} else {
				r = fmt.Sprintf(`%s, '%s'`, r, j.ToString())
			}
		}
		return fmt.Sprintf(`'[%s]'`, r)
	case map[string]interface{}:
		j := Json(v)
		return fmt.Sprintf(`'%s'`, j.ToString())
	case []map[string]interface{}:
		var r string
		for i, _v := range v {
			j := Json(_v)
			if i == 0 {
				r = fmt.Sprintf(`'%s'`, j.ToString())
			} else {
				r = fmt.Sprintf(`%s, '%s'`, r, j.ToString())
			}
		}
		return fmt.Sprintf(`'[%s]'`, r)
	case nil:
		return fmt.Sprintf(`%s`, "NULL")
	default:
		logs.Errorf("Not quoted type:%v value:%v", reflect.TypeOf(v), v)
		return val
	}
}

func DoubleQuoted(val interface{}) any {
	switch v := val.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
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
		return fmt.Sprintf(`"%s"`, v.Format("2006-01-02 15:04:05"))
	case Json:
		j := Json(v)
		return fmt.Sprintf(`%s`, j.ToQuoted())
	case []Json:
		var r string
		for i, _v := range v {
			j := Json(_v)
			if i == 0 {
				r = fmt.Sprintf(`%s`, j.ToQuoted())
			} else {
				r = fmt.Sprintf(`%s, %s`, r, j.ToQuoted())
			}
		}
		return fmt.Sprintf(`[%s]`, r)
	case []interface{}:
		var r string
		var j Json
		for i, _v := range v {
			bt, err := json.Marshal(_v)
			if err != nil {
				logs.Errorf("Not double quoted type:%v value:%v", reflect.TypeOf(v), v)
				return val
			}
			j.Scan(bt)
			if i == 0 {
				r = fmt.Sprintf(`%s`, j.ToQuoted())
			} else {
				r = fmt.Sprintf(`%s, %s`, r, j.ToQuoted())
			}
		}
		return fmt.Sprintf(`[%s]`, r)
	case map[string]interface{}:
		j := Json(v)
		return fmt.Sprintf(`%s`, j.ToQuoted())
	case []map[string]interface{}:
		var r string
		for i, _v := range v {
			j := Json(_v)
			if i == 0 {
				r = fmt.Sprintf(`%s`, j.ToQuoted())
			} else {
				r = fmt.Sprintf(`%s, %s`, r, j.ToQuoted())
			}
		}
		return fmt.Sprintf(`[%s]`, r)
	case nil:
		return fmt.Sprintf(`%s`, "NULL")
	default:
		logs.Errorf("Not double quoted type:%v value:%v", reflect.TypeOf(v), v)
		return val
	}
}

func ToUnit8Json(src interface{}) Json {
	result, err := ToJson(src)
	if err != nil {
		return nil
	}

	return result
}

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
		return nil, logs.Errorf(`Failed ToJson value: %v type: %v`, src, reflect.TypeOf(v))
	}

	t := map[string]interface{}{}
	err := json.Unmarshal(ba, &t)
	if err != nil {
		return nil, err
	}

	return Json(t), nil
}

func ToJsonArray(vals []interface{}) ([]Json, error) {
	var result []Json
	for _, val := range vals {
		v, err := ToJson(val)
		if err != nil {
			return nil, err
		}

		result = append(result, v)
	}

	return result, nil
}

func ToString(val interface{}) string {
	s, err := json.Marshal(val)
	if err != nil {
		return ""
	}

	return string(s)
}

func ByteToJson(scr interface{}) Json {
	var result Json
	result.Scan(scr)

	return result
}

func ArrayToString(vals []Json) string {
	var result string

	for k, val := range vals {
		v, err := ToJson(val)
		if err != nil {
			return "[]"
		}

		s := v.ToString()
		if k == 0 {
			result = s
		} else {
			result = fmt.Sprintf(`%s,%s`, result, s)
		}
	}

	return fmt.Sprintf(`[%s]`, result)
}

/**
*
**/
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

func ApendJson(m Json, n Json) Json {
	var result Json

	result = m
	for k, v := range n {
		result[k] = v
	}

	return result
}

/**
* Compara b contra a y se establece que es diferente si y solo si
* los valor de b no estan en a o los valores de b son diferentes en a
**/
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
					logs.Error(err)
					return false
				}
				return IsDiferent(v, _new)
			case map[string]interface{}:
				_old, err := ToJson(old)
				if err != nil {
					logs.Error(err)
					return false
				}
				_new, err := ToJson(new)
				if err != nil {
					logs.Error(err)
					return false
				}
				return IsDiferent(_old, _new)
			default:
				if fmt.Sprintf(`%v`, old) != fmt.Sprintf(`%v`, new) {
					return true
				}
			}
		}
	}

	return false
}

/**
* Compara b contra a y se establece que si ubo cambio si y solo si
* los valor de b esta en a y alguno es direfernte
**/
func IsChange(a, b Json) bool {
	for k, new := range b {
		old := a[k]

		if old != nil {
			switch v := old.(type) {
			case Json:
				_new, err := ToJson(new)
				if err != nil {
					logs.Error(err)
					return false
				}
				return IsChange(v, _new)
			case map[string]interface{}:
				_old, err := ToJson(old)
				if err != nil {
					logs.Error(err)
					return false
				}
				_new, err := ToJson(new)
				if err != nil {
					logs.Error(err)
					return false
				}
				return IsChange(_old, _new)
			default:
				if fmt.Sprintf(`%v`, old) != fmt.Sprintf(`%v`, new) {
					return true
				}
			}
		}
	}

	return false
}

func Update(a, b Json) (Json, bool) {
	var change bool
	var result Json = Json{}

	for k, val := range a {
		result[k] = val
	}

	for k, new := range b {
		old := result[k]

		if old != nil {
			switch v := new.(type) {
			case Json:
				_old, err := ToJson(old)
				if err != nil {
					logs.Error(err)
				}
				_new, ch := Update(_old, v)
				if !change {
					change = ch
				}
				result[k] = _new
			case map[string]interface{}:
				_old, err := ToJson(old)
				if err != nil {
					logs.Error(err)
				}
				_new, ch := Update(_old, v)
				if !change {
					change = ch
				}
				result[k] = _new
			case []map[string]interface{}:
				result[k] = v
			default:
				if !change {
					change = old != v
				}
				result[k] = v
			}
		}
	}

	return result, change
}

func Merge(a, b Json) (Json, bool) {
	var change bool
	var result Json = Json{}

	for k, val := range a {
		result[k] = val
	}

	for k, new := range b {
		old := a[k]

		if old == nil {
			result[k] = new
		} else if new != nil {
			switch v := new.(type) {
			case Json:
				_old, err := ToJson(old)
				if err != nil {
					logs.Error(err)
				}
				_new, ch := Merge(_old, v)
				if !change {
					change = ch
				}
				result[k] = _new
			case map[string]interface{}:
				_old, err := ToJson(old)
				if err != nil {
					logs.Error(err)
				}
				_new, ch := Merge(_old, v)
				if !change {
					change = ch
				}
				result[k] = _new
			case []map[string]interface{}:
				result[k] = v
			default:
				if !change {
					change = old != v
				}
				result[k] = v
			}
		}
	}

	return result, change
}

func OkOrNotJson(condition bool, ok Json, not Json) Json {
	if condition {
		return ok
	} else {
		return not
	}
}
