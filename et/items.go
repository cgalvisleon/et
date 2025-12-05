package et

import (
	"encoding/json"
	"fmt"
	"time"
)

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
		return fmt.Errorf(`json/Scan - Failed to unmarshal JSON value:%s`, src)
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
* Items methods
* @param src interface{}
* @return error
**/
func (s *Items) Add(item Json) {
	(*s).Result = append((*s).Result, item)
	(*s).Count = len((*s).Result)
	(*s).Ok = (*s).Count > 0
}

/**
* AddMany add many items to the items
* @param items []Json
**/
func (s *Items) AddMany(items []Json) {
	(*s).Result = append((*s).Result, items...)
	(*s).Count = len((*s).Result)
	(*s).Ok = (*s).Count > 0
}

/**
* ToJson convert a json
* @return Json
**/
func (s Items) ToJson() Json {
	return Json{
		"ok":     s.Ok,
		"count":  s.Count,
		"result": s.Result,
	}
}

/**
* ToString convert a json to a string
* @return string
**/
func (s Items) ToString() string {
	return s.ToJson().ToString()
}

/**
* ToMap convert a json to a map
* @return map[string]interface{}
**/
func (s *Items) ToMap() map[string]interface{} {
	result := make(map[string]interface{})
	result["ok"] = s.Ok
	result["count"] = s.Count
	items := make([]map[string]interface{}, 0)
	for _, item := range s.Result {
		items = append(items, item.ToMap())
	}

	result["result"] = items
	return result
}

/**
* ValAny return the value of the key
* @param idx int
* @param defaultVal any
* @param atribs ...string
* @return any
**/
func (s *Items) ValAny(idx int, defaultVal any, atribs ...string) interface{} {
	item := s.Result[idx]
	if item == nil {
		return defaultVal
	}

	return item.ValAny(defaultVal, atribs...)
}

/**
* ValStr return the value of the key
* @param idx int
* @param defaultVal string
* @param atribs ...string
* @return string
**/
func (s *Items) ValStr(idx int, defaultVal string, atribs ...string) string {
	item := s.Result[idx]
	if item == nil {
		return defaultVal
	}

	return item.ValStr(defaultVal, atribs...)
}

/**
* ValInt return int value of the key
* @param idx int
* @param defaultVal int
* @param atribs ...string
* @return int
**/
func (s *Items) ValInt(idx int, defaultVal int, atribs ...string) int {
	item := s.Result[idx]
	if item == nil {
		return defaultVal
	}

	return item.ValInt(defaultVal, atribs...)
}

/**
* ValInt64 return int64 value of the key
* @param idx int
* @param defaultVal int64
* @param atribs ...string
* @return int64
**/
func (s *Items) ValInt64(idx int64, defaultVal int64, atribs ...string) int64 {
	item := s.Result[idx]
	if item == nil {
		return defaultVal
	}

	return item.ValInt64(defaultVal, atribs...)
}

/**
* ValNum return float64 value of the key
* @param idx int
* @param defaultVal float64
* @param atribs ...string
* @return float64
**/
func (s *Items) ValNum(idx int, defaultVal float64, atribs ...string) float64 {
	item := s.Result[idx]
	if item == nil {
		return defaultVal
	}

	return item.ValNum(defaultVal, atribs...)
}

/**
* ValBool return bool value of the key
* @param idx int
* @param defaultVal bool
* @param atribs ...string
* @return bool
**/
func (s *Items) ValBool(idx int, defaultVal bool, atribs ...string) bool {
	item := s.Result[idx]
	if item == nil {
		return defaultVal
	}

	return item.ValBool(defaultVal, atribs...)
}

/**
* ValTime return time.Time value of the key
* @param idx int
* @param defaultVal time.Time
* @param atribs ...string
* @return time.Time
**/
func (s *Items) ValTime(idx int, defaultVal time.Time, atribs ...string) time.Time {
	item := s.Result[idx]
	if item == nil {
		return defaultVal
	}

	return item.ValTime(defaultVal, atribs...)
}

/**
* ValJson return the value of the key
* @param idx int
* @param defaultVal Json
* @param atribs ...string
* @return Json
**/
func (s *Items) ValJson(idx int, defaultVal Json, atribs ...string) Json {
	item := s.Result[idx]
	if item == nil {
		return defaultVal
	}

	return item.ValJson(defaultVal, atribs...)
}

/**
* ValArray return the value of the key
* @param idx int
* @param defaultVal []interface{}
* @param atribs ...string
* @return []interface{}
**/
func (s *Items) ValArray(idx int, defaultVal []interface{}, atribs ...string) []interface{} {
	if s.Result[idx] == nil {
		return defaultVal
	}

	return s.Result[idx].ValArray(defaultVal, atribs...)
}

/**
* Any return the value of the key
* @param idx int
* @param defaultVal any
* @param atribs ...string
* @return any
**/
func (s *Items) Any(idx int, defaultVal interface{}, atribs ...string) interface{} {
	item := s.Result[idx]
	if item == nil {
		return defaultVal
	}

	return item.Any(defaultVal, atribs...)
}

/**
* Key return the value of the key
* @param atribs ...string
* @return string
**/
func (s Items) Key(dx int, atribs ...string) string {
	item := s.Result[dx]
	if item == nil {
		return ""
	}

	return item.Key(atribs...)
}

/**
* Str return the value of the key
* @param atribs ...string
* @return string
**/
func (s Items) Str(dx int, atribs ...string) string {
	item := s.Result[dx]
	if item == nil {
		return ""
	}

	return item.Str(atribs...)
}

/**
* Int return the value of the key
* @param atribs ...string
* @return int
**/
func (s Items) Int(dx int, atribs ...string) int {
	item := s.Result[dx]
	if item == nil {
		return 0
	}

	return item.Int(atribs...)
}

/**
* Int64 return the value of the key
* @param atribs ...string
* @return int64
**/
func (s Items) Int64(dx int, atribs ...string) int64 {
	item := s.Result[dx]
	if item == nil {
		return 0
	}

	return item.Int64(atribs...)
}

/**
* Num return the value of the key
* @param atribs ...string
* @return float64
**/
func (s Items) Num(dx int, atribs ...string) float64 {
	item := s.Result[dx]
	if item == nil {
		return 0
	}

	return item.Num(atribs...)
}

/**
* Bool return the value of the key
* @param atribs ...string
* @return bool
**/
func (s Items) Bool(dx int, atribs ...string) bool {
	item := s.Result[dx]
	if item == nil {
		return false
	}

	return item.Bool(atribs...)
}

/**
* Time return the value of the key
* @param atribs ...string
* @return time.Time
**/
func (s Items) Time(dx int, atribs ...string) time.Time {
	item := s.Result[dx]
	if item == nil {
		return time.Time{}
	}

	return item.Time(atribs...)
}

/**
* Json return the value of the key
* @param atrib string
* @return Json
**/
func (s Items) Json(dx int, atrib string) Json {
	item := s.Result[dx]
	if item == nil {
		return Json{}
	}

	return item.Json(atrib)
}

/**
* Array return the value of the key
* @param atrib string
* @return []Json
**/
func (s Items) Array(dx int, atrib string) []interface{} {
	item := s.Result[dx]
	if item == nil {
		return []interface{}{}
	}

	return item.Array(atrib)
}

/**
* ArrayStr
* @param atribs ...string
* @return []string
**/
func (s Items) ArrayStr(dx int, atribs ...string) []string {
	item := s.Result[dx]
	if item == nil {
		return []string{}
	}

	return item.ArrayStr(atribs...)
}

/**
* ArrayInt
* @param atribs ...string
* @return []int
**/
func (s Items) ArrayInt(dx int, atribs ...string) []int {
	item := s.Result[dx]
	if item == nil {
		return []int{}
	}

	return item.ArrayInt(atribs...)
}

/**
* ArrayInt64
* @param atribs ...string
* @return []int64
**/
func (s Items) ArrayInt64(dx int, atribs ...string) []int64 {
	item := s.Result[dx]
	if item == nil {
		return []int64{}
	}

	return item.ArrayInt64(atribs...)
}

/**
* ArrayJson
* @param atribs ...string
* @return []Json
**/
func (s Items) ArrayJson(dx int, atribs ...string) []Json {
	item := s.Result[dx]
	if item == nil {
		return []Json{}
	}

	return item.ArrayJson(atribs...)
}

/**
* Get
* @param key string
* @return interface{}
**/
func (s Items) Get(dx int, key string) interface{} {
	item := s.Result[dx]
	if item == nil {
		return nil
	}

	return item.Get(key)
}

/**
* Set a value in the key
* @param key string
* @param val interface{}
* @return bool
**/
func (s *Items) Set(dx int, key string, val interface{}) {
	item := s.Result[dx]
	if item == nil {
		return
	}

	item.Set(key, val)
}

/**
* Delete a value in the key
* @param key string
* @return bool
**/
func (s *Items) Delete(dx int, keys []string) bool {
	item := s.Result[dx]
	if item == nil {
		return false
	}

	return item.Delete(keys)
}

/**
* ExistKey return if the key exist
* @param key string
* @return bool
**/
func (s Items) ExistKey(dx int, key string) bool {
	item := s.Result[dx]
	if item == nil {
		return false
	}

	return item.ExistKey(key)
}

/**
* First
* @return Item
**/
func (s Items) First() Item {
	if s.Count == 0 {
		return Item{Result: Json{}}
	}

	return Item{
		Ok:     true,
		Result: s.Result[0],
	}
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
