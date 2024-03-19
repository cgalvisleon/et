package et

import (
	"database/sql/driver"
	"encoding/json"
	"log"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cgalvisleon/et/generic"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

const TpObject = 1
const TpArray = 2

type JsonD struct {
	Type  int
	Value interface{}
}

type Json map[string]interface{}

func JsonToArrayJson(src map[string]interface{}) ([]Json, error) {
	result := []Json{}
	result = append(result, src)

	return result, nil
}

func Marshal(src interface{}) (Json, error) {
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

func (s Json) Value() (driver.Value, error) {
	j, err := json.Marshal(s)

	return j, err
}

func (s *Json) Scan(src interface{}) error {
	var ba []byte
	switch v := src.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return logs.Errorf(`json/Scan - Failed to unmarshal JSON value:%s`, src)
	}

	t := map[string]interface{}{}
	err := json.Unmarshal(ba, &t)
	if err != nil {
		return err
	}

	*s = Json(t)

	return nil
}

func (s *Json) ToScan(src interface{}) error {
	v := reflect.ValueOf(src).Elem()

	for k, val := range *s {
		field := v.FieldByName(k)
		if !field.IsValid() {
			logs.Errorf("json/ToScan - No such field:%s in struct", k)
			continue
		}
		if !field.CanSet() {
			logs.Errorf("json/ToScan - Cannot set field:%s in struct", k)
			continue
		}
		valType := reflect.ValueOf(val)
		if field.Type() != valType.Type() {
			return logs.Errorf("json/ToScan - Provided value type didn't match obj field:%s type", k)
		}
		field.Set(valType)
	}

	return nil
}

func (s Json) ToByte() []byte {
	result, err := json.Marshal(s)
	if err != nil {
		return nil
	}

	return result
}

func (s Json) ToString() string {
	bt, err := json.Marshal(s)
	if err != nil {
		return ""
	}

	result := string(bt)

	return result
}

func (s Json) ToUnquote() string {
	str := s.ToString()
	result := strs.Format(`'%v'`, str)

	return result
}

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
	// result := quote(str)

	logs.Debugf("str: %s", str)
	// logs.Debugf("quote: %s", result)

	return str
}

func (s Json) ToItem(src interface{}) Item {
	s.Scan(src)
	return Item{
		Ok:     s.Bool("Ok"),
		Result: s.Json("Result"),
	}
}

func (s Json) ValAny(_default any, atribs ...string) any {
	return Val(s, _default, atribs...)
}

func (s Json) ValStr(_default string, atribs ...string) string {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
	case string:
		return v
	default:
		return strs.Format(`%v`, v)
	}
}

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
			log.Println("ValInt value int not conver", reflect.TypeOf(v), v)
			return _default
		}
		return i
	default:
		log.Println("ValInt value is not int, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}

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
			log.Println("ValNum value float not conver", reflect.TypeOf(v), v)
			return _default
		}
		return i
	default:
		log.Println("ValNum value is not float, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}

func (s Json) ValBool(_default bool, atribs ...string) bool {
	val := s.ValAny(_default, atribs...)

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
		log.Println("ValTime value is not time, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}

func (s Json) ValJson(_default Json, atribs ...string) Json {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
	case Json:
		return v
	default:
		log.Println("ValTime value is not json, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}
func (s Json) Any(_default any, atribs ...string) *generic.Any {
	result := Val(s, _default, atribs...)
	return generic.New(result)
}

func (s Json) Id() string {
	return s.ValStr("-1", "_id")
}

func (s Json) IdT() string {
	return s.ValStr("-1", "_idT")
}

func (s Json) Index() int {
	return s.ValInt(-1, "index")
}

func (s Json) Key(atribs ...string) string {
	return s.ValStr("-1", atribs...)
}

func (s Json) Str(atribs ...string) string {
	return s.ValStr("", atribs...)
}

func (s Json) Int(atribs ...string) int {
	return s.ValInt(0, atribs...)
}

func (s Json) Num(atribs ...string) float64 {
	return s.ValNum(0.00, atribs...)
}

func (s Json) Bool(atribs ...string) bool {
	return s.ValBool(false, atribs...)
}

func (s Json) Time(atribs ...string) time.Time {
	return s.ValTime(time.Now(), atribs...)
}

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
		logs.Errorf("json/Json - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
		return JsonD{
			Type:  TpObject,
			Value: Json{},
		}
	}
}

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

func (s Json) Array(atrib string) []Json {
	val := Val(s, nil, atrib)
	if val == nil {
		return []Json{}
	}

	switch v := val.(type) {
	case []Json:
		return v
	case []interface{}:
		result, err := ToJsonArray(v)
		if err != nil {
			logs.Errorf("json/Array - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
			return []Json{}
		}

		return result
	case map[string]interface{}:
		result, err := JsonToArrayJson(v)
		if err != nil {
			return []Json{}
		}

		return result
	case string:
		if v != "[]" {
			logs.Errorf("json/Array - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
		}
		return []Json{}
	default:
		logs.Errorf("json/Array - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
		return []Json{}
	}
}

func (s Json) ArrayStr(atrib string) []string {
	result := []string{}
	vals := s[atrib]
	switch v := vals.(type) {
	case []interface{}:
		for _, val := range v {
			result = append(result, val.(string))
		}
	default:
		logs.Errorf("json/ArrayStr - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
	}

	return result
}

func (s Json) ArrayAny(atrib string) []any {
	result := []any{}
	vals := s[atrib]
	switch v := vals.(type) {
	case []interface{}:
		for _, val := range v {
			result = append(result, val)
		}
	default:
		logs.Errorf("json/ArrayAny - Type (%v) value:%v", reflect.TypeOf(v), v)
	}

	return result
}

func (s Json) Update(fromJson Json) error {
	var result bool = false
	for k, new := range fromJson {
		v := s[k]

		if v == nil {
			s[k] = new
		} else if new != nil {
			if !result && reflect.DeepEqual(v, new) {
				result = true
			}
			s[k] = new
		}
	}

	return nil
}

func (s Json) IsDiferent(new Json) bool {
	return IsDiferent(s, new)
}

func (s Json) IsChange(new Json) bool {
	return IsChange(s, new)
}

/**
*
**/
func (s Json) Get(key string) interface{} {
	v, ok := s[key]
	if !ok {
		return nil
	}

	return v
}

func (s Json) Set(key string, val interface{}) bool {
	key = strings.ToLower(key)

	if s[key] != nil {
		s[key] = val
		return true
	}

	s[key] = val
	return false
}

func (s *Json) Append(obj Json) *Json {
	var result Json = *s
	for k, v := range obj {
		if _, ok := result[k]; !ok {
			result[k] = v
		}
	}

	return &result
}

func (s Json) Del(key string) bool {
	if _, ok := s[key]; !ok {
		return false
	}

	delete(s, key)
	return true
}

func (s Json) ExistKey(key string) bool {
	return s[key] != nil
}

func (s Json) Consolidate(toField string, ruleOut ...string) Json {
	FindIndex := func(arr []string, valor string) int {
		for i, v := range arr {
			if v == valor {
				return i
			}
		}
		return -1
	}

	result := s
	if s.ExistKey(toField) {
		result = s.Json(toField)
	}

	for k, v := range s {
		if k != toField {
			idx := FindIndex(ruleOut, k)
			if idx == -1 {
				result[k] = v
			}
		}
	}

	return result
}

func (s Json) ConsolidateAndUpdate(toField string, ruleOut []string, new Json) (Json, error) {
	result := s.Consolidate(toField, ruleOut...)
	err := result.Update(new)
	if err != nil {
		return Json{}, nil
	}

	return result, nil
}
