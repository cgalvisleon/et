package jrex

import (
	"errors"
	"maps"

	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

const (
	packageName = "jrex"
)

type Jrex struct {
	ID        string                 `json:"id"`
	Tag       string                 `json:"tag"`
	Ctx       et.Json                `json:"ctx"`
	Modules   map[string]*Module     `json:"modules"`
	AuditLog  []et.Json              `json:"audit_log"`
	isChanged bool                   `json:"-"`
	store     Store                  `json:"-"`
	bindings  map[string]any         `json:"-"`
	baseDir   string                 `json:"-"`
	userId    string                 `json:"-"`
	isDebug   bool                   `json:"-"`
	onSave    func(jrex *Jrex) error `json:"-"`
}

func NewJrex(tag string) (*Jrex, error) {
	if !utility.ValidStr(tag, 0, []string{""}) {
		return nil, errors.New(MSG_TAG_REQUIRED)
	}

	tag = strs.Lowcase(tag)
	id := reg.ULID()
	result := &Jrex{
		ID:       id,
		Tag:      tag,
		Ctx:      et.Json{},
		Modules:  make(map[string]*Module),
		AuditLog: make([]et.Json, 0),
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
* OnSave
* @param onSave func(jrex *Jrex) error
* @return *Jrex
**/
func (s *Jrex) OnSave(onSave func(jrex *Jrex) error) *Jrex {
	s.onSave = onSave
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

	if s.onSave != nil {
		err := s.onSave(s)
		if err != nil {
			return err
		}
	}

	return s.store.Save(s, userId)
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
	maxAuditLog := config.GetInt("MAX_AUDIT_LOG", 1000)
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
func (s *Jrex) AddModule(module *Module) *Jrex {
	module.up(s)
	s.Modules[module.Path] = module
	s.addAuditLog(s.userId, "add_module")
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
* SetCtx
* @params ctx et.Json
**/
func (s *Jrex) SetCtx(ctx et.Json) *Jrex {
	maps.Copy(s.Ctx, ctx)
	return s
}

/**
* newInstance
* @return *Instance, error
**/
func (s *Jrex) NewInstance(module string) (*Instance, error) {
	instance := newInstance(s, module)
	wrapper(instance)
	for name, value := range s.bindings {
		instance.Set(name, value)
	}

	_, err := instance.RunString(requireRuntime)
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
	s.baseDir = ""
	code, err := s.store.GetCode(instance.Module)
	if err != nil {
		return et.Json{}, err
	}

	_, err = instance.RunScript(instance.Module, code)
	if err != nil {
		return et.Json{}, err
	}

	if s.isChanged {
		s.Save(s.userId)
	}

	return s.Ctx, nil
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
