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

/**
* RunCode
* @param code string, ctx et.Json, bindings map[string]any
* @return any, error
**/
func RunCode(code string, ctx et.Json, bindings map[string]any) (any, error) {
	if code == "" {
		return et.Json{}, nil
	}

	vm := goja.New()
	for name, value := range bindings {
		vm.Set(name, value)
	}

	result, err := vm.RunString(code)
	if err != nil {
		return et.Json{}, err
	}

	return result.Export(), nil
}

/**
* RunScripts
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
