package jsql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jwf"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/reg"
	"github.com/cgalvisleon/et/timezone"
	"github.com/cgalvisleon/et/utility"
)

type DB struct {
	ID          string                                   `json:"id"`
	Host        string                                   `json:"host"`
	Driver      string                                   `json:"driver"`
	Name        string                                   `json:"name"`
	Params      et.Json                                  `json:"params"`
	Schemas     map[string]*Schema                       `json:"schemas"`
	RecordLimit int                                      `json:"record_limit"`
	AuditLog    *et.SafeData                             `json:"audit_log"`
	isDebug     bool                                     `json:"-"`
	isChanged   bool                                     `json:"-"`
	isInit      bool                                     `json:"-"`
	driver      Driver                                   `json:"-"`
	db          *sql.DB                                  `json:"-"`
	timeout     time.Duration                            `json:"-"`
	onAuditLog  func(userId string, action string) error `json:"-"`
	muSchemas   sync.RWMutex                             `json:"-"`
	muAudit     sync.Mutex                               `json:"-"`
}

/**
* newDB: Creates a new DB instance for the given driver without initializing it (call Init afterwards).
* @param id, host, name, driver string, showLog ...bool (optional, defaults to true)
* @return *DB, error
**/
func newDB(params ConnectParams) (*DB, error) {
	drv, ok := drivers[params.Driver]
	if !ok {
		return nil, errors.New(MSG_DRIVER_NOT_FOUND)
	}

	if !utility.ValidStr(params.Host, 0, []string{""}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "host")
	}

	if !utility.ValidStr(params.Name, 0, []string{""}) {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "name")
	}

	if params.Connection == nil {
		return nil, fmt.Errorf(MSG_ATRIB_REQUIRED, "connection")
	}

	connection := params.Connection.GetParams()
	id := connection.Str("id")
	result := &DB{
		ID:          reg.GetUUID(id),
		Host:        params.Host,
		Driver:      params.Driver,
		Name:        params.Name,
		Params:      connection,
		Schemas:     make(map[string]*Schema),
		RecordLimit: params.RecordLimit,
		AuditLog:    &et.SafeData{},
		isDebug:     envar.GetBool("DEBUG", false),
		driver:      drv,
		timeout:     params.Timeout,
	}

	return result, nil
}

/**
* loadDb: Rebuilds a DB from its ToJson: the connection from params (its kind in params.driver, so a driver registered
* under another name works too), the DB with newDB (driver, host, name, record_limit), its schemas and models, and its
* audit_log. It is not initialized: call Init to connect. A timeout of params (time.Duration in a ToJson) is taken in ms
* @param def et.Json
* @return *DB, error
**/
func loadDb(def et.Json) (*DB, error) {
	params := def.Json("params").Clone()
	if timeout, ok := params["timeout"].(time.Duration); ok {
		params["timeout"] = timeout.Milliseconds()
	}

	host := def.Str("host")
	kind := params.ValStr(def.Str("driver"), "driver")
	connection, err := ConnectionFromJson(kind, host, params)
	if err != nil {
		return nil, err
	}

	result, err := newDB(ConnectParams{
		Driver:      def.Str("driver"),
		Host:        host,
		Name:        def.Str("name"),
		Connection:  connection,
		RecordLimit: def.Int("record_limit"),
	})
	if err != nil {
		return nil, err
	}

	bt, err := json.Marshal(et.Json{"schemas": def["schemas"], "audit_log": def["audit_log"]})
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(bt, &result); err != nil {
		return nil, err
	}

	for _, schema := range result.Schemas {
		if err := schema.up(result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

/**
* setOnAuditLog: Sets the audit log function.
* @param fn func(userId string, action string) error
**/
func (s *DB) setOnAuditLog(fn func(userId string, action string) error) {
	s.onAuditLog = fn
}

/**
* addAuditLog: Adds an audit log entry to the DB.
* @param userId string, action string
**/
func (s *DB) addAuditLog(userId string, action string) {
	s.muAudit.Lock()
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
	onAuditLog := s.onAuditLog
	s.muAudit.Unlock()

	// Outside the lock: the hook may log again.
	if onAuditLog != nil {
		onAuditLog(userId, action)
	}
}

/**
* toJson: Returns the DB metadata as an et.Json map.
* @return et.Json
**/
func (s *DB) toJson() et.Json {
	schemas := et.Json{}
	for _, schema := range s.schemaList() {
		schemas[schema.Name] = schema.ToJson()
	}
	s.muAudit.Lock()
	auditLog := s.AuditLog
	s.muAudit.Unlock()

	return et.Json{
		"host":         s.Host,
		"driver":       s.Driver,
		"name":         s.Name,
		"params":       s.Params,
		"schemas":      schemas,
		"record_limit": s.RecordLimit,
		"audit_log":    auditLog.All(),
	}
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

	db, err := s.driver.Connect(s, s.timeout)
	if err != nil {
		return err
	}

	s.db = db

	for _, schema := range s.schemaList() {
		err := schema.init()
		if err != nil {
			return err
		}
	}

	s.isInit = true
	logs.Logf(s.Driver, "Connected host:%s:%d:%s", s.Host, s.Params.Int("port"), s.Name)

	return nil
}

/**
* getSchema: Returns the named Schema, creating it when it does not exist (to define models in it).
* @param name string
* @return *Schema
**/
func (s *DB) getSchema(name string) *Schema {
	name = utility.Normalize(name)
	s.muSchemas.Lock()
	defer s.muSchemas.Unlock()
	result, ok := s.Schemas[name]
	if ok {
		return result
	}

	return s.newSchema(name)
}

/**
* lookupSchema: Returns the named Schema without creating it.
* @param name string
* @return *Schema, bool
**/
func (s *DB) lookupSchema(name string) (*Schema, bool) {
	name = utility.Normalize(name)
	s.muSchemas.RLock()
	defer s.muSchemas.RUnlock()
	result, ok := s.Schemas[name]
	return result, ok
}

/**
* schemaList: Returns the schemas of the DB, a copy safe to range over while others are added.
* @return []*Schema
**/
func (s *DB) schemaList() []*Schema {
	s.muSchemas.RLock()
	defer s.muSchemas.RUnlock()
	result := make([]*Schema, 0, len(s.Schemas))
	for _, schema := range s.Schemas {
		result = append(result, schema)
	}
	return result
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

	return s.driver.ExistModel(s.db, model, s.timeout)
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

	sql, err := s.driver.Load(model, s.timeout)
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

	return s.driver.Command(command, s.timeout)
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

	return s.driver.Query(query, s.timeout)
}

/**
* close: Closes the underlying *sql.DB connection pool.
* @return error
**/
func (s *DB) close() error {
	return s.db.Close()
}

/**
* newSchema: Constructs a new Schema with initialized fields and adds it to the DB; muSchemas must be held.
* @param name string
* @return *Schema
**/
func (s *DB) newSchema(name string) *Schema {
	result := &Schema{
		database: s.Name,
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
* newModel: Returns (or creates) a Model under the given schema name.
* @param schema, name string, version int, userId string
* @return *Model, error
**/
func (s *DB) newModel(schema, name string, version int, userId string) *Model {
	sch := s.getSchema(schema)
	result := sch.newModel("", name, version, userId)
	return result
}

/**
* removeModel: Drops the model's table through the driver and removes the model from its schema.
* @param schema, name string
* @return error
**/
func (s *DB) removeModel(schema, name string) error {
	sch, ok := s.lookupSchema(schema)
	if !ok {
		return fmt.Errorf(MSG_SCHEMA_NOT_FOUND, utility.Normalize(schema))
	}

	model, err := sch.getModel(name)
	if err != nil {
		return err
	}

	if s.driver == nil {
		return errors.New(MSG_DRIVER_NOT_FOUND)
	}

	err = s.driver.Drop(model, s.timeout)
	if err != nil {
		return err
	}

	sch.removeModel(model.Name)
	return nil
}

/**
* setDebug: Sets the debug flag to the given value.
* @param debug bool
**/
func (s *DB) setDebug(debug bool) {
	s.isDebug = debug
}

/**
* debug: Enables debug logging for all queries and commands.
**/
func (s *DB) debug() {
	s.isDebug = true
}

/**
* getModel: Looks up a model by schema and name, returning an error if not found.
* @param schema, name string
* @return *Model, error
**/
func (s *DB) getModel(schema, name string) (*Model, error) {
	sch, ok := s.lookupSchema(schema)
	if !ok {
		return nil, fmt.Errorf(MSG_MODEL_NOT_FOUND, utility.Normalize(schema)+"."+utility.Normalize(name))
	}
	result, err := sch.getModel(name)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* sqlTx: Executes a SQL query inside the given transaction (or directly on the pool if nil).
* @param tx *Tx, query string, args ...any
* @return et.Items, error
**/
func (s *DB) sqlTx(tx *Tx, query string, arg ...any) (et.Items, error) {
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
* sql: Executes a SQL query directly on the DB (no transaction).
* @param query string, args ...any
* @return et.Items, error
**/
func (s *DB) sql(query string, args ...any) (et.Items, error) {
	return s.SqlTx(nil, query, args...)
}

/**
* define: Creates a model from a declarative definition (delegates to DefineModel).
* @param definition Define
* @return *Model, error
**/
func (s *DB) define(define Define) (*Model, error) {
	if !utility.ValidStr(define.ID, 0, []string{}) {
		define.ID = reg.UUID()
	}
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
		result := model.defineColumn(def.Name, def.TypeColumn, def.TypeData, def.Default)
		return result, nil
	}

	result := s.NewModel(define.Schema, define.Name, define.Version, define.UserId)
	if define.IdxField != "" {
		result.DefineIdxField()
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

	result.DefineOmitUpdate(define.OmitUpdates...)

	if define.SourceField != "" {
		result.DefineSource()
	}

	for _, defDetail := range define.Details {
		detail, err := result.DefineDetail(defDetail.Name, defDetail.Keys, defDetail.Rows, defDetail.Select...)
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

		for _, rollup := range defDetail.Rollups {
			to, err := s.GetModel(rollup.To.Schema, rollup.To.Name)
			if err != nil {
				return nil, err
			}

			where, err := et.ToWheRes(rollup.Where)
			if err != nil {
				return nil, err
			}

			defined, err := detail.DefineRollup(rollup.Name, to, rollup.Keys, rollup.Select, rollup.Operation, where...)
			if err != nil {
				return nil, err
			}
			if len(defined.OrdersBy) == 0 {
				for _, order := range rollup.OrderBy {
					defined.OrderBy(order.Name, order.Sorted)
				}
			}
		}
	}

	for _, defMaster := range define.Masters {
		to, err := s.GetModel(defMaster.To.Schema, defMaster.To.Name)
		if err != nil {
			return nil, err
		}

		_, err = result.DefineMaster(defMaster.Name, to, defMaster.Keys, defMaster.ToKeys, defMaster.Select, defMaster.Rows)
		if err != nil {
			return nil, err
		}
	}

	for _, rollup := range define.Rollups {
		to, err := s.GetModel(rollup.To.Schema, rollup.To.Name)
		if err != nil {
			return nil, err
		}

		where, err := et.ToWheRes(rollup.Where)
		if err != nil {
			return nil, err
		}

		defined, err := result.DefineRollup(rollup.Name, to, rollup.Keys, rollup.Select, rollup.Operation, where...)
		if err != nil {
			return nil, err
		}
		if len(defined.OrdersBy) == 0 {
			for _, order := range rollup.OrderBy {
				defined.OrderBy(order.Name, order.Sorted)
			}
		}
	}

	result.IsDebug = s.isDebug

	return result, nil
}

/**
* Schema: Represents a database schema that owns a set of models.
**/
type Schema struct {
	Name     string            `json:"name"`
	Models   map[string]*Model `json:"models"`
	database string            `json:"-"`
	db       *DB               `json:"-"`
	isDebug  bool              `json:"-"`
	mu       *sync.RWMutex     `json:"-"`
}

/**
* up: Initializes the schema.
* @param db *DB
* @return error
**/
func (s *Schema) up(db *DB) error {
	s.db = db
	s.database = db.Name
	s.isDebug = db.isDebug
	s.mu = &sync.RWMutex{}
	for _, model := range s.Models {
		err := model.up(s)
		if err != nil {
			return err
		}
	}
	return nil
}

/**
* modelList: Returns the models of the schema, a copy safe to range over while others are added.
* @return []*Model
**/
func (s *Schema) modelList() []*Model {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Model, 0, len(s.Models))
	for _, model := range s.Models {
		result = append(result, model)
	}
	return result
}

/**
* toJson: Returns the schema metadata as an et.Json map.
* @return et.Json
**/
func (s *Schema) toJson() et.Json {
	models := et.Json{}
	for _, model := range s.modelList() {
		models[model.Name] = model.ToJson()
	}

	return et.Json{
		"name":   s.Name,
		"models": models,
	}
}

/**
* addModel: Adds a model to the schema.
* @param model *Model
* @return void
**/
func (s *Schema) addModel(model *Model) {
	s.mu.Lock()
	s.Models[model.Name] = model
	s.mu.Unlock()
}

/**
* removeModel: Removes a model from the schema.
* @param name string
* @return void
**/
func (s *Schema) removeModel(name string) {
	s.mu.Lock()
	delete(s.Models, name)
	s.mu.Unlock()
}

/**
* getModel: Returns the named model or an error if it does not exist in this schema.
* @param name string
* @return *Model, error
**/
func (s *Schema) getModel(name string) (*Model, error) {
	name = utility.Normalize(name)
	s.mu.RLock()
	result, exists := s.Models[name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf(MSG_MODEL_NOT_FOUND, name)
	}

	return result, nil
}

/**
* newModel: Constructs a new Model with initialized fields and default triggers.
* @param id string, name string, version int, userId string
* @return *Model
**/
func (s *Schema) newModel(id, name string, version int, userId string) *Model {
	name = utility.Normalize(name)
	id = reg.GetUUID(id)
	result := &Model{
		Kind:          MODEL,
		ID:            id,
		database:      s.database,
		Schema:        s.Name,
		Name:          name,
		Table:         name,
		Columns:       make([]*Column, 0),
		Indexes:       make([]*Index, 0),
		PrimaryKeys:   make([]*Index, 0),
		ForeignKeys:   make([]*Detail, 0),
		Unique:        make([]*Index, 0),
		Required:      make([]*Index, 0),
		Hiddens:       make([]string, 0),
		OmitUpdates:   make([]string, 0),
		Details:       make(map[string]*Detail, 0),
		Masters:       make(map[string]*Master, 0),
		Rollups:       make(map[string]*Rollups, 0),
		Version:       version,
		BeforeInserts: make([]*jwf.Script, 0),
		BeforeUpdates: make([]*jwf.Script, 0),
		BeforeDeletes: make([]*jwf.Script, 0),
		AfterInserts:  make([]*jwf.Script, 0),
		AfterUpdates:  make([]*jwf.Script, 0),
		AfterDeletes:  make([]*jwf.Script, 0),
		AuditLog:      &et.SafeData{},
		calcs:         make(map[string]CalcFunction, 0),
		calcScripts:   make(map[string]string, 0),
		beforeInserts: make([]TriggerFunction, 0),
		beforeUpdates: make([]TriggerFunction, 0),
		beforeDeletes: make([]TriggerFunction, 0),
		afterInserts:  make([]TriggerFunction, 0),
		afterUpdates:  make([]TriggerFunction, 0),
		afterDeletes:  make([]TriggerFunction, 0),
		db:            s.db,
	}
	result.setTable()
	result.defaultTrigger()
	s.db.addAuditLog(userId, "new_model")
	s.addModel(result)
	return result
}

/**
* loadModel: Loads a Model from the database catalog by name.
* @param params et.Json
* @return *Model, error
**/
func (s *Schema) loadModel(def et.Json) (*Model, error) {
	bt, err := def.ToByte()
	if err != nil {
		return nil, err
	}

	var result *Model
	if err := json.Unmarshal(bt, &result); err != nil {
		return nil, err
	}

	err = result.up(s)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* init: Initializes the schema.
* @return error
**/
func (s *Schema) init() error {
	for _, model := range s.modelList() {
		err := model.Init()
		if err != nil {
			return err
		}
	}

	return nil
}

type TxStatus string

const (
	TxStatusPending    TxStatus = "pending"
	TxStatusCommitted  TxStatus = "committed"
	TxStatusRolledBack TxStatus = "rolled_back"
)

type Tx struct {
	CreatedAt    time.Time `json:"created_at"`
	LastUpdateAt time.Time `json:"last_update_at"`
	Id           string    `json:"id"`
	Status       string    `json:"status"`
	Tx           *sql.Tx   `json:"-"`
}

/**
* newTx: Creates a new transaction
* @return *Tx
**/
func newTx() *Tx {
	now := timezone.Now()
	return &Tx{
		CreatedAt:    now,
		LastUpdateAt: now,
		Id:           reg.UUID(),
		Status:       string(TxStatusPending),
	}
}

/**
* getTx: Returns the given Tx unchanged, or creates a new auto-commit Tx when nil.
* The second return value is true when the caller must commit after the operation.
* @param tx *Tx
* @return *Tx, bool
**/
func getTx(tx *Tx) (*Tx, bool) {
	if tx == nil {
		return NewTx(), true
	}

	return tx, false
}

/**
* begin: Opens a database transaction on the underlying *sql.DB if not already open.
* @param db *sql.DB
* @return error
**/
func (s *Tx) begin(db *sql.DB) error {
	if s.Tx != nil {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	s.Tx = tx
	return nil
}

/**
* setStatus: Updates the transaction status and records the change time.
* @param status TxStatus
**/
func (s *Tx) setStatus(status TxStatus) {
	s.Status = string(status)
	s.LastUpdateAt = timezone.Now()
}

/**
* commit: Commits the transaction and marks it as committed.
* @return error
**/
func (s *Tx) commit() error {
	if s.Tx == nil {
		return nil
	}

	err := s.Tx.Commit()
	if err != nil {
		return err
	}

	s.setStatus(TxStatusCommitted)

	return nil
}

/**
* rollback: Rolls back the transaction and marks it as rolled back.
* @return error
**/
func (s *Tx) rollback() error {
	if s.Tx == nil {
		return nil
	}

	err := s.Tx.Rollback()
	if err != nil {
		return err
	}

	s.setStatus(TxStatusRolledBack)

	return nil
}

/**
* query: Executes a query within the transaction.
* @param db *sql.DB, query string, args ...any
* @return *sql.Rows, error
**/
func (s *Tx) query(db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	err := s.begin(db)
	if err != nil {
		return nil, err
	}

	rows, err := s.Tx.Query(query, args...)
	if err != nil {
		errR := s.Rollback()
		if errR != nil {
			err = fmt.Errorf(MSG_ROLLBACK_ERROR, errR)
		}
		return nil, err
	}

	return rows, nil
}

const (
	EVENT_MODEL_SET = "jsql:model:set"
)
