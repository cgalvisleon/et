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
* ValAny return any value of the key
* @param _default any
* @param atribs ...string
* @return any
**/
func (s *Item) ValAny(_default interface{}, atribs ...string) interface{} {
	return s.Result.ValAny(_default, atribs...)
}

/**
* ValStr return string value of the key
* @param _default string
* @param atribs ...string
* @return string
**/
func (s *Item) ValStr(_default string, atribs ...string) string {
	return s.Result.ValStr(_default, atribs...)
}

/**
* ValInt return int value of the key
* @param _default int
* @param atribs ...string
* @return int
**/
func (s *Item) ValInt(_default int, atribs ...string) int {
	return s.Result.ValInt(_default, atribs...)
}

/**
* ValInt64 return int64 value of the key
* @param _default int64
* @param atribs ...string
* @return int64
**/
func (s Item) ValInt64(_default int64, atribs ...string) int64 {
	return s.Result.ValInt64(_default, atribs...)
}

/**
* ValNum return float64 value of the key
* @param _default float64
* @param atribs ...string
* @return float64
**/
func (s *Item) ValNum(_default float64, atribs ...string) float64 {
	return s.Result.ValNum(_default, atribs...)
}

/**
* ValBool return bool value of the key
* @param _default bool
* @param atribs ...string
* @return bool
**/
func (s *Item) ValBool(_default bool, atribs ...string) bool {
	return s.Result.ValBool(_default, atribs...)
}

/**
* ValTime return time.Time value of the key
* @param _default time.Time
* @param atribs ...string
* @return time.Time
**/
func (s *Item) ValTime(_default time.Time, atribs ...string) time.Time {
	return s.Result.ValTime(_default, atribs...)
}

/**
* ValJson return Json value of the key
* @param _default Json
* @param atribs ...string
* @return Json
**/
func (s *Item) ValJson(_default Json, atribs ...string) Json {
	return s.Result.ValJson(_default, atribs...)
}

/**
* ValArray return []interface{} value of the key
* @param _default []interface{}
* @param atribs ...string
* @return []interface{}
**/
func (s Item) ValArray(_default []interface{}, atribs ...string) []interface{} {
	return s.Result.ValArray(_default, atribs...)
}

/**
* Any return any value of the key
* @param _default any
* @param atribs ...string
* @return *Any
**/
func (s Item) Any(_default interface{}, atribs ...string) interface{} {
	return s.Result.Any(_default, atribs...)
}

/**
* Id return the value of the key
* @return string
**/
func (s Item) Id() string {
	return s.Result.Id()
}

/**
* IdT return the value of the key
* @return string
**/
func (s Item) IdT() string {
	return s.Result.IdT()
}

/**
* Index return the value of the key
* @return int
**/
func (s Item) Index() int {
	return s.Result.Index()
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
* @param _default []string
* @param atribs ...string
* @return []string
**/
func (s Item) ArrayStr(_default []string, atribs ...string) []string {
	return s.Result.ArrayStr(_default, atribs...)
}

/**
* ArrayInt
* @param _default []int
* @param atribs ...string
* @return []int
**/
func (s Item) ArrayInt(_default []int, atribs ...string) []int {
	return s.Result.ArrayInt(_default, atribs...)
}

/**
* ArrayInt64
* @param _default []int64
* @param atribs ...string
* @return []int64
**/
func (s Item) ArrayInt64(_default []int64, atribs ...string) []int64 {
	return s.Result.ArrayInt64(_default, atribs...)
}

/**
* ArrayJson
* @param _default []Json
* @param atribs ...string
* @return []Json
**/
func (s Item) ArrayJson(_default []Json, atribs ...string) []Json {
	return s.Result.ArrayJson(_default, atribs...)
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
func (s *Item) Set(keys []string, val interface{}) {
	(*s).Result.Set(keys, val)
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
