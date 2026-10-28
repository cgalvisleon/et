package jsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

/**
* getConnection: Returns the Connection of driver read from the environment variables. There is no case for
* DriverJosefina (it uses PgConection): GetConnection("josefina") or DB_DRIVER=josefina gives MSG_UNSUPPORTED_DRIVER.
* @param driver, host string
* @return Connection, error
**/
func getConnection(driver, host string) (Connection, error) {
	switch driver {
	case DriverPostgres:
		config := pgConection(host)
		return config, nil
	case DriverSqlite:
		config := sqliteConection(host)
		return config, nil
	case DriverOracle:
		config := oracleConection(host)
		return config, nil
	case DriverMysql:
		config := mysqlConection(host)
		return config, nil
	case DriverMssql:
		config := mssqlConection(host)
		return config, nil
	default:
		return nil, fmt.Errorf(MSG_UNSUPPORTED_DRIVER, driver)
	}
}

type Connection interface {
	ID() string
	SetId(string)
	GetParams() et.Json
	SetDatabase(string)
	GetDatabase() string
	SetPassword(string)
}

// Pool: the connection pool of a server connection (postgres, oracle, mysql, mssql), from DB_POOL_MAX_OPEN,
// DB_POOL_MAX_IDLE, DB_POOL_LIFETIME and DB_POOL_IDLE_TIME (minutes). A zero value is left out of GetParams, so the
// driver keeps its default (3, 1, 30 and 2)
type Pool struct {
	PoolMaxOpen  int
	PoolMaxIdle  int
	PoolLifetime int
	PoolIdleTime int
}

/**
* poolFromEnv: The pool of DB_POOL_*; 0 (the driver's default) when a variable is not set
* @return Pool
**/
func poolFromEnv() Pool {
	return Pool{
		PoolMaxOpen:  envar.GetInt("DB_POOL_MAX_OPEN", 0),
		PoolMaxIdle:  envar.GetInt("DB_POOL_MAX_IDLE", 0),
		PoolLifetime: envar.GetInt("DB_POOL_LIFETIME", 0),
		PoolIdleTime: envar.GetInt("DB_POOL_IDLE_TIME", 0),
	}
}

/**
* withPool: params with the pool keys that are set (> 0)
* @param params et.Json
* @return et.Json
**/
func (s Pool) withPool(params et.Json) et.Json {
	values := map[string]int{
		"pool_max_open":  s.PoolMaxOpen,
		"pool_max_idle":  s.PoolMaxIdle,
		"pool_lifetime":  s.PoolLifetime,
		"pool_idle_time": s.PoolIdleTime,
	}
	for key, value := range values {
		if value > 0 {
			params[key] = value
		}
	}
	return params
}

/**
* readPool: The pool keys of params (ConnectionFromJson), keeping the current values when missing
* @param params et.Json
**/
func (s *Pool) readPool(params et.Json) {
	s.PoolMaxOpen = params.ValInt(s.PoolMaxOpen, "pool_max_open")
	s.PoolMaxIdle = params.ValInt(s.PoolMaxIdle, "pool_max_idle")
	s.PoolLifetime = params.ValInt(s.PoolLifetime, "pool_lifetime")
	s.PoolIdleTime = params.ValInt(s.PoolIdleTime, "pool_idle_time")
}

type PgConection struct {
	Id          string
	Host        string
	Port        int
	Database    string
	User        string
	Password    string
	Sslmode     string
	AppName     string
	RecordLimit int
	Timeout     time.Duration
	Pool
}

func pgConection(host string) *PgConection {
	port := envar.GetInt("DB_PORT", 5432)
	database := envar.GetStr("DB_NAME", "josephine")
	user := envar.GetStr("DB_USER", "test")
	password := envar.GetStr("DB_PASSWORD", "test")
	sslmode := envar.GetStr("DB_SSLMODE", "disable")
	appName := envar.GetStr("DB_APP_NAME", "josephine")
	recordLimit := envar.GetInt("DB_RECORD_LIMIT", 1000)
	return &PgConection{
		Id:          fmt.Sprintf("db:%s:%s:%s", DriverPostgres, host, database),
		Host:        host,
		Port:        port,
		Database:    database,
		User:        user,
		Password:    password,
		Sslmode:     sslmode,
		AppName:     appName,
		RecordLimit: recordLimit,
		Timeout:     1 * time.Hour,
		Pool:        poolFromEnv(),
	}
}

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *PgConection) GetParams() et.Json {
	return s.withPool(et.Json{
		"id":           s.Id,
		"driver":       DriverPostgres,
		"host":         s.Host,
		"port":         s.Port,
		"database":     s.Database,
		"user":         s.User,
		"password":     s.Password,
		"sslmode":      s.Sslmode,
		"app_name":     s.AppName,
		"record_limit": s.RecordLimit,
		"timeout":      s.Timeout,
	})
}

/**
* ID: Returns the connection ID.
* @return string
**/
func (s *PgConection) ID() string {
	return s.Id
}

/**
* SetId: Sets the connection ID.
* @param id string
**/
func (s *PgConection) SetId(id string) {
	s.Id = id
}

/**
* SetDatabase: Sets the database name in the connection parameters.
* @param name string
**/
func (s *PgConection) SetDatabase(name string) {
	s.Database = name
}

/**
* SetPassword: Sets the password in the connection parameters.
* @param password string
**/
func (s *PgConection) SetPassword(password string) {
	s.Password = password
}

/**
* GetDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *PgConection) GetDatabase() string {
	return s.Database
}

type SqliteConection struct {
	Id           string
	File         string
	RecordLimit  int
	PoolMaxOpen  int
	PoolMaxIdle  int
	PoolLifetime int
	PoolIdleTime int
	AppName      string
	Timeout      time.Duration
}

func sqliteConection(path string) *SqliteConection {
	file := envar.GetStr("DB_NAME", "josephine.db")
	if path != "" {
		file = fmt.Sprintf("%s/%s", path, file)
	}
	recordLimit := envar.GetInt("DB_RECORD_LIMIT", 1000)
	poolMaxOpen := envar.GetInt("DB_POOL_MAX_OPEN", 10)
	poolMaxIdle := envar.GetInt("DB_POOL_MAX_IDLE", 10)
	poolLifetime := envar.GetInt("DB_POOL_LIFETIME", 10)
	poolIdleTime := envar.GetInt("DB_POOL_IDLE_TIME", 10)
	appName := envar.GetStr("DB_APP_NAME", "josephine")
	return &SqliteConection{
		Id:           fmt.Sprintf("db:%s:%s", DriverSqlite, file),
		File:         file,
		RecordLimit:  recordLimit,
		PoolMaxOpen:  poolMaxOpen,
		PoolMaxIdle:  poolMaxIdle,
		PoolLifetime: poolLifetime,
		PoolIdleTime: poolIdleTime,
		AppName:      appName,
		Timeout:      1 * time.Hour,
	}
}

/**
* ID: Returns the connection ID.
* @return string
**/
func (s *SqliteConection) ID() string {
	return s.Id
}

/**
* SetId: Sets the connection ID.
* @param id string
**/
func (s *SqliteConection) SetId(id string) {
	s.Id = id
}

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *SqliteConection) GetParams() et.Json {
	return et.Json{
		"id":             s.Id,
		"driver":         DriverSqlite,
		"file":           s.File,
		"record_limit":   s.RecordLimit,
		"pool_max_open":  s.PoolMaxOpen,
		"pool_max_idle":  s.PoolMaxIdle,
		"pool_lifetime":  s.PoolLifetime,
		"pool_idle_time": s.PoolIdleTime,
		"app_name":       s.AppName,
		"timeout":        s.Timeout,
	}
}

/**
* SetDatabase: Sets the database file in the connection parameters.
* @param name string
**/
func (s *SqliteConection) SetDatabase(name string) {
	s.File = name
}

/**
* GetDatabase: Returns the database file from the connection parameters.
* @return string
**/
func (s *SqliteConection) GetDatabase() string {
	return s.File
}

/**
* SetPassword: Sets the password in the connection parameters.
* @param password string
**/
func (s *SqliteConection) SetPassword(password string) {
}

type OracleConection struct {
	Id          string
	Host        string
	Port        int
	Username    string
	Password    string
	ServiceName string
	SSL         bool
	SSLVerify   bool
	Timeout     time.Duration
	Pool
}

func oracleConection(host string) *OracleConection {
	port := envar.GetInt("DB_PORT", 1521)
	username := envar.GetStr("DB_USER", "test")
	password := envar.GetStr("DB_PASSWORD", "test")
	serviceName := envar.GetStr("DB_NAME", "josephine")
	ssl := envar.GetBool("DB_SSL", false)
	sslVerify := envar.GetBool("DB_SSL_VERIFY", true)
	id := fmt.Sprintf("db:%s:%s:%s", DriverOracle, host, serviceName)
	return &OracleConection{
		Id:          id,
		Host:        host,
		Port:        port,
		Username:    username,
		Password:    password,
		ServiceName: serviceName,
		SSL:         ssl,
		SSLVerify:   sslVerify,
		Timeout:     1 * time.Hour,
		Pool:        poolFromEnv(),
	}
}

/**
* ID: Returns the connection ID.
* @return string
**/
func (s *OracleConection) ID() string {
	return s.Id
}

/**
* SetId: Sets the connection ID.
* @param id string
**/
func (s *OracleConection) SetId(id string) {
	s.Id = id
}

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *OracleConection) GetParams() et.Json {
	return s.withPool(et.Json{
		"id":           s.Id,
		"driver":       DriverOracle,
		"host":         s.Host,
		"port":         s.Port,
		"username":     s.Username,
		"password":     s.Password,
		"service_name": s.ServiceName,
		"ssl":          s.SSL,
		"ssl_verify":   s.SSLVerify,
		"timeout":      s.Timeout,
	})
}

/**
* SetDatabase: Sets the service name in the connection parameters.
* @param name string
**/
func (s *OracleConection) SetDatabase(name string) {
	s.ServiceName = name
}

/**
* GetDatabase: Returns the service name from the connection parameters.
* @return string
**/
func (s *OracleConection) GetDatabase() string {
	return s.ServiceName
}

/**
* SetPassword: Sets the password in the connection parameters.
* @param password string
**/
func (s *OracleConection) SetPassword(password string) {
	s.Password = password
}

/**
* MysqlConection: Connection parameters of a MySQL (8.0 or later) database.
**/
type MysqlConection struct {
	Id       string
	Host     string
	Port     int
	Database string
	User     string
	Password string
	Timeout  time.Duration
	Pool
}

/**
* mysqlConection: Reads the MySQL connection parameters from the environment.
* @param host string
* @return *MysqlConection
**/
func mysqlConection(host string) *MysqlConection {
	database := envar.GetStr("DB_NAME", "josephine")
	return &MysqlConection{
		Id:       fmt.Sprintf("db:%s:%s:%s", DriverMysql, host, database),
		Host:     host,
		Port:     envar.GetInt("DB_PORT", 3306),
		Database: database,
		User:     envar.GetStr("DB_USER", "root"),
		Password: envar.GetStr("DB_PASSWORD", ""),
		Timeout:  1 * time.Hour,
		Pool:     poolFromEnv(),
	}
}

/**
* ID: Returns the connection ID.
* @return string
**/
func (s *MysqlConection) ID() string {
	return s.Id
}

/**
* SetId: Sets the connection ID.
* @param id string
**/
func (s *MysqlConection) SetId(id string) {
	s.Id = id
}

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *MysqlConection) GetParams() et.Json {
	return s.withPool(et.Json{
		"id":       s.Id,
		"driver":   DriverMysql,
		"host":     s.Host,
		"port":     s.Port,
		"database": s.Database,
		"user":     s.User,
		"password": s.Password,
		"timeout":  s.Timeout,
	})
}

/**
* SetDatabase: Sets the database name in the connection parameters.
* @param name string
**/
func (s *MysqlConection) SetDatabase(name string) {
	s.Database = name
}

/**
* GetDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *MysqlConection) GetDatabase() string {
	return s.Database
}

/**
* SetPassword: Sets the password in the connection parameters.
* @param password string
**/
func (s *MysqlConection) SetPassword(password string) {
	s.Password = password
}

/**
* MssqlConection: Connection parameters of a SQL Server (2022 or later) database.
**/
type MssqlConection struct {
	Id       string
	Host     string
	Port     int
	Database string
	User     string
	Password string
	Timeout  time.Duration
	Pool
}

/**
* mssqlConection: Reads the SQL Server connection parameters from the environment.
* @param host string
* @return *MssqlConection
**/
func mssqlConection(host string) *MssqlConection {
	database := envar.GetStr("DB_NAME", "josephine")
	return &MssqlConection{
		Id:       fmt.Sprintf("db:%s:%s:%s", DriverMssql, host, database),
		Host:     host,
		Port:     envar.GetInt("DB_PORT", 1433),
		Database: database,
		User:     envar.GetStr("DB_USER", "sa"),
		Password: envar.GetStr("DB_PASSWORD", ""),
		Timeout:  1 * time.Hour,
		Pool:     poolFromEnv(),
	}
}

/**
* ID: Returns the connection ID.
* @return string
**/
func (s *MssqlConection) ID() string {
	return s.Id
}

/**
* SetId: Sets the connection ID.
* @param id string
**/
func (s *MssqlConection) SetId(id string) {
	s.Id = id
}

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *MssqlConection) GetParams() et.Json {
	return s.withPool(et.Json{
		"id":       s.Id,
		"driver":   DriverMssql,
		"host":     s.Host,
		"port":     s.Port,
		"database": s.Database,
		"user":     s.User,
		"password": s.Password,
		"timeout":  s.Timeout,
	})
}

/**
* SetDatabase: Sets the database name in the connection parameters.
* @param name string
**/
func (s *MssqlConection) SetDatabase(name string) {
	s.Database = name
}

/**
* GetDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *MssqlConection) GetDatabase() string {
	return s.Database
}

/**
* SetPassword: Sets the password in the connection parameters.
* @param password string
**/
func (s *MssqlConection) SetPassword(password string) {
	s.Password = password
}

/**
* ConnectionFromJson: Builds the Connection of driver from params (host, port, user, pool...); host is the default host.
* @param driver, host string, params et.Json
* @return Connection, error
**/
func ConnectionFromJson(driver, host string, params et.Json) (Connection, error) {
	host = params.ValStr(host, "host")
	result, err := getConnection(driver, host)
	if err != nil {
		return nil, err
	}

	timeout := func(def time.Duration) time.Duration {
		if _, ok := params["timeout"]; !ok {
			return def
		}
		return time.Duration(params.Int64("timeout")) * time.Millisecond
	}

	switch c := result.(type) {
	case *PgConection:
		c.Id = params.ValStr(c.Id, "id")
		c.Port = params.ValInt(c.Port, "port")
		c.Database = params.ValStr(c.Database, "database")
		c.User = params.ValStr(c.User, "user")
		c.Password = params.ValStr(c.Password, "password")
		c.Sslmode = params.ValStr(c.Sslmode, "sslmode")
		c.AppName = params.ValStr(c.AppName, "app_name")
		c.RecordLimit = params.ValInt(c.RecordLimit, "record_limit")
		c.Timeout = timeout(c.Timeout)
		c.readPool(params)
	case *SqliteConection:
		c.Id = params.ValStr(c.Id, "id")
		c.File = params.ValStr(c.File, "file")
		c.RecordLimit = params.ValInt(c.RecordLimit, "record_limit")
		c.PoolMaxOpen = params.ValInt(c.PoolMaxOpen, "pool_max_open")
		c.PoolMaxIdle = params.ValInt(c.PoolMaxIdle, "pool_max_idle")
		c.PoolLifetime = params.ValInt(c.PoolLifetime, "pool_lifetime")
		c.PoolIdleTime = params.ValInt(c.PoolIdleTime, "pool_idle_time")
		c.AppName = params.ValStr(c.AppName, "app_name")
		c.Timeout = timeout(c.Timeout)
	case *OracleConection:
		c.Id = params.ValStr(c.Id, "id")
		c.Port = params.ValInt(c.Port, "port")
		c.Username = params.ValStr(c.Username, "username")
		c.Password = params.ValStr(c.Password, "password")
		c.ServiceName = params.ValStr(c.ServiceName, "service_name")
		c.SSL = params.ValBool(c.SSL, "ssl")
		c.SSLVerify = params.ValBool(c.SSLVerify, "ssl_verify")
		c.Timeout = timeout(c.Timeout)
		c.readPool(params)
	case *MysqlConection:
		c.Id = params.ValStr(c.Id, "id")
		c.Port = params.ValInt(c.Port, "port")
		c.Database = params.ValStr(c.Database, "database")
		c.User = params.ValStr(c.User, "user")
		c.Password = params.ValStr(c.Password, "password")
		c.Timeout = timeout(c.Timeout)
		c.readPool(params)
	case *MssqlConection:
		c.Id = params.ValStr(c.Id, "id")
		c.Port = params.ValInt(c.Port, "port")
		c.Database = params.ValStr(c.Database, "database")
		c.User = params.ValStr(c.User, "user")
		c.Password = params.ValStr(c.Password, "password")
		c.Timeout = timeout(c.Timeout)
		c.readPool(params)
	}

	return result, nil
}

const (
	DriverPostgres = "postgres"
	DriverSqlite   = "sqlite"
	DriverMysql    = "mysql"
	DriverMssql    = "mssql"
	DriverOracle   = "oracle"
	DriverJosefina = "josefina" // jsql/drivers/josefina, over the postgres connection (PgConection)
)

/**
* Driver: Interface that every database backend must implement to generate SQL and manage connections.
* UseSchema tells whether the engine has schemas: with them a model's Table is schema.name, without
* them schema_name (see Model.setTable).
**/
type Driver interface {
	UseSchema() bool
	CreateDB(connection *ConnectParams, timeout ...time.Duration) error
	DropDB(db *DB, timeout ...time.Duration) error
	Connect(db *DB, timeout ...time.Duration) (*sql.DB, error)
	ExistModel(db *sql.DB, model *Model, timeout ...time.Duration) (bool, error)
	Load(model *Model, timeout ...time.Duration) (string, error)
	Drop(model *Model, timeout ...time.Duration) error
	Query(query *Query, timeout ...time.Duration) (string, error)
	Command(command *Command, timeout ...time.Duration) (string, error)
}

var drivers map[string]Driver

func init() {
	drivers = make(map[string]Driver)
}

/**
* register: Registers a Driver implementation under the given name so jsql can resolve it by config.
* @param name string
* @param driver Driver
**/
func register(name string, driver Driver) {
	drivers[name] = driver
}

/**
* TimeoutContext: Returns the context a driver runs an operation with: it expires after timeout[0], or never
* when there is no timeout or it is 0 (or negative), so the operation cannot fail by timeout.
* @param timeout ...time.Duration
* @return context.Context, context.CancelFunc
**/
func TimeoutContext(timeout ...time.Duration) (context.Context, context.CancelFunc) {
	if len(timeout) == 0 || timeout[0] <= 0 {
		return context.Background(), func() {}
	}

	return context.WithTimeout(context.Background(), timeout[0])
}

/**
* Tester: A driver that can test a connection without changing anything: it opens it, pings it, runs the query (if
* any) and closes it. Unlike Connect, it never creates the database (postgres, mysql and mssql create it if missing).
**/
type Tester interface {
	Test(ctx context.Context, connection Connection, query string) error
}

/**
* TestConnection: Tests a connection with its driver (the "driver" of its params), without creating anything; the
* context bounds how long it may take. A driver that does not implement Tester answers MSG_DRIVER_CANNOT_TEST.
* @param ctx context.Context, connection Connection, query string
* @return error
**/
func TestConnection(ctx context.Context, connection Connection, query string) error {
	name := connection.GetParams().Str("driver")
	driver, ok := drivers[name]
	if !ok {
		return fmt.Errorf("%s: %s", MSG_DRIVER_NOT_FOUND, name)
	}

	tester, ok := driver.(Tester)
	if !ok {
		return fmt.Errorf(MSG_DRIVER_CANNOT_TEST, name)
	}

	return tester.Test(ctx, connection, query)
}

/**
* Probe: What every Tester does once the connection is open: runs the query (if any), reads nothing and closes it.
* @param ctx context.Context, db *sql.DB, query string
* @return error
**/
func Probe(ctx context.Context, db *sql.DB, query string) error {
	defer db.Close()
	if query == "" {
		return nil
	}

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}

	return rows.Close()
}

var (
	ErrRecordAlreadyExists = errors.New("record already exists")
	ErrUpsertWhereRequired = errors.New("upsert requires a where")
)

type ConnectParams struct {
	ID          string        `json:"id"`
	Driver      string        `json:"driver"`
	Host        string        `json:"host"`
	Name        string        `json:"name"`
	Connection  Connection    `json:"connection"`
	RecordLimit int           `json:"record_limit"`
	Timeout     time.Duration `json:"timeout"`
	IsDebug     bool          `json:"is_debug"`
}

/**
* connectTo: Creates a DB from params and initializes it (opens the connection).
* @param params ConnectParams
* @return *DB, error
**/
func connectTo(params ConnectParams) (*DB, error) {
	result, err := NewDB(params)
	if err != nil {
		return nil, err
	}

	err = result.Init()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* loadTo: Connects to dbName with the driver and settings of the environment (DB_*); hostName overrides DB_HOST.
* @param dbName string, hostName ...string
* @return *DB, error
**/
func loadTo(dbName string, hostName ...string) (*DB, error) {
	driver := envar.GetStr("DB_DRIVER", DriverPostgres)
	host := envar.GetStr("DB_HOST", "localhost")
	if len(hostName) > 0 {
		host = hostName[0]
	}

	connection, err := getConnection(driver, host)
	if err != nil {
		return nil, err
	}

	connection.SetDatabase(dbName)
	recordLimit := envar.GetInt("DB_RECORD_LIMIT", 1000)
	isDebug := envar.GetBool("DB_IS_DEBUG", false)
	result, err := ConnectTo(ConnectParams{
		ID:          connection.ID(),
		Driver:      driver,
		Host:        host,
		Name:        dbName,
		Connection:  connection,
		RecordLimit: recordLimit,
		IsDebug:     isDebug,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* load: Connects to the default database reading configuration from environment variables.
* @return *DB, error
**/
func load() (*DB, error) {
	name := envar.GetStr("DB_NAME", "josephine")
	return LoadTo(name)
}

/**
* createDB: Creates the database of the connection params with the driver, when it does not exist.
* @param connection *ConnectParams, timeout ...time.Duration (none or 0: no timeout)
* @return error
**/
func createDB(connection *ConnectParams, timeout ...time.Duration) error {
	if drivers[connection.Driver] == nil {
		return errors.New(MSG_DRIVER_NOT_FOUND)
	}

	driver := drivers[connection.Driver]
	return driver.CreateDB(connection, timeout...)
}

/**
* dropDB: Closes the connection pool (if open) and drops the database of the connection params with the driver.
* @param db *DB, timeout ...time.Duration (none or 0: no timeout)
* @return error
**/
func dropDB(db *DB, timeout ...time.Duration) error {
	if db.driver == nil {
		return errors.New(MSG_DRIVER_NOT_FOUND)
	}

	if db.db != nil {
		db.db.Close()
		db.db = nil
		db.isInit = false
	}

	return db.driver.DropDB(db, timeout...)
}
