package jsql

import (
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/utility"
)

/**
* Schema: Represents a database schema that owns a set of models.
**/
type Schema struct {
	TenantId  string            `json:"tenant_id"`
	Database  string            `json:"database"`
	Name      string            `json:"name"`
	Models    map[string]*Model `json:"models"`
	db        *DB               `json:"-"`
	historyDb *DB               `json:"-"`
	deadDb    *DB               `json:"-"`
	isDebug   bool              `json:"-"`
	mu        *sync.RWMutex     `json:"-"`
	store     Store             `json:"-"`
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
func (s *Schema) ToJson() et.Json {
	models := et.Json{}
	for _, model := range s.Models {
		models[model.Name] = model.ToJson()
	}
	return et.Json{
		"database": s.Database,
		"name":     s.Name,
		"models":   s.Models,
	}
}

/**
* SetHistoryDb: Sets the history database for this schema.
* @param db *DB
* @return void
**/
func (s *Schema) SetHistoryDb(db *DB) {
	s.TenantId = db.TenantId
	s.historyDb = db
}

/**
* SetDeadDb: Sets the dead database for this schema.
* @param db *DB
* @return void
**/
func (s *Schema) SetDeadDb(db *DB) {
	s.deadDb = db
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
