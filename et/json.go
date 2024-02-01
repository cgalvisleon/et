package et

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/console"
)

type Json map[string]interface{}

func (jb Json) Value() (driver.Value, error) {
	j, err := json.Marshal(jb)

	return j, err
}

func (jb *Json) Scan(src interface{}) error {
	var ba []byte
	switch v := src.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return console.Errorf(`Failed to unmarshal JSON value:%s`, src)
	}

	t := map[string]interface{}{}
	err := json.Unmarshal(ba, &t)
	if err != nil {
		return err
	}

	*jb = Json(t)

	return nil
}

func (jb *Json) ToScan(src interface{}) error {
	v := reflect.ValueOf(src).Elem()

	for k, val := range *jb {
		field := v.FieldByName(k)
		if !field.IsValid() {
			console.Errorf("json/ToScan - No such field:%s in struct", k)
			continue
		}
		if !field.CanSet() {
			console.Errorf("json/ToScan - Cannot set field:%s in struct", k)
			continue
		}
		valType := reflect.ValueOf(val)
		if field.Type() != valType.Type() {
			return console.Errorf("json/ToScan - Provided value type didn't match obj field:%s type", k)
		}
		field.Set(valType)
	}

	return nil
}

func (jb Json) ToByte() []byte {
	result, err := json.Marshal(jb)
	if err != nil {
		return nil
	}

	return result
}

func (jb Json) ToString() string {
	s, err := json.Marshal(jb)
	if err != nil {
		return ""
	}

	return string(s)
}

func (jb Json) ToQuoted() string {
	s, err := json.Marshal(jb)
	if err != nil {
		return ""
	}

	return string(s)
}

func (jb Json) ToItem(src interface{}) Item {
	jb.Scan(src)
	return Item{
		Ok:     jb.Bool("Ok"),
		Result: jb.Json("Result"),
	}
}

func (jb Json) ValAny(_default any, atribs ...string) any {
	return Val(jb, _default, atribs...)
}

func (jb Json) ValStr(_default string, atribs ...string) string {
	val := jb.ValAny(_default, atribs...)

	switch v := val.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf(`%v`, v)
	}
}

func (jb Json) ValInt(_default int, atribs ...string) int {
	val := jb.ValAny(_default, atribs...)

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
			log.Println("ValInt value int not conver", reflect.TypeOf(v), v)
			return _default
		}
		return i
	default:
		log.Println("ValInt value is not int, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}

func (jb Json) ValNum(_default float64, atribs ...string) float64 {
	val := jb.ValAny(_default, atribs...)

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
			log.Println("ValNum value float not conver", reflect.TypeOf(v), v)
			return _default
		}
		return i
	default:
		log.Println("ValNum value is not float, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}

func (jb Json) ValBool(_default bool, atribs ...string) bool {
	val := jb.ValAny(_default, atribs...)

	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v == 1
	case string:
		if v == "true" {
			return true
		} else if v == "false" {
			return false
		} else {
			log.Println("ValBool value is not bool, type:", reflect.TypeOf(v), "value:", v)
			return _default
		}
	default:
		log.Println("ValBool value is not bool, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}

func (jb Json) ValTime(atribs ...string) time.Time {
	_default := time.Now()
	val := jb.ValAny(_default, atribs...)

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
		log.Println("ValTime value is not time, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}

func (jb Json) Any(_default any, atribs ...string) *Any {
	result := Val(jb, _default, atribs...)
	return New(result)
}

func (jb Json) Id() string {
	return jb.ValStr("", "_id")
}

func (jb Json) IdT() string {
	return jb.ValStr("", "_idT")
}

func (jb Json) Index() int {
	return jb.ValInt(-1, "index")
}

func (jb Json) Key(atribs ...string) string {
	return jb.ValStr("", atribs...)
}

func (jb Json) Str(atribs ...string) string {
	return jb.ValStr("", atribs...)
}

func (jb Json) Int(atribs ...string) int {
	return jb.ValInt(0, atribs...)
}

func (jb Json) Num(atribs ...string) float64 {
	return jb.ValNum(0.00, atribs...)
}

func (jb Json) Bool(atribs ...string) bool {
	return jb.ValBool(false, atribs...)
}

func (jb Json) Time(atribs ...string) time.Time {
	return jb.ValTime(atribs...)
}

func (jb Json) Json(atrib string) Json {
	val := Val(jb, nil, atrib)
	if val == nil {
		return Json{}
	}

	switch v := val.(type) {
	case Json:
		return Json(v)
	case map[string]interface{}:
		return Json(v)
	default:
		console.Errorf("json/Json - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
		return Json{}
	}
}

func (jb Json) Array(atrib string) []Json {
	val := Val(jb, nil, atrib)
	if val == nil {
		return []Json{}
	}

	switch v := val.(type) {
	case []Json:
		return v
	case []interface{}:
		result, err := ToJsonArray(v)
		if err != nil {
			console.Errorf("json/Array - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
			return []Json{}
		}

		return result
	case string:
		if v != "[]" {
			console.Errorf("json/Array - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
		}
		return []Json{}
	default:
		console.Errorf("json/Array - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
		return []Json{}
	}
}

func (jb Json) ArrayStr(atrib string) []string {
	result := []string{}
	vals := jb[atrib]
	switch v := vals.(type) {
	case []interface{}:
		for _, val := range v {
			result = append(result, val.(string))
		}
	default:
		console.Errorf("json/ArrayStr - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
	}

	return result
}

func (jb Json) ArrayAny(atrib string) []any {
	result := []any{}
	vals := jb[atrib]
	switch v := vals.(type) {
	case []interface{}:
		for _, val := range v {
			result = append(result, val)
		}
	default:
		console.Errorf("json/ArrayAny - Type (%v) value:%v", reflect.TypeOf(v), v)
	}

	return result
}

func (jb Json) Update(fromJson Json) error {
	var result bool = false
	for k, new := range fromJson {
		v := jb[k]

		if v == nil {
			jb[k] = new
		} else if new != nil {
			if !result && reflect.DeepEqual(v, new) {
				result = true
			}
			jb[k] = new
		}
	}

	return nil
}

func (jb Json) Apend(n Json) error {
	for k, v := range n {
		jb[k] = v
	}

	return nil
}

func (jb Json) IsDiferent(new Json) bool {
	return IsDiferent(jb, new)
}

func (jb Json) IsChange(new Json) bool {
	return IsChange(jb, new)
}

/**
*
**/
func (jb Json) Get(key string) interface{} {
	v, ok := jb[key]
	if !ok {
		return nil
	}

	return v
}

func (jb Json) Set(key string, val interface{}) bool {
	key = strings.ToLower(key)

	if jb[key] != nil {
		jb[key] = val
		return true
	}

	jb[key] = val
	return false
}

func (jb Json) Del(key string) bool {
	if _, ok := jb[key]; !ok {
		return false
	}

	delete(jb, key)
	return true
}

func (jb Json) ExistKey(key string) bool {
	return jb[key] != nil
}

func (jb Json) Consolidate(toField string, ruleOut ...string) Json {
	FindIndex := func(arr []string, valor string) int {
		for i, v := range arr {
			if v == valor {
				return i
			}
		}
		return -1
	}

	result := jb
	if jb.ExistKey(toField) {
		result = jb.Json(toField)
	}

	for k, v := range jb {
		if k != toField {
			idx := FindIndex(ruleOut, k)
			if idx == -1 {
				result[k] = v
			}
		}
	}

	return result
}

func (jb Json) ConsolidateAndUpdate(toField string, ruleOut []string, new Json) (Json, error) {
	result := jb.Consolidate(toField, ruleOut...)
	err := result.Update(new)
	if err != nil {
		return Json{}, nil
	}

	return result, nil
}

func (jb Json) FindIndex(list []Json, key string) int {
	result := -1
	for i, element := range list {
		if jb[key] == element[key] {
			return i
		}
	}

	return result
}

func (jb Json) Append(list []Json, key string) []Json {
	idx := jb.FindIndex(list, key)
	if idx != -1 {
		list = append(list, jb)
	}

	return list
}
