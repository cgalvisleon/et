package jsql

import (
	"database/sql"
	"encoding/json"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
)

type Trigger struct {
	Name       string `json:"name"`
	Definition []byte `json:"definition"`
}

type TriggerFunction func(tx *Tx, old, new et.Json) error

type Index struct {
	Name   string `json:"name"`
	Sorted bool   `json:"sorted"`
}

type Model struct {
	Database      string             `json:"database"`
	Schema        string             `json:"schema"`
	Name          string             `json:"name"`
	Table         string             `json:"table"`
	Columns       []*Column          `json:"columns"`
	SourceField   string             `json:"source_field"`
	IdxField      string             `json:"idx_field"`
	Indexes       []*Index           `json:"indexes"`
	PrimaryKeys   []*Index           `json:"primary_keys"`
	ForeignKeys   []*Detail          `json:"foreign_keys"`
	Unique        []*Index           `json:"unique"`
	Required      []*Index           `json:"required"`
	Hidden        []string           `json:"hidden"`
	Details       map[string]*Detail `json:"details"`
	Rollups       map[string]*Detail `json:"rollups"`
	Relations     map[string]*Detail `json:"relations"`
	IsStrict      bool               `json:"is_strict"`
	Version       int                `json:"version"`
	IsCore        bool               `json:"is_core"`
	IsDebug       bool               `json:"-"`
	IsChanged     bool               `json:"-"`
	isInit        bool               `json:"-"`
	beforeInserts []TriggerFunction  `json:"-"`
	beforeUpdates []TriggerFunction  `json:"-"`
	beforeDeletes []TriggerFunction  `json:"-"`
	afterInserts  []TriggerFunction  `json:"-"`
	afterUpdates  []TriggerFunction  `json:"-"`
	afterDeletes  []TriggerFunction  `json:"-"`
	db            *DB                `json:"-"`
}

/**
* serialize
* @return []byte, error
**/
func (s *Model) serialize() ([]byte, error) {
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
func (s *Model) ToJson() et.Json {
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
* Key
* @return string
**/
func (s *Model) Key() string {
	result := s.Name
	result = strs.Append(s.Schema, result, ".")
	result = strs.Append(s.Database, result, ".")
	return result
}

/**
* save
* @return error
**/
func (s *Model) save() error {
	if s.IsCore {
		return nil
	}

	return nil
}

/**
* Debug
**/
func (s *Model) Debug() {
	s.IsDebug = true
}

/**
* Init
* @return error
**/
func (s *Model) Init() error {
	if s.isInit {
		return nil
	}

	err := s.db.load(s)
	if err != nil {
		return err
	}

	s.isInit = true
	if s.IsCore {
		return nil
	}

	if s.IsChanged {
		return s.save()
	}

	return nil
}

/**
* Stricted
* @return error
**/
func (s *Model) Stricted() {
	s.IsStrict = true
}

/**
* Db
* @return *sql.DB
**/
func (s *Model) Db() *sql.DB {
	return s.db.db
}

/**
* newColumn
* @param model *Model, name string, tpColumn TypeColumn, tpData TypeData, defaultValue interface{}, definition []byte
* @return *Column
**/
func (s *Model) newColumn(name string, tpColumn TypeColumn, tpData TypeData, defaultValue interface{}, definition []byte) *Column {
	return &Column{
		Name:       name,
		TypeColumn: tpColumn,
		TypeData:   tpData,
		Default:    defaultValue,
		Definition: definition,
		model:      s,
	}
}

/**
* idxColumn
* @param name string
* @return int
**/
func (s *Model) idxColumn(name string) int {
	return slices.IndexFunc(s.Columns, func(column *Column) bool { return column.Name == name })
}

/**
* FindColumn
* @param name string
* @return *Column
**/
func (s *Model) FindColumn(name string) *Column {
	idx := s.idxColumn(name)
	if idx != -1 {
		return s.Columns[idx]
	}

	if s.IsStrict {
		return nil
	}

	if s.SourceField == "" {
		return nil
	}

	return s.newColumn(name, ATTRIB, ANY, "", []byte{})
}

/**
* GetId
* @param id string
* @return string
**/
func (s *Model) GetId(id string) string {
	return reg.TagULID(s.Name, id)
}

/**
* BeforeInsert
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeInsert(fn TriggerFunction) *Model {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

/**
* BeforeUpdate
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeUpdate(fn TriggerFunction) *Model {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* BeforeDelete
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeDelete(fn TriggerFunction) *Model {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

/**
* BeforeInsertOrUpdate
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeInsertOrUpdate(fn TriggerFunction) *Model {
	s.beforeInserts = append(s.beforeInserts, fn)
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* AfterInsert
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterInsert(fn TriggerFunction) *Model {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

/**
* AfterUpdate
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterUpdate(fn TriggerFunction) *Model {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* AfterDelete
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterDelete(fn TriggerFunction) *Model {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

/**
* AfterInsertOrUpdate
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterInsertOrUpdate(fn TriggerFunction) *Model {
	s.afterInserts = append(s.afterInserts, fn)
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* Where
* @param cond *et.Condition
* @return *Query
**/
func (s *Model) Where(cond *et.Condition) *Query {
	result := newQuery(s)
	result.Where(cond)
	return result
}

/**
* Insert
* @param data et.Json
* @return *Command
**/
func (s *Model) Insert(data et.Json) *Command {
	result := newCommand(s, INSERT)
	result.Data = append(result.Data, data)
	return result
}

/**
* Bulk
* @param data []et.Json
* @return *Command
**/
func (s *Model) Bulk(data []et.Json) *Command {
	result := newCommand(s, BULK)
	result.Data = data
	return result
}

/**
* Update
* @param data et.Json
* @return *Command
**/
func (s *Model) Update(data et.Json) *Command {
	result := newCommand(s, UPDATE)
	result.Data = append(result.Data, data)
	return result
}

/**
* Delete
* @return *Command
**/
func (s *Model) Delete() *Command {
	result := newCommand(s, DELETE)
	return result
}

/**
* Upsert
* @param data et.Json
* @return *Command
**/
func (s *Model) Upsert(data et.Json) *Command {
	result := newCommand(s, UPSERT)
	result.Data = append(result.Data, data)
	for _, k := range s.PrimaryKeys {
		if v, ok := data[k.Name]; ok {
			result.Where(et.Eq(k.Name, v))
		}
	}
	return result
}
