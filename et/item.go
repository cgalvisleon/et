package et

import (
	"database/sql"
	"encoding/json"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

// Item struct to define a item
type Item struct {
	Ok     bool `json:"ok"`
	Result Json `json:"result"`
}

/**
* FromString
* @param src string
* @return error
**/
func (it *Item) FromString(src string) error {
	err := json.Unmarshal([]byte(src), &it)
	if err != nil {
		return err
	}

	return nil
}

/**
* ScanRows load rows to a item
* @param rows *sql.Rows
* @return error
**/
func (it *Item) ScanRows(rows *sql.Rows) error {
	it.Ok = true
	it.Result = make(Json)
	it.Result.ScanRows(rows)

	return nil
}

/**
* ValAny return any value of the key
* @param _default any
* @param atribs ...string
* @return any
**/
func (it *Item) ValAny(_default any, atribs ...string) any {
	return Val(it.Result, _default, atribs...)
}

/**
* ValStr return string value of the key
* @param _default string
* @param atribs ...string
* @return string
**/
func (it *Item) ValStr(_default string, atribs ...string) string {
	return it.Result.ValStr(_default, atribs...)
}

/**
* ValInt return int value of the key
* @param _default int
* @param atribs ...string
* @return int
**/
func (it *Item) ValInt(_default int, atribs ...string) int {
	return it.Result.ValInt(_default, atribs...)
}

/**
* ValNum return float64 value of the key
* @param _default float64
* @param atribs ...string
* @return float64
**/
func (it *Item) ValNum(_default float64, atribs ...string) float64 {
	return it.Result.ValNum(_default, atribs...)
}

/**
* ValBool return bool value of the key
* @param _default bool
* @param atribs ...string
* @return bool
**/
func (it *Item) ValBool(_default bool, atribs ...string) bool {
	return it.Result.ValBool(_default, atribs...)
}

/**
* ValTime return time.Time value of the key
* @param _default time.Time
* @param atribs ...string
* @return time.Time
**/
func (it *Item) ValTime(_default time.Time, atribs ...string) time.Time {
	return it.Result.ValTime(_default, atribs...)
}

/**
* ValJson return Json value of the key
* @param _default Json
* @param atribs ...string
* @return Json
**/
func (it *Item) ValJson(_default Json, atribs ...string) Json {
	return it.Result.ValJson(_default, atribs...)
}

/**
* Uppcase return the value of the key in uppercase
* @param _default string
* @param atribs ...string
* @return string
**/
func (it *Item) Uppcase(_default string, atribs ...string) string {
	result := Val(it.Result, _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToUpper(v)
	default:
		return strs.Format(`%v`, strings.ToUpper(_default))
	}
}

/**
* Lowcase return the value of the key in lowercase
* @param _default string
* @param atribs ...string
* @return string
**/
func (it *Item) Lowcase(_default string, atribs ...string) string {
	result := Val(it.Result, _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToLower(v)
	default:
		return strs.Format(`%v`, strings.ToLower(_default))
	}
}

/**
* Titlecase return the value of the key in titlecase
* @param _default string
* @param atribs ...string
* @return string
**/
func (it *Item) Titlecase(_default string, atribs ...string) string {
	result := Val(it.Result, _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToTitle(v)
	default:
		return strs.Format(`%v`, strings.ToTitle(_default))
	}
}

/**
* Get a value from a key
* @param key string
* @return interface{}
**/
func (it *Item) Get(key string) interface{} {
	return it.Result.Get(key)
}

/**
* Set a value from a item
* @param key string
* @param val any
* @return bool
**/
func (it *Item) Set(key string, val any) bool {
	return it.Result.Set(key, val)
}

/**
* Del a value from a item
* @param key string
* @return bool
**/
func (it *Item) Del(key string) bool {
	return it.Result.Del(key)
}

/**
* IsDiferent return if the item is diferent
* @param new Json
* @return bool
**/
func (it *Item) IsDiferent(new Json) bool {
	return IsDiferent(it.Result, new)
}

/**
* Any return any value of the key
* @param _default any
* @param atribs ...string
* @return *Any
**/
func (it *Item) Any(_default any, atribs ...string) *Any {
	return it.Result.Any(_default, atribs...)
}

/**
* Str return string value of the key
* @param _default string
* @param atribs ...string
* @return *generic.Str
**/
func (it *Item) IdT() string {
	return it.Result.IdT()
}

/**
* Index return the value of the key
* @param atribs ...string
* @return int
**/
func (it *Item) Index() int {
	return it.Result.Index()
}

/**
* Key return the value of the key
* @param atribs ...string
* @return string
**/
func (it *Item) Key(atribs ...string) string {
	return it.Result.Key(atribs...)
}

/**
* Str return the value of the key
* @param atribs ...string
* @return string
**/
func (it *Item) Str(atribs ...string) string {
	return it.Result.Str(atribs...)
}

/**
* Int return the value of the key
* @param atribs ...string
* @return int
**/
func (it *Item) Int(atribs ...string) int {
	return it.Result.Int(atribs...)
}

/**
* Int64 return the value of the key
* @param atribs ...string
* @return int
**/
func (it *Item) Int64(atribs ...string) int64 {
	return it.Result.Int64(atribs...)
}

/**
* Num return float64 value of the key
* @param atribs ...string
* @return float64
**/
func (it *Item) Num(atribs ...string) float64 {
	return it.Result.Num(atribs...)
}

/**
* Bool return boolean value of the key
* @param atribs ...string
* @return bool
**/
func (it *Item) Bool(atribs ...string) bool {
	return it.Result.Bool(atribs...)
}

/**
* Time return time.Time value of the key
* @param atribs ...string
* @return time.Time
**/
func (it *Item) Time(atribs ...string) time.Time {
	return it.Result.Time(atribs...)
}

/**
* Data return JsonD value of the key
* @param atribs ...string
* @return JsonD
**/
func (it *Item) Data(atribs ...string) JsonD {
	return it.Result.Data(atribs...)
}

/**
* Json return Json value of the key
* @param atribs ...string
* @return Json
**/
func (it *Item) Json(atribs ...string) Json {
	val := Val(it.Result, Json{}, atribs...)

	switch v := val.(type) {
	case Json:
		return Json(v)
	case map[string]interface{}:
		return Json(v)
	default:
		logs.Errorf("Not Item.Json type (%v) value:%v", reflect.TypeOf(v), v)
		return Json{}
	}
}

/**
* Array return []Json value of the key
* @param atribs ...string
* @return []Json
**/
func (it *Item) Array(atrib string) []Json {
	return it.Result.Array(atrib)
}

/**
* ToByte covert to byte values
* @return []byte
**/
func (it *Item) ToByte() []byte {
	return Json{
		"Ok":     it.Ok,
		"Result": it.Result,
	}.ToByte()
}

/**
* ToString convert to string values
* @return string
**/
func (it *Item) ToString() string {
	result := it.ToJson()
	return result.ToString()
}

/**
* ToJson covert to Json values
* @return Json
**/
func (it *Item) ToJson() Json {
	return Json{
		"Ok":     it.Ok,
		"Result": it.Result,
	}
}
