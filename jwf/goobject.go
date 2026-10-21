package jwf

import (
	"github.com/cgalvisleon/et/et"
	"github.com/dop251/goja"
)

/**
* GoValue: A Go value (a *jsql.DB, a struct with methods, a map of them) as a binding, converted with et.ToJS: methods
* and fields keep their Go names, an et.Json, et.Item or et.Items (and any map or list of data) is an object over its
* map with the methods of et.Json, and a returned error is thrown (try/catch, e.message)
*
*	const users = db.GetModel("tenancy", "users").Select("id", "name").All() // {ok, count, result: [...]}
*	users.result.forEach((user) => console.log(user.name, user.Str("email")))
*
* @param value any
* @return Binder
**/
func GoValue(value any) Binder {
	return func(vm *goja.Runtime) goja.Value {
		return et.ToJS(vm, value)
	}
}
