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
	return &SqliteConection{
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
* getParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *SqliteConection) getParams() et.Json {
	return et.Json{
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
	return &OracleConection{
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
