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
		Ctx:     make(et.Json),
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

/**
* SetCtx
* Set a context variable
**/
func (v *Vm) SetCtx(key string, value interface{}) {
	v.Ctx[key] = value
}

/**
* AddCtx
* Add a context variable
**/
func (v *Vm) AddCtx(key string, value interface{}) {
	val := v.Ctx.Array(key)
	val = append(val, value)
	v.Ctx[key] = val
}

/**
* GetCtx
* Get a context variable
**/
func (v *Vm) GetCtx(key string) interface{} {
	return v.Ctx[key]
}
