package jrex

import (
	"errors"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

const (
	packageName = "jrex"
	storeJrex   = "jrex"
	storeCode   = "code"
)

type Jrex struct {
	ID        string                                  `json:"id"`
	Tag       string                                  `json:"tag"`
	Modules   map[string]*Module                      `json:"modules"`
	AuditLog  []et.Json                               `json:"audit_log"`
	isChanged bool                                    `json:"-"`
	isDebug   bool                                    `json:"-"`
	bindings  map[string]any                          `json:"-"`
	store     Store                                   `json:"-"`
	onSave    []func(jrex *Jrex, userId string) error `json:"-"`
	onDelete  []func(jrex *Jrex, userId string) error `json:"-"`
}

/**
* newJrex: Creates a new Jrex
* @param tag, userId string
* @return *Jrex, error
**/
func newJrex(tag, userId string) (*Jrex, error) {
	id := reg.UUID()
	result := &Jrex{
		ID:       id,
		Tag:      tag,
		Modules:  make(map[string]*Module),
		AuditLog: make([]et.Json, 0),
		bindings: make(map[string]any),
		onSave:   make([]func(jrex *Jrex, userId string) error, 0),
		onDelete: make([]func(jrex *Jrex, userId string) error, 0),
	}
	_, err := result.NewModule("index", userId)
	if err != nil {
		return nil, err
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
		store, err = NewStore("./src")
		if err != nil {
			return nil, err
		}
	}

	var result *Jrex
	exists, err := store.Get(storeJrex, tag, &result)
	if err != nil {
		return nil, err
	}

	if exists {
		result.Up(store)
		return result, nil
	}

	result, err = newJrex(tag, "")
	if err != nil {
		return nil, err
	}

	err = result.Save("")
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
* OnSave
* @param onSave func(jrex *Jrex) error
* @return *Jrex
**/
func (s *Jrex) OnSave(fn func(jrex *Jrex, userId string) error) *Jrex {
	if s.onSave == nil {
		s.onSave = make([]func(jrex *Jrex, userId string) error, 0)
	}
	s.onSave = append(s.onSave, fn)
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

	s.isChanged = false
	data := s.ToJson()

	if s.isDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	err := s.store.Set(storeJrex, s.ID, s.ID, s)
	if err != nil {
		return err
	}

	for _, onSave := range s.onSave {
		err := onSave(s, userId)
		if err != nil {
			return err
		}
	}

	return nil
}

/**
* up
* @return *Jrex
**/
func (s *Jrex) Up(store Store) *Jrex {
	s.store = store
	s.bindings = make(map[string]any)
	for _, module := range s.Modules {
		module.up(s)
	}
	return s
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *Jrex) addAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
}

/**
* AddModule
* @param module *Module
* @return *Jrex
**/
func (s *Jrex) addModule(module *Module, userId string) *Jrex {
	module.up(s)
	s.Modules[module.Tag] = module
	s.addAuditLog(userId, "add_module")
	return s
}

/**
* Set
* @params name string, value interface{}
* @return error
**/
func (s *Jrex) Set(name string, value interface{}) *Jrex {
	if s.bindings == nil {
		s.bindings = make(map[string]any)
	}
	s.bindings[name] = value
	return s
}

/**
* newInstance
* @return *Instance, error
**/
func (s *Jrex) NewInstance(module string) (*Instance, error) {
	instance := NewInstance()
	wrapper(instance)
	for name, value := range s.bindings {
		instance.Set(name, value)
	}

	instance.SetCode(requireRuntime)
	_, err := instance.Run()
	if err != nil {
		return nil, err
	}

	return instance, nil
}

/**
* RunString
* @param code string
* @return goja.Value, error
**/
func (s *Jrex) RunInstance(instance *Instance) (et.Json, error) {
	return instance.Run()
}

/**
* RunModule: Runs the module
* @param module string
* @return et.Json, error
**/
func (s *Jrex) RunModule(module string) (et.Json, error) {
	instance, err := s.NewInstance(module)
	if err != nil {
		return et.Json{}, err
	}

	return s.RunInstance(instance)
}

/**
* Run: Runs the Jrex
* @return et.Json, error
**/
func (s *Jrex) Run() (et.Json, error) {
	return s.RunModule("index")
}

/**
* Notify: Notifies the Jrex
* @param kind string, message string
**/
func (s *Jrex) Notify(kind, message string) {
	logs.Log(kind, message)
}

/**
* RunDev: Runs the Jrex in development mode
* @return error
**/
func (s *Jrex) RunDev() error {
	result, err := s.Run()
	if err != nil {
		s.Notify("ERROR", err.Error())
	} else {
		s.Notify("CTX", result.ToString())
	}

	utility.AppWait()
	return nil
}
