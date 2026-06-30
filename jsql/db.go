package jsql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/event"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type Store interface {
	Set(collection, id, ownerId string, obj any) error
	Get(collection, id string, dest any) (bool, error)
	Delete(collection, id string) error
	Query(query et.Json) (et.Items, error)
}

type DB struct {
	TenantId    string             `json:"tenant_id"`
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Schemas     map[string]*Schema `json:"schemas"`
	Driver      string             `json:"driver"`
	Params      et.Json            `json:"params"`
	RecordLimit int                `json:"record_limit"`
	Version     int                `json:"version"`
	AuditLog    []et.Json          `json:"audit_log"`
	isDebug     bool               `json:"-"`
	isChanged   bool               `json:"-"`
	isInit      bool               `json:"-"`
	driver      Driver             `json:"-"`
	db          *sql.DB            `json:"-"`
	store       Store              `json:"-"`
}

/**
* newDB: Constructs a DB instance from the given config, resolving the driver by name.
* @param params et.Json
* @return *DB, error
**/
func newDB(tenantId, name string, params et.Json, store Store) (*DB, error) {
	driver := params.Str("driver")
	drv, ok := drivers[driver]
	if !ok {
		return nil, errors.New(MSG_DRIVER_NOT_FOUND)
	}

	recordLimit := params.Int("record_limit")
	version := params.ValInt(1, "version")
	result := &DB{
		TenantId:    tenantId,
		ID:          reg.UUID(),
		Name:        name,
		Schemas:     make(map[string]*Schema),
		Driver:      driver,
		Params:      params,
		RecordLimit: recordLimit,
		Version:     version,
		AuditLog:    make([]et.Json, 0),
		driver:      drv,
		store:       store,
	}
	err := result.save()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* loadDB: Loads a DB instance from the given definition.
* @param def et.Json
* @return *DB, error
**/
func (s *DB) Load() error {
	if s.store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	var def et.Json
	exists, err := s.store.Get("db", s.ID, &def)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New(MSG_DB_NOT_FOUND)
	}

	s.TenantId = def.Str("tenant_id")
	s.Name = def.Str("name")
	s.Driver = def.Str("driver")
	s.Params = def.Json("params")
	s.RecordLimit = def.Int("record_limit")
	s.Version = def.Int("version")
	s.AuditLog = def.ArrayJson("audit_log")
	s.Schemas = make(map[string]*Schema)

	schemas := def.Json("schemas")
	for name := range schemas {
		schema, ok := s.Schemas[name]
		if !ok {
			schema = s.newSchema(name)
			schema.up(s)
		}
		def := schemas.Json(name)
		models := def.Json("models")
		for modelId := range models {
			_, err := schema.loadModel(modelId)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

/**
* addAuditLog: Adds an audit log entry to the DB.
* @param userId string, action string
**/
func (s *DB) addAuditLog(userId string, action string) {
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
* Ref: Returns the DB metadata as an et.Json map.
* @return et.Json
**/
func (s *DB) Ref() et.Json {
	return et.Json{
		"id":   s.ID,
		"name": s.Name,
	}
}

/**
* ToJson: Returns the DB metadata as an et.Json map.
* @return et.Json
**/
func (s *DB) ToJson() et.Json {
	return et.Json{
		"tenant_id":    s.TenantId,
		"id":           s.ID,
		"name":         s.Name,
		"schemas":      s.Schemas,
		"driver":       s.Driver,
		"params":       s.Params,
		"record_limit": s.RecordLimit,
		"version":      s.Version,
		"audit_log":    s.AuditLog,
	}
}

/**
* ToString: Returns the DB metadata as a string.
* @return string
**/
func (s *DB) ToString() string {
	return s.ToJson().ToString()
}

/**
* save: Persists DB metadata changes (stub — no-op until storage is wired).
* @return error
**/
func (s *DB) save() error {
	if s.store == nil {
		return nil
	}

	err := s.store.Set("db", s.ID, s.TenantId, s.Ref())
	if err != nil {
		return err
	}

	json := s.ToJson()
	channel := fmt.Sprintf("db:%s", s.ID)
	event.Publish(channel, json)
	return nil
}

/**
* init: Opens the driver connection and, when UseCore is set, initializes core tables.
* @return error
**/
func (s *DB) Init() error {
	if s.isInit {
		return nil
	}

	if s.db != nil {
		return nil
	}

	if s.driver == nil {
		return errors.New(MSG_DRIVER_NOT_FOUND)
	}

	db, err := s.driver.Connect(context.Background(), s)
	if err != nil {
		return err
	}

	s.db = db
	s.isInit = true
	if s.isChanged {
		return s.save()
	}

	return nil
}

/**
* getSchema: Returns the named Schema or an error if it does not exist.
* @param name string
* @return *Schema, error
**/
func (s *DB) getSchema(name string) (*Schema, error) {
	result, ok := s.Schemas[name]
	if ok {
		return result, nil
	}

	return nil, fmt.Errorf(MSG_SCHEMA_NOT_FOUND, name)
}

/**
* existModel: Checks if a table exists in the database.
* @param schema string, name string
* @return bool, error
**/
func (s *DB) existModel(model *Model) (bool, error) {
	if s.driver == nil {
		return false, errors.New(MSG_DRIVER_NOT_FOUND)
	}

	return s.driver.ExistModel(s.db, model)
}

/**
* load: Generates DDL for the model via the driver and executes it against the DB.
* @param model *Model
* @return error
**/
func (s *DB) load(model *Model) error {
	if s.driver == nil {
		return errors.New(MSG_DRIVER_NOT_FOUND)
	}

	sql, err := s.driver.Load(model)
	if err != nil {
		return err
	}

	if model.IsDebug {
		logs.Debug("DDL:\n", sql)
	}

	_, err = s.SqlTx(nil, sql)
	if err != nil {
		return err
	}

	return nil
}

/**
* command: Asks the driver to render a Command as SQL and returns the SQL string.
* @param command *Command
* @return string, error
**/
func (s *DB) command(command *Command) (string, error) {
	if s.driver == nil {
		return "", errors.New(MSG_DRIVER_NOT_FOUND)
	}

	if s.isDebug {
		logs.Debugf("command:%s", command.ToJson().ToEscapeHTML())
	}

	return s.driver.Command(command)
}

/**
* query: Asks the driver to render a Query as SQL and returns the SQL string.
* @param query *Query
* @return string, error
**/
func (s *DB) query(query *Query) (string, error) {
	if s.driver == nil {
		return "", errors.New(MSG_DRIVER_NOT_FOUND)
	}

	if s.isDebug {
		logs.Debugf("query:%s", query.ToJson().ToEscapeHTML())
	}

	return s.driver.Query(query)
}

/**
* Close: Closes the underlying *sql.DB connection pool.
* @return error
**/
func (s *DB) Close() error {
	return s.db.Close()
}

/**
* newSchema: Constructs a new Schema with initialized fields.
* @param name string
* @return *Schema
**/
func (s *DB) newSchema(name string) *Schema {
	result := &Schema{
		TenantId: s.TenantId,
		Database: s.Name,
		Name:     name,
		Models:   make(map[string]*Model),
		db:       s,
		isDebug:  s.isDebug,
		mu:       &sync.RWMutex{},
		store:    s.store,
	}
	s.Schemas[name] = result
	return result
}

/**
* NewModel: Returns (or creates) a Model under the given schema name.
* @param schema, name string, version int, userId string
* @return *Model, error
**/
func (s *DB) NewModel(schema, name string, version int, userId string) *Model {
	schema = utility.Normalize(schema)
	sch, ok := s.Schemas[schema]
	if !ok {
		sch = s.newSchema(schema)
	}

	result := sch.newModel(name, version, userId)
	return result
}

/**
* SetDebug: Sets the debug flag to the given value.
* @param debug bool
**/
func (s *DB) SetDebug(debug bool) {
	s.isDebug = debug
}

/**
* Debug: Enables debug logging for all queries and commands.
**/
func (s *DB) Debug() {
	s.isDebug = true
}

/**
* GetModel: Looks up a model by schema and name, returning an error if not found.
* @param schema, name string
* @return *Model, error
**/
func (s *DB) GetModel(schema, name string) (*Model, error) {
	sch, err := s.getSchema(schema)
	if err != nil {
		return nil, err
	}

	result, err := sch.getModel(name)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* SqlTx: Executes a SQL query inside the given transaction (or directly on the pool if nil).
* @param tx *Tx, query string, args ...any
* @return et.Items, error
**/
func (s *DB) SqlTx(tx *Tx, query string, arg ...any) (et.Items, error) {
	query = SQLParse(query, arg...)
	if tx != nil {
		rows, err := tx.Query(s.db, query)
		if err != nil {
			return et.Items{}, err
		}

		result := RowsToItems(rows)
		return result, nil
	}

	rows, err := s.db.Query(query)
	if err != nil {
		return et.Items{}, err
	}

	result := RowsToItems(rows)
	return result, nil
}

/**
* Sql: Executes a SQL query directly on the DB (no transaction).
* @param query string, args ...any
* @return et.Items, error
**/
func (s *DB) Sql(query string, args ...any) (et.Items, error) {
	return s.SqlTx(nil, query, args...)
}

/**
* Define: Creates a model from a declarative definition (delegates to DefineModel).
* @param definition Def
* @return *Model, error
**/
func (s *DB) Define(define Def) (*Model, error) {
	if !utility.ValidStr(define.Schema, 0, []string{}) {
		return nil, errors.New(MSG_SCHEMA_REQUIRED)
	}
	if !utility.ValidStr(define.Name, 0, []string{}) {
		return nil, errors.New(MSG_NAME_REQUIRED)
	}
	if define.Version <= 0 {
		define.Version = 1
	}

	defColumns := func(model *Model, def Column) (*Column, error) {
		if def.Name == "" {
			return nil, fmt.Errorf(MSG_COLUMN_NAME_REQUIRED, "name")
		}
		if def.TypeColumn == "" {
			return nil, fmt.Errorf(MSG_TYPE_COLUMN_REQUIRED, "type_column")
		}
		if def.TypeData == "" {
			return nil, fmt.Errorf(MSG_TYPE_DATA_REQUIRED, "type_data")
		}
		result := model.defineColumn(def.Name, def.TypeColumn, def.TypeData, def.Default, def.Definition)
		return result, nil
	}

	result := s.NewModel(define.Schema, define.Name, define.Version, define.UserId)
	if define.IdxField != "" {
		result.DefineIdxField()
	}
	if define.IdtField != "" {
		result.DefineIdTField()
	}
	for _, column := range define.Columns {
		_, err := defColumns(result, column)
		if err != nil {
			return nil, err
		}
	}
	for _, primaryKey := range define.PrimaryKeys {
		result.PrimaryKeys = append(result.PrimaryKeys, &Index{
			Name:   primaryKey.Name,
			Sorted: primaryKey.Sorted,
		})
	}
	for _, index := range define.Indexes {
		result.Indexes = append(result.Indexes, &Index{
			Name:   index.Name,
			Sorted: index.Sorted,
		})
	}
	for _, unique := range define.Unique {
		result.Unique = append(result.Unique, &Index{
			Name:   unique.Name,
			Sorted: unique.Sorted,
		})
	}
	for _, required := range define.Required {
		result.Required = append(result.Required, &Index{
			Name:   required.Name,
			Sorted: required.Sorted,
		})
	}
	for _, foreignKey := range define.ForeignKeys {
		to, err := s.GetModel(foreignKey.To.Schema, foreignKey.To.Name)
		if err != nil {
			return nil, err
		}
		result.DefineForeignKeys(to, foreignKey.Keys, foreignKey.OnDeleteCascade, foreignKey.OnUpdateCascade)
	}
	for _, hidden := range define.Hiddens {
		result.DefineHidden(hidden)
	}
	if define.SourceField != "" {
		result.DefineSource()
	}
	for _, defDetail := range define.Details {
		detail, err := result.DefineDetail(defDetail.Name, defDetail.Keys, defDetail.Rows)
		if err != nil {
			return nil, err
		}
		for _, def := range defDetail.Columns {
			_, err := defColumns(detail, def)
			if err != nil {
				return nil, err
			}
		}
		if defDetail.IdxField != "" {
			detail.DefineIdxField()
		}
		if defDetail.IdtField != "" {
			detail.DefineIdTField()
		}
	}
	for _, rollup := range define.Rollups {
		to, err := s.GetModel(rollup.To.Schema, rollup.To.Name)
		if err != nil {
			return nil, err
		}
		_, err = result.DefineRollup(rollup.Name, to, rollup.Keys, rollup.Select)
		if err != nil {
			return nil, err
		}
	}
	result.IsCore = define.IsCore
	result.IsDebug = s.isDebug

	return result, nil
}

/**
* loadQuery: Creates a Query from a JSON object.
* @param tx *Tx, query et.Json
* @return *Query, error
**/
func (s *DB) loadQuery(tx *Tx, query et.Json) (et.Items, error) {
	define := query.ArrayJson("define")
	if len(define) > 0 {
		results := et.Items{}
		for _, d := range define {
			bt := []byte(d.ToString())
			def := Def{}
			err := json.Unmarshal(bt, &def)
			if err != nil {
				return et.Items{}, err
			}
			model, err := s.Define(def)
			if err != nil {
				return et.Items{}, err
			}
			if err := model.Init(); err != nil {
				return et.Items{}, err
			}
			results.Add(et.Json{"model": model.Name})
		}
		return results, nil
	}

	from := query.Str("from")
	as := ""
	args, ok := ArgWhitAs(from)
	if ok {
		from = args[0]
		as = args[1]
	}
	args, ok = ArgWhitSchema(from)
	if ok {
		return et.Items{}, fmt.Errorf(MSG_INVALID_FROM, from)
	}
	schema := args[0]
	table := args[1]
	model, err := s.GetModel(schema, table)
	if err != nil {
		return et.Items{}, fmt.Errorf(MSG_MODEL_NOT_FOUND, from)
	}

	insert := query.Json("insert")
	if insert.IsEmpty() {
		command := model.Insert(insert)
		return command.loadQuery(tx, query)
	}

	update := query.Json("update")
	if update.IsEmpty() {
		command := model.Update(update)
		return command.loadQuery(tx, query)
	}

	delete := query.Json("delete")
	if delete.IsEmpty() {
		command := model.Delete()
		return command.loadQuery(tx, delete)
	}

	upsert := query.Json("upsert")
	if upsert.IsEmpty() {
		command := model.Upsert(upsert)
		return command.loadQuery(tx, query)
	}

	q := newQuery(model, as)
	return q.loadQuery(tx, query)
}

/**
* Query: Renders a Query as SQL and returns the SQL string.
* @param query []et.Json
* @return string, error
**/
func (s *DB) Query(query []et.Json) (et.Items, error) {
	result := et.Items{}
	var tx *Tx
	var commit bool
	if len(query) > 0 {
		tx, commit = getTx(tx)
	}

	for _, q := range query {
		var err error
		result, err = s.loadQuery(tx, q)
		if err != nil {
			return et.Items{}, err
		}
	}

	if commit {
		defer tx.Commit()
	}

	return result, nil
}
