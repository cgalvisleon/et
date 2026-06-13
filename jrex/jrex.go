package jrex

import (
	"errors"
	"fmt"
	"maps"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
	"github.com/dop251/goja"
)

var (
	packageName = "jrex"
)

type Jrex struct {
	ID       string             `json:"id"`
	Tag      string             `json:"tag"`
	Ctx      et.Json            `json:"ctx"`
	Modules  map[string]*Module `json:"modules"`
	store    Store              `json:"-"`
	bindings map[string]any     `json:"-"`
	vm       *goja.Runtime      `json:"-"`
}

/**
* New
* @param name string, store Store
* @return *Jrex
**/
func New(tag string, store Store) (*Jrex, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, errors.New(MSG_TAG_REQUIRED)
	}

	if store == nil {
		var err error
		store, err = NewFileStore("./src")
		if err != nil {
			return nil, err
		}
	}

	tag = utility.Normalize(tag)
	id := fmt.Sprintf("jrex:%s", tag)
	result := &Jrex{
		ID:       id,
		Tag:      tag,
		Ctx:      et.Json{},
		Modules:  make(map[string]*Module),
		bindings: make(map[string]any),
		store:    store,
	}

	return result, nil
}

/**
* Load
* @param tag string, store Store
* @return *Jrex, error
**/
func Load(tag string, store Store) (*Jrex, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, errors.New(MSG_TAG_REQUIRED)
	}

	if store == nil {
		var err error
		store, err = NewFileStore("./src")
		if err != nil {
			return nil, err
		}
	}

	result, err := store.Load(tag)
	if err != nil {
		return nil, err
	}
	result.up(store)
	return result, nil
}

/**
* up
* @return *Jrex
**/
func (s *Jrex) up(store Store) *Jrex {
	s.store = store
	s.bindings = make(map[string]any)
	return s
}

/**
* save
* @return error
**/
func (s *Jrex) Save(userId string) error {
	return nil
}

/**
* Value
* @param value interface{}
* @return goja.Value
**/
func (s *Jrex) Value(value interface{}) goja.Value {
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
func (s *Jrex) Error(err error) *goja.Object {
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
func (s *Jrex) Get(name string) goja.Value {
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
func (s *Jrex) GetJson(name string) et.Json {
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
* Set
* @params name string, value interface{}
* @return error
**/
func (s *Jrex) Set(name string, value interface{}) *Jrex {
	s.bindings[name] = value
	return s
}

/**
* SetCtx
* @params ctx et.Json
**/
func (s *Jrex) SetCtx(ctx et.Json) *Jrex {
	maps.Copy(s.Ctx, ctx)
	return s
}

/**
* Run
* @return et.Json, error
**/
func (s *Jrex) require(module string) (et.Json, error) {
	code, err := s.store.GetCode(module)
	if err != nil {
		return nil, err
	}

	_, err = s.vm.RunScript(module, code)
	if err != nil {
		return nil, err
	}

	return s.Ctx, nil
}

/**
* up
* @return *Jrex
**/
func (s *Jrex) init() *Jrex {
	s.vm = goja.New()
	wrap(s)
	for name, value := range s.bindings {
		s.vm.Set(name, value)
	}
	return s
}

/**
* Run
* @return et.Json, error
**/
func (s *Jrex) Run() (et.Json, error) {
	s.init()
	_, err := s.vm.RunString(requireRuntime)
	if err != nil {
		return nil, err
	}

	return s.require("index")
}

/**
* RunByCode
* @params code string
* @return et.Json, error
**/
func (s *Jrex) RunByCode(code string) (et.Json, error) {
	_, err := s.vm.RunString(code)
	if err != nil {
		return nil, err
	}

	return s.Ctx, nil
}

/**
* RunByBt
* @params code []byte
* @return et.Json, error
**/
func (s *Jrex) RunByBt(code []byte) (et.Json, error) {
	return s.RunByCode(string(code))
}

/**
* Notify
* @param channel string, message string
**/
func (s *Jrex) Notify(channel string, message string) {

}
