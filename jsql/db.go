package jsql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/msg"
	"github.com/cgalvisleon/et/utility"
)

type DB struct {
	Name        string             `json:"name"`
	Schemas     map[string]*Schema `json:"schemas"`
	Driver      string             `json:"driver"`
	Params      utility.Config     `json:"params"`
	UseCore     bool               `json:"use_core"`
	RecordLimit int                `json:"record_limit"`
	IsDebug     bool               `json:"-"`
	IsChanged   bool               `json:"-"`
	driver      Driver             `json:"-"`
	db          *sql.DB            `json:"-"`
}

/**
* newDB: Constructs a DB instance from the given config, resolving the driver by name.
* @param params utility.Config
* @return *DB, error
**/
func newDB(params utility.Config) (*DB, error) {
	driverName := params.GetStr("DB_DRIVER", DriverPostgres)
	if !utility.ValidStr(driverName, 2, []string{""}) {
		return nil, errors.New(msg.MSG_DRIVER_NOT_FOUND)
	}
	driver, ok := drivers[driverName]
	if !ok {
		return nil, errors.New(msg.MSG_DRIVER_NOT_FOUND)
	}

	name := params.GetStr("DB_NAME", "test")
	useCore := params.GetBool("DB_USE_CORE", false)
	recordLimit := params.GetInt("DB_RECORD_LIMIT", 1000)
	result := &DB{
		Name:        name,
		Schemas:     make(map[string]*Schema),
		Driver:      driverName,
		Params:      params,
		UseCore:     useCore,
		RecordLimit: recordLimit,
		driver:      driver,
	}
	return result, nil
}

/**
* serialize: Marshals the DB metadata to JSON bytes.
* @return []byte, error
**/
func (s *DB) serialize() ([]byte, error) {
	bt, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}

	return bt, nil
}

/**
* ToJson: Returns the DB metadata as an et.Json map.
* @return et.Json
**/
func (s *DB) ToJson() et.Json {
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
* save: Persists DB metadata changes (stub — no-op until storage is wired).
* @return error
**/
func (s *DB) save() error {
	return nil
}

/**
* init: Opens the driver connection and, when UseCore is set, initializes core tables.
* @return error
**/
func (s *DB) init() error {
	if s.db != nil {
		return nil
	}

	if s.driver == nil {
		return errors.New(msg.MSG_DRIVER_NOT_FOUND)
	}

	db, err := s.driver.Connect(s)
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

	if s.IsChanged {
		return s.save()
	}

	return nil
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
			models:   make(map[string]*Model),
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
* getSchema: Returns the named Schema or an error if it does not exist.
* @param name string
* @return *Schema, error
**/
func (s *DB) getSchema(name string) (*Schema, error) {
	result, ok := s.Schemas[name]
	if ok {
		return result, nil
	}

	return nil, fmt.Errorf(msg.MSG_SCHEMA_NOT_FOUND, name)
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

	result, err := sch.GetModel(name)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* sqlTx: Executes a SQL query inside the given transaction (or directly on the pool if nil).
* @param tx *Tx
* @param query string
* @param arg ...any
* @return et.Items, error
**/
func (s *DB) sqlTx(tx *Tx, query string, arg ...any) (et.Items, error) {
	query = SQLParse(query, arg...)
	if tx != nil {
		err := tx.begin(s.db)
		if err != nil {
			return et.Items{}, err
		}

		rows, err := tx.Tx.Query(query)
		if err != nil {
			errR := tx.rollback()
			if errR != nil {
				err = fmt.Errorf(msg.MSG_ROLLBACK_ERROR, errR)
			}
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
* load: Generates DDL for the model via the driver and executes it against the DB.
* @param model *Model
* @return error
**/
func (s *DB) load(model *Model) error {
	if s.driver == nil {
		return errors.New(msg.MSG_DRIVER_NOT_FOUND)
	}

	sql, err := s.driver.Load(model)
	if err != nil {
		return err
	}

	if model.IsDebug {
		logs.Debug("DDL:\n", sql)
		return nil
	}

	if !model.isTest {
		_, err = s.sqlTx(nil, sql)
		if err != nil {
			return err
		}
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
		return "", errors.New(msg.MSG_DRIVER_NOT_FOUND)
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
		return "", errors.New(msg.MSG_DRIVER_NOT_FOUND)
	}

	if s.IsDebug {
		logs.Debugf("query:%s", query.ToJson().ToEscapeHTML())
	}

	return s.driver.Query(query)
}

/**
* Define: Creates a model from a declarative definition (delegates to DefineModel).
* @param definition Define
* @return *Model, error
**/
func (s *DB) Define(define Define) (*Model, error) {
	if !utility.ValidStr(define.Schema, 0, []string{}) {
		return nil, errors.New(msg.MSG_SCHEMA_REQUIRED)
	}
	if !utility.ValidStr(define.Name, 0, []string{}) {
		return nil, errors.New(msg.MSG_NAME_REQUIRED)
	}
	if define.Version <= 0 {
		define.Version = 1
	}

	result, err := s.NewModel(define.Schema, define.Name, define.Version)
	if err != nil {
		return nil, err
	}

	for _, column := range define.Columns {
		result.defineColumn(column.Name, column.TypeColumn, column.TypeData, column.Default, column.Definition)
	}

	if define.SourceField != "" {
		result.defineSource()
	}

	if define.IdxField != "" {
		result.defineIdxField()
	}
	for _, primaryKey := range define.PrimaryKeys {
		result.DefinePrimaryKey(primaryKey.Name, primaryKey.TypeData, primaryKey.Default)
	}
	for _, foreignKey := range define.ForeignKeys {
		to, err := s.GetModel(foreignKey.To.Schema, foreignKey.To.Name)
		if err != nil {
			return nil, err
		}
		result.DefineForeignKeys(to, foreignKey.Keys, foreignKey.OnDeleteCascade, foreignKey.OnUpdateCascade)
	}
	for _, index := range define.Indexes {
		result.DefineIndex(index.Name, index.TypeData, index.Default)
	}
	for _, unique := range define.Unique {
		result.DefineUnique(unique.Name, unique.TypeData, unique.Default)
	}
	for _, required := range define.Required {
		result.DefineRequired(required.Name, required.TypeData, required.Default)
	}
	for _, hidden := range define.Hiddens {
		result.DefineHidden(hidden)
	}
	for _, detail := range define.Details {
		result.DefineDetail(detail.Name, detail.Keys)
	}
	for _, rollup := range define.Rollups {
		to, err := s.GetModel(rollup.To.Schema, rollup.To.Name)
		if err != nil {
			return nil, err
		}
		result.DefineRollup(rollup.Name, to, rollup.Keys, rollup.Select)
	}
	for _, relation := range define.Relations {
		result.DefineRelation(relation.Name, nil, relation.Keys)
	}

	return result, nil
}
