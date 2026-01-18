package et

import (
	"encoding/json"
	"time"
)

type Items struct {
	Ok     bool   `json:"ok"`
	Count  int    `json:"count"`
	Result []Json `json:"result"`
}

/**
* ToByte convert a json to a []byte
* @return []byte, error
**/
func (s Items) ToByte() ([]byte, error) {
	result, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return result, nil
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
* Add add an item to the items
* @param item Json
**/
func (s Items) Add(item Json) {
	s.Result = append(s.Result, item)
	s.Count = len(s.Result)
	s.Ok = s.Count > 0
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
* @param idx int, def any, atribs ...string
* @return any
**/
func (s *Items) ValAny(idx int, def any, atribs ...string) interface{} {
	item := s.Result[idx]
	if item == nil {
		return def
	}

	return item.ValAny(def, atribs...)
}

/**
* ValStr return the value of the key
* @param idx int, def string, atribs ...string
* @return string
**/
func (s *Items) ValStr(idx int, def string, atribs ...string) string {
	item := s.Result[idx]
	if item == nil {
		return def
	}

	return item.ValStr(def, atribs...)
}

/**
* ValInt return int value of the key
* @param idx int, def int, atribs ...string
* @return int
**/
func (s *Items) ValInt(idx int, def int, atribs ...string) int {
	item := s.Result[idx]
	if item == nil {
		return def
	}

	return item.ValInt(def, atribs...)
}

/**
* ValInt64 return int64 value of the key
* @param idx int64, def int64, atribs ...string
* @return int64
**/
func (s *Items) ValInt64(idx int64, def int64, atribs ...string) int64 {
	item := s.Result[idx]
	if item == nil {
		return def
	}

	return item.ValInt64(def, atribs...)
}

/**
* ValNum return float64 value of the key
* @param idx int, def float64, atribs ...string
* @return float64
**/
func (s *Items) ValNum(idx int, def float64, atribs ...string) float64 {
	item := s.Result[idx]
	if item == nil {
		return def
	}

	return item.ValNum(def, atribs...)
}

/**
* ValBool return bool value of the key
* @param idx int, def bool, atribs ...string
* @return bool
**/
func (s *Items) ValBool(idx int, def bool, atribs ...string) bool {
	item := s.Result[idx]
	if item == nil {
		return def
	}

	return item.ValBool(def, atribs...)
}

/**
* ValTime return time.Time value of the key
* @param idx int, def time.Time, atribs ...string
* @return time.Time
**/
func (s *Items) ValTime(idx int, def time.Time, atribs ...string) time.Time {
	item := s.Result[idx]
	if item == nil {
		return def
	}

	return item.ValTime(def, atribs...)
}

/**
* ValJson return the value of the key
* @param idx int, def Json, atribs ...string
* @return Json
**/
func (s *Items) ValJson(idx int, def Json, atribs ...string) Json {
	item := s.Result[idx]
	if item == nil {
		return def
	}

	return item.ValJson(def, atribs...)
}

/**
* ValArray return the value of the key
* @param idx int, def []interface{}, atribs ...string
* @return []interface{}
**/
func (s *Items) ValArray(idx int, def []interface{}, atribs ...string) []interface{} {
	if s.Result[idx] == nil {
		return def
	}

	return s.Result[idx].ValArray(def, atribs...)
}

/**
* Any return the value of the key
* @param idx int, def interface{}, atribs ...string
* @return interface{}
**/
func (s *Items) Any(idx int, def interface{}, atribs ...string) interface{} {
	item := s.Result[idx]
	if item == nil {
		return def
	}

	return item.Any(def, atribs...)
}

/**
* Str return the value of the key
* @param idx int, atribs ...string
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
* @param idx int, atribs ...string
* @return int
**/
func (s Items) Int(idx int, atribs ...string) int {
	item := s.Result[idx]
	if item == nil {
		return 0
	}

	return item.Int(atribs...)
}

/**
* Int64 return the value of the key
* @param idx int, atribs ...string
* @return int64
**/
func (s Items) Int64(idx int, atribs ...string) int64 {
	item := s.Result[idx]
	if item == nil {
		return 0
	}

	return item.Int64(atribs...)
}

/**
* Num return the value of the key
* @param idx int, atribs ...string
* @return float64
**/
func (s Items) Num(idx int, atribs ...string) float64 {
	item := s.Result[idx]
	if item == nil {
		return 0
	}

	return item.Num(atribs...)
}

/**
* Bool return the value of the key
* @param idx int, atribs ...string
* @return bool
**/
func (s Items) Bool(idx int, atribs ...string) bool {
	item := s.Result[idx]
	if item == nil {
		return false
	}

	return item.Bool(atribs...)
}

/**
* Time return the value of the key
* @param idx int, atribs ...string
* @return time.Time
**/
func (s Items) Time(idx int, atribs ...string) time.Time {
	item := s.Result[idx]
	if item == nil {
		return time.Time{}
	}

	return item.Time(atribs...)
}

/**
* Json return the value of the key
* @param idx int, atrib string
* @return Json
**/
func (s Items) Json(idx int, atrib string) Json {
	item := s.Result[idx]
	if item == nil {
		return Json{}
	}

	return item.Json(atrib)
}

/**
* Array return the value of the key
* @param idx int, atrib string
* @return []Json
**/
func (s Items) Array(idx int, atrib string) []interface{} {
	item := s.Result[idx]
	if item == nil {
		return []interface{}{}
	}

	return item.Array(atrib)
}

/**
* ArrayStr
* @param idx int, atribs ...string
* @return []string
**/
func (s Items) ArrayStr(idx int, atribs ...string) []string {
	item := s.Result[idx]
	if item == nil {
		return []string{}
	}

	return item.ArrayStr(atribs...)
}

/**
* ArrayInt
* @param idx int, atribs ...string
* @return []int
**/
func (s Items) ArrayInt(idx int, atribs ...string) []int {
	item := s.Result[idx]
	if item == nil {
		return []int{}
	}

	return item.ArrayInt(atribs...)
}

/**
* ArrayInt64
* @param idx int, atribs ...string
* @return []int64
**/
func (s Items) ArrayInt64(idx int, atribs ...string) []int64 {
	item := s.Result[idx]
	if item == nil {
		return []int64{}
	}

	return item.ArrayInt64(atribs...)
}

/**
* ArrayJson
* @param idx int, atribs ...string
* @return []Json
**/
func (s Items) ArrayJson(idx int, atribs ...string) []Json {
	item := s.Result[idx]
	if item == nil {
		return []Json{}
	}

	return item.ArrayJson(atribs...)
}

/**
* Get
* @param idx int, key string
* @return interface{}
**/
func (s Items) Get(idx int, key string) interface{} {
	item := s.Result[idx]
	if item == nil {
		return nil
	}

	return item.Get(key)
}

/**
* Set a value in the key
* @param idx int, key string, val interface{}
* @return bool
**/
func (s *Items) Set(idx int, key string, val interface{}) {
	item := s.Result[idx]
	if item == nil {
		return
	}

	item.Set(key, val)
}

/**
* Delete a value in the key
* @param idx int, keys []string
* @return bool
**/
func (s *Items) Delete(idx int, keys []string) bool {
	item := s.Result[idx]
	if item == nil {
		return false
	}

	return item.Delete(keys)
}

/**
* ExistKey return if the key exist
* @param idx int, key string
* @return bool
**/
func (s Items) ExistKey(idx int, key string) bool {
	item := s.Result[idx]
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
* @param all int, page int, rows int
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
