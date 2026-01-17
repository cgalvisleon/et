package et

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
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
	case map[string]interface{}:
		*s = Json(v)
		return nil
	default:
		return fmt.Errorf(msg.MSG_FAILED_TO_UNMARSHAL_JSON_VALUE, src)
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

	byteVal := func(val interface{}) interface{} {
		switch v := val.(type) {
		case []uint8:
			return string(v)
		default:
			logs.Debugf(`ScanRows: []byte Type:%v Value:%v`, reflect.TypeOf(v), v)
			return v
		}
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
			if err != nil {
				result[col] = byteVal(v)
				continue
			}
			result[col] = bt
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
func (s Json) ToByte() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return result, nil
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

	return string(bt)
}

/**
* ToEscapeHTML convert a json to a string without escape html
* @return string
**/
func (s Json) ToEscapeHTML() string {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err := encoder.Encode(s)
	if err != nil {
		panic(err)
	}

	return buf.String()
}

/**
* ToMap convert a json to a map
* @return map[string]interface{}
**/
func (s Json) ToMap() map[string]interface{} {
	return s
}

/**
* IsEmpty return if the json is empty
* @return bool
**/
func (s Json) IsEmpty() bool {
	return len(s) == 0
}

/**
* IsExist return if the json has the key
* @param key string
* @return bool
**/
func (s Json) IsExist(key string) bool {
	_, ok := s[key]
	return ok
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

/**
* ValAny
* @param def interface{}, atribs ...string
* @return any
**/
func (s Json) ValAny(def interface{}, atribs ...string) interface{} {
	if len(atribs) == 0 {
		return def
	}

	src := s.Clone()
	result := def
	array := []Json{}
	n := len(atribs) - 1
	for i, atrib := range atribs {
		idx, err := strconv.Atoi(atrib)
		if err == nil && len(array) > idx {
			result = array[idx]
			array = []Json{}
			if i == n {
				return result
			} else {
				continue
			}
		}

		val, ok := src[atrib]
		if !ok {
			return nil
		}

		result = val
		switch v := val.(type) {
		case Json:
			src = v
		case map[string]interface{}:
			src = v
		case []Json:
			array = v
		case []map[string]interface{}:
			for _, item := range v {
				array = append(array, item)
			}
		}

		if i == n {
			return result
		}
	}

	return result
}

/**
* ValStr return string value of the key
* @param def string
* @param atribs ...string
* @return string
**/
func (s Json) ValStr(def string, atribs ...string) string {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf(`%v`, v)
	}
}

/**
* ValInt return int value of the key
* @param def int
* @param atribs ...string
* @return int
**/
func (s Json) ValInt(def int, atribs ...string) int {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

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
			return def
		}
		return i
	default:
		return def
	}
}

func (s Json) ValInt64(def int64, atribs ...string) int64 {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

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
			return def
		}
		return i
	default:
		return def
	}
}

/**
* ValNum return float64 value of the key
* @param def float64
* @param atribs ...string
* @return float64
**/
func (s Json) ValNum(def float64, atribs ...string) float64 {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

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
			return def
		}
		return i
	default:
		return def
	}
}

/**
* ValBool return bool value of the key
* @param def bool
* @param atribs ...string
* @return bool
**/
func (s Json) ValBool(def bool, atribs ...string) bool {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v == 1
	case string:
		switch strings.ToLower(v) {
		case "true":
			return true
		case "false":
			return false
		default:
			return def
		}
	default:
		return def
	}
}

/**
* ValTime return time value of the key
* @param def time.Time
* @param atribs ...string
* @return time.Time
**/
func (s Json) ValTime(def time.Time, atribs ...string) time.Time {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	switch v := val.(type) {
	case string:
		layout := "2006-01-02T15:04:05.000Z"
		result, err := time.Parse(layout, v)
		if err != nil {
			return def
		}
		return result
	case time.Time:
		return v
	default:
		return def
	}
}

/**
* ValJson
* @param def Json, atribs ...string
* @return Json
**/
func (s Json) ValJson(def Json, atribs ...string) Json {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	switch v := val.(type) {
	case Json:
		return v
	case map[string]interface{}:
		return Json(v)
	case string:
		var result Json
		err := json.Unmarshal([]byte(v), &result)
		if err != nil {
			logs.Error(fmt.Errorf("ValJson:%s", err.Error()))
			return def
		}

		return result
	default:
		src, err := json.Marshal(v)
		if err != nil {
			logs.Error(fmt.Errorf("ValJson:%s", err.Error()))
			return def
		}

		var result Json
		err = json.Unmarshal(src, &result)
		if err != nil {
			logs.Error(fmt.Errorf("ValJson:%s", err.Error()))
			return def
		}

		return result
	}
}

/**
* ValArray
* @param def []interface{}, atribs ...string
* @return []interface{}
**/
func (s Json) ValArray(def []interface{}, atribs ...string) []interface{} {
	val := s.ValAny(def, atribs...)
	if val == nil {
		return def
	}

	switch v := val.(type) {
	case []interface{}:
		return v
	case []Json:
		result := []interface{}{}
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []map[string]interface{}:
		result := []interface{}{}
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []string:
		result := []interface{}{}
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []int:
		result := []interface{}{}
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []int64:
		result := []interface{}{}
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []float32:
		result := []interface{}{}
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []float64:
		result := []interface{}{}
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []bool:
		result := []interface{}{}
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []time.Time:
		result := []interface{}{}
		for _, item := range v {
			result = append(result, item)
		}

		return result
	default:
		result := []interface{}{}
		src := fmt.Sprintf(`%v`, v)
		err := json.Unmarshal([]byte(src), &result)
		if err != nil {
			logs.Error(fmt.Errorf("ValJson:%s", err.Error()))
			return def
		}

		return result
	}
}

/**
* Any return any value of the key
* @param def any
* @param atribs ...string
* @return *Any
**/
func (s Json) Any(def interface{}, atribs ...string) interface{} {
	return s.ValAny(def, atribs...)
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
* String return the value of the key
* @param atribs ...string
* @return string
**/
func (s Json) String(atribs ...string) string {
	return s.Str(atribs...)
}

/**
* FromBase64 decode base64 value to string
* @param atribs ...string
* @return string
**/
func (s Json) FromBase64(atribs ...string) string {
	result := s.Str(atribs...)
	bt, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		return result
	}

	return string(bt)
}

/**
* ToBase64 encode string value to base64
* @param atribs ...string
* @return string
**/
func (s Json) ToBase64(atribs ...string) string {
	result := s.Str(atribs...)
	bt, err := json.Marshal(result)
	if err != nil {
		return result
	}

	return base64.StdEncoding.EncodeToString(bt)
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
	value := s.ValAny([]byte{}, atribs...)
	bytes, err := json.Marshal(value)
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
func (s Json) Array(atrib ...string) []interface{} {
	return s.ValArray([]interface{}{}, atrib...)
}

/**
* ArrayStr
* @return []string
**/
func (s Json) ArrayStr(atribs ...string) []string {
	var result = []string{}
	vals := s.Array(atribs...)
	for _, val := range vals {
		src := fmt.Sprintf(`%v`, val)
		result = append(result, src)
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
	vals := s.Array(atribs...)
	for _, val := range vals {
		v, ok := val.(int)
		if ok {
			result = append(result, v)
		}
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
	vals := s.Array(atribs...)
	for _, val := range vals {
		v, ok := val.(int64)
		if ok {
			result = append(result, v)
		}
	}

	return result
}

/**
* ArrayNumber
* @param atribs ...string
* @return []float64
**/
func (s Json) ArrayNumber(atribs ...string) []float64 {
	var result = []float64{}
	vals := s.Array(atribs...)
	for _, val := range vals {
		v, ok := val.(float64)
		if ok {
			result = append(result, v)
		}
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
	vals := s.Array(atribs...)
	for _, val := range vals {
		switch v := val.(type) {
		case Json:
			result = append(result, v)
		case map[string]interface{}:
			result = append(result, v)
		case string:
			bt, err := json.Marshal([]byte(v))
			if err != nil {
				logs.Error(fmt.Errorf("ArrayJson: %s", err.Error()))
			}

			if err := json.Unmarshal(bt, &result); err != nil {
				logs.Error(fmt.Errorf("ArrayJson: %s", err.Error()))
			}
		default:
			logs.Error(fmt.Errorf("ArrayJson: value: %v type:%T", v, v))
		}
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
* IsChanged: This method return true if the values in s are different to the values in from.
* @param from Json
* @return bool
**/
func (s Json) IsChanged(from Json) bool {
	return !EqualJSON(s, from)
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
			current = next
		} else {
			return false
		}
	}

	lastKey := keys[len(keys)-1]
	if _, exists := current[lastKey]; exists {
		delete(current, lastKey)
		return true
	}

	return false
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
* Select
* @param keys []string
* @return Json
**/
func (s Json) Select(keys []string) Json {
	result := Json{}
	for _, key := range keys {
		val, ok := s[key]
		if ok {
			result[key] = val
		}
	}

	return result
}

/**
* ArrayJsonToString
* @param vals []Json
* @return string
**/
func ArrayJsonToString(vals []Json) string {
	result, err := json.Marshal(vals)
	if err != nil {
		return ""
	}

	return string(result)
}
