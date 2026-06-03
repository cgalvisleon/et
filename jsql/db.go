package jsql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
)

type DB struct {
	Name        string             `json:"name"`
	Schemas     map[string]*Schema `json:"schemas"`
	Driver      string             `json:"driver"`
	Params      et.Json            `json:"params"`
	UseCore     bool               `json:"use_core"`
	RecordLimit int                `json:"record_limit"`
	Version     int                `json:"version"`
	IsDebug     bool               `json:"-"`
	IsChanged   bool               `json:"-"`
	isInit      bool               `json:"-"`
	driver      Driver             `json:"-"`
	db          *sql.DB            `json:"-"`
	catalog     *Model             `json:"-"`
	series      *Model             `json:"-"`
}

/**
* newDB: Constructs a DB instance from the given config, resolving the driver by name.
* @param params et.Json
* @return *DB, error
**/
func newDB(params et.Json) (*DB, error) {
	driverName := params.Str("driver")
	if !utility.ValidStr(driverName, 2, []string{""}) {
		return nil, errors.New(MSG_DRIVER_NOT_FOUND)
	}
	driver, ok := drivers[driverName]
	if !ok {
		return nil, errors.New(MSG_DRIVER_NOT_FOUND)
	}

	name := params.Str("database")
	useCore := params.Bool("use_core")
	recordLimit := params.Int("record_limit")
	version := params.ValInt(1, "version")
	result := &DB{
		Name:        name,
		Schemas:     make(map[string]*Schema),
		Driver:      driverName,
		Params:      params,
		UseCore:     useCore,
		RecordLimit: recordLimit,
		Version:     version,
		driver:      driver,
	}
	return result, nil
}

/**
* ToJson: Returns the DB metadata as an et.Json map.
* @return et.Json
**/
func (s *DB) ToJson() et.Json {
	bt, err := json.Marshal(s)
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
* save: Persists DB metadata changes (stub — no-op until storage is wired).
* @return error
**/
func (s *DB) save() error {
	return s.setCatalog(s.Name, "db", s.Version, s)
}

/**
* init: Opens the driver connection and, when UseCore is set, initializes core tables.
* @return error
**/
func (s *DB) init() error {
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
	if s.UseCore {
		err := s.initCore()
		if err != nil {
			return err
		}
	}

	s.isInit = true
	if s.IsChanged {
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
func (s *DB) existModel(schema string, name string) (bool, error) {
	if s.driver == nil {
		return false, errors.New(MSG_DRIVER_NOT_FOUND)
	}

	return s.driver.ExistModel(s.db, schema, name)
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

	if model.isTest {
		return nil
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

	if s.IsDebug {
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

	if s.IsDebug {
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
* NewModel: Returns (or creates) a Model under the given schema name.
* @param schema string
* @param name string
* @param version int
* @return *Model, error
**/
func (s *DB) NewModel(schema, name string, version int) (*Model, error) {
	schema = utility.Normalize(schema)
	sch, ok := s.Schemas[schema]
	if !ok {
		sch = &Schema{
			Database: s.Name,
			Name:     schema,
			Models:   make(map[string]*Model),
			db:       s,
			mu:       &sync.RWMutex{},
		}
		s.Schemas[schema] = sch
	}

	result, err := sch.newModel(name, version)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* SetDebug: Sets the debug flag to the given value.
* @param debug bool
**/
func (s *DB) SetDebug(debug bool) {
	s.IsDebug = debug
}

/**
* Debug: Enables debug logging for all queries and commands.
**/
func (s *DB) Debug() {
	s.IsDebug = true
}

/**
* GetModel: Looks up a model by schema and name, returning an error if not found.
* @param schema string
* @param name string
* @return *Model, error
**/
func (s *DB) GetModel(schema string, name string) (*Model, error) {
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
* @param tx *Tx
* @param query string
* @param arg ...any
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
* @param query string
* @param args ...any
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

	result, err := s.NewModel(define.Schema, define.Name, define.Version)
	if err != nil {
		return nil, err
	}
	if define.IdxField != "" {
		result.DefineIdxField()
	}
	if define.IdtField != "" {
		result.DefineIdTField()
	}
	for _, column := range define.Columns {
		if column.Name == "" {
			return nil, fmt.Errorf(MSG_COLUMN_NAME_REQUIRED, result.Name)
		}
		if column.TypeColumn == "" {
			return nil, fmt.Errorf(MSG_TYPE_COLUMN_REQUIRED, result.Name)
		}
		if column.TypeData == "" {
			return nil, fmt.Errorf(MSG_TYPE_DATA_REQUIRED, result.Name)
		}
		result.defineColumn(column.Name, column.TypeColumn, column.TypeData, column.Default, column.Definition)
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
	for _, detail := range define.Details {
		_, err := result.DefineDetail(detail.Name, detail.Keys, detail.Rows)
		if err != nil {
			return nil, err
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
	result.IsDebug = define.IsDebug
	result.isTest = define.IsTest

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
		err := tx.commit()
		if err != nil {
			return et.Items{}, err
		}
	}

	return result, nil
}
