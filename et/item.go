package et

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/cgalvisleon/et/console"
)

type Item struct {
	Ok     bool `json:"ok"`
	Result Json `json:"result"`
}

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

func (it *Item) ToScan(src interface{}) error {
	v := reflect.ValueOf(src).Elem()
	for k, val := range it.Result {
		field := v.FieldByName(k)
		if !field.IsValid() {
			console.Errorf("No such field:%s in struct", k)
			continue
		}
		if !field.CanSet() {
			console.Errorf("Cannot set field:%s in struct", k)
			continue
		}
		valType := reflect.ValueOf(val)
		if field.Type() != valType.Type() {
			return console.Errorf(`Provided value type didn't match obj field:%s type`, k)
		}
		field.Set(valType)
	}

	return nil
}

func (it *Item) ValAny(_default any, atribs ...string) any {
	return Val(it.Result, _default, atribs...)
}

func (it *Item) ValStr(_default string, atribs ...string) string {
	return it.Result.ValStr(_default, atribs...)
}

func (it *Item) ValInt(_default int, atribs ...string) int {
	return it.Result.ValInt(_default, atribs...)
}

func (it *Item) ValNum(_default float64, atribs ...string) float64 {
	return it.Result.ValNum(_default, atribs...)
}

func (it *Item) ValBool(_default bool, atribs ...string) bool {
	return it.Result.ValBool(_default, atribs...)
}

func (it *Item) ValTime(atribs ...string) time.Time {
	return it.Result.ValTime(atribs...)
}

func (it *Item) Uppcase(_default string, atribs ...string) string {
	result := Val(it.Result, _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToUpper(v)
	default:
		return fmt.Sprintf(`%v`, strings.ToUpper(_default))
	}
}

func (it *Item) Lowcase(_default string, atribs ...string) string {
	result := Val(it.Result, _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToLower(v)
	default:
		return fmt.Sprintf(`%v`, strings.ToLower(_default))
	}
}

func (it *Item) Titlecase(_default string, atribs ...string) string {
	result := Val(it.Result, _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToTitle(v)
	default:
		return fmt.Sprintf(`%v`, strings.ToTitle(_default))
	}
}

func (it *Item) Get(key string) interface{} {
	return it.Result.Get(key)
}

func (it *Item) Set(key string, val any) bool {
	return it.Result.Set(key, val)
}

func (it *Item) Del(key string) bool {
	return it.Result.Del(key)
}

func (it *Item) IsDiferent(new Json) bool {
	return IsDiferent(it.Result, new)
}

func (it *Item) IsChange(new Json) bool {
	return IsChange(it.Result, new)
}

func (it *Item) Any(_default any, atribs ...string) *Any {
	return it.Result.Any(_default, atribs...)
}

func (it *Item) Id() string {
	return it.Result.Id()
}

func (it *Item) IdT() string {
	return it.Result.IdT()
}

func (it *Item) Index() int {
	return it.Result.Index()
}

func (it *Item) Key(atribs ...string) string {
	return it.Result.Key(atribs...)
}

func (it *Item) Str(atribs ...string) string {
	return it.Result.Str(atribs...)
}

func (it *Item) Int(atribs ...string) int {
	return it.Result.Int(atribs...)
}

func (it *Item) Num(atribs ...string) float64 {
	return it.Result.Num(atribs...)
}

func (it *Item) Bool(atribs ...string) bool {
	return it.Result.Bool(atribs...)
}

func (it *Item) Time(atribs ...string) time.Time {
	return it.Result.Time(atribs...)
}

func (it *Item) Json(atribs ...string) Json {
	val := Val(it.Result, Json{}, atribs...)

	switch v := val.(type) {
	case Json:
		return Json(v)
	case map[string]interface{}:
		return Json(v)
	default:
		console.Errorf("Not Item.Json type (%v) value:%v", reflect.TypeOf(v), v)
		return Json{}
	}
}

func (it *Item) Array(atrib string) []Json {
	return it.Result.Array(atrib)
}

func (it *Item) ArrayStr(atrib string) []string {
	return it.Result.ArrayStr(atrib)
}

func (it *Item) ToString() string {
	return it.Result.ToString()
}

func (it *Item) ToJson() Json {
	return Json{
		"Ok":     it.Ok,
		"Result": it.Result,
	}
}

func (it *Item) ToByte() []byte {
	return Json{
		"Ok":     it.Ok,
		"Result": it.Result,
	}.ToByte()
}

func (it *Item) Consolidate(toField string, ruleOut ...string) Json {
	result := it.Result.Consolidate(toField, ruleOut...)

	return result
}

func (it *Item) ConsolidateAndUpdate(toField string, ruleOut []string, new Json) (Json, error) {
	return it.Result.ConsolidateAndUpdate(toField, ruleOut, new)
}
