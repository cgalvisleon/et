package et

import (
	"encoding/json"
	"time"
)

type Item struct {
	Ok     bool `json:"ok"`
	Result Json `json:"result"`
}

/**
* ToByte convert a json to a []byte
* @return []byte, error
**/
func (s Item) ToByte() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* ToJson convert a json to a Json
* @return Json
**/
func (s Item) ToJson() Json {
	result := Json{
		"ok":     s.Ok,
		"result": s.Result,
	}
	return result
}

/**
* ToString convert a json to a string
* @return string
**/
func (s Item) ToString() string {
	return s.ToJson().ToString()
}

/**
* IsEmpty return true if the json is empty
* @return bool
**/
func (s *Item) IsEmpty() bool {
	return s.Result.IsEmpty()
}

/**
* ToMap convert a json to a map
* @return map[string]interface{}
**/
func (s Item) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	result["ok"] = s.Ok
	result["result"] = s.Result.ToMap()

	return result
}

/**
* ValAny return any value of the key
* @param def interface{}, atribs ...string
* @return interface{}
**/
func (s *Item) ValAny(def interface{}, atribs ...string) interface{} {
	return s.Result.ValAny(def, atribs...)
}

/**
* ValStr return string value of the key
* @param def string, atribs ...string
* @return string
**/
func (s *Item) ValStr(def string, atribs ...string) string {
	return s.Result.ValStr(def, atribs...)
}

/**
* ValInt return int value of the key
* @param def int, atribs ...string
* @return int
**/
func (s *Item) ValInt(def int, atribs ...string) int {
	return s.Result.ValInt(def, atribs...)
}

/**
* ValInt64 return int64 value of the key
* @param def int64, atribs ...string
* @return int64
**/
func (s Item) ValInt64(def int64, atribs ...string) int64 {
	return s.Result.ValInt64(def, atribs...)
}

/**
* ValNum return float64 value of the key
* @param def float64, atribs ...string
* @return float64
**/
func (s *Item) ValNum(def float64, atribs ...string) float64 {
	return s.Result.ValNum(def, atribs...)
}

/**
* ValBool return bool value of the key
* @param def bool, atribs ...string
* @return bool
**/
func (s *Item) ValBool(def bool, atribs ...string) bool {
	return s.Result.ValBool(def, atribs...)
}

/**
* ValTime return time.Time value of the key
* @param def time.Time, atribs ...string
* @return time.Time
**/
func (s *Item) ValTime(def time.Time, atribs ...string) time.Time {
	return s.Result.ValTime(def, atribs...)
}

/**
* ValJson return Json value of the key
* @param def Json, atribs ...string
* @return Json
**/
func (s *Item) ValJson(def Json, atribs ...string) Json {
	return s.Result.ValJson(def, atribs...)
}

/**
* ValArray return []interface{} value of the key
* @param def []interface{}, atribs ...string
* @return []interface{}
**/
func (s Item) ValArray(def []interface{}, atribs ...string) []interface{} {
	return s.Result.ValArray(def, atribs...)
}

/**
* Any return any value of the key
* @param def interface{}, atribs ...string
* @return interface{}
**/
func (s Item) Any(def interface{}, atribs ...string) interface{} {
	return s.Result.Any(def, atribs...)
}

/**
* Str return the value of the key
* @param atribs ...string
* @return string
**/
func (s Item) Str(atribs ...string) string {
	return s.Result.Str(atribs...)
}

/**
* ToBase64 return the value of the key
* @param atribs ...string
* @return []string
**/
func (s Item) ToBase64(atribs ...string) string {
	return s.Result.ToBase64(atribs...)
}

/**
* FromBase64 return the value of the key
* @param atribs ...string
* @return []string
**/
func (s Item) FromBase64(atribs ...string) string {
	return s.Result.FromBase64(atribs...)
}

/**
* Int return the value of the key
* @param atribs ...string
* @return int
**/
func (s Item) Int(atribs ...string) int {
	return s.Result.Int(atribs...)
}

/**
* Int64 return the value of the key
* @param atribs ...string
* @return int64
**/
func (s Item) Int64(atribs ...string) int64 {
	return s.Result.Int64(atribs...)
}

/**
* Num return the value of the key
* @param atribs ...string
* @return float64
**/
func (s Item) Num(atribs ...string) float64 {
	return s.Result.Num(atribs...)
}

/**
* Bool return the value of the key
* @param atribs ...string
* @return bool
**/
func (s Item) Bool(atribs ...string) bool {
	return s.Result.Bool(atribs...)
}

/**
* Byte return the value of the key
* @param atribs ...string
* @return []byte
**/
func (s Item) Byte(atribs ...string) ([]byte, error) {
	return s.Result.Byte(atribs...)
}

/**
* Time return the value of the key
* @param atribs ...string
* @return time.Time
**/
func (s Item) Time(atribs ...string) time.Time {
	return s.Result.Time(atribs...)
}

/**
* Json return the value of the key
* @param atrib string
* @return Json
**/
func (s Item) Json(atrib string) Json {
	return s.Result.Json(atrib)
}

/**
* Array return the value of the key
* @param atrib string
* @return []Json
**/
func (s Item) Array(atrib string) []interface{} {
	return s.Result.Array(atrib)
}

/**
* ArrayStr
* @param atribs ...string
* @return []string
**/
func (s Item) ArrayStr(atribs ...string) []string {
	return s.Result.ArrayStr(atribs...)
}

/**
* ArrayInt
* @param atribs ...string
* @return []int
**/
func (s Item) ArrayInt(atribs ...string) []int {
	return s.Result.ArrayInt(atribs...)
}

/**
* ArrayInt64
* @param atribs ...string
* @return []int64
**/
func (s Item) ArrayInt64(atribs ...string) []int64 {
	return s.Result.ArrayInt64(atribs...)
}

/**
* ArrayJson
* @param atribs ...string
* @return []Json
**/
func (s Item) ArrayJson(atribs ...string) []Json {
	return s.Result.ArrayJson(atribs...)
}

/**
* Get
* @param key string
* @return interface{}
**/
func (s Item) Get(key string) interface{} {
	return s.Result.Get(key)
}

/**
* Set a value in the key
* @param key string
* @param val interface{}
* @return bool
**/
func (s Item) Set(key string, val interface{}) {
	if s.Result == nil {
		s.Result = Json{}
	}

	s.Result.Set(key, val)
}

/**
* Delete a value in the key
* @param key string
* @return bool
**/
func (s Item) Delete(keys []string) bool {
	return s.Result.Delete(keys)
}

/**
* ExistKey return if the key exist
* @param key string
* @return bool
**/
func (s Item) ExistKey(key string) bool {
	return s.Result.ExistKey(key)
}
