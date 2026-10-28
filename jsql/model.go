package jsql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwf"
	"github.com/cgalvisleon/et/timezone"
)

const (
	MODEL = "model:model"
	TABLE = "model:table"
)

/**
* TriggerFunction: Callback invoked before or after a data-mutation command.
**/
type TriggerFunction func(tx *Tx, old, new et.Json) error
type CalcFunction func(tx *Tx, data et.Json)

/**
* Model: The definition of a table: columns, indexes, keys, SourceField, relations and triggers.
*
* Concurrency (the one place for these rules):
*   - DB.muSchemas guards DB.Schemas and DB.muAudit the DB audit log; getModel never creates a schema.
*   - Model.muDef serializes definitions. Columns, Details, Masters, Rollups and the calcs are copy on
*     write (defineColumn, defineDetail, defineMaster, defineRollup, defineCalc, defineCalcFunc, withEntry):
*     a query that already read them keeps a consistent value, because the drivers read without locks.
*   - Not copy on write yet (append in place, without muDef): Indexes, PrimaryKeys, Unique, Required,
*     ForeignKeys, Hiddens, OmitUpdates, IdxField and the before/after trigger scripts. Define them before the
*     model is queried concurrently (at start-up, as every consumer does).
*   - Model.muReplace serializes Replace. Lock order: DB.muSchemas → Model.muDef; defineMaster takes the
*     muDef of each model one after the other, never both at once.
**/
type Model struct {
	Kind          string                  `json:"kind"`
	ID            string                  `json:"id"`
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
	Masters       map[string]*Master      `json:"master"`
	Rollups       map[string]*Rollups     `json:"rollups"`
	OmitUpdates   []string                `json:"omit_updates"`
	IsStrict      bool                    `json:"is_strict"`
	Version       int                     `json:"version"`
	Deployed      bool                    `json:"deployed"`
	BeforeInserts []*jwf.Script           `json:"before_inserts"`
	BeforeUpdates []*jwf.Script           `json:"before_updates"`
	BeforeDeletes []*jwf.Script           `json:"before_deletes"`
	AfterInserts  []*jwf.Script           `json:"after_inserts"`
	AfterUpdates  []*jwf.Script           `json:"after_updates"`
	AfterDeletes  []*jwf.Script           `json:"after_deletes"`
	AuditLog      *et.SafeData            `json:"audit_log"`
	IsDebug       bool                    `json:"-"`
	IsInit        bool                    `json:"-"`
	database      string                  `json:"-"`
	isChanged     bool                    `json:"-"`
	calcs         map[string]CalcFunction `json:"-"`
	calcScripts   map[string]string       `json:"-"`
	beforeInserts []TriggerFunction       `json:"-"`
	beforeUpdates []TriggerFunction       `json:"-"`
	beforeDeletes []TriggerFunction       `json:"-"`
	afterInserts  []TriggerFunction       `json:"-"`
	afterUpdates  []TriggerFunction       `json:"-"`
	afterDeletes  []TriggerFunction       `json:"-"`
	db            *DB                     `json:"-"`
	schema        *Schema                 `json:"-"`
	muReplace     sync.Mutex              `json:"-"`
	muDef         sync.Mutex              `json:"-"` // serializes definitions; they replace maps and slices (copy on write)
}

/**
* UnmarshalJSON: A model from its json; its masters are read from "master" (the key the struct writes)
* or, when it is missing, from "masters" (the key of Define and of toJson).
* @param data []byte
* @return error
**/
func (s *Model) UnmarshalJSON(data []byte) error {
	type plain Model
	value := struct {
		*plain
		Masters map[string]*Master `json:"masters"`
	}{plain: (*plain)(s)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(s.Masters) == 0 && len(value.Masters) > 0 {
		s.Masters = value.Masters
	}
	return nil
}

/**
* up: Initializes the model.
* @param db *DB
* @return void
**/
func (s *Model) up(schema *Schema) error {
	s.schema = schema
	s.database = schema.database
	s.IsDebug = schema.isDebug
	s.calcs = make(map[string]CalcFunction, 0)
	s.calcScripts = make(map[string]string, 0)
	s.beforeInserts = make([]TriggerFunction, 0)
	s.beforeUpdates = make([]TriggerFunction, 0)
	s.beforeDeletes = make([]TriggerFunction, 0)
	s.afterInserts = make([]TriggerFunction, 0)
	s.afterUpdates = make([]TriggerFunction, 0)
	s.afterDeletes = make([]TriggerFunction, 0)
	s.db = schema.db

	if s.AuditLog == nil {
		s.AuditLog = et.NewSafeData()
	}

	if s.AuditLog.Data == nil {
		s.AuditLog.Data = make([]et.Json, 0)
	}

	for _, foreignKey := range s.ForeignKeys {
		to := foreignKey.To
		toModel, err := s.db.GetModel(to.Schema, to.Name)
		if err != nil {
			return err
		}
		foreignKey.To.Model = toModel
	}

	for _, detail := range s.Details {
		to := detail.To
		toModel, err := s.db.GetModel(to.Schema, to.Name)
		if err != nil {
			return err
		}
		detail.To.Model = toModel
	}

	for _, rollup := range s.Rollups {
		to := rollup.To
		toModel, err := s.db.GetModel(to.Schema, to.Name)
		if err != nil {
			return err
		}
		rollup.To.Model = toModel
	}

	s.defaultTrigger()
	schema.addModel(s)
	return nil
}

/**
* getSchema: Returns the schema of the model.
* @return *Schema
**/
func (s *Model) getSchema() *Schema {
	return s.schema
}

/**
* Replace: Replaces the content of the model with other's, keeping its pointer and its mutex.
* @param other *Model
**/
func (s *Model) Replace(other *Model) {
	if other == nil || other == s {
		return
	}

	s.muReplace.Lock()
	defer s.muReplace.Unlock()
	s.Kind = other.Kind
	s.ID = other.ID
	s.Schema = other.Schema
	s.Name = other.Name
	s.Table = other.Table
	s.Columns = other.Columns
	s.SourceField = other.SourceField
	s.IdxField = other.IdxField
	s.Indexes = other.Indexes
	s.PrimaryKeys = other.PrimaryKeys
	s.ForeignKeys = other.ForeignKeys
	s.Unique = other.Unique
	s.Required = other.Required
	s.Hiddens = other.Hiddens
	s.Details = other.Details
	s.Masters = other.Masters
	s.Rollups = other.Rollups
	s.OmitUpdates = other.OmitUpdates
	s.IsStrict = other.IsStrict
	s.Version = other.Version
	s.Deployed = other.Deployed
	s.BeforeInserts = other.BeforeInserts
	s.BeforeUpdates = other.BeforeUpdates
	s.BeforeDeletes = other.BeforeDeletes
	s.AfterInserts = other.AfterInserts
	s.AfterUpdates = other.AfterUpdates
	s.AfterDeletes = other.AfterDeletes
	s.AuditLog = other.AuditLog
	s.IsDebug = other.IsDebug
	s.IsInit = other.IsInit
	s.database = other.database
	s.isChanged = other.isChanged
	s.calcs = other.calcs
	s.calcScripts = other.calcScripts
	s.beforeInserts = other.beforeInserts
	s.beforeUpdates = other.beforeUpdates
	s.beforeDeletes = other.beforeDeletes
	s.afterInserts = other.afterInserts
	s.afterUpdates = other.afterUpdates
	s.afterDeletes = other.afterDeletes
	s.db = other.db
	s.schema = other.schema
}

/**
* getIdx: Returns the index for the model.
* @return string
**/
func (s *Model) getIdx() string {
	now := timezone.Now()
	return fmt.Sprintf("%d", now.UnixMilli())
}

/**
* addAuditLog: Adds an audit log to the model.
* @param userId string, action string
**/
func (s *Model) addAuditLog(userId string, action string) {
	if s.AuditLog == nil {
		s.AuditLog = &et.SafeData{}
	}

	now := timezone.Now()
	s.AuditLog.Add(et.Json{
		"created_at": now,
		"user_id":    userId,
		"action":     action,
	})
	s.AuditLog.Trim(envar.GetInt("MAX_AUDIT_LOG", 1000))
	s.isChanged = true
	if s.db != nil {
		s.db.addAuditLog(userId, action)
	}
}

/**
* getKind: Returns the kind of the model.
* @return string
**/
func (s *Model) getKind() string {
	return s.Kind
}

/**
* toJson: Returns the model metadata as an et.Json map.
* @return et.Json
**/
func (s *Model) toJson() et.Json {
	columns := make([]et.Json, 0)
	for _, column := range s.Columns {
		columns = append(columns, column.ToJson())
	}

	return et.Json{
		"kind":           s.Kind,
		"id":             s.ID,
		"schema":         s.Schema,
		"name":           s.Name,
		"table":          s.Table,
		"columns":        columns,
		"source_field":   s.SourceField,
		"idx_field":      s.IdxField,
		"indexes":        s.Indexes,
		"primary_keys":   s.PrimaryKeys,
		"foreign_keys":   s.ForeignKeys,
		"unique":         s.Unique,
		"required":       s.Required,
		"hiddens":        s.Hiddens,
		"omit_updates":   s.OmitUpdates,
		"details":        s.Details,
		"masters":        s.Masters,
		"master":         s.Masters, // the key before "masters", still read by its consumers (core-studio)
		"rollups":        s.Rollups,
		"is_strict":      s.IsStrict,
		"version":        s.Version,
		"before_inserts": s.BeforeInserts,
		"before_updates": s.BeforeUpdates,
		"before_deletes": s.BeforeDeletes,
		"after_inserts":  s.AfterInserts,
		"after_updates":  s.AfterUpdates,
		"after_deletes":  s.AfterDeletes,
		"audit_log":      s.AuditLog.All(),
	}
}

/**
* debug: Enables debug logging and returns the model for chaining.
* @return *Model
**/
func (s *Model) debug() *Model {
	s.IsDebug = true
	return s
}

/**
* getCalcFunc: Returns the CalcFunction for the given name, if it exists.
* @param name string
* @return CalcFunction, bool
**/
func (s *Model) getCalcFunc(name string) (CalcFunction, bool) {
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
* bindings: What a script of the model sees: the standard bindings of jwf, the jsql wrapper of its db and, in a trigger,
* TX, the transaction of the command that runs it; what the script writes goes in it (with SQLite, a write outside it
* would wait for the only connection forever)
* @param tx *Tx
* @return map[string]any
**/
func (s *Model) bindings(tx *Tx) map[string]any {
	result := jwf.Bindings(nil)
	if s == nil {
		return result
	}
	for name, binding := range Wrapper(s.db) {
		result[name] = binding
	}
	if tx != nil {
		result["TX"] = wrapTx(tx)
	}
	return result
}

/**
* setTable: Sets Table from Schema and Name according to the driver: schema.name when it has schemas
* (Driver.UseSchema), schema_name when it doesn't, and just name without schema. Without a driver
* it keeps schema.name.
**/
func (s *Model) setTable() {
	switch {
	case s.Schema == "":
		s.Table = s.Name
	case s.db != nil && s.db.driver != nil && !s.db.driver.UseSchema():
		s.Table = s.Schema + "_" + s.Name
	default:
		s.Table = s.Schema + "." + s.Name
	}
}

/**
* init: Runs DDL for the model the first time it is called; subsequent calls are no-ops.
* @return error
**/
func (s *Model) init() error {
	if s.IsInit {
		return nil
	}

	s.setTable()
	_, err := s.initModel(s.db)
	if err != nil {
		return err
	}

	// Marked before the relations: a master is registered on both models, so initializing
	// it calls back into this model, which must not start over.
	s.IsInit = true
	for _, detail := range s.Details {
		err = detail.init()
		if err != nil {
			s.IsInit = false
			return err
		}
	}

	for _, master := range s.Masters {
		err = master.init()
		if err != nil {
			s.IsInit = false
			return err
		}
	}

	return nil
}

/**
* stricted: Enables strict mode — unknown field names are not treated as ATTRIBs.
**/
func (s *Model) stricted() {
	s.IsStrict = true
}

/**
* getDb: Returns the DB the model belongs to.
* @return *DB
**/
func (s *Model) getDb() *DB {
	return s.db
}

/**
* setDb: Sets the primary DB connection for the model.
* @param db *DB
**/
func (s *Model) setDb(db *DB) {
	s.IsInit = false
	s.db = db
}

/**
* sqlDB: Returns the underlying *sql.DB connection pool.
* @return *sql.DB
**/
func (s *Model) sqlDB() *sql.DB {
	return s.db.db
}

/**
* getModel: Returns the model for the given schema and name.
* @param schema string
* @param name string
* @return *Model, error
**/
func (s *Model) getModel(schema, name string) (*Model, error) {
	return s.db.GetModel(schema, name)
}

/**
* newColumn: Constructs a Column bound to this model without adding it to the Columns slice.
* @param name string, tpColumn TypeColumn, tpData et.TypeData, defaultValue interface{}, definition []byte
* @return *Column
**/
func (s *Model) newColumn(name string, tpColumn TypeColumn, tpData et.TypeData, deFault any) *Column {
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
* getColumn: Returns the Column for the given name. For non-strict models with a SourceField,
* unknown names are returned as synthetic ATTRIB columns.
* @param name string
* @return *Column
**/
func (s *Model) getColumn(name string) (*Column, bool) {
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

	return s.newColumn(name, ATTRIB, et.ANY, ""), true
}

/**
* getField: Returns the Field for the given name.
* @param name string
* @return *Field, bool
**/
func (s *Model) getField(name string) (*Field, bool) {
	col, ok := s.GetColumn(name)
	if !ok {
		return nil, false
	}
	return &Field{
		Field:      et.Field{Name: name},
		TypeColumn: col.TypeColumn,
		TypeData:   col.TypeData,
		From:       getFrom(s, ""),
	}, true
}

/**
* beforeInsert: Registers a trigger function to run before each INSERT.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) beforeInsert(fn TriggerFunction) *Model {
	s.beforeInserts = append(s.beforeInserts, fn)
	return s
}

/**
* beforeUpdate: Registers a trigger function to run before each UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) beforeUpdate(fn TriggerFunction) *Model {
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* beforeDelete: Registers a trigger function to run before each DELETE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) beforeDelete(fn TriggerFunction) *Model {
	s.beforeDeletes = append(s.beforeDeletes, fn)
	return s
}

/**
* beforeInsertOrUpdate: Registers a trigger function to run before INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) beforeInsertOrUpdate(fn TriggerFunction) *Model {
	s.beforeInserts = append(s.beforeInserts, fn)
	s.beforeUpdates = append(s.beforeUpdates, fn)
	return s
}

/**
* afterInsert: Registers a trigger function to run after each INSERT.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) afterInsert(fn TriggerFunction) *Model {
	s.afterInserts = append(s.afterInserts, fn)
	return s
}

/**
* afterUpdate: Registers a trigger function to run after each UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) afterUpdate(fn TriggerFunction) *Model {
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* afterDelete: Registers a trigger function to run after each DELETE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) afterDelete(fn TriggerFunction) *Model {
	s.afterDeletes = append(s.afterDeletes, fn)
	return s
}

/**
* afterInsertOrUpdate: Registers a trigger function to run after INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) afterInsertOrUpdate(fn TriggerFunction) *Model {
	s.afterInserts = append(s.afterInserts, fn)
	s.afterUpdates = append(s.afterUpdates, fn)
	return s
}

/**
* getFrom: Returns the From for the model.
* @return *From
**/
func (s *Model) getFrom() *From {
	return getFrom(s, "")
}

/**
* as: Creates a new Query for this model with the given alias.
* @param as ...string
* @return *Query
**/
func (s *Model) as(as ...string) *Query {
	result := NewQuery(s, as...)
	return result
}

/**
* detail: Returns the detail for the given name.
* @param name string
* @return *Model, bool
**/
func (s *Model) detail(name string) (*Model, bool) {
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
* master: Returns the master for the given name.
* @param name string
* @return *Query, bool
**/
func (s *Model) master(name string) (*Query, bool) {
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
* bridge: Returns the bridge for the given name.
* @param name string
* @return *Model, bool
**/
func (s *Model) bridge(name string) (*Model, bool) {
	master, ok := s.Masters[name]
	if !ok {
		return nil, false
	}

	return master.Bridge.Model, true
}

/**
* join: Creates a new Query for this model with the given model as the INNER JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Model) join(to *Model, as string, on []*et.Condition) *Query {
	result := s.As("A")
	result.Join(to, as, on)
	return result
}

/**
* selects: Creates a new Query for this model with the given fields as the SELECT clause.
* @param fields ...string
* @return *Query
**/
func (s *Model) selects(fields ...string) *Query {
	result := s.As("")
	result.Select(fields...)
	return result
}

/**
* calc: Creates a new Query for this model with the given fields as the CALC clause.
* @param fields ...string
* @return *Query
**/
func (s *Model) calc(fields ...string) *Query {
	result := s.As("")
	result.Calc(fields...)
	return result
}

/**
* where: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param cond *et.Condition
* @return *Query
**/
func (s *Model) where(cond *et.Condition) *Query {
	result := s.As("")
	result.Where(cond)
	return result
}

/**
* whereByJson: Creates a new Query for this model with the given conditions as the WHERE clause.
* @param cond []et.Json
* @return *Query
**/
func (s *Model) whereByJson(cond []et.Json) *Query {
	result := s.As("")
	result.WhereByJson(cond)
	return result
}

/**
* count: Returns the count of records in the model.
* @return (int, error)
**/
func (s *Model) count() (int, error) {
	return s.
		As().
		Count()
}

/**
* first: Returns the first record in the model.
* @return (et.Item, error)
**/
func (s *Model) first(n int) (et.Items, error) {
	return s.
		As().
		Limit(1, n)
}

/**
* insert: Creates a Command of type INSERT pre-loaded with the given data row.
* @param data et.Json
* @return *Command
**/
func (s *Model) insert(data et.Json) *Command {
	result := newCommand(s, INSERT)
	result.Data = append(result.Data, data)
	return result
}

/**
* bulk: Creates a Command of type BULK pre-loaded with multiple data rows.
* @param data []et.Json
* @return *Command
**/
func (s *Model) bulk(data []et.Json) *Command {
	result := newCommand(s, BULK)
	result.Data = data
	return result
}

/**
* update: Creates a Command of type UPDATE pre-loaded with the given data row.
* @param data et.Json
* @return *Command
**/
func (s *Model) update(data et.Json) *Command {
	result := newCommand(s, UPDATE)
	result.Data = append(result.Data, data)
	return result
}

/**
* delete: Creates a Command of type DELETE (conditions must be added with Where/And).
* @return *Command
**/
func (s *Model) delete() *Command {
	result := newCommand(s, DELETE)
	return result
}

/**
* upsert: Creates a Command of type UPSERT pre-loaded with data and primary-key conditions.
* @param data et.Json
* @return *Command
**/
func (s *Model) upsert(data et.Json) *Command {
	result := newCommand(s, UPSERT)
	result.Data = append(result.Data, data)
	return result
}

/**
* query: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param query et.Json
* @return *Query
**/
func (s *Model) query(query et.Json) *Query {
	result := s.As("")
	_, result.err = result.loadQuery(query)
	return result
}

/**
* defaultTrigger: Registers the triggers that enforce the unique fields of the model (DefineUnique)
* before each insert and update. The check runs inside the command's transaction, so it also sees
* the rows written earlier in the same transaction (e.g. a bulk insert) and does not open another
* connection (SQLite has a single writer connection). A duplicate returns ErrRecordAlreadyExists
* with the field and value. The unique index of the database remains as the final guarantee.
* @return *Model
**/
func (s *Model) defaultTrigger() *Model {
	s.BeforeInsert(func(tx *Tx, old, new et.Json) error {
		for _, unique := range s.Unique {
			value, ok := new[unique.Name]
			if !ok || isEmptyUnique(value) {
				continue
			}
			exists, err := s.Where(Eq(unique.Name, value)).ExistsTx(tx)
			if err != nil {
				return err
			}
			if exists {
				return uniqueError(unique.Name, value)
			}
		}
		return nil
	})

	s.BeforeUpdate(func(tx *Tx, old, new et.Json) error {
		for _, unique := range s.Unique {
			value, ok := new[unique.Name]
			if !ok || isEmptyUnique(value) || !old.IsDifferent(unique.Name, value) {
				continue
			}
			// Other rows with the value; the row being updated is told apart by its primary key.
			rows, err := s.Where(Eq(unique.Name, value)).LimitTx(tx, 1, 2)
			if err != nil {
				return err
			}
			for _, row := range rows.Result {
				if !s.samePrimaryKey(row, old) {
					return uniqueError(unique.Name, value)
				}
			}
		}
		return nil
	})

	return s
}

/**
* samePrimaryKey: Reports whether two records have the same primary key values.
* @param a, b et.Json
* @return bool
**/
func (s *Model) samePrimaryKey(a, b et.Json) bool {
	if len(s.PrimaryKeys) == 0 {
		return false
	}
	for _, pk := range s.PrimaryKeys {
		if fmt.Sprint(a[pk.Name]) != fmt.Sprint(b[pk.Name]) {
			return false
		}
	}
	return true
}

/**
* isEmptyUnique: Reports whether a unique value is empty (nil or ""), which is not checked.
* @param value any
* @return bool
**/
func isEmptyUnique(value any) bool {
	return value == nil || value == ""
}

/**
* uniqueError: Returns ErrRecordAlreadyExists with the duplicated field and value.
* @param field string, value any
* @return error
**/
func uniqueError(field string, value any) error {
	return fmt.Errorf("%w: %s = %v", ErrRecordAlreadyExists, field, value)
}

/**
* withoutOmitUpdates: Returns data without the fields of OmitUpdates (and their nested paths "field->…").
* @param data et.Json
* @return et.Json
**/
func (s *Model) withoutOmitUpdates(data et.Json) et.Json {
	if len(s.OmitUpdates) == 0 {
		return data
	}
	result := et.Json{}
	for key, value := range data {
		if s.isOmitUpdate(key) {
			continue
		}
		result[key] = value
	}
	return result
}

/**
* isOmitUpdate: Reports whether key is an OmitUpdates field or a nested path of one.
* @param key string
* @return bool
**/
func (s *Model) isOmitUpdate(key string) bool {
	for _, name := range s.OmitUpdates {
		if key == name || strings.HasPrefix(key, name+"->") {
			return true
		}
	}
	return false
}

/**
* restoreOmitUpdates: Puts back in new the value that old had for each OmitUpdates field (or removes it
* when old did not have it), undoing the changes a before-update trigger made to them.
* @param old, new et.Json
**/
func (s *Model) restoreOmitUpdates(old, new et.Json) {
	for _, name := range s.OmitUpdates {
		if value, ok := old[name]; ok {
			new[name] = value
			continue
		}
		delete(new, name)
	}
	for key := range new {
		if strings.Contains(key, "->") && s.isOmitUpdate(key) {
			delete(new, key)
		}
	}
}
