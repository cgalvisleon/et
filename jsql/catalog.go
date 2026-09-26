package jsql

import (
	"database/sql"

	"github.com/cgalvisleon/et/et"
)

// catalog.go is the public API of jsql. Every exported function and method is declared
// here as a thin wrapper over its private implementation in the rest of the package,
// so the whole surface of the package can be read in one place, grouped by functionality.
//
// Índice de la API pública (funciones y métodos por sección):
//
//	#   Sección                         Cantidad
//	1   Conexión y base de datos        24
//	2   Definición de modelos (DDL)     20
//	3   Relaciones y campos calculados  19
//	4   Consultas                       46
//	5   Condiciones                     18
//	6   Comandos                        16
//	7   Triggers                        22
//	8   Transacciones                   4
//	9   Series                          6
//	10  Auditoría                       3
//	11  Utilidades                      17
//	    Total                           195

// =============================================================================
// 1. Conexión y base de datos (24)
// =============================================================================

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *PgConection) GetParams() et.Json {
	return s.getParams()
}

/**
* SetDatabase: Sets the database name in the connection parameters.
* @param name string
**/
func (s *PgConection) SetDatabase(name string) {
	s.setDatabase(name)
}

/**
* GetDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *PgConection) GetDatabase() string {
	return s.getDatabase()
}

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *SqliteConection) GetParams() et.Json {
	return s.getParams()
}

/**
* SetDatabase: Sets the database name in the connection parameters
* @param name string
**/
func (s *SqliteConection) SetDatabase(name string) {
	s.setDatabase(name)
}

/**
* GetDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *SqliteConection) GetDatabase() string {
	return s.getDatabase()
}

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *OracleConection) GetParams() et.Json {
	return s.getParams()
}

/**
* SetDatabase: Sets the database name in the connection parameters
* @param name string
**/
func (s *OracleConection) SetDatabase(name string) {
	s.setDatabase(name)
}

/**
* GetDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *OracleConection) GetDatabase() string {
	return s.getDatabase()
}

/**
* NewDB: Creates a new DB instance for the given driver without initializing it (call Init afterwards).
* @param id, host, name, driver string, showLog ...bool (optional, defaults to true)
* @return *DB, error
**/
func NewDB(params ConnectParams) (*DB, error) {
	return newDB(params)
}

/**
* LoadDb
* @param params et.Json
* @return *DB, error
**/
func LoadDb(params et.Json) (*DB, error) {
	return loadDb(params)
}

/**
* ToJson: Returns the DB metadata as an et.Json map.
* @return et.Json
**/
func (s *DB) ToJson() et.Json {
	return s.toJson()
}

/**
* Init: Opens the driver connection and, when UseCore is set, initializes core tables.
* @return error
**/
func (s *DB) Init() error {
	return s.init()
}

/**
* Close: Closes the underlying *sql.DB connection pool.
* @return error
**/
func (s *DB) Close() error {
	return s.close()
}

/**
* SetDebug: Sets the debug flag to the given value.
* @param debug bool
**/
func (s *DB) SetDebug(debug bool) {
	s.setDebug(debug)
}

/**
* Debug: Enables debug logging for all queries and commands.
**/
func (s *DB) Debug() {
	s.debug()
}

/**
* SqlTx: Executes a SQL query inside the given transaction (or directly on the pool if nil).
* @param tx *Tx, query string, args ...any
* @return et.Items, error
**/
func (s *DB) SqlTx(tx *Tx, query string, arg ...any) (et.Items, error) {
	return s.sqlTx(tx, query, arg...)
}

/**
* Sql: Executes a SQL query directly on the DB (no transaction).
* @param query string, args ...any
* @return et.Items, error
**/
func (s *DB) Sql(query string, args ...any) (et.Items, error) {
	return s.sql(query, args...)
}

/**
* Register: Registers a Driver implementation under the given name so jsql can resolve it by config.
* @param name string
* @param driver Driver
**/
func Register(name string, driver Driver) {
	register(name, driver)
}

/**
* Query: Executes a query.
* @param sql et.Json
* @return et.Items, error
**/
func (s *DB) Query(sql et.Json) (et.Items, error) {
	return s.queryJson(sql)
}

/**
* QueryTx: Executes a query with a transaction.
* @param tx *Tx, sql ry.Json
* @return et.Items, error
**/
func (s *DB) QueryTx(tx *Tx, sql et.Json) (et.Items, error) {
	return s.queryJsonTx(tx, sql)
}

/**
* ConnectTo: Returns an existing DB by name, or creates and initialises a new one from params.
* @param tenantId, host, driver, name string, showLog bool
* @return *DB, error
**/
func ConnectTo(params ConnectParams) (*DB, error) {
	return connectTo(params)
}

/**
* LoadTo: Returns an existing DB by name.
* @param name, hostName string
* @return *DB, error
**/
func LoadTo(dbName string, hostName ...string) (*DB, error) {
	return loadTo(dbName, hostName...)
}

/**
* Load: Connects to the default database reading configuration from environment variables.
* @return *DB, error
**/
func Load() (*DB, error) {
	return load()
}

// =============================================================================
// 2. Definición de modelos (DDL) (20)
// =============================================================================

/**
* NewModel: Returns (or creates) a Model under the given schema name.
* @param schema, name string, version int, userId string
* @return *Model, error
**/
func (s *DB) NewModel(schema, name string, version int, userId string) *Model {
	return s.newModel(schema, name, version, userId)
}

/**
* RemoveModel: Removes a model from the database.
* @param schema, name string
* @return error
**/
func (s *DB) RemoveModel(schema, name string) error {
	return s.removeModel(schema, name)
}

/**
* GetModel: Looks up a model by schema and name, returning an error if not found.
* @param schema, name string
* @return *Model, error
**/
func (s *DB) GetModel(schema, name string) (*Model, error) {
	return s.getModel(schema, name)
}

/**
* Define: Creates a model from a declarative definition (delegates to DefineModel).
* @param definition Define
* @return *Model, error
**/
func (s *DB) Define(define Define) (*Model, error) {
	return s.define(define)
}

/**
* DefineSource: Defines the source column for the model.
* @return *Column
**/
func (s *Model) DefineSource() *Column {
	return s.defineSource()
}

/**
* DefineIdxField: Defines the idx field column for the model.
* @return *Index
**/
func (s *Model) DefineIdxField() *Index {
	return s.defineIdxField()
}

/**
* DefineIndex: Defines a new index column for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Index
**/
func (s *Model) DefineIndex(name string, tp et.TypeData, deFault any) *Index {
	return s.defineIndex(name, tp, deFault)
}

/**
* DefinePrimaryKey: Defines a new primary key column for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Index
**/
func (s *Model) DefinePrimaryKey(name string, tp et.TypeData, deFault any) *Index {
	return s.definePrimaryKey(name, tp, deFault)
}

/**
* DefineForeignKeys: Defines a new foreign key column for the model.
* @param to *Model, keys map[string]string, onDeleteCascade bool, onUpdateCascade bool
* @return *Detail
**/
func (s *Model) DefineForeignKeys(to *Model, keys map[string]string, onDeleteCascade, onUpdateCascade bool) *Detail {
	return s.defineForeignKeys(to, keys, onDeleteCascade, onUpdateCascade)
}

/**
* DefineUnique: Defines a new unique index for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Index
**/
func (s *Model) DefineUnique(name string, tp et.TypeData, deFault any) *Index {
	return s.defineUnique(name, tp, deFault)
}

/**
* DefineRequired: Defines a new required column for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Index
**/
func (s *Model) DefineRequired(name string, tp et.TypeData, deFault any) *Index {
	return s.defineRequired(name, tp, deFault)
}

/**
* DefineHidden: Defines a new hidden column for the model.
* @param name ...string
**/
func (s *Model) DefineHidden(name ...string) {
	s.defineHidden(name...)
}

/**
* DefineColumn: Defines a new column for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Column
**/
func (s *Model) DefineColumn(name string, tp et.TypeData, deFault any) *Column {
	return s.defineRealColumn(name, tp, deFault)
}

/**
* DefineAttrib: Defines a new attribute for the model.
* @param name string, tp et.TypeData, deFault any
* @return *Column
**/
func (s *Model) DefineAttrib(name string, tp et.TypeData, deFault any) *Column {
	return s.defineAttrib(name, tp, deFault)
}

/**
* DefineModel: Defines the standard columns for the model.
* @return *Model
**/
func (s *Model) DefineModel() *Model {
	return s.defineModel()
}

/**
* DefineModel: Defines a new model for the database.
* @param schema string, name string, version int
* @return *Model, error
**/
func (s *DB) DefineModel(schema, name string, version int, userId string) (*Model, error) {
	return s.defineModel(schema, name, version, userId)
}

/**
* DefineTenantModel: Defines a new tenant model for the database.
* @param schema string, name string, version int
* @return *Model, error
**/
func (s *DB) DefineTenantModel(schema, name string, version int, userId string) (*Model, error) {
	return s.defineTenantModel(schema, name, version, userId)
}

/**
* DefineProjectModel: Defines a new project model for the database.
* @param schema string, name string, version int
* @return *Model, error
**/
func (s *DB) DefineProjectModel(schema, name string, version int, userId string) (*Model, error) {
	return s.defineProjectModel(schema, name, version, userId)
}

/**
* Init: Runs DDL for the model the first time it is called; subsequent calls are no-ops.
* @return error
**/
func (s *Model) Init() error {
	return s.init()
}

/**
* Stricted: Enables strict mode — unknown field names are not treated as ATTRIBs.
**/
func (s *Model) Stricted() {
	s.stricted()
}

// =============================================================================
// 3. Relaciones y campos calculados (19)
// =============================================================================

/**
* DefineRollup: Defines a new rollup for the model.
* Selects are required except for RollupCount (counts rows) and RollupObject (whole row).
* An empty operation defaults to RollupObject.
* @param name string, to *Model, keys map[string]string, selects []string, operation RollupOperation
* @return (*Rollups, error)
**/
func (s *Model) DefineRollup(name string, to *Model, keys map[string]string, selects []string, operation RollupOperation) (*Rollups, error) {
	return s.defineRollup(name, to, keys, selects, operation)
}

/**
* DefineDetail: Defines a new detail for the model: keys join the master with the detail, rows is how
* many records are shown and selects (optional) the fields shown; without selects, all fields.
* @param name string, keys map[string]string, rows int, selects ...string
* @return (*Model, error)
**/
func (s *Model) DefineDetail(name string, keys map[string]string, rows int, selects ...string) (*Model, error) {
	return s.defineDetail(name, keys, rows, selects...)
}

/**
* DefineMaster: Defines a new master for the model through a bridge model: keys go from the master to
* the bridge, toKeys from the destination to the bridge, selects are the fields shown and rows (optional)
* how many records are shown. With rows = 1 the relation is 1 to 1 and the row gets an object;
* otherwise it gets a list (up to DB_RECORD_LIMIT when rows is not set).
* @param name string, to *Model, keys, toKeys map[string]string, selects []string, rows ...int
* @return (*Model, error)
**/
func (s *Model) DefineMaster(name string, to *Model, keys, toKeys map[string]string, selects []string, rows ...int) (*Model, error) {
	return s.defineMaster(name, to, keys, toKeys, selects, rows...)
}

/**
* DefineCalcFunc: Defines a new calculation for the model.
* @param name string, calc CalcFunction
* @return *Model
**/
func (s *Model) DefineCalcFunc(name string, calc CalcFunction) *Model {
	return s.defineCalcFunc(name, calc)
}

/**
* DefineCalc: Defines a new calculation for the model using a bytecode definition.
* @param name string, code string
* @return *Model
**/
func (s *Model) DefineCalc(name, script string) *Model {
	return s.defineCalc(name, script)
}

/**
* DetailKeys
**/
func DetailKeys(key, foreignKey string) map[string]string {
	return detailKeys(key, foreignKey)
}

/**
* RollupKeys
**/
func RollupKeys(key, foreignKey string) map[string]string {
	return rollupKeys(key, foreignKey)
}

/**
* Ref: Returns the reference of the detail.
* @return et.Json
**/
func (s *Detail) Ref() et.Json {
	return s.ref()
}

/**
* GetQuery: Returns the query for the detail.
* @param item et.Json
* @return *Query
**/
func (s *Detail) GetQuery(item et.Json, page, rows int) *Query {
	return s.getQuery(item, page, rows)
}

/**
* IsAggregate: Returns true if the operation is a SQL aggregate (count, sum, avg, min, max).
* @return bool
**/
func (s RollupOperation) IsAggregate() bool {
	return s.isAggregate()
}

/**
* IsValid: Returns true if the operation is one of the defined rollup operations.
* @return bool
**/
func (s RollupOperation) IsValid() bool {
	return s.isValid()
}

/**
* Ref: Returns the reference of the master.
* @return et.Json
**/
func (s *Master) Ref() et.Json {
	return s.ref()
}

/**
* GetCalcFunc: Returns the CalcFunction for the given name, if it exists.
* @param name string
* @return CalcFunction, bool
**/
func (s *Model) GetCalcFunc(name string) (CalcFunction, bool) {
	return s.getCalcFunc(name)
}

/**
* Detail: Returns the detail for the given name.
* @param name string
* @return *Model, bool
**/
func (s *Model) Detail(name string) (*Model, bool) {
	return s.detail(name)
}

/**
* Master: Returns the master for the given name.
* @param name string
* @return *Query, bool
**/
func (s *Model) Master(name string) (*Query, bool) {
	return s.master(name)
}

/**
* Bridge: Returns the bridge for the given name.
* @param name string
* @return *Model, bool
**/
func (s *Model) Bridge(name string) (*Model, bool) {
	return s.bridge(name)
}

/**
* Ref: Returns the reference of the from.
* @return et.Json
**/
func (s *From) Ref() et.Json {
	return s.ref()
}

/**
* GetQuery: Returns the query for the rollup filtered by the keys of the given row.
* Returns false when the row lacks any of the keys, so the rollup is not applied.
* @param item et.Json
* @return *Query, bool
**/
func (s *QueryRollups) GetQuery(item et.Json) (*Query, bool) {
	return s.getQuery(item)
}

/**
* GetQuery: Returns the query for the detail.
* @param item et.Json
* @return *Query
**/
func (s *QueryDetail) GetQuery(item et.Json) *Query {
	return s.getQuery(item)
}

// =============================================================================
// 4. Consultas (46)
// =============================================================================

/**
* GetColumn: Returns the Column for the given name. For non-strict models with a SourceField,
* unknown names are returned as synthetic ATTRIB columns.
* @param name string
* @return *Column
**/
func (s *Model) GetColumn(name string) (*Column, bool) {
	return s.getColumn(name)
}

/**
* GetField: Returns the Field for the given name.
* @param name string
* @return *Field, bool
**/
func (s *Model) GetField(name string) (*Field, bool) {
	return s.getField(name)
}

/**
* GetFrom: Returns the From for the model.
* @return *From
**/
func (s *Model) GetFrom() *From {
	return s.getFrom()
}

/**
* As: Creates a new Query for this model with the given alias.
* @param as ...string
* @return *Query
**/
func (s *Model) As(as ...string) *Query {
	return s.as(as...)
}

/**
* Join: Creates a new Query for this model with the given model as the INNER JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Model) Join(to *Model, as string, on []*et.Condition) *Query {
	return s.join(to, as, on)
}

/**
* Select: Creates a new Query for this model with the given fields as the SELECT clause.
* @param fields ...string
* @return *Query
**/
func (s *Model) Select(fields ...string) *Query {
	return s.selects(fields...)
}

/**
* Calc: Creates a new Query for this model with the given fields as the CALC clause.
* @param fields ...string
* @return *Query
**/
func (s *Model) Calc(fields ...string) *Query {
	return s.calc(fields...)
}

/**
* Where: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param cond *et.Condition
* @return *Query
**/
func (s *Model) Where(cond *et.Condition) *Query {
	return s.where(cond)
}

/**
* Count: Returns the count of records in the model.
* @return (int, error)
**/
func (s *Model) Count() (int, error) {
	return s.count()
}

/**
* First: Returns the first record in the model.
* @return (et.Item, error)
**/
func (s *Model) First(n int) (et.Items, error) {
	return s.first(n)
}

/**
* QueryTx: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param query et.Json
* @return *Query
**/
func (s *Model) QueryTx(tx *Tx, query et.Json) *Query {
	return s.queryTx(tx, query)
}

/**
* Query: Creates a new Query for this model with the given condition as the first WHERE clause.
* @param query et.Json
* @return *Query
**/
func (s *Model) Query(query et.Json) *Query {
	return s.query(query)
}

/**
* NewQuery: Creates a Query with the model as the primary FROM source.
* @param model *Model, as ...string
* @return *Query
**/
func NewQuery(model *Model, as ...string) *Query {
	return newQuery(model, as...)
}

/**
* ToJson: Returns the query metadata as an et.Json map.
* @return et.Json
**/
func (s *Query) ToJson() et.Json {
	return s.toJson()
}

/**
* Debug: Enables SQL logging for this query and returns it for chaining.
* @return *Query
**/
func (s *Query) Debug() *Query {
	return s.debug()
}

/**
* Test: Enables test mode — SQL is generated but not executed.
* @return *Query
**/
func (s *Query) Test() *Query {
	return s.test()
}

/**
* GetField: Parses a field reference with et.ToField and resolves it against the query's origins.
* Supports "field", "field:as", "from.field", "from.field:as", "agg(field)", "agg(field):as",
* "field->a->b" and the "|page:n" suffix.
* @param field string
* @return (*Field, bool)
**/
func (s *Query) GetField(field string) (*Field, bool) {
	return s.getField(field)
}

/**
* Join: Appends an INNER JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) Join(model *Model, as string, on []*et.Condition) *Query {
	return s.joinInner(model, as, on)
}

/**
* LeftJoin: Appends a LEFT JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) LeftJoin(model *Model, as string, on []*et.Condition) *Query {
	return s.leftJoin(model, as, on)
}

/**
* RightJoin: Appends a RIGHT JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) RightJoin(model *Model, as string, on []*et.Condition) *Query {
	return s.rightJoin(model, as, on)
}

/**
* FullJoin: Appends a FULL JOIN clause.
* @param model *Model, as string, on *et.Condition
* @return *Query
**/
func (s *Query) FullJoin(model *Model, as string, on []*et.Condition) *Query {
	return s.fullJoin(model, as, on)
}

/**
* Select: Appends fields to the SELECT clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) Select(fields ...string) *Query {
	return s.selects(fields...)
}

/**
* Calc: Appends fields to the DETAIL clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) Calc(fields ...string) *Query {
	return s.calc(fields...)
}

/**
* Detail: Appends fields to the DETAIL clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) Detail(fields ...string) *Query {
	return s.detail(fields...)
}

/**
* Master: Appends fields to the MASTER clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) Master(fields ...string) *Query {
	return s.master(fields...)
}

/**
* Hidden: Appends fields to the HIDDEN clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) Hidden(fields ...string) *Query {
	return s.hidden(fields...)
}

/**
* AddCondition: Appends a condition to the WHERE clause.
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) AddCondition(cond *et.Condition) *Query {
	return s.addOneCondition(cond)
}

/**
* Where: Appends a condition to the WHERE clause and sets the active section to where.
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Where(cond *et.Condition) *Query {
	return s.where(cond)
}

/**
* And: Appends an AND condition to the active clause section (WHERE, JOIN ON, or HAVING).
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) And(cond *et.Condition) *Query {
	return s.and(cond)
}

/**
* Or: Appends an OR condition to the active clause section (WHERE, JOIN ON, or HAVING).
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Or(cond *et.Condition) *Query {
	return s.or(cond)
}

/**
* GroupBy: Adds one or more fields to the GROUP BY clause.
* @param fields ...string
* @return *Query
**/
func (s *Query) GroupBy(fields ...string) *Query {
	return s.groupBy(fields...)
}

/**
* Having: Appends a condition to the HAVING clause and sets the active section to having.
* @param cond *et.Condition
* @return *Query
**/
func (s *Query) Having(cond *et.Condition) *Query {
	return s.having(cond)
}

/**
* Page: Sets the result offset based on the 1-based page number and current Rows limit.
* @param page int
* @return *Query
**/
func (s *Query) Page(page int) *Query {
	return s.page(page)
}

/**
* OrderBy: Appends a field to the ORDER BY clause; sorted=true means ASC, false means DESC.
* @param field string
* @param sorted bool
* @return *Query
**/
func (s *Query) OrderBy(field string, sorted ...bool) *Query {
	return s.orderBy(field, sorted...)
}

/**
* AllTx: Generates and executes a SELECT query inside the given transaction.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Query) AllTx(tx *Tx) (et.Items, error) {
	return s.allTx(tx)
}

/**
* All: Generates and executes a SELECT query without an explicit transaction.
* @return et.Items, error
**/
func (s *Query) All() (et.Items, error) {
	return s.all()
}

/**
* OneTx: Executes the query limited to one row inside the given transaction.
* @param tx *Tx
* @return et.Item, error
**/
func (s *Query) OneTx(tx *Tx) (et.Item, error) {
	return s.oneTx(tx)
}

/**
* One: Executes the query limited to one row without an explicit transaction.
* @return et.Item, error
**/
func (s *Query) One() (et.Item, error) {
	return s.one()
}

/**
* FirstTx: Executes the query limited to the first n rows inside the given transaction.
* @param tx *Tx, n int
* @return et.Items, error
**/
func (s *Query) FirstTx(tx *Tx, n int) (et.Items, error) {
	return s.firstTx(tx, n)
}

/**
* First: Executes the query limited to the first n rows without an explicit transaction.
* @param n int
* @return et.Items, error
**/
func (s *Query) First(n int) (et.Items, error) {
	return s.first(n)
}

/**
* LimitTx: Sets the maximum number of rows to return.
* @param tx *Tx, page int, rows int
* @return et.Items, error
**/
func (s *Query) LimitTx(tx *Tx, page, rows int) (et.Items, error) {
	return s.limitTx(tx, page, rows)
}

/**
* Limit: Sets the maximum number of rows to return.
* @param page int, rows int
* @return et.Items, error
**/
func (s *Query) Limit(page, rows int) (et.Items, error) {
	return s.limit(page, rows)
}

/**
* ExistsTx: Returns the Model for the primary FROM source, or nil if not found.
* @return *Model
**/
func (s *Query) ExistsTx(tx *Tx) (bool, error) {
	return s.existsTx(tx)
}

/**
* Exists: Checks if any rows match the query conditions.
* @return bool, error
**/
func (s *Query) Exists() (bool, error) {
	return s.exists()
}

/**
* CountTx: Executes the query and returns the count of matching rows within the given transaction.
* @param tx *Tx
* @return int, error
**/
func (s *Query) CountTx(tx *Tx) (int, error) {
	return s.countTx(tx)
}

/**
* Count: Executes the query and returns the count of matching rows.
* @return int, error
**/
func (s *Query) Count() (int, error) {
	return s.count()
}

// =============================================================================
// 5. Condiciones (18)
// =============================================================================

/**
* Where: Returns a WHERE condition (field = value).
* @param field, value interface{}
* @return *et.Condition
**/
func Where(field, value interface{}) *et.Condition {
	return where(field, value)
}

/**
* And: Returns a AND condition (field = value).
* @param field, operator et.Operator, value interface{}
* @return *et.Condition
**/
func And(field interface{}, operator et.Operator, value interface{}) *et.Condition {
	return and(field, operator, value)
}

/**
* Or: Returns a OR condition (field = value).
* @param field, operator et.Operator, value interface{}
* @return *et.Condition
**/
func Or(field interface{}, operator et.Operator, value interface{}) *et.Condition {
	return or(field, operator, value)
}

/**
* Eq: Returns an equality condition (field = value).
* @param field string, value interface{}
* @return *et.Condition
**/
func Eq(field string, value interface{}) *et.Condition {
	return eq(field, value)
}

/**
* Neg: Returns a not-equal condition (field <> value).
* @param field string, value interface{}
* @return *et.Condition
**/
func Neg(field string, value interface{}) *et.Condition {
	return neg(field, value)
}

/**
* Less: Returns a less-than condition (field < value).
* @param field string, value interface{}
* @return *et.Condition
**/
func Less(field string, value interface{}) *et.Condition {
	return less(field, value)
}

/**
* LessEq: Returns a less-than-or-equal condition (field <= value).
* @param field string, value interface{}
* @return *et.Condition
**/
func LessEq(field string, value interface{}) *et.Condition {
	return lessEq(field, value)
}

/**
* More: Returns a greater-than condition (field > value).
* @param field string, value interface{}
* @return *et.Condition
**/
func More(field string, value interface{}) *et.Condition {
	return more(field, value)
}

/**
* MoreEq: Returns a greater-than-or-equal condition (field >= value).
* @param field string, value interface{}
* @return *et.Condition
**/
func MoreEq(field string, value interface{}) *et.Condition {
	return moreEq(field, value)
}

/**
* Like: Returns a case-insensitive pattern match condition (field ILIKE value).
* @param field string, value interface{}
* @return *et.Condition
**/
func Like(field string, value interface{}) *et.Condition {
	return like(field, value)
}

/**
* In: Returns an inclusion condition (field IN (values...)).
* @param field string, value []interface{}
* @return *et.Condition
**/
func In(field string, value []interface{}) *et.Condition {
	return in(field, value)
}

/**
* NotIn: Returns an exclusion condition (field NOT IN (values...)).
* @param field string, value []interface{}
* @return *et.Condition
**/
func NotIn(field string, value []interface{}) *et.Condition {
	return notIn(field, value)
}

/**
* Is: Returns an IS condition (field IS value), typically used with NULL or booleans.
* @param field string, value interface{}
* @return *et.Condition
**/
func Is(field string, value interface{}) *et.Condition {
	return is(field, value)
}

/**
* IsNot: Returns an IS NOT condition (field IS NOT value).
* @param field string, value interface{}
* @return *et.Condition
**/
func IsNot(field string, value interface{}) *et.Condition {
	return isNot(field, value)
}

/**
* Null: Returns an IS NULL condition (field IS NULL).
* @param field string
* @return *et.Condition
**/
func Null(field string) *et.Condition {
	return null(field)
}

/**
* NotNull: Returns an IS NOT NULL condition (field IS NOT NULL).
* @param field string
* @return *et.Condition
**/
func NotNull(field string) *et.Condition {
	return notNull(field)
}

/**
* Between: Returns a range condition (field BETWEEN min AND max).
* @param field string, min any, max any
* @return *et.Condition
**/
func Between(field string, min, max any) *et.Condition {
	return between(field, min, max)
}

/**
* NotBetween: Returns a negated range condition (field NOT BETWEEN min AND max).
* @param field string, min any, max any
* @return *et.Condition
**/
func NotBetween(field string, min, max any) *et.Condition {
	return notBetween(field, min, max)
}

// =============================================================================
// 6. Comandos (16)
// =============================================================================

/**
* ToJson: Returns the command metadata as an et.Json map.
* @return et.Json
**/
func (s *Command) ToJson() et.Json {
	return s.toJson()
}

/**
* Debug: Enables debug mode — SQL is logged to stdout.
* @return *Command
**/
func (s *Command) Debug() *Command {
	return s.debug()
}

/**
* Test: Enables test mode — SQL is generated but not executed.
* @return *Command
**/
func (s *Command) Test() *Command {
	return s.test()
}

/**
* Where: Sets the first WHERE condition and returns the command for chaining.
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) Where(cond *et.Condition) *Command {
	return s.where(cond)
}

/**
* And: Appends a condition joined with AND to the WHERE clause.
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) And(cond *et.Condition) *Command {
	return s.and(cond)
}

/**
* Or: Appends a condition joined with OR to the WHERE clause.
* @param cond *et.Condition
* @return *Command
**/
func (s *Command) Or(cond *et.Condition) *Command {
	return s.or(cond)
}

/**
* Return: Sets the fields to return in the result.
* @param fields ...string
* @return *Command
**/
func (s *Command) Return(fields ...string) *Command {
	return s.returning(fields...)
}

/**
* ExecTx: Dispatches the command to the appropriate handler and commits if no external Tx was given.
* @param tx *Tx
* @return et.Items, error
**/
func (s *Command) ExecTx(tx *Tx) (et.Items, error) {
	return s.execTx(tx)
}

/**
* Exec: Executes the command without an explicit transaction.
* @return et.Items, error
**/
func (s *Command) Exec() (et.Items, error) {
	return s.execute()
}

/**
* OneTx: Executes the command and returns the first result within the given transaction.
* @param tx *Tx
* @return et.Item, error
**/
func (s *Command) OneTx(tx *Tx) (et.Item, error) {
	return s.oneTx(tx)
}

/**
* One: Executes the command and returns the first result without an explicit transaction.
* @return et.Item, error
**/
func (s *Command) One() (et.Item, error) {
	return s.one()
}

/**
* Insert: Creates a Command of type INSERT pre-loaded with the given data row.
* @param data et.Json
* @return *Command
**/
func (s *Model) Insert(data et.Json) *Command {
	return s.insert(data)
}

/**
* Bulk: Creates a Command of type BULK pre-loaded with multiple data rows.
* @param data []et.Json
* @return *Command
**/
func (s *Model) Bulk(data []et.Json) *Command {
	return s.bulk(data)
}

/**
* Update: Creates a Command of type UPDATE pre-loaded with the given data row.
* @param data et.Json
* @return *Command
**/
func (s *Model) Update(data et.Json) *Command {
	return s.update(data)
}

/**
* Delete: Creates a Command of type DELETE (conditions must be added with Where/And).
* @return *Command
**/
func (s *Model) Delete() *Command {
	return s.delete()
}

/**
* Upsert: Creates a Command of type UPSERT pre-loaded with data and primary-key conditions.
* @param data et.Json
* @return *Command
**/
func (s *Model) Upsert(data et.Json) *Command {
	return s.upsert(data)
}

// =============================================================================
// 7. Triggers (22)
// =============================================================================

/**
* BeforeInsert: Registers a trigger function to run before each INSERT execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeInsert(fn TriggerFunction) *Command {
	return s.beforeInsert(fn)
}

/**
* BeforeUpdate: Registers a trigger function to run before each UPDATE execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeUpdate(fn TriggerFunction) *Command {
	return s.beforeUpdate(fn)
}

/**
* BeforeDelete: Registers a trigger function to run before each DELETE execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeDelete(fn TriggerFunction) *Command {
	return s.beforeDelete(fn)
}

/**
* BeforeInsertOrUpdate: Registers a trigger function to run before INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) BeforeInsertOrUpdate(fn TriggerFunction) *Command {
	return s.beforeInsertOrUpdate(fn)
}

/**
* AfterInsert: Registers a trigger function to run after each INSERT execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterInsert(fn TriggerFunction) *Command {
	return s.afterInsert(fn)
}

/**
* AfterUpdate: Registers a trigger function to run after each UPDATE execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterUpdate(fn TriggerFunction) *Command {
	return s.afterUpdate(fn)
}

/**
* AfterInsertOrUpdate: Registers a trigger function to run after INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterInsertOrUpdate(fn TriggerFunction) *Command {
	return s.afterInsertOrUpdate(fn)
}

/**
* AfterDelete: Registers a trigger function to run after each DELETE execution.
* @param fn TriggerFunction
* @return *Command
**/
func (s *Command) AfterDelete(fn TriggerFunction) *Command {
	return s.afterDelete(fn)
}

/**
* DefineBeforeInsert: Defines a new before insert hook for the model.
* @param name string
* @return *Model
**/
func (s *Model) DefineBeforeInsert(name string) *Model {
	return s.defineBeforeInsert(name)
}

/**
* DefineBeforeUpdate: Defines a new before update hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) DefineBeforeUpdate(name, code string) *Model {
	return s.defineBeforeUpdate(name, code)
}

/**
* DefineBeforeDelete: Defines a new before delete hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) DefineBeforeDelete(name, code string) *Model {
	return s.defineBeforeDelete(name, code)
}

/**
* DefineAfterInsert: Defines a new after insert hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) DefineAfterInsert(name, code string) *Model {
	return s.defineAfterInsert(name, code)
}

/**
* DefineAfterUpdate: Defines a new after update hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) DefineAfterUpdate(name, code string) *Model {
	return s.defineAfterUpdate(name, code)
}

/**
* DefineAfterDelete: Defines a new after delete hook for the model using a bytecode definition.
* @param module string
* @return *Model
**/
func (s *Model) DefineAfterDelete(name, code string) *Model {
	return s.defineAfterDelete(name, code)
}

/**
* BeforeInsert: Registers a trigger function to run before each INSERT.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeInsert(fn TriggerFunction) *Model {
	return s.beforeInsert(fn)
}

/**
* BeforeUpdate: Registers a trigger function to run before each UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeUpdate(fn TriggerFunction) *Model {
	return s.beforeUpdate(fn)
}

/**
* BeforeDelete: Registers a trigger function to run before each DELETE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeDelete(fn TriggerFunction) *Model {
	return s.beforeDelete(fn)
}

/**
* BeforeInsertOrUpdate: Registers a trigger function to run before INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) BeforeInsertOrUpdate(fn TriggerFunction) *Model {
	return s.beforeInsertOrUpdate(fn)
}

/**
* AfterInsert: Registers a trigger function to run after each INSERT.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterInsert(fn TriggerFunction) *Model {
	return s.afterInsert(fn)
}

/**
* AfterUpdate: Registers a trigger function to run after each UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterUpdate(fn TriggerFunction) *Model {
	return s.afterUpdate(fn)
}

/**
* AfterDelete: Registers a trigger function to run after each DELETE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterDelete(fn TriggerFunction) *Model {
	return s.afterDelete(fn)
}

/**
* AfterInsertOrUpdate: Registers a trigger function to run after INSERT and UPDATE.
* @param fn TriggerFunction
* @return *Model
**/
func (s *Model) AfterInsertOrUpdate(fn TriggerFunction) *Model {
	return s.afterInsertOrUpdate(fn)
}

// =============================================================================
// 8. Transacciones (4)
// =============================================================================

/**
* NewTx: Creates a new transaction
* @return *Tx
**/
func NewTx() *Tx {
	return newTx()
}

/**
* Commit: Commits the transaction and marks it as committed.
* @return error
**/
func (s *Tx) Commit() error {
	return s.commit()
}

/**
* Rollback: Rolls back the transaction and marks it as rolled back.
* @return error
**/
func (s *Tx) Rollback() error {
	return s.rollback()
}

/**
* Query: Executes a query within the transaction.
* @param db *sql.DB, query string, args ...any
* @return *sql.Rows, error
**/
func (s *Tx) Query(db *sql.DB, query string, args ...any) (*sql.Rows, error) {
	return s.query(db, query, args...)
}

// =============================================================================
// 9. Series (6)
// =============================================================================

/**
* DefineSeries
* @param schema string
* @return error
**/
func DefineSeries(db *DB, schema string) (*Series, error) {
	return defineSeries(db, schema)
}

/**
* SetSeries
* @param string tag, format string, value int
* @return error
**/
func (s *Series) SetSeries(tag string, format string, value int) error {
	return s.setSeries(tag, format, value)
}

/**
* GetSeries
* @param string tag, ownerId string
* @return (et.Item, error)
**/
func (s *Series) GetSeries(tag string) (et.Item, error) {
	return s.getSeries(tag)
}

/**
* DeleteSeries
* @param string tag, ownerId string
* @return error
**/
func (s *Series) DeleteSeries(tag string) error {
	return s.deleteSeries(tag)
}

/**
* GenSerie
* @param string tag
* @return (string, error)
**/
func (s *Series) GenSerie(tag string) (string, error) {
	return s.genSerie(tag)
}

/**
* GenValue
* @param string tag
* @return (int, error)
**/
func (s *Series) GenValue(tag string) (int, error) {
	return s.genValue(tag)
}

// =============================================================================
// 10. Auditoría (3)
// =============================================================================

/**
* OnAuditLog
**/
func (s *DB) OnAuditLog(fn func(userId string, action string) error) {
	s.setOnAuditLog(fn)
}

/**
* AddAuditLog
* @param userId string, action string
**/
func AddAuditLog(auditLog []et.Json, userId string, action string) []et.Json {
	return addAuditLog(auditLog, userId, action)
}

/**
* AddAuditLog: Adds an audit log to the model.
* @param userId string, action string
**/
func (s *Model) AddAuditLog(userId string, action string) {
	s.addAuditLog(userId, action)
}

// =============================================================================
// 11. Utilidades (17)
// =============================================================================

/**
* Str: Returns the string representation of the TypeColumn.
* @return string
**/
func (s TypeColumn) Str() string {
	return s.str()
}

/**
* StatusList
**/
func StatusList() []interface{} {
	return statusList()
}

/**
* ToJson: Returns the column metadata as an et.Json map.
* @return et.Json
**/
func (s *Column) ToJson() et.Json {
	return s.toJson()
}

/**
* SQLParse: Replaces $N positional placeholders in sql with their quoted argument values.
* @param sql string
* @param args ...any
* @return string
**/
func SQLParse(sql string, args ...any) string {
	return sqlParse(sql, args...)
}

/**
* Quoted: Returns val formatted as a SQL literal (quoted string, bare number, NULL, etc.).
* @param val any
* @return any
**/
func Quoted(val any) any {
	return quoted(val)
}

/**
* EscapeSQLString: Escapes single quotes in s by doubling them (standard SQL
* string-literal escaping), so a value can be safely embedded between the
* surrounding '...' produced by Quoted.
* @param s string
* @return string
**/
func EscapeSQLString(s string) string {
	return escapeSQLString(s)
}

/**
* JsonString: Serializes val as compact JSON without HTML escaping, so characters
* such as <, > and & are stored as-is instead of \u003c, \u003e and \u0026.
* The result still needs EscapeSQLString before being embedded in a SQL literal.
* @param val any
* @return string, error
**/
func JsonString(val any) (string, error) {
	return jsonString(val)
}

/**
* RowsToItems: Scans all rows from a *sql.Rows result set into an et.Items collection.
* @param rows *sql.Rows
* @return et.Items
**/
func RowsToItems(rows *sql.Rows) et.Items {
	return rowsToItems(rows)
}

/**
* ArgWhitAs: Returns an array with the argument and its alias.
* @param arg string
* @return []string, bool
**/
func ArgWhitAs(arg string) ([]string, bool) {
	return argWhitAs(arg)
}

/**
* ArgWhitSchema: Returns an array with the argument and its schema.
* @param arg string
* @return []string, bool
**/
func ArgWhitSchema(arg string) ([]string, bool) {
	return argWhitSchema(arg)
}

/**
* ToJson: Returns the model metadata as an et.Json map.
* @return et.Json
**/
func (s *Model) ToJson() et.Json {
	return s.toJson()
}

/**
* Debug: Enables debug logging and returns the model for chaining.
* @return *Model
**/
func (s *Model) Debug() *Model {
	return s.debug()
}

/**
* Db: Returns the underlying *sql.DB connection pool.
* @return *sql.DB
**/
func (s *Model) Db() *DB {
	return s.getDb()
}

/**
* SetDb: Sets the primary DB connection for the model.
* @param db *DB
**/
func (s *Model) SetDb(db *DB) {
	s.setDb(db)
}

/**
* SqlDB: Returns the underlying *sql.DB connection pool.
* @return *sql.DB
**/
func (s *Model) SqlDB() *sql.DB {
	return s.sqlDB()
}

/**
* GetModel: Returns the model for the given schema and name.
* @param schema string
* @param name string
* @return *Model, error
**/
func (s *Model) GetModel(schema, name string) (*Model, error) {
	return s.getModel(schema, name)
}

/**
* ToJson: Returns the schema metadata as an et.Json map.
* @return et.Json
**/
func (s *Schema) ToJson() et.Json {
	return s.toJson()
}
