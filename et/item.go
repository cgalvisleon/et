package et

import (
	"database/sql"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/generic"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/strs"
)

// Item struct to define a item
type Item struct {
	Ok     bool `json:"ok"`
	Result Json `json:"result"`
}

// Scan a row from a sql query
func (it *Item) Scan(rows *sql.Rows) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]interface{}, len(cols))
	pointers := make([]interface{}, len(cols))
	for i := range values {
		pointers[i] = &values[i]
	}

	if err := rows.Scan(pointers...); err != nil {
		return err
	}

	it.Ok = true
	it.Result = make(Json)
	for i, colName := range cols {
		if values[i] == nil {
			it.Result[colName] = nil
		} else if reflect.TypeOf(values[i]).String() == "[]uint8" {
			it.Result[colName] = ToUnit8Json(values[i])
		} else {
			it.Result[colName] = values[i]
		}
	}

	return nil
}

// ToScan a struct from a item
func (it *Item) ToScan(src interface{}) error {
	v := reflect.ValueOf(src).Elem()
	for k, val := range it.Result {
		field := v.FieldByName(k)
		if !field.IsValid() {
			logs.Errorf("No such field:%s in struct", k)
			continue
		}
		if !field.CanSet() {
			logs.Errorf("Cannot set field:%s in struct", k)
			continue
		}
		valType := reflect.ValueOf(val)
		if field.Type() != valType.Type() {
			return logs.Errorf(`Provided value type didn't match obj field:%s type`, k)
		}
		field.Set(valType)
	}

	return nil
}

// ValAny get a value from a item
func (it *Item) ValAny(_default any, atribs ...string) any {
	return Val(it.Result, _default, atribs...)
}

// ValStr get a string value from a item
func (it *Item) ValStr(_default string, atribs ...string) string {
	return it.Result.ValStr(_default, atribs...)
}

// ValInt get a int value from a item
func (it *Item) ValInt(_default int, atribs ...string) int {
	return it.Result.ValInt(_default, atribs...)
}

// ValNum get a float64 value from a item
func (it *Item) ValNum(_default float64, atribs ...string) float64 {
	return it.Result.ValNum(_default, atribs...)
}

// ValBool get a bool value from a item
func (it *Item) ValBool(_default bool, atribs ...string) bool {
	return it.Result.ValBool(_default, atribs...)
}

// ValTime get a time.Time value from a item
func (it *Item) ValTime(_default time.Time, atribs ...string) time.Time {
	return it.Result.ValTime(_default, atribs...)
}

// ValJson get a Json value from a item
func (it *Item) ValJson(_default Json, atribs ...string) Json {
	return it.Result.ValJson(_default, atribs...)
}

// Uppcase a string value from a item
func (it *Item) Uppcase(_default string, atribs ...string) string {
	result := Val(it.Result, _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToUpper(v)
	default:
		return strs.Format(`%v`, strings.ToUpper(_default))
	}
}

// Lowcase a string value from a item
func (it *Item) Lowcase(_default string, atribs ...string) string {
	result := Val(it.Result, _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToLower(v)
	default:
		return strs.Format(`%v`, strings.ToLower(_default))
	}
}

// Titlecase a string value from a item
func (it *Item) Titlecase(_default string, atribs ...string) string {
	result := Val(it.Result, _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToTitle(v)
	default:
		return strs.Format(`%v`, strings.ToTitle(_default))
	}
}

// Get a value from a item
func (it *Item) Get(key string) interface{} {
	return it.Result.Get(key)
}

// Set a value from a item
func (it *Item) Set(key string, val any) bool {
	return it.Result.Set(key, val)
}

// Del a value from a item
func (it *Item) Del(key string) bool {
	return it.Result.Del(key)
}

// IsDiferent compare two items
func (it *Item) IsDiferent(new Json) bool {
	return IsDiferent(it.Result, new)
}

// Any get a value from a item
func (it *Item) Any(_default any, atribs ...string) *generic.Any {
	return it.Result.Any(_default, atribs...)
}

// Id get a string value from a item
func (it *Item) Id() string {
	return it.Result.Id()
}

// IdT get a string value from a item
func (it *Item) IdT() string {
	return it.Result.IdT()
}

// Index get a int value from a item
func (it *Item) Index() int {
	return it.Result.Index()
}

// Key get a string value from a item
func (it *Item) Key(atribs ...string) string {
	return it.Result.Key(atribs...)
}

// Str get a string value from a item
func (it *Item) Str(atribs ...string) string {
	return it.Result.Str(atribs...)
}

// Int get a int value from a item
func (it *Item) Int(atribs ...string) int {
	return it.Result.Int(atribs...)
}

// Num get a float64 value from a item
func (it *Item) Num(atribs ...string) float64 {
	return it.Result.Num(atribs...)
}

// Bool get a bool value from a item
func (it *Item) Bool(atribs ...string) bool {
	return it.Result.Bool(atribs...)
}

// Time get a time.Time value from a item
func (it *Item) Time(atribs ...string) time.Time {
	return it.Result.Time(atribs...)
}

// Data get a JsonD value from a item
func (it *Item) Data(atribs ...string) JsonD {
	return it.Result.Data(atribs...)
}

// Json get a Json value from a item
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

// Array get a []Json value from a item
func (it *Item) Array(atrib string) []Json {
	return it.Result.Array(atrib)
}

// ArrayStr get a []string value from a item
func (it *Item) ArrayStr(atrib string) []string {
	return it.Result.ArrayStr(atrib)
}

// ToString get a string value from a item
func (it *Item) ToString() string {
	return it.Result.ToString()
}

// ToJson get a Json value from a item
func (it *Item) ToJson() Json {
	return Json{
		"Ok":     it.Ok,
		"Result": it.Result,
	}
}

// ToByte get a []byte value from a item
func (it *Item) ToByte() []byte {
	return Json{
		"Ok":     it.Ok,
		"Result": it.Result,
	}.ToByte()
}
