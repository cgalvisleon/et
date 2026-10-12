package jwf

import (
	"reflect"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/dop251/goja"
)

/**
* GoValue: A Go value (a *jsql.DB, a struct with methods, a map of them) as a binding whose methods and fields keep
* their Go names, but whose results come back as JavaScript: an et.Json, et.Item or et.Items (and any map or list of
* data) is a plain object (item.name), a struct with methods is again a Go object (db.GetModel(…).Where(…)), and a
* returned error is thrown (try/catch, e.message)
*
*	const users = db.GetModel("tenancy", "users").Select("id", "name").All() // {ok, count, result: [...]}
*	users.result.forEach((user) => console.log(user.name))
*
* @param value any
* @return Binder
**/
func GoValue(value any) Binder {
	return func(vm *goja.Runtime) goja.Value {
		return goValueOf(vm, value)
	}
}

var (
	errorType = reflect.TypeOf((*error)(nil)).Elem()
	timeType  = reflect.TypeOf(time.Time{})
)

/**
* goValueOf: value as JavaScript: data (et.Json, et.Item, et.Items, maps, lists) as plain values, structs with methods
* as a goObject, functions with their results converted the same way
* @param vm *goja.Runtime, value any
* @return goja.Value
**/
func goValueOf(vm *goja.Runtime, value any) goja.Value {
	switch v := value.(type) {
	case nil:
		return goja.Null()
	case goja.Value:
		return v
	case et.Item:
		return goValueOf(vm, v.ToJson())
	case *et.Item:
		return goValueOf(vm, v.ToJson())
	case et.Items:
		return goValueOf(vm, v.ToJson())
	case *et.Items:
		return goValueOf(vm, v.ToJson())
	case time.Time, []byte:
		return vm.ToValue(v)
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map:
		if rv.IsNil() {
			return goja.Null()
		}
		obj := vm.NewObject()
		iter := rv.MapRange()
		for iter.Next() {
			obj.Set(iter.Key().String(), goValueOf(vm, iter.Value().Interface()))
		}
		return obj
	case reflect.Slice, reflect.Array:
		if rv.Kind() == reflect.Slice && rv.IsNil() {
			return vm.NewArray()
		}
		items := make([]any, rv.Len())
		for i := range items {
			items[i] = goValueOf(vm, rv.Index(i).Interface())
		}
		return vm.NewArray(items...)
	case reflect.Func:
		if rv.IsNil() {
			return goja.Null()
		}
		return vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return callOf(vm, rv, call.Arguments)
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
* callOf: Calls a Go function with JavaScript arguments: each one goes to its Go type (a goObject gives back its Go
* value), a returned error is thrown and the rest come back with goValueOf (several results, a list)
* @param vm *goja.Runtime, fn reflect.Value, args []goja.Value
* @return goja.Value
**/
func callOf(vm *goja.Runtime, fn reflect.Value, args []goja.Value) goja.Value {
	tp := fn.Type()
	in := make([]reflect.Value, 0, len(args))
	for i := 0; i < tp.NumIn(); i++ {
		if tp.IsVariadic() && i == tp.NumIn()-1 {
			elem := tp.In(i).Elem()
			for j := i; j < len(args); j++ {
				in = append(in, argOf(vm, args[j], elem))
			}
			break
		}

		arg := goja.Undefined()
		if i < len(args) {
			arg = args[i]
		}
		in = append(in, argOf(vm, arg, tp.In(i)))
	}

	out := fn.Call(in)
	if n := len(out); n > 0 && tp.Out(n-1) == errorType {
		if err, ok := out[n-1].Interface().(error); ok && err != nil {
			panic(vm.NewGoError(err))
		}
		out = out[:n-1]
	}

	switch len(out) {
	case 0:
		return goja.Undefined()
	case 1:
		return goValueOf(vm, out[0].Interface())
	}
	items := make([]any, len(out))
	for i, item := range out {
		items[i] = goValueOf(vm, item.Interface())
	}
	return vm.NewArray(items...)
}

/**
* argOf: A JavaScript argument as a Go value of type tp: a goObject gives its Go value, params and ctx their et.Json;
* undefined or null, the zero value. A value that does not fit is a TypeError
* @param vm *goja.Runtime, arg goja.Value, tp reflect.Type
* @return reflect.Value
**/
func argOf(vm *goja.Runtime, arg goja.Value, tp reflect.Type) reflect.Value {
	if arg == nil || goja.IsUndefined(arg) || goja.IsNull(arg) {
		return reflect.Zero(tp)
	}

	if obj, ok := arg.(*goja.Object); ok {
		var native any
		switch v := obj.Export().(type) {
		case *goObject:
			native = v.value.Interface()
		case *jsonObject:
			native = v.data()
		}
		if native != nil {
			value := reflect.ValueOf(native)
			if value.Type().AssignableTo(tp) {
				return value
			}
			if value.Type().ConvertibleTo(tp) {
				return value.Convert(tp)
			}
			panic(vm.NewTypeError("cannot use %s as %s", value.Type(), tp))
		}
	}

	ptr := reflect.New(tp)
	if err := vm.ExportTo(arg, ptr.Interface()); err != nil {
		panic(vm.NewTypeError(err.Error()))
	}
	return ptr.Elem()
}

// goObject: a Go pointer to a struct seen from JavaScript: its exported methods and fields by their Go name
type goObject struct {
	vm    *goja.Runtime
	value reflect.Value
}

func (o *goObject) Get(key string) goja.Value {
	if method := o.value.MethodByName(key); method.IsValid() {
		return o.vm.ToValue(func(call goja.FunctionCall) goja.Value {
			return callOf(o.vm, method, call.Arguments)
		})
	}

	field := o.value.Elem().FieldByName(key)
	if !field.IsValid() || !field.CanInterface() {
		return goja.Undefined()
	}
	return goValueOf(o.vm, field.Interface())
}

// Set: only an exported field, with a value of its type
func (o *goObject) Set(key string, val goja.Value) bool {
	field := o.value.Elem().FieldByName(key)
	if !field.IsValid() || !field.CanSet() {
		return false
	}
	field.Set(argOf(o.vm, val, field.Type()))
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
	for i := 0; i < tp.NumField(); i++ {
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
	if tp == timeType {
		return true
	}
	switch tp.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return false
	case reflect.Map, reflect.Slice, reflect.Array:
		return isData(tp.Elem()) || tp.Elem().Kind() == reflect.Interface
	case reflect.Struct:
		for i := 0; i < tp.NumField(); i++ {
			if tp.Field(i).IsExported() && !isData(tp.Field(i).Type) {
				return false
			}
		}
	}
	return true
}
