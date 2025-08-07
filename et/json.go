package et

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/logs"
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
* ToMap convert a json to a map
* @return map[string]interface{}
**/
func (s Json) ToMap() map[string]interface{} {
	return s
}

/**
* ToItem convert a json to a Item
* @return Item
**/
func (s Json) ToItem() Item {
	return Item{
		Ok:     s.Bool("ok"),
		Result: s.Json("result"),
	}
}

/**
* ToItems convert a json to a Items
* @return Items
**/
func (s Json) ToItems() Items {
	result := Items{}
	result.Ok = s.Bool("ok")
	result.Result = s.ArrayJson("result")
	result.Count = len(result.Result)

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
* @param defaultVal interface{}, atribs ...string
* @return any
**/
func (s Json) ValAny(defaultVal interface{}, atribs ...string) interface{} {
	return valAny(s, defaultVal, atribs...)
}

/**
* ValStr return string value of the key
* @param defaultVal string
* @param atribs ...string
* @return string
**/
func (s Json) ValStr(defaultVal string, atribs ...string) string {
	val := s.ValAny(defaultVal, atribs...)

	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf(`%v`, v)
	}
}

/**
* ValInt return int value of the key
* @param defaultVal int
* @param atribs ...string
* @return int
**/
func (s Json) ValInt(defaultVal int, atribs ...string) int {
	val := s.ValAny(defaultVal, atribs...)

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
			return defaultVal
		}
		return i
	default:
		return defaultVal
	}
}

func (s Json) ValInt64(defaultVal int64, atribs ...string) int64 {
	val := s.ValAny(defaultVal, atribs...)

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
			return defaultVal
		}
		return i
	default:
		return defaultVal
	}
}

/**
* ValNum return float64 value of the key
* @param defaultVal float64
* @param atribs ...string
* @return float64
**/
func (s Json) ValNum(defaultVal float64, atribs ...string) float64 {
	val := s.ValAny(defaultVal, atribs...)

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
			return defaultVal
		}
		return i
	default:
		return defaultVal
	}
}

/**
* ValBool return bool value of the key
* @param defaultVal bool
* @param atribs ...string
* @return bool
**/
func (s Json) ValBool(defaultVal bool, atribs ...string) bool {
	val := s.ValAny(defaultVal, atribs...)

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
			return defaultVal
		}
	default:
		return defaultVal
	}
}

/**
* ValTime return time value of the key
* @param defaultVal time.Time
* @param atribs ...string
* @return time.Time
**/
func (s Json) ValTime(defaultVal time.Time, atribs ...string) time.Time {
	val := s.ValAny(defaultVal, atribs...)

	switch v := val.(type) {
	case string:
		layout := "2006-01-02T15:04:05.000Z"
		result, err := time.Parse(layout, v)
		if err != nil {
			return defaultVal
		}
		return result
	case time.Time:
		return v
	default:
		return defaultVal
	}
}

/**
* ValJson
* @param defaultVal Json, atribs ...string
* @return Json
**/
func (s Json) ValJson(defaultVal Json, atribs ...string) Json {
	result, err := valJson(s, defaultVal, atribs...)
	if err != nil {
		logs.Debugf(`ValJson: %v error:%v type:%T`, s, err.Error(), s)
		return defaultVal
	}

	return result
}

/**
* Any return any value of the key
* @param defaultVal any
* @param atribs ...string
* @return *Any
**/
func (s Json) Any(defaultVal interface{}, atribs ...string) interface{} {
	return s.ValAny(defaultVal, atribs...)
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
* String return the value of the key
* @param atribs ...string
* @return string
**/
func (s Json) String(atribs ...string) string {
	return s.Str(atribs...)
}

/**
* Decode return the value of the key
* @param atribs ...string
* @return string
**/
func (s Json) Decode(atribs ...string) string {
	result := s.Str(atribs...)
	bt, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		return result
	}

	return string(bt)
}

/**
* Encode return the value of the key
* @param atribs ...string
* @return string
**/
func (s Json) Encode(atribs ...string) string {
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
	value := s.ValAny("", atribs...)
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
* ValArray
* @param defaultVal []interface{}, atribs ...string
* @return []interface{}
**/
func (s Json) ValArray(defaultVal []interface{}, atribs ...string) []interface{} {
	var result []interface{}
	val := s.ValAny(defaultVal, atribs...)

	switch v := val.(type) {
	case []interface{}:
		return v
	case []Json:
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []map[string]interface{}:
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []string:
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []int:
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []int64:
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []float64:
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []float32:
		for _, item := range v {
			result = append(result, item)
		}

		return result
	case []bool:
		for _, item := range v {
			result = append(result, item)
		}

		return result
	default:
		src := fmt.Sprintf(`%v`, v)
		err := json.Unmarshal([]byte(src), &result)
		if err != nil {
			err := fmt.Errorf(`valor: %v error:%v type:%T`, val, err.Error(), val)
			logs.Tracer("ValArray", err)
			return defaultVal
		}

		return result
	}
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
* ArrayJson
* @param atribs ...string
* @return []Json
**/
func (s Json) ArrayJson(atribs ...string) []Json {
	var result = []Json{}
	vals := s.Array(atribs...)
	for _, val := range vals {
		v, ok := val.(Json)
		if ok {
			result = append(result, v)
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
func (s *Json) IsChanged(from Json) bool {
	for key, fromValue := range from {
		if (*s)[key] == nil {
			return true
		}

		if strings.EqualFold(fmt.Sprintf(`%v`, (*s)[key]), fmt.Sprintf(`%v`, fromValue)) {
			return true
		}
	}

	return false
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
