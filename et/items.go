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

/**
* Items methods
* @param src interface{}
* @return error
**/
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

/**
* FromString
* @param src string
* @return error
**/
func (it *Items) FromString(src string) error {
	err := json.Unmarshal([]byte(src), &it)
	if err != nil {
		return err
	}

	return nil
}

/**
* ValAny return the value of the key
* @param idx int
* @param _default any
* @param atribs ...string
* @return any
**/
func (it *Items) ValAny(idx int, _default any, atribs ...string) any {
	if it.Result[idx] == nil {
		return _default
	}

	return it.Result[idx].ValAny(_default, atribs...)
}

/**
* ValStr return the value of the key
* @param idx int
* @param _default string
* @param atribs ...string
* @return string
**/
func (it *Items) ValStr(idx int, _default string, atribs ...string) string {
	if it.Result[idx] == nil {
		return _default
	}

	return it.Result[idx].ValStr(_default, atribs...)
}

/**
* Uppcase return the value of the key in uppercase
* @param idx int
* @param _default string
* @param atribs ...string
* @return string
**/
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

/**
* Lowcase return the value of the key in lowercase
* @param idx int
* @param _default string
* @param atribs ...string
* @return string
**/
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

/**
* Titlecase return the value of the key in titlecase
* @param idx int
* @param _default string
* @param atribs ...string
* @return string
**/
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

/**
* Get a value from the key
* @param idx int
* @param key string
* @return interface{}
**/
func (it *Items) Get(idx int, key string) interface{} {
	if it.Result[idx] == nil {
		return nil
	}

	return it.Result[idx].Get(key)
}

/**
* Set a value from the key
* @param idx int
* @param key string
* @param val interface{}
* @return bool
**/
func (it *Items) Set(idx int, key string, val interface{}) bool {
	if it.Result[idx] == nil {
		return false
	}

	return it.Result[idx].Set(key, val)
}

/**
* Del a value from the key
* @param idx int
* @param key string
* @return bool
**/
func (it *Items) Del(idx int, key string) bool {
	if it.Result[idx] == nil {
		return false
	}

	return it.Result[idx].Del(key)
}

/**
* IdT return the value of the key
* @param idx int
* @return string
**/
func (it *Items) IdT(idx int) string {
	return it.Result[idx].IdT()
}

/**
* Index return the value of the key
* @param idx int
* @atrib ...string
* @return int
**/
func (it *Items) Key(idx int, atribs ...string) string {
	return it.Result[idx].Key()
}

/**
* Str return the value of the key
* @param idx int
* @atrib ...string
* @return string
**/
func (it *Items) Str(idx int, atribs ...string) string {
	return it.Result[idx].Str()
}

/**
* Int return the value of the key
* @param idx int
* @atrib ...string
* @return int
**/
func (it *Items) Int(idx int, atribs ...string) int {
	return it.Result[idx].Int()
}

/**
* Num return the value of the key
* @param idx int
* @atrib ...string
* @return float64
**/
func (it *Items) Num(idx int, atribs ...string) float64 {
	return it.Result[idx].Num()
}

/**
* Bool return the value of the key
* @param idx int
* @atrib ...string
* @return bool
**/
func (it *Items) Bool(idx int, atribs ...string) bool {
	return it.Result[idx].Bool()
}

/**
* Json return the value of the key
* @param idx int
* @atrib ...string
* @return Json
**/
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

/**
* ToByte
* @return []byte
**/
func (it *Items) ToByte() []byte {
	return Json{
		"Ok":     it.Ok,
		"Count":  it.Count,
		"Result": it.Result,
	}.ToByte()
}

/**
* ToString
* @return string
**/
func (it *Items) ToString() string {
	result := it.ToJson()
	return result.ToString()
}

/**
* ToJson return the value type Json
* @return Json
**/
func (it *Items) ToJson() Json {
	return Json{
		"Ok":     it.Ok,
		"Count":  it.Count,
		"Result": it.Result,
	}
}

/**
* ToList return the value type List
* @param all int
* @param page int
* @param rows int
* @return List
**/
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
