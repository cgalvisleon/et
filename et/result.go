package et

import (
	"encoding/json"
	"errors"
	"time"
)

type Item struct {
	Ok     bool `json:"ok"`
	Result Json `json:"result"`
}

/**
* NewItem
* @param data Json
* @return Item
**/
func NewItem(data Json) Item {
	return Item{
		Ok:     true,
		Result: data,
	}
}

/**
* ToObject convert a json to a object
* @return []byte, error
**/
func (s Item) ToObject(dest any) error {
	return s.Result.ToObject(dest)
}

/**
* ToByte convert a json to a []byte
* @return []byte, error
**/
func (s Item) ToByte() ([]byte, error) {
	return json.Marshal(s)
}

/**
* ToJson convert a json to a Json
* @return Json
**/
func (s Item) ToJson() Json {
	bt, err := s.ToByte()
	if err != nil {
		return Json{}
	}

	var result Json
	err = decodeJSON(bt, &result)
	if err != nil {
		return Json{}
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
* String return the value of the key
* @param atribs ...string
* @return string
**/
func (s Item) String(atribs ...string) string {
	return s.Result.String(atribs...)
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
func (s *Item) Set(key string, val interface{}) {
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
* Exist return if the key exist
* @param key string
* @return bool
**/
func (s Item) Exist(key string) bool {
	return s.Result.Exist(key)
}

/**
* From
* @param as string
* @return *Where
**/
func (s Item) From(as string) *Wheres {
	return From([]Json{s.Result}, as)
}

type Items struct {
	Ok     bool   `json:"ok"`
	Count  int    `json:"count"`
	Result []Json `json:"result"`
}

func NewItems(data []Json) Items {
	result := Items{
		Count:  len(data),
		Result: data,
	}
	result.Ok = result.Count > 0
	return result
}

/**
* ToByte convert a json to a []byte
* @return []byte, error
**/
func (s Items) ToByte() ([]byte, error) {
	return json.Marshal(s)
}

/**
* ToJson convert a json
* @return Json
**/
func (s Items) ToJson() Json {
	bt, err := s.ToByte()
	if err != nil {
		return Json{}
	}

	var result Json
	err = decodeJSON(bt, &result)
	if err != nil {
		return Json{}
	}
	return result
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
func (s *Items) Add(item ...Json) {
	if s.Result == nil {
		s.Result = make([]Json, 0)
	}
	s.Result = append(s.Result, item...)
	s.Count = len(s.Result)
	s.Ok = s.Count > 0
}

/**
* AddMany add a slice of items with pre-allocated capacity
* @param items []Json
**/
func (s *Items) AddMany(items []Json) {
	if len(items) == 0 {
		return
	}
	if s.Result == nil {
		s.Result = make([]Json, 0, len(items))
	}
	s.Result = append(s.Result, items...)
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
	items := make([]map[string]interface{}, 0, len(s.Result))
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
	if item == nil {
		return def
	}

	return item.ValInt(def, atribs...)
}

/**
* ValInt64 return int64 value of the key
* @param idx int, def int64, atribs ...string
* @return int64
**/
func (s *Items) ValInt64(idx int, def int64, atribs ...string) int64 {
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	if s.at(idx) == nil {
		return def
	}

	return s.at(idx).ValArray(def, atribs...)
}

/**
* Any return the value of the key
* @param idx int, def interface{}, atribs ...string
* @return interface{}
**/
func (s *Items) Any(idx int, def interface{}, atribs ...string) interface{} {
	item := s.at(idx)
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
	item := s.at(dx)
	if item == nil {
		return ""
	}

	return item.Str(atribs...)
}

/**
* String return the value of the key
* @param idx int, atribs ...string
* @return string
**/
func (s Items) String(idx int, atribs ...string) string {
	return s.Str(idx, atribs...)
}

/**
* Int return the value of the key
* @param idx int, atribs ...string
* @return int
**/
func (s Items) Int(idx int, atribs ...string) int {
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
	item := s.at(idx)
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
func (s Items) Exist(idx int, key string) bool {
	item := s.at(idx)
	if item == nil {
		return false
	}

	return item.Exist(key)
}

/**
* One
* @param idx int
* @return Item, error
**/
func (s Items) One(idx int) (Item, error) {
	n := len(s.Result)
	if idx < 0 {
		idx = n + idx
	} else if idx > 0 {
		idx = idx - 1
	}

	if idx < 0 || idx >= n {
		return Item{}, errors.New(MSG_INDEX_OUT_OF_RANGE)
	}

	return NewItem(s.Result[idx]), nil
}

/**
* First
* @return Item, error
**/
func (s *Items) First() (Item, error) {
	return s.One(1)
}

/**
* Last
* @return Item, error
**/
func (s *Items) Last() (Item, error) {
	return s.One(-1)
}

/**
* at: The item at index idx (0-based) of Result, or nil when it is out of range, so the accessors give their default
* value instead of a panic
* @param idx int
* @return Json
**/
func (s Items) at(idx int) Json {
	if idx < 0 || idx >= len(s.Result) {
		return nil
	}
	return s.Result[idx]
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

/**
* From
* @param as string
* @return *Where
**/
func (s Items) From(as string) *Wheres {
	return From(s.Result, as)
}

/* List struct to use in et */
type List struct {
	Rows   int    `json:"rows"`
	All    int    `json:"all"`
	Count  int    `json:"count"`
	Page   int    `json:"page"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Result []Json `json:"result"`
}

func (s *List) ToByte() ([]byte, error) {
	return json.Marshal(s)
}

/**
* ToJson
* @return Json
**/
func (s *List) ToJson() Json {
	bt, err := s.ToByte()
	if err != nil {
		return Json{}
	}

	var result Json
	err = decodeJSON(bt, &result)
	if err != nil {
		return Json{}
	}
	return result
}

/**
* ToString
* @return string
**/
func (s *List) ToString() string {
	return s.ToJson().ToString()
}

/**
* ToMap
* @return map[string]interface{}
**/
func (s *List) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"rows":   s.Rows,
		"all":    s.All,
		"count":  s.Count,
		"page":   s.Page,
		"start":  s.Start,
		"end":    s.End,
		"result": s.Result,
	}
}

/**
* From
* @param as string
* @return *Where
**/
func (s List) From(as string) *Wheres {
	return From(s.Result, as)
}
