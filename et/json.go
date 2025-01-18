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
* IsEmpty return if the json is empty
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

	n := len(atribs)
	for i := 0; i < n; i++ {
		switch v := current.(type) {
		case Json:
			current = v[atribs[i]]
		case map[string]interface{}:
			current = v[atribs[i]]
		default:
			return _default
		}

		if i == n-1 {
			return current
		}
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

	switch v := val.(type) {
	case string:
		return v
	default:
		return strs.Format(`%v`, v)
	}
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
* Byte return the value of the key
* @param atribs ...string
* @return []byte
**/
func (s Json) Byte(atribs ...string) ([]byte, error) {
	data := s.ValAny("", atribs...)
	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return bytes, nil
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
* @return []string
**/
func (s Json) ArrayStr(atribs ...string) []string {
	var result = []string{}
	vals := s.ValArray([]interface{}{}, atribs...)
	for i, val := range vals {
		v, ok := val.(string)
		if !ok {
			return result
		}

		if i == 0 {
			result = []string{}
		}

		result = append(result, v)
	}

	return result
}

/**
* ArrayInt
* @param atribs ...string
* @return []int
**/
func (s Json) ArrayInt(atribs ...string) []int {
	var result = []int{}
	vals := s.ValArray([]interface{}{}, atribs...)
	for i, val := range vals {
		v, ok := val.(int)
		if !ok {
			return result
		}

		if i == 0 {
			result = []int{}
		}

		result = append(result, v)
	}

	return result
}

/**
* ArrayInt64
* @param atribs ...string
* @return []int64
**/
func (s Json) ArrayInt64(atribs ...string) []int64 {
	var result = []int64{}
	vals := s.ValArray([]interface{}{}, atribs...)
	for i, val := range vals {
		v, ok := val.(int64)
		if !ok {
			return result
		}

		if i == 0 {
			result = []int64{}
		}

		result = append(result, v)
	}

	return result
}

/**
* ArrayJson
* @param atribs ...string
* @return []Json
**/
func (s Json) ArrayJson(atribs ...string) []Json {
	var result = []Json{}
	vals := s.ValArray([]interface{}{}, atribs...)
	for i, val := range vals {
		v, err := Object(val)
		if err != nil {
			return result
		}

		if i == 0 {
			result = []Json{}
		}

		result = append(result, v)
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
* Get
* @param key string
* @return interface{}
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
func (s Json) Set(key string, val interface{}) {
	s[key] = val
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
