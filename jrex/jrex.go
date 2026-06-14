package jrex

import (
	"errors"
	"fmt"
	"maps"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
	"github.com/dop251/goja"
)

const (
	packageName = "jrex"
)

type Jrex struct {
	TenantId  string             `json:"tenant_id"`
	ID        string             `json:"id"`
	Tag       string             `json:"tag"`
	Ctx       et.Json            `json:"ctx"`
	Modules   map[string]*Module `json:"modules"`
	AuditLog  []et.Json          `json:"audit_log"`
	store     Store              `json:"-"`
	bindings  map[string]any     `json:"-"`
	baseDir   string             `json:"-"`
	userId    string             `json:"-"`
	isChanged bool               `json:"-"`
	isDebug   bool               `json:"-"`
	vm        *goja.Runtime      `json:"-"`
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
		store, err = NewStore("./src")
		if err != nil {
			return nil, err
		}
	}

	result, err := store.Load(tag)
	if err != nil {
		return nil, err
	}
	result.Up(store)
	return result, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Jrex) ToJson() et.Json {
	return et.Json{
		"id":        s.ID,
		"tag":       s.Tag,
		"ctx":       s.Ctx,
		"modules":   s.Modules,
		"audit_log": s.AuditLog,
	}
}

/**
* ToString
* @return string
**/
func (s *Jrex) ToString() string {
	return s.ToJson().ToString()
}

/**
* Debug
* @return *Jrex
**/
func (s *Jrex) Debug() *Jrex {
	s.isDebug = true
	return s
}

/**
* save
* @return error
**/
func (s *Jrex) Save(userId string) error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     "save",
	})
	maxAuditLog := config.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}

	s.isChanged = false
	data := s.ToJson()

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	channel := fmt.Sprintf("%s:%s", EVENT_JREX_SET, s.TenantId)
	event.Publish(channel, data)

	return s.store.Save(s, userId)
}

/**
* up
* @return *Jrex
**/
func (s *Jrex) Up(store Store) *Jrex {
	s.store = store
	s.bindings = make(map[string]any)
	return s
}

/**
* AddModule
* @param module *Module
* @return *Jrex
**/
func (s *Jrex) AddModule(module *Module) *Jrex {
	s.Modules[module.Path] = module
	s.isChanged = true
	return s
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
* Run: Runs the Jrex
* @return et.Json, error
**/
func (s *Jrex) Run() (et.Json, error) {
	s.vm = goja.New()
	wrap(s)
	for name, value := range s.bindings {
		s.vm.Set(name, value)
	}

	_, err := s.vm.RunString(requireRuntime)
	if err != nil {
		return nil, err
	}

	s.baseDir = ""
	module := "index"
	code, err := s.store.GetCode(module)
	if err != nil {
		return nil, err
	}

	_, err = s.vm.RunScript(module, code)
	if err != nil {
		return nil, err
	}

	if s.isChanged {
		s.Save(s.userId)
	}

	return s.Ctx, nil
}

/**
* RunDev: Runs the Jrex in development mode
* @param userId string
* @return error
**/
func (s *Jrex) RunDev(userId string) error {
	s.userId = userId
	result, err := s.Run()
	if err != nil {
		return err
	}

	logs.Log("CTX", result.ToString())
	utility.AppWait()
	return nil
}
