package jrex

import (
	"errors"
	"maps"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/dop251/goja"
)

type Instance struct {
	ID    string        `json:"id"`
	Ctx   et.Json       `json:"ctx"`
	code  string        `json:"-"`
	store Store         `json:"-"`
	vm    *goja.Runtime `json:"-"`
}

func NewInstance() *Instance {
	result := &Instance{
		ID:    reg.UUID(),
		Ctx:   et.Json{},
		store: nil,
		vm:    goja.New(),
	}
	wrapper(result)
	return result
}

/**
* SetStore
* @param store Store
* @return *Instance
**/
func (s *Instance) SetStore(store Store) *Instance {
	s.store = store
	return s
}

/**
* SetCode
* @param code string
* @return *Instance
**/
func (s *Instance) SetCode(code string) *Instance {
	s.code = code
	return s
}

/**
* GetCode
* @param module string
* @return string, error
**/
func (s *Instance) GetCode() string {
	return s.code
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
* Run
* @return et.Json, error
**/
func (s *Instance) Run() (et.Json, error) {
	if s.code == "" {
		return et.Json{}, errors.New(MSG_CODE_IS_EMPTY)
	}
	_, err := s.vm.RunString(s.code)
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
	value := s.Get(name)
	result, ok := value.Export().(et.Json)
	if !ok {
		return et.Json{}
	}
	return result
}
