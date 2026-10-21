package jwf

import (
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/strs"
	"github.com/dop251/goja"
)

type Script struct {
	Code        string        `json:"code"`
	Language    string        `json:"language"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Version     int           `json:"version"`
	vm          *goja.Runtime `json:"-"`
}

// Binder: a binding built with the runtime of the script (a goja dynamic object, like params)
type Binder func(vm *goja.Runtime) goja.Value

/**
* Run: Runs code in a new runtime with the bindings (a Binder is built with the runtime; the rest go with et.ToJS, so an
* et.Json is an object over the same map) and gives back its last value and, after it, the globals named in reads
* (as et.FromJS sees them: a binding reassigned by the script, NEW = {...}, comes back too)
* @param code string, bindings map[string]any, reads ...string
* @return any, et.Json, error
**/
func Run(code string, bindings map[string]any, reads ...string) (any, et.Json, error) {
	globals := et.Json{}
	if code == "" {
		return et.Json{}, globals, nil
	}

	vm := goja.New()
	aliasJson(vm)
	for name, value := range bindings {
		if bind, ok := value.(Binder); ok {
			vm.Set(name, bind(vm))
			continue
		}
		vm.Set(name, et.ToJS(vm, value))
	}

	result, err := vm.RunString(code)
	if err != nil {
		return et.Json{}, globals, err
	}

	for _, name := range reads {
		globals[name] = et.FromJS(vm.Get(name))
	}

	return et.FromJS(result), globals, nil
}

/**
* RunCode: Runs code with the bindings and ctx (as the ctx binding, unless the bindings bring their own) and gives back
* its last value
* @param code string, ctx et.Json, bindings map[string]any
* @return any, error
**/
func RunCode(code string, ctx et.Json, bindings map[string]any) (any, error) {
	if _, ok := bindings["ctx"]; !ok && ctx != nil {
		with := make(map[string]any, len(bindings)+1)
		for name, value := range bindings {
			with[name] = value
		}
		with["ctx"] = ctx
		bindings = with
	}

	result, _, err := Run(code, bindings)
	return result, err
}

/**
* RunScripts: Runs the scripts together, in order, as one code (RunCode)
* @param scripts []*Script, ctx et.Json, bindings map[string]any
* @return any, error
**/
func RunScripts(scripts []*Script, ctx et.Json, bindings map[string]any) (any, error) {
	if len(scripts) == 0 {
		return et.Json{}, nil
	}

	script := ""
	for _, scr := range scripts {
		script = strs.Append(script, scr.Code, "\n")
	}

	return RunCode(script, ctx, bindings)
}
