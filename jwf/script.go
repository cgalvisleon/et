package jwf

import (
	"errors"

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
	if code == "" {
		return et.Json{}, et.Json{}, nil
	}
	return NewRunner(bindings).Run(code, nil, reads...)
}

/**
* Runner: A runtime with fixed bindings to run scripts many times; globals persist; not concurrent.
**/
type Runner struct {
	vm       *goja.Runtime
	programs map[string]*goja.Program
}

/**
* NewRunner: Creates a runtime with the bindings, set as Run sets them.
* @param bindings map[string]any
* @return *Runner
**/
func NewRunner(bindings map[string]any) *Runner {
	vm := goja.New()
	aliasJson(vm)
	for name, value := range bindings {
		if bind, ok := value.(Binder); ok {
			vm.Set(name, bind(vm))
			continue
		}
		vm.Set(name, et.ToJS(vm, value))
	}
	return &Runner{vm: vm, programs: map[string]*goja.Program{}}
}

/**
* Run: Sets the values of set as globals (with et.ToJS), runs code (compiled once per Runner) and gives back
* its last value and the globals named in reads.
* @param code string, set map[string]any, reads ...string
* @return any, et.Json, error
**/
func (s *Runner) Run(code string, set map[string]any, reads ...string) (any, et.Json, error) {
	globals := et.Json{}
	if code == "" {
		return et.Json{}, globals, nil
	}

	program, ok := s.programs[code]
	if !ok {
		var err error
		program, err = goja.Compile("", code, false)
		if err != nil {
			return et.Json{}, globals, err
		}
		s.programs[code] = program
	}

	for name, value := range set {
		s.vm.Set(name, et.ToJS(s.vm, value))
	}

	result, err := s.vm.RunProgram(program)
	if err != nil {
		return et.Json{}, globals, scriptError(err)
	}

	for _, name := range reads {
		globals[name] = et.FromJS(s.vm.Get(name))
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

/**
* scriptError: The error of a script as its message: a thrown Error (or a Go error of a binding) gives its message, a
* ReferenceError, TypeError… "name: message", any other thrown value its text; without the "at <eval>:line:col" of goja,
* which points into the scripts joined and means nothing to whoever reads it. An error that is not a thrown exception (an
* interrupt) stays as it is
* @param err error
* @return error
**/
func scriptError(err error) error {
	var exception *goja.Exception
	if !errors.As(err, &exception) {
		return err
	}

	value := exception.Value()
	if object, ok := value.(*goja.Object); ok {
		message := object.Get("message")
		if message != nil && !goja.IsUndefined(message) && !goja.IsNull(message) {
			name := ""
			if value := object.Get("name"); value != nil && !goja.IsUndefined(value) {
				name = value.String()
			}
			if name == "" || name == "Error" || name == "GoError" {
				return errors.New(message.String())
			}
			return errors.New(name + ": " + message.String())
		}
	}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return err
	}
	return errors.New(value.String())
}
