package jsql

import (
	"fmt"
	"time"

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
	GetParams() et.Json
	SetDatabase(string)
	GetDatabase() string
}

type PgConection struct {
	Host        string
	Port        int
	Database    string
	User        string
	Password    string
	Sslmode     string
	AppName     string
	RecordLimit int
	Timeout     time.Duration
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
		Host:        host,
		Port:        port,
		Database:    database,
		User:        user,
		Password:    password,
		Sslmode:     sslmode,
		AppName:     appName,
		RecordLimit: recordLimit,
		Timeout:     1 * time.Hour,
	}
}

/**
* getParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *PgConection) getParams() et.Json {
	return et.Json{
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
	}
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
* getParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *SqliteConection) getParams() et.Json {
	return et.Json{
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
* setDatabase: Sets the database name in the connection parameters
* @param name string
**/
func (s *SqliteConection) setDatabase(name string) {
	s.File = name
}

/**
* getDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *SqliteConection) getDatabase() string {
	return s.File
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
		"timeout":      s.Timeout,
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
	Timeout  time.Duration
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
		"timeout":  s.Timeout,
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
	Timeout  time.Duration
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
		"timeout":  s.Timeout,
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

/**
* connectionFromJson: Builds the Connection of a driver from its JSON, the inverse of GetParams (same keys,
* timeout in milliseconds). Missing keys keep the values read from the environment (DB_*), as in loadTo.
* @param driver, host string, params et.Json
* @return Connection, error
**/
func connectionFromJson(driver, host string, params et.Json) (Connection, error) {
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
		c.Port = params.ValInt(c.Port, "port")
		c.Database = params.ValStr(c.Database, "database")
		c.User = params.ValStr(c.User, "user")
		c.Password = params.ValStr(c.Password, "password")
		c.Sslmode = params.ValStr(c.Sslmode, "sslmode")
		c.AppName = params.ValStr(c.AppName, "app_name")
		c.RecordLimit = params.ValInt(c.RecordLimit, "record_limit")
		c.Timeout = timeout(c.Timeout)
	case *SqliteConection:
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
	case *MysqlConection:
		c.Id = params.ValStr(c.Id, "id")
		c.Port = params.ValInt(c.Port, "port")
		c.Database = params.ValStr(c.Database, "database")
		c.User = params.ValStr(c.User, "user")
		c.Password = params.ValStr(c.Password, "password")
		c.Timeout = timeout(c.Timeout)
	case *MssqlConection:
		c.Id = params.ValStr(c.Id, "id")
		c.Port = params.ValInt(c.Port, "port")
		c.Database = params.ValStr(c.Database, "database")
		c.User = params.ValStr(c.User, "user")
		c.Password = params.ValStr(c.Password, "password")
		c.Timeout = timeout(c.Timeout)
	}

	return result, nil
}
