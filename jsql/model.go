package jsql

import (
	"database/sql"
	"fmt"
	"slices"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jrex"
	"github.com/cgalvisleon/et/timezone"
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
	ID            string                  `json:"id"`
	database      string                  `json:"-"`
	Schema        string                  `json:"schema"`
	Name          string                  `json:"name"`
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
	Masters       map[string]*Master      `json:"master"`
	Rollups       map[string]*Detail      `json:"rollups"`
	IsStrict      bool                    `json:"is_strict"`
	Version       int                     `json:"version"`
	IsDebug       bool                    `json:"-"`
	isInit        bool                    `json:"-"`
	BeforeInserts []string                `json:"before_inserts"`
	BeforeUpdates []string                `json:"before_updates"`
	BeforeDeletes []string                `json:"before_deletes"`
	AfterInserts  []string                `json:"after_inserts"`
	AfterUpdates  []string                `json:"after_updates"`
	AfterDeletes  []string                `json:"after_deletes"`
	AuditLog      *et.SafeData            `json:"audit_log"`
	isChanged     bool                    `json:"-"`
	calcs         map[string]CalcFunction `json:"-"`
	beforeInserts []TriggerFunction       `json:"-"`
	beforeUpdates []TriggerFunction       `json:"-"`
	beforeDeletes []TriggerFunction       `json:"-"`
	afterInserts  []TriggerFunction       `json:"-"`
	afterUpdates  []TriggerFunction       `json:"-"`
	afterDeletes  []TriggerFunction       `json:"-"`
	db            *DB                     `json:"-"`
}

/**
* AddAuditLog: Adds an audit log to the model.
* @param userId string, action string
**/
func (s *Model) AddAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = &et.SafeData{}
	}

	now := timezone.Now()
	s.AuditLog.Add(et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	maxAuditLog := envar.GetInt("MAX_AUDIT_LOG", 1000)
	if s.AuditLog.Len() > maxAuditLog {
		s.AuditLog.Delete(maxAuditLog)
	}
	s.isChanged = true
	if s.db != nil {
		s.db.addAuditLog(userId, action)
	}
}

/**
* ToJson: Returns the model metadata as an et.Json map.
* @return et.Json
**/
func (s *Model) ToJson() et.Json {
	columns := make([]et.Json, 0)
	for _, column := range s.Columns {
		columns = append(columns, column.ToJson())
	}

	return et.Json{
		"id":             s.ID,
		"schema":         s.Schema,
		"name":           s.Name,
		"table":          s.Table,
		"columns":        columns,
		"source_field":   s.SourceField,
		"idx_field":      s.IdxField,
		"idt_field":      s.IdtField,
		"indexes":        s.Indexes,
		"primary_keys":   s.PrimaryKeys,
		"foreign_keys":   s.ForeignKeys,
		"unique":         s.Unique,
		"required":       s.Required,
		"hiddens":        s.Hiddens,
		"details":        s.Details,
		"master":         s.Masters,
		"rollups":        s.Rollups,
		"is_strict":      s.IsStrict,
		"version":        s.Version,
		"before_inserts": s.BeforeInserts,
		"before_updates": s.BeforeUpdates,
		"before_deletes": s.BeforeDeletes,
		"after_inserts":  s.AfterInserts,
		"after_updates":  s.AfterUpdates,
		"after_deletes":  s.AfterDeletes,
		"audit_log":      s.AuditLog,
	}
}

/**
* loadColumns: Loads the columns from a JSON object.
* @param columns []et.Json
* @return void
**/
func (s *Model) loadColumns(columns []et.Json) {
	for _, column := range columns {
		col := loadColumn(column)
		s.Columns = append(s.Columns, col)
	}
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
* initModel: Checks if the model exists in the database and loads it if not.
* @param db *DB
* @return (bool, error) where bool indicates if the model already existed
**/
func (s *Model) initModel(db *DB) (bool, error) {
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
* @param instance *jrex.Instance, model *Model
* @return void
**/
func (s *Model) wrapper(instance *jrex.Instance) {
	instance.Set("db", s.db)
	instance.Set("newTx", NewTx)
}

/**
* Clone: Clones the model
* @return *Model
**/
func (s *Model) Clone() *Model {
	result := new(Model)
	*result = *s
	result.isInit = false
	return result
}

/**
* Init: Runs DDL for the model the first time it is called; subsequent calls are no-ops.
* @return error
**/
func (s *Model) Init() error {
	if s.isInit {
		return nil
	}

	_, err := s.initModel(s.db)
	if err != nil {
		return err
	}

	for _, detail := range s.Details {
		err = detail.init()
		if err != nil {
			return err
		}
	}

	for _, master := range s.Masters {
		err = master.init()
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
func (s *Model) Db() *DB {
	return s.db
}

/**
* SetDb: Sets the primary DB connection for the model.
* @param db *DB
**/
func (s *Model) SetDb(db *DB) {
	s.isInit = false
	s.db = db
}

/**
* SqlDb: Returns the underlying *sql.DB connection pool.
* @return *sql.DB
**/
func (s *Model) SqlDB() *sql.DB {
	return s.db.db
}

/**
* GetModel: Returns the model for the given schema and name.
* @param schema string
* @param name string
* @return *Model, error
**/
func (s *Model) GetModel(schema, name string) (*Model, error) {
	return s.db.GetModel(schema, name)
}

/**
* newColumn: Constructs a Column bound to this model without adding it to the Columns slice.
* @param name string, tpColumn TypeColumn, tpData TypeData, defaultValue interface{}, definition []byte
* @return *Column
**/
func (s *Model) newColumn(name string, tpColumn TypeColumn, tpData TypeData, deFault any) *Column {
	return &Column{
		Name:       name,
		TypeColumn: tpColumn,
		TypeData:   tpData,
		Default:    deFault,
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

	return s.newColumn(name, ATTRIB, ANY, ""), true
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
* GetFrom: Returns the From for the model.
* @return *From
**/
func (s *Model) GetFrom() *From {
	return getFrom(s, "")
}

/**
* From: Creates a new Query for this model with the given alias.
* @param as ...string
* @return *Query
**/
func (s *Model) As(as ...string) *Query {
	result := newQuery(s, as...)
	return result
}

/**
* Detail: Returns the detail for the given name.
* @param name string
* @return *Model, bool
**/
func (s *Model) Detail(name string) (*Model, bool) {
	detail, ok := s.Details[name]
	if !ok {
		return nil, false
	}

	if detail.To == nil {
		return nil, false
	}

	if detail.To.Model == nil {
		return nil, false
	}

	return detail.To.Model, true
}

/**
* Master: Returns the master for the given name.
* @param name string
* @return *Query, bool
**/
func (s *Model) Master(name string) (*Query, bool) {
	master, ok := s.Masters[name]
	if !ok {
		return nil, false
	}

	if master.To == nil {
		return nil, false
	}

	if master.To.Model == nil {
		return nil, false
	}

	conditions := make([]*et.Condition, 0)
	for k, fK := range master.ToKeys {
		k = fmt.Sprintf("A.%s", k)
		fK = fmt.Sprintf("B.%s", fK)
		conditions = append(conditions, Eq(k, fK))
	}

	result := master.To.Model.As("A")
	result.Join(master.Bridge.Model, "B", conditions)
	return result, true
}

/**
* Bridge: Returns the bridge for the given name.
* @param name string
* @return *Model, bool
**/
func (s *Model) Bridge(name string) (*Model, bool) {
	master, ok := s.Masters[name]
	if !ok {
		return nil, false
	}

	return master.Bridge.Model, true
}

/**
* InnerJoin: Creates a new Query for this model with the given model as the INNER JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Model) Join(to *Model, as string, on []*et.Condition) *Query {
	result := s.As("A")
	result.Join(to, as, on)
	return result
}

/**
* Select: Creates a new Query for this model with the given fields as the SELECT clause.
* @param fields ...string
* @return *Query
**/
func (s *Model) Select(fields ...string) *Query {
	result := s.As("")
	result.Select(fields...)
	return result
}

/**
* Calc: Creates a new Query for this model with the given fields as the CALC clause.
* @param fields ...string
* @return *Query
**/
func (s *Model) Calc(fields ...string) *Query {
	result := s.As("")
	result.Calc(fields...)
	return result
}

/**
* Where: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param cond *et.Condition
* @return *Query
**/
func (s *Model) Where(cond *et.Condition) *Query {
	result := s.As("")
	result.Where(cond)
	return result
}

/**
* Count: Returns the count of records in the model.
* @return (int, error)
**/
func (s *Model) Count() (int, error) {
	return s.
		As().
		Count()
}

/**
* First: Returns the first record in the model.
* @return (et.Item, error)
**/
func (s *Model) First(n int) (et.Items, error) {
	return s.
		As().
		Limit(1, n)
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
* @return *Query
**/
func (s *Model) QueryTx(tx *Tx, query et.Json) *Query {
	result := s.As("")
	result.loadQuery(query)
	return result
}

/**
* Query: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param query et.Json
* @return *Query
**/
func (s *Model) Query(query et.Json) *Query {
	return s.QueryTx(nil, query)
}
