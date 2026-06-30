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
	store    Store             `json:"-"`
}

/**
* up: Updates the schema metadata after loading from catalog.
* @param db *DB
* @return *Schema
**/
func (s *Schema) up(db *DB) (*Schema, error) {
	s.db = db
	s.isDebug = db.isDebug
	for _, model := range s.Models {
		_, err := s.loadModel(model.ID)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}

/**
* ToJson: Returns the schema metadata as an et.Json map.
* @return et.Json
**/
func (s *Schema) Ref() et.Json {
	models := et.Json{}
	for _, model := range s.Models {
		models[model.ID] = model.Ref()
	}
	return et.Json{
		"database": s.Database,
		"name":     s.Name,
		"models":   models,
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
		isChanged:     true,
	}
	result.addAuditLog(userId, "new_model")
	result.up(s)
	return result
}

/**
* loadModel: Loads a Model from the database catalog by name.
* @param schema *Schema, name string
* @return *Model, error
**/
func (s *Schema) loadModel(id string) (*Model, error) {
	if s.store == nil {
		return nil, errors.New(MSG_DB_STORE_IS_NIL)
	}

	var result *Model
	exists, err := s.store.Get("model", id, &result)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New(MSG_MODEL_NOT_FOUND)
	}

	result.up(s)
	return result, nil
}
