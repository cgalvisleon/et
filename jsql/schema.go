package jsql

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
)

/**
* Schema: Represents a database schema that owns a set of models.
**/
type Schema struct {
	TenantId string            `json:"tenant_id"`
	Database string            `json:"database"`
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"models"`
	db       *DB               `json:"-"`
	isDebug  bool              `json:"-"`
	mu       *sync.RWMutex     `json:"-"`
}

/**
* ToJson: Returns the schema metadata as an et.Json map.
* @return et.Json
**/
func (s *Schema) Ref() et.Json {
	models := []et.Json{}
	for _, model := range s.Models {
		models = append(models, model.Ref())
	}

	return et.Json{
		"name":   s.Name,
		"models": models,
	}
}

/**
* addModel: Adds a model to the schema.
* @param model *Model
* @return void
**/
func (s *Schema) addModel(model *Model) {
	s.mu.Lock()
	s.Models[model.Name] = model
	s.mu.Unlock()
}

/**
* removeModel: Removes a model from the schema.
* @param name string
* @return void
**/
func (s *Schema) removeModel(name string) {
	s.mu.Lock()
	delete(s.Models, name)
	s.mu.Unlock()
}

/**
* getModel: Returns the named model or an error if it does not exist in this schema.
* @param name string
* @return *Model, error
**/
func (s *Schema) getModel(name string) (*Model, error) {
	name = utility.Normalize(name)
	s.mu.RUnlock()
	result, exists := s.Models[name]
	s.mu.RLock()

	if !exists {
		return nil, fmt.Errorf(MSG_MODEL_NOT_FOUND, name)
	}

	return result, nil
}

/**
* newModel: Constructs a new Model with initialized fields and default triggers.
* @param schema *Schema, name string, version int, userId string
* @return *Model
**/
func (s *Schema) newModel(name string, version int, userId string) *Model {
	name = utility.Normalize(name)
	result := &Model{
		TenantId:      s.TenantId,
		ID:            reg.UUID(),
		Database:      s.Database,
		Schema:        s.Name,
		DatabaseId:    s.db.ID,
		Name:          name,
		Columns:       make([]*Column, 0),
		Indexes:       make([]*Index, 0),
		PrimaryKeys:   make([]*Index, 0),
		ForeignKeys:   make([]*Detail, 0),
		Unique:        make([]*Index, 0),
		Required:      make([]*Index, 0),
		Hiddens:       make([]string, 0),
		Details:       make(map[string]*Detail, 0),
		Rollups:       make(map[string]*Detail, 0),
		calcs:         make(map[string]CalcFunction, 0),
		Version:       version,
		BeforeInserts: make([]string, 0),
		BeforeUpdates: make([]string, 0),
		BeforeDeletes: make([]string, 0),
		AfterInserts:  make([]string, 0),
		AfterUpdates:  make([]string, 0),
		AfterDeletes:  make([]string, 0),
		AuditLog:      make([]et.Json, 0),
		beforeInserts: make([]TriggerFunction, 0),
		beforeUpdates: make([]TriggerFunction, 0),
		beforeDeletes: make([]TriggerFunction, 0),
		afterInserts:  make([]TriggerFunction, 0),
		afterUpdates:  make([]TriggerFunction, 0),
		afterDeletes:  make([]TriggerFunction, 0),
		db:            s.db,
	}
	result.addAuditLog(userId, "new_model")
	s.addModel(result)
	return result
}

/**
* loadModel: Loads a Model from the database catalog by name.
* @param store Store, id string
* @return *Model, error
**/
func (s *Schema) loadModel(store Store, id string) (*Model, error) {
	if store == nil {
		return nil, errors.New(MSG_DB_STORE_IS_NIL)
	}

	ref, err := store.Get("model", id)
	if err != nil {
		return nil, err
	}

	result := &Model{
		TenantId:      ref.Str("tenant_id"),
		ID:            ref.Str("id"),
		Database:      ref.Str("database"),
		Schema:        ref.Str("schema"),
		DatabaseId:    ref.Str("database_id"),
		Name:          ref.Str("name"),
		Columns:       make([]*Column, 0),
		SourceField:   ref.Str("source_field"),
		IdxField:      ref.Str("idx_field"),
		IdtField:      ref.Str("idt_field"),
		Indexes:       make([]*Index, 0),
		PrimaryKeys:   make([]*Index, 0),
		ForeignKeys:   make([]*Detail, 0),
		Unique:        make([]*Index, 0),
		Required:      make([]*Index, 0),
		Hiddens:       make([]string, 0),
		Details:       make(map[string]*Detail, 0),
		Rollups:       make(map[string]*Detail, 0),
		calcs:         make(map[string]CalcFunction, 0),
		IsStrict:      ref.Bool("is_strict"),
		Version:       ref.Int("version"),
		IsCore:        ref.Bool("is_core"),
		IsDebug:       s.db.isDebug,
		BeforeInserts: ref.ArrayStr("before_inserts"),
		BeforeUpdates: ref.ArrayStr("before_updates"),
		BeforeDeletes: ref.ArrayStr("before_deletes"),
		AfterInserts:  ref.ArrayStr("after_inserts"),
		AfterUpdates:  ref.ArrayStr("after_updates"),
		AfterDeletes:  ref.ArrayStr("after_deletes"),
		AuditLog:      ref.ArrayJson("audit_log"),
		beforeInserts: make([]TriggerFunction, 0),
		beforeUpdates: make([]TriggerFunction, 0),
		beforeDeletes: make([]TriggerFunction, 0),
		afterInserts:  make([]TriggerFunction, 0),
		afterUpdates:  make([]TriggerFunction, 0),
		afterDeletes:  make([]TriggerFunction, 0),
		db:            s.db,
	}

	columns := ref.ArrayJson("columns")
	for _, column := range columns {
		definition, err := column.Byte("definition")
		if err != nil {
			return nil, err
		}
		result.Columns = append(result.Columns, &Column{
			Name:       column.Str("name"),
			TypeColumn: TypeColumn(column.Str("type_column")),
			TypeData:   TypeData(column.Str("type_data")),
			Default:    column.ValAny("default"),
			Definition: definition,
			model:      result,
		})
	}

	indexes := ref.ArrayJson("indexes")
	for _, index := range indexes {
		result.Indexes = append(result.Indexes, &Index{
			Name:   index.Str("name"),
			Sorted: index.Bool("sorted"),
		})
	}

	primaryKeys := ref.ArrayJson("primary_keys")
	for _, primaryKey := range primaryKeys {
		result.PrimaryKeys = append(result.PrimaryKeys, &Index{
			Name:   primaryKey.Str("name"),
			Sorted: primaryKey.Bool("sorted"),
		})
	}

	foreignKeys := ref.ArrayJson("foreign_keys")
	for _, foreignKey := range foreignKeys {
		to := foreignKey.Json("to")
		toSchema := to.Str("schema")
		toName := to.Str("name")
		toModel, err := s.db.GetModel(toSchema, toName)
		if err != nil {
			return nil, err
		}
		result.ForeignKeys = append(result.ForeignKeys, &Detail{
			To: &From{
				Database: to.Str("database"),
				Schema:   toSchema,
				Name:     toName,
				Table:    to.Str("table"),
				As:       to.Str("as"),
				Model:    toModel,
			},
			Keys:            foreignKey.MapStr("keys"),
			Select:          foreignKey.ArrayStr("select"),
			OnDeleteCascade: foreignKey.Bool("on_delete_cascade"),
			OnUpdateCascade: foreignKey.Bool("on_update_cascade"),
			Rows:            foreignKey.Int("rows"),
		})
	}

	unique := ref.ArrayJson("unique")
	for _, unique := range unique {
		result.Unique = append(result.Unique, &Index{
			Name:   unique.Str("name"),
			Sorted: unique.Bool("sorted"),
		})
	}

	required := ref.ArrayJson("required")
	for _, required := range required {
		result.Required = append(result.Required, &Index{
			Name:   required.Str("name"),
			Sorted: required.Bool("sorted"),
		})
	}

	details := ref.Json("details")
	for name := range details {
		detail := details.Json(name)
		to := detail.Json("to")
		toSchema := to.Str("schema")
		toName := to.Str("name")
		toModel, err := s.db.GetModel(toSchema, toName)
		if err != nil {
			return nil, err
		}
		result.Details[name] = &Detail{
			To: &From{
				Database: to.Str("database"),
				Schema:   toSchema,
				Name:     toName,
				Table:    detail.Str("table"),
				As:       detail.Str("as"),
				Model:    toModel,
			},
			Keys:            detail.MapStr("keys"),
			Select:          detail.ArrayStr("select"),
			OnDeleteCascade: detail.Bool("on_delete_cascade"),
			OnUpdateCascade: detail.Bool("on_update_cascade"),
			Rows:            detail.Int("rows"),
		}
	}

	rollups := ref.Json("rollups")
	for name := range rollups {
		rollup := rollups.Json(name)
		to := rollup.Json("to")
		toSchema := to.Str("schema")
		toName := to.Str("name")
		toModel, err := s.db.GetModel(toSchema, toName)
		if err != nil {
			return nil, err
		}
		result.Rollups[name] = &Detail{
			To: &From{
				Database: to.Str("database"),
				Schema:   toSchema,
				Name:     toName,
				Table:    rollup.Str("table"),
				As:       rollup.Str("as"),
				Model:    toModel,
			},
			Keys:            rollup.MapStr("keys"),
			Select:          rollup.ArrayStr("select"),
			OnDeleteCascade: rollup.Bool("on_delete_cascade"),
			OnUpdateCascade: rollup.Bool("on_update_cascade"),
			Rows:            rollup.Int("rows"),
		}
	}

	result.defaultTrigger()
	s.addModel(result)

	return result, nil
}

/**
* init: Initializes the schema.
* @return error
**/
func (s *Schema) init() error {
	for _, model := range s.Models {
		err := model.Init()
		if err != nil {
			return err
		}
	}
	return nil
}
