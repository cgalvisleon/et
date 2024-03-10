package et

import (
	"fmt"
	"reflect"
	"strings"
)

type Items struct {
	Ok     bool   `json:"ok"`
	Count  int    `json:"count"`
	Result []Json `json:"result"`
}

func (it *Items) ValAny(idx int, _default any, atribs ...string) any {
	if it.Result[idx] == nil {
		return _default
	}

	return it.Result[idx].ValAny(_default, atribs...)
}

func (it *Items) ValStr(idx int, _default string, atribs ...string) string {
	if it.Result[idx] == nil {
		return _default
	}

	return it.Result[idx].ValStr(_default, atribs...)
}

func (it *Items) Uppcase(idx int, _default string, atribs ...string) string {
	if it.Result[idx] == nil {
		return _default
	}

	result := Val(it.Result[idx], _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToUpper(v)
	default:
		return fmt.Sprintf(`%v`, strings.ToUpper(_default))
	}
}

func (it *Items) Lowcase(idx int, _default string, atribs ...string) string {
	if it.Result[idx] == nil {
		return _default
	}

	result := Val(it.Result[idx], _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToLower(v)
	default:
		return fmt.Sprintf(`%v`, strings.ToLower(_default))
	}
}

func (it *Items) Titlecase(idx int, _default string, atribs ...string) string {
	if it.Result[idx] == nil {
		return _default
	}

	result := Val(it.Result[idx], _default, atribs...)

	switch v := result.(type) {
	case string:
		return strings.ToTitle(v)
	default:
		return fmt.Sprintf(`%v`, strings.ToTitle(_default))
	}
}

func (it *Items) Get(idx int, key string) interface{} {
	if it.Result[idx] == nil {
		return nil
	}

	return it.Result[idx].Get(key)
}

func (it *Items) Set(idx int, key string, val interface{}) bool {
	if it.Result[idx] == nil {
		return false
	}

	return it.Result[idx].Set(key, val)
}

func (it *Items) Del(idx int, key string) bool {
	if it.Result[idx] == nil {
		return false
	}

	return it.Result[idx].Del(key)
}

func (it *Items) Id(idx int) string {
	return it.Result[idx].Id()
}

func (it *Items) IdT(idx int) string {
	return it.Result[idx].IdT()
}

func (it *Items) Key(idx int, atribs ...string) string {
	return it.Result[idx].Key()
}

func (it *Items) Str(idx int, atribs ...string) string {
	return it.Result[idx].Str()
}

func (it *Items) Int(idx int, atribs ...string) int {
	return it.Result[idx].Int()
}

func (it *Items) Num(idx int, atribs ...string) float64 {
	return it.Result[idx].Num()
}

func (it *Items) Bool(idx int, atribs ...string) bool {
	return it.Result[idx].Bool()
}

func (it *Items) Json(idx int, atribs ...string) Json {
	if it.Result[idx] == nil {
		return Json{}
	}

	val := Val(it.Result[idx], Json{}, atribs...)

	switch v := val.(type) {
	case Json:
		return Json(v)
	case map[string]interface{}:
		return Json(v)
	default:
		Errorf("Not Items.Json type (%v) value:%v", reflect.TypeOf(v), v)
		return Json{}
	}
}

func (it *Items) ToStrings(idx int) string {
	return it.Result[idx].ToString()
}

func (it *Items) ToString() string {
	var result string
	for _, item := range it.Result {
		str := item.ToString()
		result = AppendStr(result, str, ",")
	}

	return fmt.Sprintf(`[%s]`, result)
}

func (it *Items) ToJson() Json {
	return Json{
		"Ok":     it.Ok,
		"Count":  it.Count,
		"Result": it.Result,
	}
}

func (it *Items) ToList(all, page, rows int) List {
	var start int
	var end int
	count := it.Count

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
		Result: it.Result,
	}
}
