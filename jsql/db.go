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
}

/**
* NewDB
* @param tenantId, host, name, driver string
* @return *DB, error
**/
func NewDB(tenantId, host, name, driver string) (*DB, error) {
	drv, ok := drivers[driver]
	if !ok {
		return nil, errors.New(MSG_DRIVER_NOT_FOUND)
	}

	if !utility.ValidStr(name, 0, []string{""}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "name")
	}

	if !utility.ValidStr(host, 0, []string{""}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "host")
	}

	connect, err := GetConnection(driver, host)
	if err != nil {
		return nil, err
	}

	connect.SetDatabase(name)
	params := connect.GetParams()
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
		isDebug:     envar.GetBool("DEBUG", false),
	}

	return result, nil
}

/**
* LoadDb
* @param store *Store, id string
* @return *DB, error
**/
func LoadDb(store *Store, id string) (*DB, error) {
	if store == nil {
		return nil, errors.New(MSG_STORE_IS_NIL)
	}

	ref := et.Json{}
	exists, err := store.Get("db", id, &ref)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf(MSG_RECORD_NOT_FOUND, "db", id)
	}

	recordLimit := envar.GetInt("RECORD_LIMIT", 1000)
	params := ref.Json("params")
	driver := params.Str("driver")
	drv, ok := drivers[driver]
	if !ok {
		return nil, errors.New(MSG_DRIVER_NOT_FOUND)
	}

	result := &DB{
		TenantId:    ref.Str("tenant_id"),
		ID:          ref.Str("id"),
		Name:        ref.Str("name"),
		Schemas:     make(map[string]*Schema),
		Driver:      ref.Str("driver"),
		Params:      params,
		RecordLimit: ref.ValInt(recordLimit, "record_limit"),
		Version:     ref.Int("version"),
		AuditLog:    ref.ArrayJson("audit_log"),
		isDebug:     envar.GetBool("DEBUG", false),
		driver:      drv,
	}

	if !utility.ValidStr(result.ID, 0, []string{""}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "id")
	}
	if !utility.ValidStr(result.Name, 0, []string{""}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "name")
	}

	schemas := ref.ArrayJson("schemas")
	for _, schemaRef := range schemas {
		name := schemaRef.Str("name")
		schema, ok := result.Schemas[name]
		if !ok {
			schema = result.newSchema(name)
		}
		models := schemaRef.ArrayJson("models")
		for _, modelRef := range models {
			modelId := modelRef.Str("id")
			_, ok := schema.Models[modelId]
			if !ok {
				_, err := schema.loadModel(store, modelId)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return result, nil
}

/**
* saveDb
* @return error
**/
func (s *DB) Save(store *Store) error {
	if store == nil {
		return errors.New(MSG_STORE_IS_NIL)
	}

	err := store.Set("db", s.ID, s.TenantId, s.Ref())
	if err != nil {
		return err
	}

	json := s.ToJson()
	channel := fmt.Sprintf("db:%s", s.ID)
	event.Publish(channel, json)
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
	schemas := []et.Json{}
	for _, schema := range s.Schemas {
		schemas = append(schemas, schema.Ref())
	}

	return et.Json{
		"tenant_id":    s.TenantId,
		"id":           s.ID,
		"name":         s.Name,
		"schemas":      schemas,
		"params":       s.Params,
		"record_limit": s.RecordLimit,
		"version":      s.Version,
		"audit_log":    s.AuditLog,
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

	for _, schema := range s.Schemas {
		err := schema.init()
		if err != nil {
			return err
		}
	}

	s.isInit = true

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

		for _, primaryKey := range defDetail.PrimaryKeys {
			detail.PrimaryKeys = append(detail.PrimaryKeys, &Index{
				Name:   primaryKey.Name,
				Sorted: primaryKey.Sorted,
			})
		}

		for _, index := range defDetail.Indexes {
			detail.Indexes = append(detail.Indexes, &Index{
				Name:   index.Name,
				Sorted: index.Sorted,
			})
		}

		if defDetail.IdxField != "" {
			detail.DefineIdxField()
		}

		if defDetail.IdtField != "" {
			detail.DefineIdTField()
		}

		for _, rollup := range defDetail.Rollups {
			to, err := s.GetModel(rollup.To.Schema, rollup.To.Name)
			if err != nil {
				return nil, err
			}
			
			_, err = detail.DefineRollup(rollup.Name, to, rollup.Keys, rollup.Select)
			if err != nil {
				return nil, err
			}
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
		results := et.Items{Result: []et.Json{}}
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
			results.Add(et.Json{model.Table: model.ToJson()})
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
func (s *DB) Query(query []et.Json) ([][]et.Json, error) {
	result := [][]et.Json{}
	var tx *Tx
	var commit bool
	if len(query) > 0 {
		tx, commit = getTx(tx)
	}

	for _, q := range query {
		items, err := s.loadQuery(tx, q)
		if err != nil {
			return nil, err
		}
		result = append(result, items.Result)
	}

	if commit {
		defer tx.Commit()
	}

	return result, nil
}
