package vm

import (
	"github.com/cgalvisleon/et/et"
	"github.com/dop251/goja"
)

type Vm struct {
	*goja.Runtime
	Ctx et.Json
}

/**
* New
* Create a new vm
**/
func New() *Vm {
	result := &Vm{
		Runtime: goja.New(),
		Ctx:     et.Json{},
	}

	Console(result)
	Fetch(result)
	Event(result)
	Cache(result)
	return result
}

/**
* Run
* Run a script
**/
func (v *Vm) Run(script string) (goja.Value, error) {
	if script == "" {
		return nil, nil
	}

	return v.RunString(script)
}
