package et

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

// Items struct to define a items
type Items struct {
	Ok     bool   `json:"ok"`
	Count  int    `json:"count"`
	Result []Json `json:"result"`
}

// Scan a row from a sql query
func (it *Items) Scan(src interface{}) error {
	var ba []byte
	switch v := src.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return logs.Errorf(`json/Scan - Failed to unmarshal JSON value:%s`, src)
	}

	var t []Json
	err := json.Unmarshal(ba, &t)
	if err != nil {
		return err
	}

	*it = Items{
		Ok:     len(t) > 0,
		Count:  len(t),
		Result: t,
	}

	return nil
}

// ValAny return the value of the key
func (it *Items) ValAny(idx int, _default any, atribs ...string) any {
	if it.Result[idx] == nil {
		return _default
	}

	return it.Result[idx].ValAny(_default, atribs...)
}

// ValStr return the value of the key
func (it *Items) ValStr(idx int, _default string, atribs ...string) string {
	if it.Result[idx] == nil {
		return _default
	}

	return it.Result[idx].ValStr(_default, atribs...)
}

// Uppcase return the value of the key in uppercase
func (it *Items) Uppcase(idx int, _default string, atribs ...string) string {
	if it.Result[idx] == nil {
		return _default
	}

	result := Val(it.Result[idx], _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToUpper(v)
	default:
		return strs.Format(`%v`, strings.ToUpper(_default))
	}
}

// Lowcase return the value of the key in lowercase
func (it *Items) Lowcase(idx int, _default string, atribs ...string) string {
	if it.Result[idx] == nil {
		return _default
	}

	result := Val(it.Result[idx], _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToLower(v)
	default:
		return strs.Format(`%v`, strings.ToLower(_default))
	}
}

// Titlecase return the value of the key in titlecase
func (it *Items) Titlecase(idx int, _default string, atribs ...string) string {
	if it.Result[idx] == nil {
		return _default
	}

	result := Val(it.Result[idx], _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToTitle(v)
	default:
		return strs.Format(`%v`, strings.ToTitle(_default))
	}
}

// Get return the value of the key
func (it *Items) Get(idx int, key string) interface{} {
	if it.Result[idx] == nil {
		return nil
	}

	return it.Result[idx].Get(key)
}

// Set a value to the key
func (it *Items) Set(idx int, key string, val interface{}) bool {
	if it.Result[idx] == nil {
		return false
	}

	return it.Result[idx].Set(key, val)
}

// Del a value from the key
func (it *Items) Del(idx int, key string) bool {
	if it.Result[idx] == nil {
		return false
	}

	return it.Result[idx].Del(key)
}

// Id return the value of the key
func (it *Items) Id(idx int) string {
	return it.Result[idx].Id()
}

// IdT return the value of the key
func (it *Items) IdT(idx int) string {
	return it.Result[idx].IdT()
}

// Key return the value of the key
func (it *Items) Key(idx int, atribs ...string) string {
	return it.Result[idx].Key()
}

// Str return the value of the key
func (it *Items) Str(idx int, atribs ...string) string {
	return it.Result[idx].Str()
}

// Int return the value of the key
func (it *Items) Int(idx int, atribs ...string) int {
	return it.Result[idx].Int()
}

// Num return the value of the key
func (it *Items) Num(idx int, atribs ...string) float64 {
	return it.Result[idx].Num()
}

// Bool return the value of the key
func (it *Items) Bool(idx int, atribs ...string) bool {
	return it.Result[idx].Bool()
}

// Json return the value of the key
func (it *Items) Json(idx int, atribs ...string) Json {
	if it.Result[idx] == nil {
		return Json{}
	}

	val := Val(it.Result[idx], Json{}, atribs...)

	switch v := val.(type) {
	case Json:
		return Json(v)
	case map[string]interface{}:
		return Json(v)
	default:
		logs.Errorf("Not Items.Json type (%v) value:%v", reflect.TypeOf(v), v)
		return Json{}
	}
}

// ToString return the value of the key
func (it *Items) ToString() string {
	var result string
	for _, item := range it.Result {
		str := item.ToString()
		result = strs.Append(result, str, ",")
	}

	return strs.Format(`[%s]`, result)
}

// ToJson return the value of the key
func (it *Items) ToJson() Json {
	return Json{
		"Ok":     it.Ok,
		"Count":  it.Count,
		"Result": it.Result,
	}
}

// ToList return the value of the key
func (it *Items) ToList(all, page, rows int) List {
	var start int
	var end int
	count := it.Count

	if count <= 0 {
		start = 0
		end = 0
	} else {
		offset := (page - 1) * rows

		if offset > 0 {
			start = offset + 1
			end = offset + count
		} else {
			start = 1
			end = count
		}
	}

	return List{
		Rows:   rows,
		All:    all,
		Count:  count,
		Page:   page,
		Start:  start,
		End:    end,
		Result: it.Result,
	}
}
