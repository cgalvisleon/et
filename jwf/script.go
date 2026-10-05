package jwf

import (
	"github.com/cgalvisleon/et/et"
	"github.com/dop251/goja"
)

type Script struct {
	Code        string         `json:"code"`
	Language    string         `json:"language"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Version     int            `json:"version"`
	bindings    map[string]any `json:"-"`
	vm          *goja.Runtime  `json:"-"`
}

func (s *Script) SetBinding(name string, value any) *Script {
	if s.bindings == nil {
		s.bindings = make(map[string]any)
	}
	s.bindings[name] = value
	return s
}

/**
* RunScript
* @param code string, ctx et.Json, bindings map[string]any
* @return any, error
**/
func RunScript(code string, ctx et.Json, bindings map[string]any) (any, error) {
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
