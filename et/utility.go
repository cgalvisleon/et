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
*
**/
func unquote(str string) string {
	result, err := strconv.Unquote(str)
	if err != nil {
		result = str
	}

	return result
}

func quote(str string) string {
	result := strconv.Quote(str)
	result = strs.Replace(result, `'`, ``)

	logs.Debug("str", str)
	logs.Debug("quote", result)

	return result
}

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
		j := Json(v)
		return strs.Format(`%s`, j.ToUnquote())
	case []Json:
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
			result = strs.Format(`%s,%s`, result, s)
		}
	}

	return strs.Format(`[%s]`, result)
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
	result := m
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
				if strs.Format(`%v`, old) != strs.Format(`%v`, new) {
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
				if strs.Format(`%v`, old) != strs.Format(`%v`, new) {
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
