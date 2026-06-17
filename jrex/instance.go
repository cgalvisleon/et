package jrex

import (
	"errors"
	"maps"

	"github.com/cgalvisleon/et/et"
	"github.com/dop251/goja"
)

type Instance struct {
	Module string        `json:"module"`
	Ctx    et.Json       `json:"ctx"`
	store  Store         `json:"-"`
	jrex   *Jrex         `json:"-"`
	vm     *goja.Runtime `json:"-"`
}

func NewInstance() *Instance {
	result := &Instance{
		Module: "index",
		Ctx:    et.Json{},
		store:  nil,
		jrex:   nil,
		vm:     goja.New(),
	}
	wrapper(result)
	return result
}

/**
* newInstance
* @param jrex *Jrex, module string
* @return *Instance
**/
func newInstance(jrex *Jrex, module string) *Instance {
	return &Instance{
		Module: module,
		Ctx:    et.Json{},
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
* GetCode
* @param module string
* @return string, error
**/
func (s *Instance) GetCode(module string) (string, error) {
	if s.jrex == nil {
		return "", errors.New(MSG_JREX_IS_NIL)
	}

	mod, exists := s.jrex.Modules[module]
	if !exists {
		return "", errors.New(MSG_MODULE_NOT_FOUND)
	}

	return mod.getCode()
}

/**
* Run
* @return et.Json, error
**/
func (s *Instance) Run() (et.Json, error) {
	code, err := s.GetCode("index")
	if err != nil {
		return et.Json{}, err
	}

	_, err = s.RunString(code)
	if err != nil {
		return et.Json{}, err
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
