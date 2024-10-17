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

// TpObject and TpArray type
const TpObject = 1
const TpArray = 2

// JsonD struct to define a json data
type JsonD struct {
	Type  int
	Value interface{}
}

/**
* Json type
**/
type Json map[string]interface{}

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
func Array(src interface{}) ([]Json, error) {
	j, err := json.Marshal(src)
	if err != nil {
		return []Json{}, err
	}

	result := []Json{}
	err = json.Unmarshal(j, &result)
	if err != nil {
		return []Json{}, err
	}

	return result, nil
}

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
* Value return drive value of json
* @return driver.Value, error
**/

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
* ToItem convert a json to a item
* @param src interface{}
* @return Item
**/
func (s Json) ToItem(src interface{}) Item {
	s.Scan(src)
	return Item{
		Ok:     s.Bool("Ok"),
		Result: s.Json("Result"),
	}
}

/**
* Empty return if the json is empty
* @return bool
**/
func (s Json) IsEmpty() bool {
	return len(s) == 0
}

/**
* ValAny return any value of the key
* @param _default any
* @param atribs ...string
* @return any
**/
func (s Json) ValAny(_default any, atribs ...string) any {
	return Val(s, _default, atribs...)
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
		switch v {
		case "true", "True", "TRUE":
			return true
		case "false", "False", "FALSE":
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

	switch v := val.(type) {
	case Json:
		return v
	default:
		return _default
	}
}

/**
* Any return any value of the key
* @param _default any
* @param atribs ...string
* @return *Any
**/
func (s Json) Any(_default any, atribs ...string) *Any {
	result := Val(s, _default, atribs...)
	return NewAny(result)
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
func (s Json) Data(atrib ...string) JsonD {
	val := Val(s, nil, atrib...)
	if val == nil {
		return JsonD{
			Type:  TpObject,
			Value: Json{},
		}
	}

	switch v := val.(type) {
	case Json:
		return JsonD{
			Type:  TpObject,
			Value: v,
		}
	case map[string]interface{}:
		return JsonD{
			Type:  TpObject,
			Value: Json(v),
		}
	case []Json:
		return JsonD{
			Type:  TpArray,
			Value: v,
		}
	case []interface{}:
		return JsonD{
			Type:  TpArray,
			Value: v,
		}
	default:
		logs.Errorf("Json/Json - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
		return JsonD{
			Type:  TpObject,
			Value: Json{},
		}
	}
}

/**
* Json return the value of the key
* @param atrib string
* @return Json
**/
func (s Json) Json(atrib string) Json {
	val := Val(s, nil, atrib)
	if val == nil {
		return Json{}
	}

	switch v := val.(type) {
	case Json:
		return Json(v)
	case map[string]interface{}:
		return Json(v)
	case []interface{}:
		result := Json{
			atrib: v,
		}

		return result
	default:
		logs.Errorf("json/Json - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
		return Json{}
	}
}

/**
* Array return the value of the key
* @param atrib string
* @return []Json
**/
func (s Json) Array(atrib string) []Json {
	val := Val(s, nil, atrib)
	if val == nil {
		return []Json{}
	}

	data, err := json.MarshalIndent(val, "", "  ")
	if err != nil {
		return []Json{}
	}

	var result []Json
	err = json.Unmarshal(data, &result)
	if err != nil {
		return []Json{}
	}

	return result
}

/**
* Update a json with a other json
* @param fromJson Json
* @return error
**/
func (s *Json) Update(fromJson Json) error {
	obj := *s
	var result bool = false
	for k, new := range fromJson {
		v := obj[k]

		if v == nil {
			obj[k] = new
		} else if new != nil {
			if !result && reflect.DeepEqual(v, new) {
				result = true
			}
			obj[k] = new
		}
	}

	*s = obj

	return nil
}

/**
* IsDiferent return if the json is diferent
* @param new Json
* @return bool
**/
func (s Json) IsDiferent(new Json) bool {
	return IsDiferent(s, new)
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
func (s *Json) Set(key string, val interface{}) bool {
	obj := *s
	created := false
	key = strings.ToLower(key)

	if obj[key] != nil {
		obj[key] = val
		created = true
	} else {
		obj[key] = val
	}

	*s = obj

	return created
}

/**
* Del a value in the key
* @param key string
* @return bool
**/
func (s *Json) Del(key string) bool {
	obj := *s
	key = strings.ToLower(key)
	if _, ok := obj[key]; !ok {
		return false
	}

	delete(obj, key)

	*s = obj

	return true
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
	var result Json = Json{}
	for k, v := range s {
		result[k] = v
	}

	return result
}

/**
* Append s json with a other json
* @param obj Json
* @return *Json
* @return bool
**/
func (s Json) Append(obj Json) (*Json, bool) {
	result := s.Clone()
	var change bool
	var ch bool

	changed := func(v bool) {
		if !change {
			change = v
		}
	}

	for k, v := range obj {
		if _, ok := result[k]; !ok {
			result[k] = v
			changed(true)
			continue
		}

		switch v := v.(type) {
		case Json:
			val := result.Json(k)
			result[k], ch = val.Append(v)
			changed(ch)
		case *Json:
			val := result.Json(k)
			result[k], ch = val.Append(*v)
			changed(ch)
		case map[string]interface{}:
			val := result.Json(k)
			result[k], ch = val.Append(Json(v))
			changed(ch)
		}
	}

	return &result, change
}

/**
* Merge s json with a other json
* @param obj Json
* @return *Json
* @return bool
**/
func (s Json) Merge(obj Json) (*Json, bool) {
	result := s.Clone()
	var change bool
	var ch bool

	changed := func(v bool) {
		if !change {
			change = v
		}
	}

	for k, v := range obj {
		if _, ok := result[k]; !ok {
			result[k] = v
			changed(true)
			continue
		}

		switch v := v.(type) {
		case Json:
			val := result.Json(k)
			result[k], ch = val.Merge(v)
			changed(ch)
		case *Json:
			val := result.Json(k)
			result[k], ch = val.Merge(*v)
			changed(ch)
		case map[string]interface{}:
			val := result.Json(k)
			result[k], ch = val.Merge(Json(v))
			changed(ch)
		default:
			ch := !reflect.DeepEqual(result[k], v)
			result[k] = v
			changed(ch)
		}
	}

	return &result, change
}

/**
* Chage s json with a other json
* @param obj Json
* @return *Json
* @return bool
**/
func (s Json) Chage(obj Json) (*Json, bool) {
	var changes *Json = &Json{}
	var change bool

	changed := func(v bool, key string, value interface{}) {
		if !change {
			change = v
		}

		if v {
			changes.Set(key, value)
		}
	}

	for k, v := range obj {
		if _, ok := s[k]; !ok {
			changed(true, k, v)
			continue
		}

		switch v := v.(type) {
		case Json:
			val := s.Json(k)
			pv, ch := val.Chage(v)
			changed(ch, k, *pv)
		case *Json:
			val := s.Json(k)
			pv, ch := val.Chage(*v)
			changed(ch, k, *pv)
		case map[string]interface{}:
			val := s.Json(k)
			pv, ch := val.Chage(Json(v))
			changed(ch, k, *pv)
		default:
			ch := !reflect.DeepEqual(s[k], v)
			s[k] = v
			changed(ch, k, v)
		}
	}

	return changes, change
}

/**
* Append s json with a other json
* @param obj Json
* @return *Json
* @return bool
**/
func Append(a, b Json) (Json, bool) {
	c, ch := a.Append(b)

	return *c, ch
}

/**
* Merge s json with a other json
* @param obj Json
* @return *Json
* @return bool
**/
func Merge(a, b Json) (Json, bool) {
	c, ch := a.Merge(b)

	return *c, ch
}

/**
* Chage s json with a other json
* @param obj Json
* @return *Json
* @return bool
**/
func Chage(a, b Json) (Json, bool) {
	c, ch := a.Chage(b)

	return *c, ch
}
