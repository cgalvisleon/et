package jsql

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
)

/**
* Schema: Represents a database schema that owns a set of models.
**/
type Schema struct {
	Database string            `json:"database"`
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"-"`
	db       *DB               `json:"-"`
	mu       *sync.RWMutex     `json:"-"`
}

/**
* serialize: Marshals the schema metadata to JSON bytes.
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
* ToJson: Returns the schema metadata as an et.Json map.
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
* save: Persists schema metadata changes (stub — no-op until storage is wired).
* @return error
**/
func (s *Schema) save() error {
	if s.db == nil {
		return errors.New(msg.MSG_DB_IS_NIL)
	}
	return nil
}

/**
* newModel: Returns an existing model by name or creates and registers a new one.
* @param name string
* @param version int
* @return *Model, error
**/
func (s *Schema) newModel(name string, version int) (*Model, error) {
	name = utility.Normalize(name)

	s.mu.Lock()
	result, ok := s.Models[name]
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
		Hiddens:       make([]string, 0),
		Details:       make(map[string]*Detail, 0),
		Rollups:       make(map[string]*Detail, 0),
		Calcs:         make(map[string]CalcFunction, 0),
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
	s.Models[name] = result
	s.mu.Unlock()

	result.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		var results sync.Map
		var hasError error
		var wg sync.WaitGroup
		for _, validate := range result.Unique {
			wg.Add(1)
			go func(field string) {
				defer wg.Done()

				if hasError != nil {
					return
				}

				val := new.Str(field)
				if len(val) == 0 {
					results.Store(field, false)
					return
				}

				exists, err := result.
					Where(Eq(field, val)).
					Exists()
				if err != nil {
					hasError = err
					return
				}
				results.Store(field, exists)
			}(validate.Name)
		}
		wg.Wait()

		results.Range(func(key, value interface{}) bool {
			if ok, _ := value.(bool); ok {
				return false
			}
			return true
		})

		return hasError
	})

	result.BeforeUpdate(func(tx *Tx, old, new et.Json) error {
		var results sync.Map
		var hasError error
		var wg sync.WaitGroup
		for _, validate := range result.Unique {
			wg.Add(1)
			go func(field string) {
				defer wg.Done()

				if hasError != nil {
					return
				}

				newVal := new[field]
				chage := old.IsDeferent(field, newVal)
				if !chage {
					results.Store(field, false)
					return
				}

				ql := result.Where(Eq(field, newVal))
				for _, pk := range result.PrimaryKeys {
					val := old[pk.Name]
					ql = ql.And(Neg(pk.Name, val))
				}

				exists, err := ql.Exists()
				if err != nil {
					hasError = err
					return
				}
				results.Store(field, exists)
			}(validate.Name)
		}
		wg.Wait()

		results.Range(func(key, value interface{}) bool {
			if ok, _ := value.(bool); ok {
				return false
			}
			return true
		})

		return hasError
	})

	return result, nil
}

/**
* GetModel: Returns the named model or an error if it does not exist in this schema.
* @param name string
* @return *Model, error
**/
func (s *Schema) GetModel(name string) (*Model, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	name = utility.Normalize(name)
	s.mu.RUnlock()
	result, exists := s.Models[name]
	s.mu.RLock()
	if !exists {
		return nil, errors.New(msg.MSG_MODEL_NOT_FOUND)
	}

	return result, nil
}
