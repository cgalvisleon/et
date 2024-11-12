package et

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
)

/**
* Json type
**/
type Json map[string]interface{}

/**
* Scan load rows to a json
* @param src interface{}
* @return error
**/
func (s *Json) Scan(src interface{}) error {
	var ba []byte
	switch v := src.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return logs.Errorf(`Json/Scan - Failed to unmarshal JSON value:%s`, src)
	}

	t := map[string]interface{}{}
	err := json.Unmarshal(ba, &t)
	if err != nil {
		return err
	}

	*s = Json(t)

	return nil
}

/**
* ScanRows load rows to a json
* @param rows *sql.Rows
* @return error
**/
func (s *Json) ScanRows(rows *sql.Rows) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(cols))
	pointers := make([]interface{}, len(cols))
	for i := range values {
		pointers[i] = &values[i]
	}

	if err := rows.Scan(pointers...); err != nil {
		return err
	}

	result := make(Json)
	for i, col := range cols {
		src := values[i]
		switch v := src.(type) {
		case nil:
			result[col] = nil
		case []byte:
			var bt interface{}
			err = json.Unmarshal(v, &bt)
			if err == nil {
				result[col] = bt
				continue
			}
			result[col] = src
			logs.Debugf(`[]byte Col:%s Type:%v Value:%v`, col, reflect.TypeOf(v), v)
		default:
			result[col] = src
		}
	}

	*s = result

	return nil
}

/**
* ToByte convert a json to a []byte
* @return []byte
**/
func (s Json) ToByte() []byte {
	result, err := json.Marshal(s)
	if err != nil {
		return nil
	}

	return result
}

/**
* ToString convert a json to a string
* @return string
**/
func (s Json) ToString() string {
	bt, err := json.Marshal(s)
	if err != nil {
		return ""
	}

	result := string(bt)

	return result
}

/**
* ToUnquote convert a json to a unquote string
* @return string
**/
func (s Json) ToUnquote() string {
	str := s.ToString()
	result := strs.Format(`'%v'`, str)

	return result
}

/**
* ToQuote convert a json to a quote string
* @return string
**/
func (s Json) ToQuote() string {
	for k, v := range s {
		if str, ok := s["mensaje"].(string); ok {
			ustr, err := strconv.Unquote(`"` + str + `"`)
			if err != nil {
				s[k] = v
			} else {
				s[k] = ustr
			}
		} else {
			s[k] = v
		}
	}
	str := s.ToString()

	return str
}

/**
* Empty return if the json is empty
* @return bool
**/
func (s Json) IsEmpty() bool {
	return len(s) == 0
}

/**
* ValAny
* @param _default interface{}
* @param atribs ...string
* @return any
**/
func (s Json) ValAny(_default interface{}, atribs ...string) interface{} {
	var current interface{} = s

	for _, atrib := range atribs {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[atrib]
		} else {
			return _default
		}
	}

	if current == nil {
		return _default
	}

	return current
}

/**
* ValStr return string value of the key
* @param _default string
* @param atribs ...string
* @return string
**/
func (s Json) ValStr(_default string, atribs ...string) string {
	val := s.ValAny(_default, atribs...)

	result, ok := val.(string)
	if !ok {
		return _default
	}

	return result
}

/**
* ValInt return int value of the key
* @param _default int
* @param atribs ...string
* @return int
**/
func (s Json) ValInt(_default int, atribs ...string) int {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
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
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return _default
		}
		return i
	default:
		return _default
	}
}

func (s Json) ValInt64(_default int64, atribs ...string) int64 {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case string:
		i, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return _default
		}
		return i
	default:
		return _default
	}
}

/**
* ValNum return float64 value of the key
* @param _default float64
* @param atribs ...string
* @return float64
**/
func (s Json) ValNum(_default float64, atribs ...string) float64 {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
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
	case string:
		i, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return _default
		}
		return i
	default:
		return _default
	}
}

/**
* ValBool return bool value of the key
* @param _default bool
* @param atribs ...string
* @return bool
**/
func (s Json) ValBool(_default bool, atribs ...string) bool {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v == 1
	case string:
		switch strings.ToUpper(v) {
		case "TRUE":
			return true
		case "FALSE":
			return false
		default:
			return _default
		}
	default:
		return _default
	}
}

/**
* ValTime return time value of the key
* @param _default time.Time
* @param atribs ...string
* @return time.Time
**/
func (s Json) ValTime(_default time.Time, atribs ...string) time.Time {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
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
		return _default
	}
}

/**
* ValJson return Json value of the key
* @param _default Json
* @param atribs ...string
* @return Json
**/
func (s Json) ValJson(_default Json, atribs ...string) Json {
	val := s.ValAny(_default, atribs...)

	result, err := Object(val)
	if err != nil {
		return _default
	}

	return result
}

/**
* ValJson return Json value of the key
* @param _default []interface{}
* @param atribs ...string
* @return []interface{}
**/
func (s Json) ValArray(_default []interface{}, atribs ...string) []interface{} {
	val := s.ValAny(_default, atribs...)

	result, err := Array(val)
	if err != nil {
		return _default
	}

	return result
}

/**
* Any return any value of the key
* @param _default any
* @param atribs ...string
* @return *Any
**/
func (s Json) Any(_default interface{}, atribs ...string) interface{} {
	return s.ValAny(_default, atribs...)
}

/**
* Id return the value of the key
* @return string
**/
func (s Json) Id() string {
	return s.ValStr("-1", "_id")
}

/**
* IdT return the value of the key
* @return string
**/
func (s Json) IdT() string {
	return s.ValStr("-1", "_idT")
}

/**
* Index return the value of the key
* @return int
**/
func (s Json) Index() int {
	return s.ValInt(-1, "index")
}

/**
* Key return the value of the key
* @param atribs ...string
* @return string
**/
func (s Json) Key(atribs ...string) string {
	return s.ValStr("-1", atribs...)
}

/**
* Str return the value of the key
* @param atribs ...string
* @return string
**/
func (s Json) Str(atribs ...string) string {
	return s.ValStr("", atribs...)
}

/**
* Int return the value of the key
* @param atribs ...string
* @return int
**/
func (s Json) Int(atribs ...string) int {
	return s.ValInt(0, atribs...)
}

/**
* Int64 return the value of the key
* @param atribs ...string
* @return int64
**/
func (s Json) Int64(atribs ...string) int64 {
	return s.ValInt64(0, atribs...)
}

/**
* Num return the value of the key
* @param atribs ...string
* @return float64
**/
func (s Json) Num(atribs ...string) float64 {
	return s.ValNum(0.00, atribs...)
}

/**
* Bool return the value of the key
* @param atribs ...string
* @return bool
**/
func (s Json) Bool(atribs ...string) bool {
	return s.ValBool(false, atribs...)
}

/**
* Time return the value of the key
* @param atribs ...string
* @return time.Time
**/
func (s Json) Time(atribs ...string) time.Time {
	return s.ValTime(timezone.NowTime(), atribs...)
}

/**
* Json return the value of the key
* @param atrib string
* @return Json
**/
func (s Json) Json(atrib string) Json {
	return s.ValJson(Json{}, atrib)
}

/**
* Array return the value of the key
* @param atrib string
* @return []Json
**/
func (s Json) Array(atrib string) []interface{} {
	return s.ValArray([]interface{}{}, atrib)
}

/**
* ArrayStr
* @param _default []string
* @param atribs ...string
* @return []string
**/
func (s Json) ArrayStr(_default []string, atribs ...string) []string {
	var result = _default
	vals := s.ValArray([]interface{}{}, atribs...)

	for i, val := range vals {
		v, ok := val.(string)
		if !ok {
			return _default
		}

		if i == 0 {
			result = []string{}
		}

		result[i] = v
	}

	return result
}

/**
* ArrayInt
* @param _default []int
* @param atribs ...string
* @return []int
**/
func (s Json) ArrayInt(_default []int, atribs ...string) []int {
	var result = _default
	vals := s.ValArray([]interface{}{}, atribs...)

	for i, val := range vals {
		v, ok := val.(int)
		if !ok {
			return _default
		}

		if i == 0 {
			result = []int{}
		}

		result[i] = v
	}

	return result
}

/**
* ArrayInt64
* @param _default []int64
* @param atribs ...string
* @return []int64
**/
func (s Json) ArrayInt64(_default []int64, atribs ...string) []int64 {
	var result = _default
	vals := s.ValArray([]interface{}{}, atribs...)

	for i, val := range vals {
		v, ok := val.(int64)
		if !ok {
			return _default
		}

		if i == 0 {
			result = []int64{}
		}

		result[i] = v
	}

	return result
}

/**
* ArrayJson
* @param _default []Json
* @param atribs ...string
* @return []Json
**/
func (s Json) ArrayJson(_default []Json, atribs ...string) []Json {
	var result = _default
	vals := s.ValArray([]interface{}{}, atribs...)

	for i, val := range vals {
		v, err := Object(val)
		if err != nil {
			return _default
		}

		if i == 0 {
			result = []Json{}
		}

		result[i] = v
	}

	return result
}

/**
* Update: This method update s with values in from. If the key exist in s, the value is replaced with the value in from.
* @param fromJson Json
* @return error
**/
func (s *Json) Update(from Json) {
	for key, value := range from {
		(*s)[key] = value
	}
}

/**
* Compare: This method return a new json with the diferent values between s and from. Also include the keys that not exist in s.
* @param from Json
* @return bool
**/
func (s *Json) Compare(from Json) Json {
	diff := Json{}
	for key, fromValue := range from {
		if sValue, exists := (*s)[key]; !exists || sValue != fromValue {
			diff[key] = fromValue
		}
	}

	return diff
}

/**
* Append: This method append the values in from to s. If the key exist in s, the value is not replaced.
* @param from Json
**/
func (s *Json) Append(from Json) {
	for key, value := range from {
		if _, exists := (*s)[key]; !exists {
			(*s)[key] = value
		}
	}
}

/**
* IsDiferent return if the json is diferent
* @param old Json
* @param new Json
* @return bool
**/
func (s Json) Get(key string) interface{} {
	v, ok := s[key]
	if !ok {
		return nil
	}

	return v
}

/**
* Set a value in the key
* @param key string
* @param val interface{}
* @return bool
**/
func (s *Json) Set(keys []string, val interface{}) {
	if *s == nil {
		*s = make(Json)
	}

	current := *s
	for i, k := range keys {
		if i == len(keys)-1 {
			current[k] = val
		} else {
			if _, exists := current[k]; !exists {
				current[k] = Json{}
			}

			if nextMap, ok := current[k].(Json); ok {
				current = nextMap // Avanzamos al siguiente nivel
			} else {
				return
			}
		}
	}
}

/**
* Delete a value in the key
* @param key string
* @return bool
**/
func (s *Json) Delete(keys []string) bool {
	if len(keys) == 0 {
		return false
	}

	current := *s
	for i := 0; i < len(keys)-1; i++ {
		if next, ok := current[keys[i]].(Json); ok {
			current = next // Nos movemos al siguiente nivel
		} else {
			return false // La clave no existe, no se puede eliminar
		}
	}

	lastKey := keys[len(keys)-1]
	if _, exists := current[lastKey]; exists {
		delete(current, lastKey)
		return true // Se eliminó con éxito
	}

	return false // La clave no existe
}

/**
* ExistKey return if the key exist
* @param key string
* @return bool
**/
func (s Json) ExistKey(key string) bool {
	return s[key] != nil
}

/**
* Clone a json
* @return Json
**/
func (s Json) Clone() Json {
	result := Json{}
	for k, v := range s {
		result[k] = v
	}

	return result
}
