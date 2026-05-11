package jql

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
)

/**
* Schema: Represents a schema in the database
**/
type Schema struct {
	Database string            `json:"database"` // Database name
	Name     string            `json:"name"`     // Schema name
	models   map[string]*Model `json:"-"`        // Models
	db       *DB               `json:"-"`        // Database
	mu       *sync.RWMutex     `json:"-"`        // Mutex
}

/**
* serialize
* @return []byte, error
**/
func (s *Schema) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson
* @return et.Json
**/
func (s *Schema) ToJson() et.Json {
	bt, err := s.serialize()
	if err != nil {
		return et.Json{}
	}

	var result et.Json
	err = json.Unmarshal(bt, &result)
	if err != nil {
		return et.Json{}
	}

	return result
}

/**
* save: Save schema data
* @return error
**/
func (s *Schema) save() error {
	if s.db == nil {
		return errors.New(msg.MSG_DB_IS_NIL)
	}
	return nil
}

/**
* newModel
* @param schema, name string, version int
* @return *Model
**/
func (s *Schema) newModel(name string, version int) (*Model, error) {
	name = utility.Normalize(name)

	s.mu.Lock()
	result, ok := s.models[name]
	s.mu.Unlock()
	if ok {
		return result, nil
	}

	name = utility.Normalize(name)
	result = &Model{
		Database:      s.Database,
		Schema:        s.Name,
		Name:          name,
		Columns:       make([]*Column, 0),
		Indexes:       make([]*Index, 0),
		PrimaryKeys:   make([]*Index, 0),
		ForeignKeys:   make([]*Detail, 0),
		Unique:        make([]*Index, 0),
		Required:      make([]*Index, 0),
		Hidden:        make([]string, 0),
		Details:       make(map[string]*Detail, 0),
		Rollups:       make(map[string]*Detail, 0),
		Relations:     make(map[string]*Detail, 0),
		Version:       version,
		beforeInserts: make([]TriggerFunction, 0),
		beforeUpdates: make([]TriggerFunction, 0),
		beforeDeletes: make([]TriggerFunction, 0),
		afterInserts:  make([]TriggerFunction, 0),
		afterUpdates:  make([]TriggerFunction, 0),
		afterDeletes:  make([]TriggerFunction, 0),
		db:            s.db,
		IsDebug:       s.db.IsDebug,
	}
	s.mu.Lock()
	s.models[name] = result
	s.mu.Unlock()

	return result, nil
}

/**
* GetModel: Returns a model
* @param string name
* @return *Model, error
**/
func (s *Schema) GetModel(name string) (*Model, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	name = utility.Normalize(name)
	result, exists := s.models[name]
	if !exists {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return result, nil
}
