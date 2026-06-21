package jsql

import (
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/strs"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

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
	TenantId      string                  `json:"tenant_id"`
	ID            string                  `json:"id"`
	Database      string                  `json:"database"`
	Schema        string                  `json:"schema"`
	Name          string                  `json:"name"`
	DatabaseId    string                  `json:"database_id"`
	Table         string                  `json:"table"`
	Columns       []*Column               `json:"columns"`
	SourceField   string                  `json:"source_field"`
	IdxField      string                  `json:"idx_field"`
	IdtField      string                  `json:"idt_field"`
	Indexes       []*Index                `json:"indexes"`
	PrimaryKeys   []*Index                `json:"primary_keys"`
	ForeignKeys   []*Detail               `json:"foreign_keys"`
	Unique        []*Index                `json:"unique"`
	Required      []*Index                `json:"required"`
	Hiddens       []string                `json:"hiddens"`
	Details       map[string]*Detail      `json:"details"`
	Rollups       map[string]*Detail      `json:"rollups"`
	calcs         map[string]CalcFunction `json:"-"`
	IsStrict      bool                    `json:"is_strict"`
	Version       int                     `json:"version"`
	IsCore        bool                    `json:"is_core"`
	IsDebug       bool                    `json:"-"`
	isInit        bool                    `json:"-"`
	Calcs         map[string]string       `json:"calcs"`
	BeforeInserts []string                `json:"before_inserts"`
	BeforeUpdates []string                `json:"before_updates"`
	BeforeDeletes []string                `json:"before_deletes"`
	AfterInserts  []string                `json:"after_inserts"`
	AfterUpdates  []string                `json:"after_updates"`
	AfterDeletes  []string                `json:"after_deletes"`
	AuditLog      []et.Json               `json:"audit_log"`
	isChanged     bool                    `json:"-"`
	beforeInserts []TriggerFunction       `json:"-"`
	beforeUpdates []TriggerFunction       `json:"-"`
	beforeDeletes []TriggerFunction       `json:"-"`
	afterInserts  []TriggerFunction       `json:"-"`
	afterUpdates  []TriggerFunction       `json:"-"`
	afterDeletes  []TriggerFunction       `json:"-"`
	db            *DB                     `json:"-"`
	historyDb     *DB                     `json:"-"`
	deadDb        *DB                     `json:"-"`
	store         Store                   `json:"-"`
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
		ID:            reg.ULID(),
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

/**
* up: Updates the model's database and schema references after loading from catalog.
* @param schema *Schema
* @return *Model
**/
func (s *Model) up(schema *Schema) *Model {
	s.store = schema.store
	s.calcs = make(map[string]CalcFunction, 0)
	s.IsDebug = schema.isDebug
	s.beforeInserts = make([]TriggerFunction, 0)
	s.beforeUpdates = make([]TriggerFunction, 0)
	s.beforeDeletes = make([]TriggerFunction, 0)
	s.afterInserts = make([]TriggerFunction, 0)
	s.afterUpdates = make([]TriggerFunction, 0)
	s.afterDeletes = make([]TriggerFunction, 0)
	s.db = schema.db
	s.historyDb = schema.historyDb
	s.deadDb = schema.deadDb
	for _, column := range s.Columns {
		column.up(s)
	}
	s.defaultTrigger()
	schema.addModel(s)
	return s
}

/**
* addAuditLog
* @param userId string, action string
**/
func (s *Model) addAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
}

/**
* ToJson: Returns the model metadata as an et.Json map.
* @return et.Json
**/
func (s *Model) ToJson() et.Json {
	return et.Json{
		"id":           s.ID,
		"database":     s.Database,
		"schema":       s.Schema,
		"name":         s.Name,
		"table":        s.Table,
		"columns":      s.Columns,
		"source_field": s.SourceField,
		"idx_field":    s.IdxField,
		"idt_field":    s.IdtField,
		"indexes":      s.Indexes,
		"primary_keys": s.PrimaryKeys,
		"foreign_keys": s.ForeignKeys,
		"unique":       s.Unique,
		"required":     s.Required,
		"hiddens":      s.Hiddens,
		"details":      s.Details,
		"rollups":      s.Rollups,
		"calcs":        s.Calcs,
		"audit_log":    s.AuditLog,
	}
}

/**
* ToString: Returns the model metadata as a string.
* @return string
**/
func (s *Model) ToString() string {
	return s.ToJson().ToString()
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

	if s.store == nil {
		return nil
	}

	s.isChanged = false

	data := s.ToJson()
	if s.IsDebug {
		logs.Log(packageName, "save:", data.ToString())
	}

	err := s.store.Set("model", s.ID, s.TenantId, s.DatabaseId, s, s.ID)
	if err != nil {
		return err
	}

	channel := fmt.Sprintf("model:%s", s.ID)
	event.Publish(channel, data)
	return nil
}

/**
* AddAuditLog
* @param userId string, action string
**/
func (s *Model) AddAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = make([]et.Json, 0)
	}

	now := timezone.Now()
	s.AuditLog = append(s.AuditLog, et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if len(s.AuditLog) > maxAuditLog {
		s.AuditLog = s.AuditLog[len(s.AuditLog)-maxAuditLog:]
	}
	s.isChanged = true
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
* GetCalcFunc: Returns the CalcFunction for the given name, if it exists.
* @param name string
* @return CalcFunction, bool
**/
func (s *Model) GetCalcFunc(name string) (CalcFunction, bool) {
	result, ok := s.calcs[name]
	return result, ok
}

/**
* initInDb: Checks if the model exists in the database and loads it if not.
* @param db *DB
* @return (bool, error) where bool indicates if the model already existed
**/
func (s *Model) initInDb(db *DB) (bool, error) {
	exist, err := db.existModel(s)
	if err != nil {
		return false, err
	}

	if !exist {
		err = s.db.load(s)
		if err != nil {
			return false, err
		}
	}

	return exist, nil
}

/**
* wrapper: Wraps the jrex with the model
* @param rex *jrex.Jrex, model *Model
* @return void
**/
func (s *Model) wrapper(instance *jrex.Instance) {
	instance.Set("db", s.db)
	instance.Set("getDb", GetDb)
	instance.Set("newTx", NewTx)
}

/**
* Init: Runs DDL for the model the first time it is called; subsequent calls are no-ops.
* @return error
**/
func (s *Model) Init() error {
	if s.isInit {
		return nil
	}

	n := 1
	if s.historyDb != nil {
		n++
	}

	if s.deadDb != nil {
		n++
	}

	var err error
	var wg sync.WaitGroup
	wg.Add(n)

	go func() {
		defer wg.Done()
		var exist bool
		exist, err = s.initInDb(s.db)
		if err != nil {
			return
		}

		if !exist && !s.IsCore {
			err = s.save()
			if err != nil {
				return
			}
		}
	}()

	if s.historyDb != nil {
		go func() {
			defer wg.Done()
			_, err = s.initInDb(s.historyDb)
			if err != nil {
				return
			}
		}()
	}

	if s.deadDb != nil {
		go func() {
			defer wg.Done()
			_, err = s.initInDb(s.deadDb)
			if err != nil {
				return
			}
		}()
	}

	wg.Wait()
	if err != nil {
		return err
	}

	for _, detail := range s.Details {
		err = detail.init()
		if err != nil {
			return err
		}
	}

	s.isInit = true
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
* SetDb: Sets the primary DB connection for the model.
* @param db *DB
**/
func (s *Model) SetDb(db *DB) {
	s.db = db
}

/*** SetHistoryDb: Sets the history DB connection for the model.
* @param db *DB
**/
func (s *Model) SetHistoryDb(db *DB) {
	s.historyDb = db
}

/**
* SetDeadDb: Sets the dead DB connection for the model.
* @param db *DB
**/
func (s *Model) SetDeadDb(db *DB) {
	s.deadDb = db
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
* Count: Returns the count of records in the model.
* @return (int, error)
**/
func (s *Model) Count() (int, error) {
	return s.
		From().
		Count()
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
	return result
}

/**
* Query: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param query et.Json
* @return et.Items, error
**/
func (s *Model) QueryTx(tx *Tx, query et.Json) (et.Items, error) {
	query.Set("from", fmt.Sprintf("%s", s.Table))
	return s.db.loadQuery(tx, query)
}

/**
* Query: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param query et.Json
* @return et.Items, error
**/
func (s *Model) Query(query et.Json) (et.Items, error) {
	return s.QueryTx(nil, query)
}

/**
* HistoryQuery: Executes a query against the model's history database, if configured.
* @param query et.Json
* @return et.Items, error
**/
func (s *Model) HistoryQueryTx(tx *Tx, query et.Json) (et.Items, error) {
	if s.historyDb == nil {
		return et.Items{}, fmt.Errorf(MSG_HISTORY_DB_NOT_CONFIGURED, s.Name)
	}
	query.Set("from", fmt.Sprintf("%s", s.Table))
	return s.historyDb.loadQuery(tx, query)
}

/**
* HistoryQuery: Executes a query against the model's history database, if configured.
* @param query et.Json
* @return et.Items, error
**/
func (s *Model) HistoryQuery(query et.Json) (et.Items, error) {
	return s.HistoryQueryTx(nil, query)
}

/**
* DeadQuery: Executes a query against the model's dead database, if configured.
* @param query et.Json
* @return et.Items, error
**/
func (s *Model) DeadQueryTx(tx *Tx, query et.Json) (et.Items, error) {
	if s.deadDb == nil {
		return et.Items{}, fmt.Errorf(MSG_DEAD_DB_NOT_CONFIGURED, s.Name)
	}
	query.Set("from", fmt.Sprintf("%s", s.Table))
	return s.deadDb.loadQuery(tx, query)
}

/**
* DeadQuery: Executes a query against the model's dead database, if configured.
* @param query et.Json
* @return et.Items, error
**/
func (s *Model) DeadQuery(query et.Json) (et.Items, error) {
	return s.DeadQueryTx(nil, query)
}

/**
* SetSeries: Creates a Command of type SET_SERIES pre-loaded with the given data.
* @param tag, format string, val int
* @return error
**/
func (s *Model) SetSeries(tag, format string, val int) error {
	return s.db.SetSeries(tag, format, val)
}

/**
* GetSeries: Returns the series data for the given tag and owner.
* @param tag string
* @return (et.Item, error)
**/
func (s *Model) GetSeries(tag string) (et.Item, error) {
	return s.db.GetSeries(tag)
}

/**
* DeleteSeries: Deletes the series data for the given tag and owner.
* @param tag, ownerId string
* @return error
**/
func (s *Model) DeleteSeries(tag string) error {
	return s.db.DeleteSeries(tag)
}

/**
* GenSerie: Returns the next series value for the given tag and owner.
* @param tag, ownerId string
* @return (string, error)
**/
func (s *Model) GenSerie(tag string) (string, error) {
	return s.db.GenSerie(tag)
}

/**
* GenValue: Returns the next value for the given tag and owner.
* @param tag, ownerId string
* @return (int, error)
**/
func (s *Model) GenValue(tag string) (int, error) {
	return s.db.GenValue(tag)
}
