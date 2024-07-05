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

// TpObject and TpArray type
const TpObject = 1
const TpArray = 2

// JsonD struct to define a json data
type JsonD struct {
	Type  int
	Value interface{}
}

// Json type
type Json map[string]interface{}

// JsonToArrayJson convert a map to a json array
func JsonToArrayJson(src map[string]interface{}) ([]Json, error) {
	result := []Json{}
	result = append(result, src)

	return result, nil
}

// Marshal convert a interface to a json
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

// Value convert a json to a driver value
func (s Json) Value() (driver.Value, error) {
	j, err := json.Marshal(s)

	return j, err
}

// Scan convert a driver value to a json
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

// ToScan convert a json to a struct
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

// ToByte convert a json to a byte
func (s Json) ToByte() []byte {
	result, err := json.Marshal(s)
	if err != nil {
		return nil
	}

	return result
}

// ToString convert a json to a string
func (s Json) ToString() string {
	bt, err := json.Marshal(s)
	if err != nil {
		return ""
	}

	result := string(bt)

	return result
}

// ToUnquote convert a json to a unquote string
func (s Json) ToUnquote() string {
	str := s.ToString()
	result := strs.Format(`'%v'`, str)

	return result
}

// ToQuote convert a json to a quote string
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

// ToItem convert a json to a item
func (s Json) ToItem(src interface{}) Item {
	s.Scan(src)
	return Item{
		Ok:     s.Bool("Ok"),
		Result: s.Json("Result"),
	}
}

// Empty return if the json is empty
func (s Json) Emptyt() bool {
	return len(s) == 0
}

// ValAny return the value of the key
func (s Json) ValAny(_default any, atribs ...string) any {
	return Val(s, _default, atribs...)
}

// ValStr return the value of the key
func (s Json) ValStr(_default string, atribs ...string) string {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
	case string:
		return v
	default:
		return strs.Format(`%v`, v)
	}
}

// ValInt return the value of the key
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

// ValNum return the value of the key
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

// ValBool return the value of the key
func (s Json) ValBool(_default bool, atribs ...string) bool {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v == 1
	case string:
		switch v {
		case "true":
			return true
		case "false":
			return false
		default:
			log.Println("ValBool value is not bool, type:", reflect.TypeOf(v), "value:", v)
			return _default
		}
	default:
		log.Println("ValBool value is not bool, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}

// ValTime return the value of the key
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

// ValJson return the value of the key
func (s Json) ValJson(_default Json, atribs ...string) Json {
	val := s.ValAny(_default, atribs...)

	switch v := val.(type) {
	case Json:
		return v
	default:
		log.Println("ValJson value is not json, type:", reflect.TypeOf(v), "value:", v)
		return _default
	}
}

// Any return the value of the key
func (s Json) Any(_default any, atribs ...string) *generic.Any {
	result := Val(s, _default, atribs...)
	return generic.New(result)
}

// Id return the value of the key
func (s Json) Id() string {
	return s.ValStr("-1", "_id")
}

// IdT return the value of the key
func (s Json) IdT() string {
	return s.ValStr("-1", "_idT")
}

// Index return the value of the key
func (s Json) Index() int {
	return s.ValInt(-1, "index")
}

// Key return the value of the key
func (s Json) Key(atribs ...string) string {
	return s.ValStr("-1", atribs...)
}

// Str return the value of the key
func (s Json) Str(atribs ...string) string {
	return s.ValStr("", atribs...)
}

// Int return the value of the key
func (s Json) Int(atribs ...string) int {
	return s.ValInt(0, atribs...)
}

// Num return the value of the key
func (s Json) Num(atribs ...string) float64 {
	return s.ValNum(0.00, atribs...)
}

// Bool return the value of the key
func (s Json) Bool(atribs ...string) bool {
	return s.ValBool(false, atribs...)
}

// Time return the value of the key
func (s Json) Time(atribs ...string) time.Time {
	return s.ValTime(time.Now(), atribs...)
}

// Data return the value of the key
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

// Json return the value of the key
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

// Array return the value of the key
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

// ArrayStr return the value of the key
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

// ArrayAny return the value of the key
func (s Json) ArrayAny(atrib string) []any {
	result := []any{}
	vals := s[atrib]
	switch v := vals.(type) {
	case []interface{}:
		result = append(result, v...)
	default:
		logs.Errorf("json/ArrayAny - Type (%v) value:%v", reflect.TypeOf(v), v)
	}

	return result
}

// Update a json with a new json
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

// IsDiferent compare two json
func (s Json) IsDiferent(new Json) bool {
	return IsDiferent(s, new)
}

// Get return the value of the key
func (s Json) Get(key string) interface{} {
	v, ok := s[key]
	if !ok {
		return nil
	}

	return v
}

// Set a value from a json
func (s Json) Set(key string, val interface{}) bool {
	key = strings.ToLower(key)

	if s[key] != nil {
		s[key] = val
		return true
	}

	s[key] = val
	return false
}

// Del a value from the key
func (s Json) Del(key string) bool {
	if _, ok := s[key]; !ok {
		return false
	}

	delete(s, key)
	return true
}

// ExistKey return if the key exist in the json
func (s Json) ExistKey(key string) bool {
	return s[key] != nil
}

// Clone a json
func (s Json) Clone() Json {
	var result Json = Json{}
	for k, v := range s {
		result[k] = v
	}

	return result
}

// Append s json with a other json
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

// Merge s json with a other json
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

// Cange s json with a other json and return the change and is changed a json
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

func Append(a, b Json) (Json, bool) {
	c, ch := a.Append(b)

	return *c, ch
}

func Merge(a, b Json) (Json, bool) {
	c, ch := a.Merge(b)

	return *c, ch
}

func Chage(a, b Json) (Json, bool) {
	c, ch := a.Chage(b)

	return *c, ch
}
