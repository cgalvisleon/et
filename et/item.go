package et

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Item struct to define a item
type Item struct {
	Ok     bool `json:"ok"`
	Result Json `json:"result"`
}

/**
* Scan load rows to a json
* @param src interface{}
* @return error
**/
func (s *Item) Scan(src interface{}) error {
	err := s.Result.Scan(src)
	if err != nil {
		return err
	}

	(*s).Ok = len(s.Result) > 0

	return nil
}

/**
* ScanRows load rows to a json
* @param rows *sql.Rows
* @return error
**/
func (s *Item) ScanRows(rows *sql.Rows) error {
	err := s.Result.ScanRows(rows)
	if err != nil {
		return err
	}

	(*s).Ok = len(s.Result) > 0

	return nil
}

/**
* ToByte convert a json to a []byte
* @return []byte
**/
func (s Item) ToByte() []byte {
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
func (s Item) ToString() string {
	bt, err := json.Marshal(s)
	if err != nil {
		return ""
	}

	result := string(bt)

	return result
}

/**
* ToJson convert a json to a Json
* @return Json
**/
func (s Item) ToJson() Json {
	return Json{
		"ok":     s.Ok,
		"result": s.Result,
	}
}

/**
* ValAny return any value of the key
* @param defaultVal any
* @param atribs ...string
* @return any
**/
func (s *Item) ValAny(defaultVal interface{}, atribs ...string) interface{} {
	return s.Result.ValAny(defaultVal, atribs...)
}

/**
* ValStr return string value of the key
* @param defaultVal string
* @param atribs ...string
* @return string
**/
func (s *Item) ValStr(defaultVal string, atribs ...string) string {
	return s.Result.ValStr(defaultVal, atribs...)
}

/**
* ValInt return int value of the key
* @param defaultVal int
* @param atribs ...string
* @return int
**/
func (s *Item) ValInt(defaultVal int, atribs ...string) int {
	return s.Result.ValInt(defaultVal, atribs...)
}

/**
* ValInt64 return int64 value of the key
* @param defaultVal int64
* @param atribs ...string
* @return int64
**/
func (s Item) ValInt64(defaultVal int64, atribs ...string) int64 {
	return s.Result.ValInt64(defaultVal, atribs...)
}

/**
* ValNum return float64 value of the key
* @param defaultVal float64
* @param atribs ...string
* @return float64
**/
func (s *Item) ValNum(defaultVal float64, atribs ...string) float64 {
	return s.Result.ValNum(defaultVal, atribs...)
}

/**
* ValBool return bool value of the key
* @param defaultVal bool
* @param atribs ...string
* @return bool
**/
func (s *Item) ValBool(defaultVal bool, atribs ...string) bool {
	return s.Result.ValBool(defaultVal, atribs...)
}

/**
* ValTime return time.Time value of the key
* @param defaultVal time.Time
* @param atribs ...string
* @return time.Time
**/
func (s *Item) ValTime(defaultVal time.Time, atribs ...string) time.Time {
	return s.Result.ValTime(defaultVal, atribs...)
}

/**
* ValJson return Json value of the key
* @param defaultVal Json
* @param atribs ...string
* @return Json
**/
func (s *Item) ValJson(defaultVal Json, atribs ...string) Json {
	return s.Result.ValJson(defaultVal, atribs...)
}

/**
* ValArray return []interface{} value of the key
* @param defaultVal []interface{}
* @param atribs ...string
* @return []interface{}
**/
func (s Item) ValArray(defaultVal []interface{}, atribs ...string) []interface{} {
	return s.Result.ValArray(defaultVal, atribs...)
}

/**
* Any return any value of the key
* @param defaultVal any
* @param atribs ...string
* @return *Any
**/
func (s Item) Any(defaultVal interface{}, atribs ...string) interface{} {
	return s.Result.Any(defaultVal, atribs...)
}

/**
* Key return the value of the key
* @param atribs ...string
* @return string
**/
func (s Item) Key(atribs ...string) string {
	return s.Result.Key(atribs...)
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
func (s *Item) Set(key string, val interface{}) {
	if (*s).Result == nil {
		(*s).Result = Json{}
	}

	(*s).Result.Set(key, val)
}

/**
* Delete a value in the key
* @param key string
* @return bool
**/
func (s *Item) Delete(keys []string) bool {
	return (*s).Result.Delete(keys)
}

/**
* ExistKey return if the key exist
* @param key string
* @return bool
**/
func (s Item) ExistKey(key string) bool {
	return s.Result.ExistKey(key)
}
