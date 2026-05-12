package jsql

import (
	"database/sql"
	"encoding/json"
	"slices"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
)

/**
* Trigger: Stores a named trigger definition as raw bytes.
**/
type Trigger struct {
	Name       string `json:"name"`
	Definition []byte `json:"definition"`
}

/**
* TriggerFunction: Callback invoked before or after a data-mutation command.
**/
type TriggerFunction func(tx *Tx, old, new et.Json) error

/**
* Index: Represents a named index or primary-key field; Sorted selects BTREE over HASH.
**/
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
	isTest        bool               `json:"-"`
	beforeInserts []TriggerFunction  `json:"-"`
	beforeUpdates []TriggerFunction  `json:"-"`
	beforeDeletes []TriggerFunction  `json:"-"`
	afterInserts  []TriggerFunction  `json:"-"`
	afterUpdates  []TriggerFunction  `json:"-"`
	afterDeletes  []TriggerFunction  `json:"-"`
	db            *DB                `json:"-"`
}

/**
* serialize: Marshals the model metadata to JSON bytes.
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
* ToJson: Returns the model metadata as an et.Json map.
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
* Key: Returns the fully-qualified model identifier (database.schema.name).
* @return string
**/
func (s *Model) Key() string {
	result := s.Name
	result = strs.Append(s.Schema, result, ".")
	result = strs.Append(s.Database, result, ".")
	return result
}

/**
* save: Persists model metadata changes (stub — no-op until storage is wired).
* @return error
**/
func (s *Model) save() error {
	if s.IsCore {
		return nil
	}

	return nil
}

/**
* Debug: Enables debug logging and returns the model for chaining.
* @return *Model
**/
func (s *Model) Debug() *Model {
	s.IsDebug = true
	return s
}

/**
* Test: Enables test mode (DDL and DML are logged but not executed) and returns the model.
* @return *Model
**/
func (s *Model) Test() *Model {
	s.isTest = true
	return s
}

/**
* Init: Runs DDL for the model the first time it is called; subsequent calls are no-ops.
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
* Stricted: Enables strict mode — unknown field names are not treated as ATTRIBs.
**/
func (s *Model) Stricted() {
	s.IsStrict = true
}

/**
* Db: Returns the underlying *sql.DB connection pool.
* @return *sql.DB
**/
func (s *Model) Db() *sql.DB {
	return s.db.db
}

/**
* newColumn: Constructs a Column bound to this model without adding it to the Columns slice.
* @param name string, tpColumn TypeColumn, tpData TypeData, defaultValue interface{}, definition []byte
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
* idxColumn: Returns the slice index of the column with the given name, or -1 if not found.
* @param name string
* @return int
**/
func (s *Model) idxColumn(name string) int {
	return slices.IndexFunc(s.Columns, func(column *Column) bool { return column.Name == name })
}

/**
* FindColumn: Returns the Column for the given name. For non-strict models with a SourceField,
* unknown names are returned as synthetic ATTRIB columns.
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
* GetId: Returns a ULID tagged with the model name, used as a stable record identifier.
* @param id string
* @return string
**/
func (s *Model) GetId(id string) string {
	return reg.TagULID(s.Name, id)
}

/**
* BeforeInsert: Registers a trigger function to run before each INSERT.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeInsert(fn TriggerFunction) *Model {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

/**
* BeforeUpdate: Registers a trigger function to run before each UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeUpdate(fn TriggerFunction) *Model {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* BeforeDelete: Registers a trigger function to run before each DELETE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeDelete(fn TriggerFunction) *Model {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

/**
* BeforeInsertOrUpdate: Registers a trigger function to run before INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeInsertOrUpdate(fn TriggerFunction) *Model {
	s.beforeInserts = append(s.beforeInserts, fn)
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* AfterInsert: Registers a trigger function to run after each INSERT.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterInsert(fn TriggerFunction) *Model {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

/**
* AfterUpdate: Registers a trigger function to run after each UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterUpdate(fn TriggerFunction) *Model {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* AfterDelete: Registers a trigger function to run after each DELETE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterDelete(fn TriggerFunction) *Model {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

/**
* AfterInsertOrUpdate: Registers a trigger function to run after INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterInsertOrUpdate(fn TriggerFunction) *Model {
	s.afterInserts = append(s.afterInserts, fn)
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* Where: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param cond *et.Condition
* @return *Query
**/
func (s *Model) Where(cond *et.Condition) *Query {
	result := newQuery(s)
	result.Where(cond)
	return result
}

/**
* Insert: Creates a Command of type INSERT pre-loaded with the given data row.
* @param data et.Json
* @return *Command
**/
func (s *Model) Insert(data et.Json) *Command {
	result := newCommand(s, INSERT)
	result.Data = append(result.Data, data)
	return result
}

/**
* Bulk: Creates a Command of type BULK pre-loaded with multiple data rows.
* @param data []et.Json
* @return *Command
**/
func (s *Model) Bulk(data []et.Json) *Command {
	result := newCommand(s, BULK)
	result.Data = data
	return result
}

/**
* Update: Creates a Command of type UPDATE pre-loaded with the given data row.
* @param data et.Json
* @return *Command
**/
func (s *Model) Update(data et.Json) *Command {
	result := newCommand(s, UPDATE)
	result.Data = append(result.Data, data)
	return result
}

/**
* Delete: Creates a Command of type DELETE (conditions must be added with Where/And).
* @return *Command
**/
func (s *Model) Delete() *Command {
	result := newCommand(s, DELETE)
	return result
}

/**
* Upsert: Creates a Command of type UPSERT pre-loaded with data and primary-key conditions.
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
