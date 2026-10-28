package et

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
)

/**
* Json in goja (jrex, jwf): one representation both ways.
*
* Go → JavaScript (ToJS): a Json (or a map[string]any, an Item, an Items, a List) is an object over the same Go map: its
* keys are properties (obj.name, obj.user.email, obj.name = "x", delete obj.name, Object.keys, JSON.stringify) and the
* methods of Json are there with their Go names (obj.Str("name"), obj.ValStr("", "user", "email"), obj.Set("a", 1)).
* A change in JavaScript is a change in the Go map. Lists are arrays whose objects follow the same rule.
*
* JavaScript → Go (FromJS, JsonOf): an object of ToJS gives back its Json (the same map); any other object is a new Json
* with its values converted the same way, an array is a []any, a Date a time.Time, and integers come as int64.
*
* A Go function is called with its arguments converted with GoArg and its results with ToJS (a returned error is thrown);
* one with the goja signatures (func(goja.FunctionCall) …) goes as is. A struct (or a pointer to one) is an object with
* its exported methods and fields by their Go name (db.GetModel(…).Where(…).All()).
**/

// jsonPrototypeKey: where a runtime keeps the prototype with the methods of Json (one per runtime)
var jsonPrototypeKey = goja.NewSymbol("et.Json")

// jsObject: a Json seen from JavaScript (a goja.DynamicObject over the Go map)
type jsObject struct {
	vm   *goja.Runtime
	data Json
	self *goja.Object
	kids map[string]*jsObject // the nested objects already wrapped, so obj.user === obj.user
}

/**
* JsonValue: The Json behind the object (the same Go map); with it, Json reads a nested object that goja exported as is
* @return Json
**/
func (o *jsObject) JsonValue() Json {
	return o.data
}

func (o *jsObject) Get(key string) goja.Value {
	value, ok := o.data[key]
	if !ok {
		return nil // nil: goja looks for it in the prototype (the methods)
	}

	if data, ok := asJson(value); ok {
		kid := o.kids[key]
		if kid == nil || !sameMap(kid.data, data) {
			kid = newJsObject(o.vm, data)
			o.kids[key] = kid
		}
		return kid.self
	}

	return ToJS(o.vm, value)
}

func (o *jsObject) Set(key string, val goja.Value) bool {
	o.data[key] = FromJS(val)
	delete(o.kids, key)
	return true
}

func (o *jsObject) Has(key string) bool {
	_, ok := o.data[key]
	return ok
}

func (o *jsObject) Delete(key string) bool {
	delete(o.data, key)
	delete(o.kids, key)
	return true
}

func (o *jsObject) Keys() []string {
	return slices.Sorted(maps.Keys(o.data))
}

// String and MarshalJSON: the object handed to Go as is (a log, json.Marshal) shows its data
func (o *jsObject) String() string {
	return o.data.ToString()
}

func (o *jsObject) MarshalJSON() ([]byte, error) {
	return json.Marshal(o.data)
}

/**
* newJsObject: The object of a Json in a runtime, with the prototype of the methods
* @param vm *goja.Runtime, data Json
* @return *jsObject
**/
func newJsObject(vm *goja.Runtime, data Json) *jsObject {
	if data == nil {
		data = Json{}
	}
	result := &jsObject{vm: vm, data: data, kids: map[string]*jsObject{}}
	result.self = vm.NewDynamicObject(result)
	result.self.SetPrototype(jsonPrototype(vm))
	return result
}

/**
* sameMap: Whether a and b are the same Go map (not just equal)
* @param a, b Json
* @return bool
**/
func sameMap(a, b Json) bool {
	return reflect.ValueOf(a).UnsafePointer() == reflect.ValueOf(b).UnsafePointer()
}

/**
* asJson: The Json of a value that is one: a Json, a map[string]any or an object of a script (its same map)
* @param value any
* @return Json, bool
**/
func asJson(value any) (Json, bool) {
	switch v := value.(type) {
	case Json:
		return v, v != nil
	case map[string]any:
		return Json(v), v != nil
	case interface{ JsonValue() Json }:
		return v.JsonValue(), true
	}
	return nil, false
}

/**
* ToJS: A Go value as JavaScript: a Json, a map[string]any, an Item, an Items or a List as an object over its map with
* the methods of Json; a list as an array of values converted the same way; the rest as goja converts it
* @param vm *goja.Runtime, value any
* @return goja.Value
**/
func ToJS(vm *goja.Runtime, value any) goja.Value {
	switch v := value.(type) {
	case nil:
		return goja.Null()
	case goja.Value:
		return v
	case *jsObject:
		if v.vm == vm {
			return v.self
		}
		return newJsObject(vm, v.data).self
	case Json:
		if v == nil {
			return goja.Null()
		}
		return newJsObject(vm, v).self
	case map[string]any:
		if v == nil {
			return goja.Null()
		}
		return newJsObject(vm, Json(v)).self
	case Item:
		return ToJS(vm, v.ToJson())
	case *Item:
		return ToJS(vm, v.ToJson())
	case Items:
		return ToJS(vm, v.ToJson())
	case *Items:
		return ToJS(vm, v.ToJson())
	case List:
		return ToJS(vm, v.ToJson())
	case *List:
		return ToJS(vm, v.ToJson())
	case []byte, time.Time:
		return vm.ToValue(v)
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return vm.NewArray()
		}
		items := make([]any, rv.Len())
		for i := range items {
			items[i] = ToJS(vm, rv.Index(i).Interface())
		}
		return vm.NewArray(items...)
	case reflect.Func:
		if rv.IsNil() {
			return goja.Null()
		}
		if isGojaFunc(rv.Type()) {
			return vm.ToValue(value)
		}
		return vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return callGo(vm, rv, call.Arguments)
		})
	case reflect.Pointer:
		if rv.IsNil() {
			return goja.Null()
		}
		if rv.Elem().Kind() == reflect.Struct {
			return vm.NewDynamicObject(&goObject{vm: vm, value: rv})
		}
	case reflect.Struct:
		// A copy by pointer, so its pointer methods are there too
		ptr := reflect.New(rv.Type())
		ptr.Elem().Set(rv)
		return vm.NewDynamicObject(&goObject{vm: vm, value: ptr})
	}

	return vm.ToValue(value)
}

/**
* isGojaFunc: Whether a function has the signature goja calls as is (its first argument a goja.FunctionCall or a
* goja.ConstructorCall)
* @param tp reflect.Type
* @return bool
**/
func isGojaFunc(tp reflect.Type) bool {
	if tp.NumIn() == 0 {
		return false
	}
	first := tp.In(0)
	return first == reflect.TypeFor[goja.FunctionCall]() || first == reflect.TypeFor[goja.ConstructorCall]()
}

/**
* FromJS: A JavaScript value as Go data: an object of ToJS gives its Json (the same map), another object a new Json,
* an array a []any (each value converted the same way), a Date a time.Time, undefined and null nil, the rest as goja
* exports it (integers as int64, other numbers as float64)
* @param value goja.Value
* @return any
**/
func FromJS(value goja.Value) any {
	return fromJS(value, map[*goja.Object]any{})
}

/**
* fromJS: FromJS with the objects already converted (a cycle gives back the same value)
* @param value goja.Value, seen map[*goja.Object]any
* @return any
**/
func fromJS(value goja.Value, seen map[*goja.Object]any) any {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}

	obj, ok := value.(*goja.Object)
	if !ok {
		return value.Export()
	}

	if done, ok := seen[obj]; ok {
		return done
	}

	exported := obj.Export()
	if data, ok := exported.(*jsObject); ok {
		return data.data
	}
	if goObj, ok := exported.(*goObject); ok {
		return goObj.value.Interface()
	}

	switch obj.ClassName() {
	case "Array":
		n := int(obj.Get("length").ToInteger())
		result := make([]any, n)
		seen[obj] = result
		for i := range n {
			result[i] = fromJS(obj.Get(strconv.Itoa(i)), seen)
		}
		return result
	case "Object":
		if _, isFunc := goja.AssertFunction(obj); isFunc {
			return exported
		}
		result := Json{}
		seen[obj] = result
		for _, key := range obj.Keys() {
			result[key] = fromJS(obj.Get(key), seen)
		}
		return result
	}

	return exported
}

/**
* JsonOf: The Json of a JavaScript value (FromJS); false if it is not an object
* @param value goja.Value
* @return Json, bool
**/
func JsonOf(value goja.Value) (Json, bool) {
	return asJson(FromJS(value))
}

/**
* JsonPrototype: The prototype of the Json objects of a runtime (the methods of Json), to add methods of its own
* @param vm *goja.Runtime
* @return *goja.Object
**/
func JsonPrototype(vm *goja.Runtime) *goja.Object {
	return jsonPrototype(vm)
}

/**
* jsonPrototype: The prototype of the Json objects of a runtime: every method of Json by its Go name (except the ones
* of database/sql), made the first time it is asked for
* @param vm *goja.Runtime
* @return *goja.Object
**/
func jsonPrototype(vm *goja.Runtime) *goja.Object {
	global := vm.GlobalObject()
	if proto, ok := global.GetSymbol(jsonPrototypeKey).(*goja.Object); ok {
		return proto
	}

	proto := vm.NewObject()
	tp := reflect.TypeFor[*Json]()
	for i := range tp.NumMethod() {
		method := tp.Method(i)
		if method.Name == "Scan" || method.Name == "ScanRows" {
			continue
		}
		proto.Set(method.Name, jsonMethod(vm, method.Name))
	}
	global.SetSymbol(jsonPrototypeKey, proto)
	return proto
}

/**
* jsonMethod: A method of Json called from JavaScript on this: the arguments go to its Go types, a returned error is
* thrown (try/catch, e.message) and the results come back with ToJS (several, an array)
* @param vm *goja.Runtime, name string
* @return func(call goja.FunctionCall) goja.Value
**/
func jsonMethod(vm *goja.Runtime, name string) func(call goja.FunctionCall) goja.Value {
	return func(call goja.FunctionCall) goja.Value {
		data, ok := JsonOf(call.This)
		if !ok {
			panic(vm.NewTypeError("%s: this is not an et.Json", name))
		}

		fn := reflect.ValueOf(&data).MethodByName(name)
		return callGo(vm, fn, call.Arguments)
	}
}

/**
* callGo: Calls a Go function with JavaScript arguments converted to its types (GoArg)
* @param vm *goja.Runtime, fn reflect.Value, args []goja.Value
* @return goja.Value
**/
func callGo(vm *goja.Runtime, fn reflect.Value, args []goja.Value) goja.Value {
	tp := fn.Type()
	in := make([]reflect.Value, 0, len(args))
	for i := range tp.NumIn() {
		if tp.IsVariadic() && i == tp.NumIn()-1 {
			elem := tp.In(i).Elem()
			for j := i; j < len(args); j++ {
				in = append(in, GoArg(vm, args[j], elem))
			}
			break
		}

		arg := goja.Undefined()
		if i < len(args) {
			arg = args[i]
		}
		in = append(in, GoArg(vm, arg, tp.In(i)))
	}

	out := fn.Call(in)
	if n := len(out); n > 0 && tp.Out(n-1) == reflect.TypeFor[error]() {
		if err, ok := out[n-1].Interface().(error); ok && err != nil {
			panic(vm.NewGoError(err))
		}
		out = out[:n-1]
	}

	switch len(out) {
	case 0:
		return goja.Undefined()
	case 1:
		return ToJS(vm, out[0].Interface())
	}
	items := make([]any, len(out))
	for i, item := range out {
		items[i] = ToJS(vm, item.Interface())
	}
	return vm.NewArray(items...)
}

/**
* GoArg: A JavaScript argument as a Go value of type tp: FromJS and then that type (a Json, a []Json, numbers of any
* kind…); undefined and null, the zero value. A value that does not fit is a TypeError
* @param vm *goja.Runtime, arg goja.Value, tp reflect.Type
* @return reflect.Value
**/
func GoArg(vm *goja.Runtime, arg goja.Value, tp reflect.Type) reflect.Value {
	if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
		return reflect.Zero(tp)
	}

	if result, ok := convertTo(FromJS(arg), tp); ok {
		return result
	}

	ptr := reflect.New(tp)
	if err := vm.ExportTo(arg, ptr.Interface()); err != nil {
		panic(vm.NewTypeError(err.Error()))
	}
	return ptr.Elem()
}

/**
* convertTo: value as a Go value of type tp when it fits: the same type or an interface it meets, a map to a map type
* (Json ↔ map[string]any), a number to another kind of number, a list element by element
* @param value any, tp reflect.Type
* @return reflect.Value, bool
**/
func convertTo(value any, tp reflect.Type) (reflect.Value, bool) {
	if value == nil {
		return reflect.Zero(tp), true
	}

	rv := reflect.ValueOf(value)
	if rv.Type().AssignableTo(tp) {
		return rv, true
	}

	switch {
	case rv.Kind() == reflect.Map && tp.Kind() == reflect.Map && rv.Type().ConvertibleTo(tp):
		return rv.Convert(tp), true
	case isNumber(rv.Kind()) && isNumber(tp.Kind()):
		return rv.Convert(tp), true
	case rv.Kind() == reflect.Slice && tp.Kind() == reflect.Slice:
		result := reflect.MakeSlice(tp, rv.Len(), rv.Len())
		for i := range rv.Len() {
			item, ok := convertTo(rv.Index(i).Interface(), tp.Elem())
			if !ok {
				return reflect.Value{}, false
			}
			result.Index(i).Set(item)
		}
		return result, true
	}

	return reflect.Value{}, false
}

/**
* isNumber: Whether a kind is an integer or a float
* @param kind reflect.Kind
* @return bool
**/
func isNumber(kind reflect.Kind) bool {
	return kind >= reflect.Int && kind <= reflect.Float64
}

// goObject: a Go pointer to a struct seen from JavaScript: its exported methods and fields by their Go name
type goObject struct {
	vm    *goja.Runtime
	value reflect.Value
}

func (o *goObject) Get(key string) goja.Value {
	if method := o.value.MethodByName(key); method.IsValid() {
		return o.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return callGo(o.vm, method, call.Arguments)
		})
	}

	field := o.value.Elem().FieldByName(key)
	if !field.IsValid() || !field.CanInterface() {
		return goja.Undefined()
	}
	return ToJS(o.vm, field.Interface())
}

// Set: only an exported field, with a value of its type
func (o *goObject) Set(key string, val goja.Value) bool {
	field := o.value.Elem().FieldByName(key)
	if !field.IsValid() || !field.CanSet() {
		return false
	}
	field.Set(GoArg(o.vm, val, field.Type()))
	return true
}

func (o *goObject) Has(key string) bool {
	if o.value.MethodByName(key).IsValid() {
		return true
	}
	field := o.value.Elem().FieldByName(key)
	return field.IsValid() && field.CanInterface()
}

func (o *goObject) Delete(key string) bool {
	return false
}

// Keys: the exported fields that are data (JSON.stringify, console.log); not the ones that point to other objects
// (db.Models → model.db…), which would never end
func (o *goObject) Keys() []string {
	tp := o.value.Elem().Type()
	result := make([]string, 0, tp.NumField())
	for i := range tp.NumField() {
		field := tp.Field(i)
		if field.IsExported() && isData(field.Type) {
			result = append(result, field.Name)
		}
	}
	return result
}

/**
* isData: Whether a type holds data and not references to other objects (pointers, interfaces, functions, channels)
* @param tp reflect.Type
* @return bool
**/
func isData(tp reflect.Type) bool {
	if tp == reflect.TypeFor[time.Time]() {
		return true
	}
	switch tp.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return false
	case reflect.Map, reflect.Slice, reflect.Array:
		return isData(tp.Elem()) || tp.Elem().Kind() == reflect.Interface
	case reflect.Struct:
		for i := range tp.NumField() {
			if tp.Field(i).IsExported() && !isData(tp.Field(i).Type) {
				return false
			}
		}
	}
	return true
}

/**
* EvaluateObjectWrapper: EvaluateObject as a script (goja, jrex/jwf) sees it: evaluateObject(obj, where) is true when
* obj meets the conditions. where is a list of conditions as JSON, as in a jsql where; an invalid condition comes back
* as a Go error, which goja throws as a JavaScript Error:
*
*	evaluateObject(params, [{status: {eq: "active"}}, {and: {age: {more: 18}}}, {or: {role: {eq: "admin"}}}])
*	evaluateObject(params, [{age: {more_eq: field("min_age")}}, {"extract_year(created_at)": {eq: 2026}}])
*
* @return func(obj Json, where []Json) (bool, error)
**/
func EvaluateObjectWrapper() func(obj Json, where []Json) (bool, error) {
	return func(obj Json, where []Json) (bool, error) {
		conditions, err := ToConditions(where)
		if err != nil {
			return false, err
		}
		return EvaluateObject(obj, conditions), nil
	}
}

/**
* EvaluateValueWrapper: EvaluateValue as a script (goja, jrex/jwf) sees it: evaluateValue(value, where, obj?) is true
* when value meets the conditions. where is a list of conditions as JSON, as in a jsql where; their field name is not
* used. A field reference ({field: "min_age"}, field("min_age"), sum("items")…) is read from obj; without obj it does
* not hold. An invalid condition is thrown as an Error:
*
*	evaluateValue(age, [{value: {more_eq: 18}}, {and: {value: {less: 65}}}])
*	evaluateValue(age, [{value: {more_eq: field("min_age")}}], params)
*
* @return func(value any, where []Json, obj ...Json) (bool, error)
**/
func EvaluateValueWrapper() func(value any, where []Json, obj ...Json) (bool, error) {
	return func(value any, where []Json, obj ...Json) (bool, error) {
		conditions, err := ToConditions(where)
		if err != nil {
			return false, err
		}
		if len(obj) > 0 && obj[0] != nil {
			for i, cond := range conditions {
				conditions[i] = cond.resolve(obj[0])
			}
		}
		return EvaluateValue(value, conditions), nil
	}
}

/**
* FieldWrapper: The field references of a script (goja, jrex/jwf), to set as globals: field(name) and the aggregates
* count, sum, avg, min, max, year, month, day, hour, minute, second, intValue and floatValue. Each gives the JSON of a
* reference, {field: "sum(items)"}, so it can go in a condition, be stored and be printed:
*
*	evaluateObject(params, [{age: {more_eq: field("min_age")}}, {and: {total: {eq: sum("items->price")}}}])
*
* @return map[string]any
**/
func FieldWrapper() map[string]any {
	ref := func(fn Function) func(name string) map[string]any {
		return func(name string) map[string]any {
			return map[string]any{FIELD.Str(): fmt.Sprintf("%s(%s)", fn.Str(), strings.TrimSpace(name))}
		}
	}
	return map[string]any{
		"field": func(name string) map[string]any {
			return map[string]any{FIELD.Str(): strings.TrimSpace(name)}
		},
		"count":      ref(COUNT),
		"sum":        ref(SUM),
		"avg":        ref(AVG),
		"min":        ref(MIN),
		"max":        ref(MAX),
		"year":       ref(EXTRACT_YEAR),
		"month":      ref(EXTRACT_MONTH),
		"day":        ref(EXTRACT_DAY),
		"hour":       ref(EXTRACT_HOUR),
		"minute":     ref(EXTRACT_MINUTE),
		"second":     ref(EXTRACT_SECOND),
		"intValue":   ref(INT_VALUE),
		"floatValue": ref(FLOAT_VALUE),
	}
}
