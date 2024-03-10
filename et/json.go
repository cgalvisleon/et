package et

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Json map[string]interface{}

// Marshal functions
// Convert a struct to a Json
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

// Convert a Unit8 to a Json
func ToUnit8Json(src interface{}) Json {
	result, err := ToJson(src)
	if err != nil {
		return nil
	}

	return result
}

// Convert a struct to a Json
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
		return nil, Errorf(`Failed convert value: %v type: %v`, src, reflect.TypeOf(v))
	}

	t := map[string]interface{}{}
	err := json.Unmarshal(ba, &t)
	if err != nil {
		return nil, err
	}

	return Json(t), nil
}

// Convert a struct to a array Json
func ToJsonArray(vals []interface{}) ([]Json, error) {
	result := []Json{}
	for _, val := range vals {
		v, err := ToJson(val)
		if err != nil {
			return nil, err
		}

		result = append(result, v)
	}

	return result, nil
}

// Convert map[string]interface{} to a array Json
func ToArrayJson(src map[string]interface{}) ([]Json, error) {
	result := []Json{}
	result = append(result, src)

	return result, nil
}

// Convert a struct to a string
func ToString(val interface{}) string {
	s, err := json.Marshal(val)
	if err != nil {
		return ""
	}

	return string(s)
}

// Convert byte to a Json
func ByteToJson(scr interface{}) Json {
	var result Json
	result.Scan(scr)

	return result
}

// Convert array to string
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

// Extract a value from a levels value Json
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
				Errorf("Val. Type (%v) value:%v", reflect.TypeOf(v), v)
				return _default
			}
		}
		if result == nil {
			return _default
		}
	}

	return result
}

// Append a Json to another Json
func ApendJson(a Json, b Json) Json {
	for k, v := range b {
		a[k] = v
	}

	return a
}

// Compare a against b and set that is different if and only if a is different from b
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
					Error(err)
					return false
				}
				return IsDiferent(v, _new)
			case map[string]interface{}:
				_old, err := ToJson(old)
				if err != nil {
					Error(err)
					return false
				}
				_new, err := ToJson(new)
				if err != nil {
					Error(err)
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

// Compare a against b and get is change
func IsChange(a, b Json) bool {
	for k, new := range b {
		old := a[k]

		if old != nil {
			switch v := old.(type) {
			case Json:
				_new, err := ToJson(new)
				if err != nil {
					Error(err)
					return false
				}
				return IsChange(v, _new)
			case map[string]interface{}:
				_old, err := ToJson(old)
				if err != nil {
					Error(err)
					return false
				}
				_new, err := ToJson(new)
				if err != nil {
					Error(err)
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

// Uodate a against b and get the result
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
					Error(err)
				}
				_new, ch := Update(_old, v)
				if !change {
					change = ch
				}
				result[k] = _new
			case map[string]interface{}:
				_old, err := ToJson(old)
				if err != nil {
					Error(err)
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

// Merge a against b and get the result
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
					Error(err)
				}
				_new, ch := Merge(_old, v)
				if !change {
					change = ch
				}
				result[k] = _new
			case map[string]interface{}:
				_old, err := ToJson(old)
				if err != nil {
					Error(err)
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

// OkOrNotJson
func OkOrNotJson(condition bool, ok Json, not Json) Json {
	if condition {
		return ok
	} else {
		return not
	}
}

// Json functions
// Get a value driver.Value from a Json
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
		return Errorf(`Failed to unmarshal JSON value:%s`, src)
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
			Errorf("json/ToScan - No such field:%s in struct", k)
			continue
		}
		if !field.CanSet() {
			Errorf("json/ToScan - Cannot set field:%s in struct", k)
			continue
		}
		valType := reflect.ValueOf(val)
		if field.Type() != valType.Type() {
			return Errorf("json/ToScan - Provided value type didn't match obj field:%s type", k)
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
	return ToString(jb)
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
			fmt.Println("ValInt value int not conver", reflect.TypeOf(v), v)
			return _default
		}
		return i
	default:
		fmt.Println("ValInt value is not int, type:", reflect.TypeOf(v), "value:", v)
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
			fmt.Println("ValNum value float not conver", reflect.TypeOf(v), v)
			return _default
		}
		return i
	default:
		fmt.Println("ValNum value is not float, type:", reflect.TypeOf(v), "value:", v)
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
			fmt.Println("ValBool value is not bool, type:", reflect.TypeOf(v), "value:", v)
			return _default
		}
	default:
		fmt.Println("ValBool value is not bool, type:", reflect.TypeOf(v), "value:", v)
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
		fmt.Println("ValTime value is not time, type:", reflect.TypeOf(v), "value:", v)
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
		Errorf("json/Json - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
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
			Errorf("json/Array - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
			return []Json{}
		}

		return result
	case string:
		if v != "[]" {
			Errorf("json/Array - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
		}
		return []Json{}
	default:
		Errorf("json/Array - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
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
		Errorf("json/ArrayStr - Atrib:%s Type:%v Value:%v", atrib, reflect.TypeOf(v), v)
	}

	return result
}

func (jb Json) ArrayAny(atrib string) []any {
	result := []any{}
	vals := jb[atrib]
	switch v := vals.(type) {
	case []interface{}:
		result = append(result, v...)
	default:
		Errorf("json/ArrayAny - Type (%v) value:%v", reflect.TypeOf(v), v)
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
