package stores

import (
	"fmt"

	"github.com/cgalvisleon/et/dt"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	. "github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
)

type Config struct {
	TenantId string
	Stage    string
	Tag      string
	model    *Model
}

/**
* defineConfig
* @param db *DB, tenantId, schema string
* @return (*Config, error)
**/
func defineConfig(db *DB, tenantId, schema, stage, tag string) (*Config, error) {
	columns := []Column{
		{Name: CREATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: UPDATED_AT, TypeColumn: COLUMN, TypeData: DATETIME, Default: ""},
		{Name: TENANT_ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: ID, TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "tag", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "stage", TypeColumn: COLUMN, TypeData: KEY, Default: ""},
		{Name: "title", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: "definition", TypeColumn: COLUMN, TypeData: TEXT, Default: ""},
		{Name: "params", TypeColumn: COLUMN, TypeData: JSON, Default: et.Json{}},
	}

	def := Def{
		Schema:  schema,
		Name:    "configs",
		Version: 1,
		Columns: columns,
		PrimaryKeys: []DefIndex{
			{Name: TENANT_ID, Sorted: true},
			{Name: "stage", Sorted: true},
			{Name: "tag", Sorted: true},
		},
		Unique: []DefIndex{
			{Name: ID, Sorted: true},
		},
		IdxField: IDX,
		IdtField: IDT,
	}

	model, err := db.Define(def)
	if err != nil {
		return nil, err
	}
	model.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		now := timezone.Now()
		new.Set(CREATED_AT, now)
		new.Set(UPDATED_AT, now)
		new.Set(ID, reg.UUID())
		return nil
	})
	model.BeforeUpdate(func(tx *Tx, old, new et.Json) error {
		now := timezone.Now()
		new.Set(UPDATED_AT, now)
		return nil
	})
	err = model.Init()
	if err != nil {
		return nil, err
	}

	result := &Config{
		TenantId: tenantId,
		Stage:    stage,
		Tag:      tag,
		model:    model,
	}
	result.initEvent()

	return result, nil
}

/**
* LoadConfig
* @param db *DB, tenantId, schema, stage, tag string
* @return (*Config, error)
**/
func LoadConfig(db *DB, tenantId, schema, stage, tag string) (*Config, error) {
	config, err := defineConfig(db, tenantId, schema, stage, tag)
	if err != nil {
		return nil, err
	}

	return config, nil
}

/**
* SetConfig
* @param tag, stage string, params et.Json
* @return error
**/
func (s *Config) SetConfig(tag, stage string, params et.Json) error {
	_, err := s.model.
		Upsert(et.Json{
			"tenant_id": s.TenantId,
			"stage":     stage,
			"tag":       tag,
			"params":    params,
		}).
		Where(Eq(TENANT_ID, s.TenantId)).
		And(Eq("stage", stage)).
		And(Eq("tag", tag)).
		One()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("config:%s:%s:%s", s.TenantId, s.Stage, s.Tag)
	dt.Drop(key)

	channel := fmt.Sprintf("%s:%s", EVENT_SET_CONFIG, key)
	event.Publish(channel, params)

	return nil
}

/**
* deleteConfig
* @param stage, tag string
* @return error
**/
func (s *Config) deleteConfig(stage, tag string) error {
	_, err := s.model.
		Delete().
		Where(Eq(TENANT_ID, s.TenantId)).
		And(Eq("stage", stage)).
		And(Eq("tag", tag)).
		Exec()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("config:%s:%s:%s", s.TenantId, stage, tag)
	dt.Drop(key)
	return nil
}

/**
* DeleteConfig
* @param tag, stage string
* @return error
**/
func (s *Config) DeleteConfig(tag, stage string) error {
	err := s.deleteConfig(stage, tag)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("config:%s:%s:%s", s.TenantId, stage, tag)
	channel := fmt.Sprintf("%s:%s", EVENT_DEL_CONFIG, key)
	event.Publish(channel, et.Json{
		"tenant_id": s.TenantId,
		"tag":       tag,
		"stage":     stage,
	})
	return nil
}

/**
* GetConfig
* @param tag, stage string
* @return (et.Json, error)
**/
func (s *Config) Get(name string, def interface{}) interface{} {
	config := et.Json{}
	key := fmt.Sprintf("config:%s:%s:%s", s.TenantId, s.Stage, s.Tag)
	item := dt.Get(key)
	if item.Ok {
		val, ok := item.Json()
		if ok {
			config = val
		}
	} else {
		item, err := s.model.
			Where(Eq(TENANT_ID, s.TenantId)).
			And(Eq("stage", s.Stage)).
			And(Eq("tag", s.Tag)).
			One()
		if err != nil {
			return def
		}

		if !item.Ok {
			return def
		}

		dt.Up(key, item.Result)
		config = item.Result
	}

	return config.ValAny(def, name)
}

/**
* initEvent
* @return error
**/
func (s *Config) initEvent() {
	key := fmt.Sprintf("config:%s:%s:%s", s.TenantId, s.Stage, s.Tag)
	channel := fmt.Sprintf("%s:%s", EVENT_SET_CONFIG, key)
	err := event.Stack(channel, s.eventActionSetConfig)
	if err != nil {
		logs.Error(err)
	}

	channel = fmt.Sprintf("%s:%s", EVENT_DEL_CONFIG, key)
	err = event.Stack(channel, s.eventActionDelConfig)
	if err != nil {
		logs.Error(err)
	}
}

/**
* eventActionSetConfig
* @param m event.Message
**/
func (s *Config) eventActionSetConfig(m event.Message) {
	data := m.Data
	stage := data.Str("stage")
	tag := data.Str("tag")
	key := fmt.Sprintf("config:%s:%s:%s", s.TenantId, stage, tag)
	dt.Drop(key)
}

/**
* eventActionDelConfig
* @param m event.Message
**/
func (s *Config) eventActionDelConfig(m event.Message) {
	data := m.Data
	stage := data.Str("stage")
	tag := data.Str("tag")
	err := s.deleteConfig(stage, tag)
	if err != nil {
		logs.Error(err)
	}
}
