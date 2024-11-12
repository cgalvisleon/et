package et

import (
	"encoding/json"
	"time"

	"github.com/cgalvisleon/et/logs"
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
func (s *Items) Scan(src interface{}) error {
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

	*s = Items{
		Ok:     len(t) > 0,
		Count:  len(t),
		Result: t,
	}

	return nil
}

/**
* ToByte convert a json to a []byte
* @return []byte
**/
func (s Items) ToByte() []byte {
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
func (s Items) ToString() string {
	bt, err := json.Marshal(s)
	if err != nil {
		return ""
	}

	result := string(bt)

	return result
}

/**
* ValAny return the value of the key
* @param idx int
* @param _default any
* @param atribs ...string
* @return any
**/
func (s *Items) ValAny(idx int, _default any, atribs ...string) interface{} {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].ValAny(_default, atribs...)
}

/**
* ValStr return the value of the key
* @param idx int
* @param _default string
* @param atribs ...string
* @return string
**/
func (s *Items) ValStr(idx int, _default string, atribs ...string) string {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].ValStr(_default, atribs...)
}

/**
* ValInt return int value of the key
* @param idx int
* @param _default int
* @param atribs ...string
* @return int
**/
func (s *Items) ValInt(idx int, _default int, atribs ...string) int {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].ValInt(_default, atribs...)
}

/**
* ValInt64 return int64 value of the key
* @param idx int
* @param _default int64
* @param atribs ...string
* @return int64
**/
func (s *Items) ValInt64(idx int64, _default int64, atribs ...string) int64 {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].ValInt64(_default, atribs...)
}

/**
* ValNum return float64 value of the key
* @param idx int
* @param _default float64
* @param atribs ...string
* @return float64
**/
func (s *Items) ValNum(idx int, _default float64, atribs ...string) float64 {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].ValNum(_default, atribs...)
}

/**
* ValBool return bool value of the key
* @param idx int
* @param _default bool
* @param atribs ...string
* @return bool
**/
func (s *Items) ValBool(idx int, _default bool, atribs ...string) bool {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].ValBool(_default, atribs...)
}

/**
* ValTime return time.Time value of the key
* @param idx int
* @param _default time.Time
* @param atribs ...string
* @return time.Time
**/
func (s *Items) ValTime(idx int, _default time.Time, atribs ...string) time.Time {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].ValTime(_default, atribs...)
}

/**
* ValJson return the value of the key
* @param idx int
* @param _default Json
* @param atribs ...string
* @return Json
**/
func (s *Items) ValJson(idx int, _default Json, atribs ...string) Json {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].ValJson(_default, atribs...)
}

/**
* ValArray return the value of the key
* @param idx int
* @param _default []interface{}
* @param atribs ...string
* @return []interface{}
**/
func (s *Items) ValArray(idx int, _default []interface{}, atribs ...string) []interface{} {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].ValArray(_default, atribs...)
}

/**
* Any return the value of the key
* @param idx int
* @param _default any
* @param atribs ...string
* @return any
**/
func (s *Items) Any(idx int, _default interface{}, atribs ...string) interface{} {
	if s.Result[idx] == nil {
		return _default
	}

	return s.Result[idx].Any(_default, atribs...)
}

/**
* Id return the value of the key
* @return string
**/
func (s Items) Id(idx int) string {
	return s.Result[idx].Id()
}

/**
* IdT return the value of the key
* @return string
**/
func (s Items) IdT(dx int) string {
	return s.Result[dx].IdT()
}

/**
* Index return the value of the key
* @return int
**/
func (s Items) Index(dx int) int {
	return s.Result[dx].Index()
}

/**
* Key return the value of the key
* @param atribs ...string
* @return string
**/
func (s Items) Key(dx int, atribs ...string) string {
	return s.Result[dx].Key(atribs...)
}

/**
* Str return the value of the key
* @param atribs ...string
* @return string
**/
func (s Items) Str(dx int, atribs ...string) string {
	return s.Result[dx].Str(atribs...)
}

/**
* Int return the value of the key
* @param atribs ...string
* @return int
**/
func (s Items) Int(dx int, atribs ...string) int {
	return s.Result[dx].Int(atribs...)
}

/**
* Int64 return the value of the key
* @param atribs ...string
* @return int64
**/
func (s Items) Int64(dx int, atribs ...string) int64 {
	return s.Result[dx].Int64(atribs...)
}

/**
* Num return the value of the key
* @param atribs ...string
* @return float64
**/
func (s Items) Num(dx int, atribs ...string) float64 {
	return s.Result[dx].Num(atribs...)
}

/**
* Bool return the value of the key
* @param atribs ...string
* @return bool
**/
func (s Items) Bool(dx int, atribs ...string) bool {
	return s.Result[dx].Bool(atribs...)
}

/**
* Time return the value of the key
* @param atribs ...string
* @return time.Time
**/
func (s Items) Time(dx int, atribs ...string) time.Time {
	return s.Result[dx].Time(atribs...)
}

/**
* Json return the value of the key
* @param atrib string
* @return Json
**/
func (s Items) Json(dx int, atrib string) Json {
	return s.Result[dx].Json(atrib)
}

/**
* Array return the value of the key
* @param atrib string
* @return []Json
**/
func (s Items) Array(dx int, atrib string) []interface{} {
	return s.Result[dx].Array(atrib)
}

/**
* ArrayStr
* @param _default []string
* @param atribs ...string
* @return []string
**/
func (s Items) ArrayStr(dx int, _default []string, atribs ...string) []string {
	return s.Result[dx].ArrayStr(_default, atribs...)
}

/**
* ArrayInt
* @param _default []int
* @param atribs ...string
* @return []int
**/
func (s Items) ArrayInt(dx int, _default []int, atribs ...string) []int {
	return s.Result[dx].ArrayInt(_default, atribs...)
}

/**
* ArrayInt64
* @param _default []int64
* @param atribs ...string
* @return []int64
**/
func (s Items) ArrayInt64(dx int, _default []int64, atribs ...string) []int64 {
	return s.Result[dx].ArrayInt64(_default, atribs...)
}

/**
* ArrayJson
* @param _default []Json
* @param atribs ...string
* @return []Json
**/
func (s Items) ArrayJson(dx int, _default []Json, atribs ...string) []Json {
	return s.Result[dx].ArrayJson(_default, atribs...)
}

/**
* Get
* @param key string
* @return interface{}
**/
func (s Items) Get(dx int, key string) interface{} {
	return s.Result[dx].Get(key)
}

/**
* Set a value in the key
* @param key string
* @param val interface{}
* @return bool
**/
func (s *Items) Set(dx int, keys []string, val interface{}) {
	(*s).Result[dx].Set(keys, val)
}

/**
* Delete a value in the key
* @param key string
* @return bool
**/
func (s *Items) Delete(dx int, keys []string) bool {
	return s.Result[dx].Delete(keys)
}

/**
* ExistKey return if the key exist
* @param key string
* @return bool
**/
func (s Items) ExistKey(dx int, key string) bool {
	return s.Result[dx].ExistKey(key)
}

/**
* ToList return the value type List
* @param all int
* @param page int
* @param rows int
* @return List
**/
func (s *Items) ToList(all, page, rows int) List {
	var start int
	var end int
	count := s.Count

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
		Result: s.Result,
	}
}
