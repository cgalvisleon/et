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
	ID      string             `json:"id"`
	Tag     string             `json:"tag"`
	Ctx     et.Json            `json:"ctx"`
	Modules map[string]*Module `json:"modules"`
	store   Store              `json:"-"`
	vm      *goja.Runtime      `json:"-"`
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
		ID:      id,
		Tag:     tag,
		Ctx:     et.Json{},
		Modules: make(map[string]*Module),
		store:   store,
	}

	return result, nil
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
	return s.vm.ToValue(value)
}

/**
* Error
* @param err error
* @return *goja.Object
**/
func (s *Jrex) Error(err error) *goja.Object {
	return s.vm.NewGoError(err)
}

/**
* Get
* @param name string
* @return goja.Value
**/
func (s *Jrex) Get(name string) goja.Value {
	return s.vm.Get(name)
}

/**
* GetJson
* @param name string
* @return et.Json
**/
func (s *Jrex) GetJson(name string) et.Json {
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
func (s *Jrex) Set(name string, value interface{}) error {
	return s.vm.Set(name, value)
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
func (s *Jrex) Require(name string) (et.Json, error) {
	module, exists := s.Modules[name]
	if !exists {
		return nil, errors.New(MSG_INDEX_MODULE_NOT_FOUND)
	}

	_, err := s.vm.RunScript(module.Name, module.Code)
	if err != nil {
		return nil, err
	}

	return s.Ctx, nil
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
* Run
* @return et.Json, error
**/
func (s *Jrex) Run() (et.Json, error) {
	s.vm = goja.New()
	wrap(s)

	_, err := s.vm.RunString(requireRuntime)
	if err != nil {
		return nil, err
	}

	return s.Require("index.js")
}

/**
* Notify
* @param channel string, message string
**/
func (s *Jrex) Notify(channel string, message string) {

}

/**
* Resolve
* @param modulePath string
* @return (string, error)
**/
func (s *Jrex) Resolve(modulePath string) (string, error) {
	s.Notify("LOG", fmt.Sprintf("Resolve: %s", modulePath))
	return "", nil
}
