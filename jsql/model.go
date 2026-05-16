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
type CalcFunction func(tx *Tx, data et.Json)

/**
* Index: Represents a named index or primary-key field; Sorted selects BTREE over HASH.
**/
type Index struct {
	Name   string `json:"name"`
	Sorted bool   `json:"sorted"`
}

type Model struct {
	Database      string                  `json:"database"`
	Schema        string                  `json:"schema"`
	Name          string                  `json:"name"`
	Table         string                  `json:"table"`
	Columns       []*Column               `json:"columns"`
	SourceField   string                  `json:"source_field"`
	IdxField      string                  `json:"idx_field"`
	Indexes       []*Index                `json:"indexes"`
	PrimaryKeys   []*Index                `json:"primary_keys"`
	ForeignKeys   []*Detail               `json:"foreign_keys"`
	Unique        []*Index                `json:"unique"`
	Required      []*Index                `json:"required"`
	Hiddens       []string                `json:"hiddens"`
	Details       map[string]*Detail      `json:"details"`
	Rollups       map[string]*Detail      `json:"rollups"`
	Calcs         map[string]CalcFunction `json:"calcs"`
	IsStrict      bool                    `json:"is_strict"`
	Version       int                     `json:"version"`
	IsCore        bool                    `json:"is_core"`
	IsDebug       bool                    `json:"-"`
	IsChanged     bool                    `json:"-"`
	isInit        bool                    `json:"-"`
	isTest        bool                    `json:"-"`
	beforeInserts []TriggerFunction       `json:"-"`
	beforeUpdates []TriggerFunction       `json:"-"`
	beforeDeletes []TriggerFunction       `json:"-"`
	afterInserts  []TriggerFunction       `json:"-"`
	afterUpdates  []TriggerFunction       `json:"-"`
	afterDeletes  []TriggerFunction       `json:"-"`
	db            *DB                     `json:"-"`
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

	return s.db.setCatalog(s.Name, "model", s.Version, s)
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
func (s *Model) GetColumn(name string) (*Column, bool) {
	idx := s.idxColumn(name)
	if idx != -1 {
		return s.Columns[idx], true
	}

	if s.IsStrict {
		return nil, false
	}

	if s.SourceField == "" {
		return nil, false
	}

	return s.newColumn(name, ATTRIB, ANY, "", []byte{}), true
}

/**
* GetField: Returns the Field for the given name.
* @param name string
* @return *Field, bool
**/
func (s *Model) GetField(name string) (*Field, bool) {
	col, ok := s.GetColumn(name)
	if !ok {
		return nil, false
	}
	return &Field{
		TypeColumn: col.TypeColumn,
		TypeData:   col.TypeData,
		Name:       name,
		As:         "",
		From:       getFrom(s, ""),
	}, true
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
* From: Creates a new Query for this model with the given alias.
* @param as ...string
* @return *Query
**/
func (s *Model) From(as ...string) *Query {
	result := newQuery(s, as...)
	return result
}

/**
* Where: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param cond *et.Condition
* @return *Query
**/
func (s *Model) Where(cond *et.Condition) *Query {
	result := s.From()
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

/**
* Query: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param query et.Json
* @return et.Items, error
**/
func (s *Model) Query(query et.Json) (et.Items, error) {
	return et.Items{}, nil
}

/**
* SetSeries: Creates a Command of type SET_SERIES pre-loaded with the given data.
* @param tag, ownerId, format string, val int
* @return error
**/
func (s *Model) SetSeries(tag, ownerId, format string, val int) error {
	return s.db.SetSeries(tag, ownerId, format, val)
}

/**
* GetSeries: Returns the series data for the given tag and owner.
* @param tag, ownerId string
* @return (et.Item, error)
**/
func (s *Model) GetSeries(tag, ownerId string) (et.Item, error) {
	return s.db.GetSeries(tag, ownerId)
}

/**
* DeleteSeries: Deletes the series data for the given tag and owner.
* @param tag, ownerId string
* @return error
**/
func (s *Model) DeleteSeries(tag, ownerId string) error {
	return s.db.DeleteSeries(tag, ownerId)
}

/**
* NextSeries: Returns the next series value for the given tag and owner.
* @param tag, ownerId string
* @return (string, error)
**/
func (s *Model) NextSeries(tag, ownerId string) (string, error) {
	return s.db.NextSeries(tag, ownerId)
}

/**
* NextValue: Returns the next value for the given tag and owner.
* @param tag, ownerId string
* @return (int, error)
**/
func (s *Model) NextValue(tag, ownerId string) (int, error) {
	return s.db.NextValue(tag, ownerId)
}
