package jrex

import (
	"fmt"
	"maps"

	"github.com/cgalvisleon/et/et"
	"github.com/dop251/goja"
)

type Instance struct {
	Module  string        `json:"module"`
	Ctx     et.Json       `json:"ctx"`
	store   Store         `json:"-"`
	jrex    *Jrex         `json:"-"`
	baseDir string        `json:"-"`
	vm      *goja.Runtime `json:"-"`
}

func newInstance(jrex *Jrex, module string) *Instance {
	return &Instance{
		Module: module,
		Ctx:    jrex.Ctx.Clone(),
		store:  jrex.store,
		jrex:   jrex,
		vm:     goja.New(),
	}
}

/**
* Set
* @param name string, value interface{}
* @return *Instance
**/
func (s *Instance) Set(name string, value interface{}) *Instance {
	s.vm.Set(name, value)
	return s
}

/**
* SetCtx
* @param ctx et.Json
* @return *Instance
**/
func (s *Instance) SetCtx(ctx et.Json) *Instance {
	maps.Copy(s.Ctx, ctx)
	return s
}

/**
* RunString
* @param code string
* @return goja.Value, error
**/
func (s *Instance) RunString(code string) (goja.Value, error) {
	return s.vm.RunString(code)
}

/**
* RunScript
* @param module string, code string
* @return goja.Value, error
**/
func (s *Instance) RunScript(module string, code string) (goja.Value, error) {
	return s.vm.RunScript(module, code)
}

/**
* Run
* @return et.Json, error
**/
func (s *Instance) Run() (et.Json, error) {
	code, err := s.store.GetCode(s.Module)
	if err != nil {
		return et.Json{}, err
	}

	_, err = s.RunScript(s.Module, code)
	if err != nil {
		return et.Json{}, err
	}

	if s.jrex.isChanged {
		s.jrex.Save(s.jrex.userId)
	}

	return s.Ctx, nil
}

/**
* Value
* @param value interface{}
* @return goja.Value
**/
func (s *Instance) Value(value interface{}) goja.Value {
	if s.vm == nil {
		return goja.Undefined()
	}
	return s.vm.ToValue(value)
}

/**
* Error
* @param err error
* @return *goja.Object
**/
func (s *Instance) Error(err error) *goja.Object {
	if s.vm == nil {
		return nil
	}
	return s.vm.NewGoError(err)
}

/**
* Get
* @param name string
* @return goja.Value
**/
func (s *Instance) Get(name string) goja.Value {
	if s.vm == nil {
		return goja.Undefined()
	}
	return s.vm.Get(name)
}

/**
* GetJson
* @param name string
* @return et.Json
**/
func (s *Instance) GetJson(name string) et.Json {
	if s.vm == nil {
		return et.Json{}
	}
	result, ok := s.vm.Get(name).Export().(et.Json)
	if !ok {
		return et.Json{}
	}
	return result
}

/**
* wrapperRunTime: Wraps the runtime
* @param vm *VM
**/
func (s *Instance) wrapperModules(module *Module) {
	module.Set("version", func(value string) string {
		part, ok := ToPart(value)
		if !ok {
			panic(s.Error(fmt.Errorf("invalid part: %s", value)))
		}
		module.SetVersion(part)
		return module.Version
	})
	module.Set("metadata", func(value et.Json) et.Json {
		module.SetMetadata(value)
		return value
	})
}
