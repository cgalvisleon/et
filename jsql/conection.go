package jsql

import (
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

/**
* getConnection: Returns a Connection object based on the specified driver and environment variables.
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
	GetParams() et.Json
	SetDatabase(string)
	GetDatabase() string
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
}

func pgConection(host string) *PgConection {
	port := envar.GetInt("DB_PORT", 5432)
	database := envar.GetStr("DB_NAME", "josephine")
	user := envar.GetStr("DB_USER", "test")
	password := envar.GetStr("DB_PASSWORD", "test")
	sslmode := envar.GetStr("DB_SSLMODE", "disable")
	appName := envar.GetStr("DB_APP_NAME", "josephine")
	recordLimit := envar.GetInt("DB_RECORD_LIMIT", 1000)
	id := fmt.Sprintf("db:%s:%s:%s", DriverPostgres, host, database)
	return &PgConection{
		Id:          id,
		Host:        host,
		Port:        port,
		Database:    database,
		User:        user,
		Password:    password,
		Sslmode:     sslmode,
		AppName:     appName,
		RecordLimit: recordLimit,
	}
}

/**
* getParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *PgConection) getParams() et.Json {
	return et.Json{
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
	}
}

/**
* ID: Returns the connection ID.
* @return string
**/
func (s *PgConection) id() string {
	return s.Id
}

/**
* setDatabase: Sets the database name in the connection parameters.
* @param name string
**/
func (s *PgConection) setDatabase(name string) {
	s.Database = name
}

/**
* getDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *PgConection) getDatabase() string {
	return s.Database
}

type SqliteConection struct {
	Id           string
	Name         string
	RecordLimit  int
	PoolMaxOpen  int
	PoolMaxIdle  int
	PoolLifetime int
	PoolIdleTime int
	AppName      string
}

func sqliteConection(path string) *SqliteConection {
	name := envar.GetStr("DB_NAME", "josephine.db")
	if path != "" {
		name = fmt.Sprintf("%s/%s", path, name)
	}
	recordLimit := envar.GetInt("DB_RECORD_LIMIT", 1000)
	poolMaxOpen := envar.GetInt("DB_POOL_MAX_OPEN", 10)
	poolMaxIdle := envar.GetInt("DB_POOL_MAX_IDLE", 10)
	poolLifetime := envar.GetInt("DB_POOL_LIFETIME", 10)
	poolIdleTime := envar.GetInt("DB_POOL_IDLE_TIME", 10)
	appName := envar.GetStr("DB_APP_NAME", "josephine")
	id := fmt.Sprintf("db:%s:%s", DriverSqlite, name)
	return &SqliteConection{
		Id:           id,
		Name:         name,
		RecordLimit:  recordLimit,
		PoolMaxOpen:  poolMaxOpen,
		PoolMaxIdle:  poolMaxIdle,
		PoolLifetime: poolLifetime,
		PoolIdleTime: poolIdleTime,
		AppName:      appName,
	}
}

/**
* ID: Returns the connection ID.
* @return string
**/
func (s *SqliteConection) id() string {
	return s.Id
}

/**
* getParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *SqliteConection) getParams() et.Json {
	return et.Json{
		"id":             s.Id,
		"driver":         DriverSqlite,
		"name":           s.Name,
		"record_limit":   s.RecordLimit,
		"pool_max_open":  s.PoolMaxOpen,
		"pool_max_idle":  s.PoolMaxIdle,
		"pool_lifetime":  s.PoolLifetime,
		"pool_idle_time": s.PoolIdleTime,
		"app_name":       s.AppName,
	}
}

/**
* setDatabase: Sets the database name in the connection parameters
* @param name string
**/
func (s *SqliteConection) setDatabase(name string) {
	s.Name = name
}

/**
* getDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *SqliteConection) getDatabase() string {
	return s.Name
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
	}
}

/**
* getParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *OracleConection) getParams() et.Json {
	return et.Json{
		"id":           s.Id,
		"driver":       DriverOracle,
		"host":         s.Host,
		"port":         s.Port,
		"username":     s.Username,
		"password":     s.Password,
		"service_name": s.ServiceName,
		"ssl":          s.SSL,
		"ssl_verify":   s.SSLVerify,
	}
}

/**
* id: Returns the connection ID.
* @return string
**/
func (s *OracleConection) id() string {
	return s.Id
}

/**
* setDatabase: Sets the database name in the connection parameters
* @param name string
**/
func (s *OracleConection) setDatabase(name string) {
	s.ServiceName = name
}

/**
* getDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *OracleConection) getDatabase() string {
	return s.ServiceName
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
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *MysqlConection) GetParams() et.Json {
	return et.Json{
		"id":       s.Id,
		"driver":   DriverMysql,
		"host":     s.Host,
		"port":     s.Port,
		"database": s.Database,
		"user":     s.User,
		"password": s.Password,
	}
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
* MssqlConection: Connection parameters of a SQL Server (2022 or later) database.
**/
type MssqlConection struct {
	Id       string
	Host     string
	Port     int
	Database string
	User     string
	Password string
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
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *MssqlConection) GetParams() et.Json {
	return et.Json{
		"id":       s.Id,
		"driver":   DriverMssql,
		"host":     s.Host,
		"port":     s.Port,
		"database": s.Database,
		"user":     s.User,
		"password": s.Password,
	}
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
