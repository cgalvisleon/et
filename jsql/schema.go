package jsql

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwf"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/utility"
)

/**
* Schema: Represents a database schema that owns a set of models.
**/
type Schema struct {
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"models"`
	database string            `json:"-"`
	db       *DB               `json:"-"`
	isDebug  bool              `json:"-"`
	mu       *sync.RWMutex     `json:"-"`
}

/**
* up: Initializes the schema.
* @param db *DB
* @return error
**/
func (s *Schema) up(db *DB) error {
	s.db = db
	s.database = db.Name
	s.isDebug = db.isDebug
	s.mu = &sync.RWMutex{}
	for _, model := range s.Models {
		err := model.up(s)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
* toJson: Returns the schema metadata as an et.Json map.
* @return et.Json
**/
func (s *Schema) toJson() et.Json {
	models := et.Json{}
	for name, model := range s.Models {
		models[name] = model.ToJson()
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
* @param id string, name string, version int, userId string
* @return *Model
**/
func (s *Schema) newModel(id, name string, version int, userId string) *Model {
	name = utility.Normalize(name)
	id = reg.GetUUID(id)
	result := &Model{
		Kind:          MODEL,
		ID:            id,
		database:      s.database,
		Schema:        s.Name,
		Name:          name,
		Table:         name,
		Columns:       make([]*Column, 0),
		Indexes:       make([]*Index, 0),
		PrimaryKeys:   make([]*Index, 0),
		ForeignKeys:   make([]*Detail, 0),
		Unique:        make([]*Index, 0),
		Required:      make([]*Index, 0),
		Hiddens:       make([]string, 0),
		OmitUpdates:   make([]string, 0),
		Details:       make(map[string]*Detail, 0),
		Masters:       make(map[string]*Master, 0),
		Rollups:       make(map[string]*Rollups, 0),
		Version:       version,
		BeforeInserts: make([]*jwf.Script, 0),
		BeforeUpdates: make([]*jwf.Script, 0),
		BeforeDeletes: make([]*jwf.Script, 0),
		AfterInserts:  make([]*jwf.Script, 0),
		AfterUpdates:  make([]*jwf.Script, 0),
		AfterDeletes:  make([]*jwf.Script, 0),
		AuditLog:      &et.SafeData{},
		calcs:         make(map[string]CalcFunction, 0),
		calcScripts:   make(map[string]string, 0),
		beforeInserts: make([]TriggerFunction, 0),
		beforeUpdates: make([]TriggerFunction, 0),
		beforeDeletes: make([]TriggerFunction, 0),
		afterInserts:  make([]TriggerFunction, 0),
		afterUpdates:  make([]TriggerFunction, 0),
		afterDeletes:  make([]TriggerFunction, 0),
		db:            s.db,
	}
	result.defaultTrigger()
	s.db.addAuditLog(userId, "new_model")
	s.addModel(result)
	return result
}

/**
* loadModel: Loads a Model from the database catalog by name.
* @param params et.Json
* @return *Model, error
**/
func (s *Schema) loadModel(def et.Json) (*Model, error) {
	bt, err := def.ToByte()
	if err != nil {
		return nil, err
	}

	var result *Model
	if err := json.Unmarshal(bt, &result); err != nil {
		return nil, err
	}

	result.database = s.database
	result.IsDebug = s.isDebug
	result.calcs = make(map[string]CalcFunction, 0)
	result.calcScripts = make(map[string]string, 0)
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
