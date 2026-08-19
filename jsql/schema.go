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
	s.mu.RLock()
	result, exists := s.Models[name]
	s.mu.RUnlock()

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
		Masters:        make(map[string]*Master, 0),
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
	s.db.addAuditLog(userId, "new_model")
	s.addModel(result)
	return result
}

/**
* loadModel: Loads a Model from the database catalog by name.
* @param store *Store, id string
* @return *Model, error
**/
func (s *Schema) loadModel(store *Store, id string) (*Model, error) {
	if store == nil {
		return nil, errors.New(MSG_DB_STORE_IS_NIL)
	}

	var result *Model
	exists, err := store.Get("model", id, &result)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf(MSG_RECORD_NOT_FOUND, "model", id)
	}

	result.beforeInserts = make([]TriggerFunction, 0)
	result.beforeUpdates = make([]TriggerFunction, 0)
	result.beforeDeletes = make([]TriggerFunction, 0)
	result.afterInserts = make([]TriggerFunction, 0)
	result.afterUpdates = make([]TriggerFunction, 0)
	result.afterDeletes = make([]TriggerFunction, 0)
	result.db = s.db

	for _, column := range result.Columns {
		column.model = result
	}

	for _, foreignKey := range result.ForeignKeys {
		to := foreignKey.To
		toModel, err := s.db.GetModel(to.Schema, to.Name)
		if err != nil {
			return nil, err
		}
		foreignKey.To.Model = toModel
	}

	for _, detail := range result.Details {
		to := detail.To
		toModel, err := s.db.GetModel(to.Schema, to.Name)
		if err != nil {
			return nil, err
		}
		detail.To.Model = toModel
	}

	for _, rollup := range result.Rollups {
		to := rollup.To
		toModel, err := s.db.GetModel(to.Schema, to.Name)
		if err != nil {
			return nil, err
		}
		rollup.To.Model = toModel
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
