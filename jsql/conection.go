package jsql

import (
	"github.com/cgalvisleon/et/config"
	"github.com/cgalvisleon/et/et"
)

type Connection interface {
	GetParams() et.Json
	SetDatabase(string)
	GetDatabase() string
}

type PgConection struct {
	Database    string
	Host        string
	Port        int
	User        string
	Password    string
	Sslmode     string
	UseCore     bool
	AppName     string
	RecordLimit int
}

func pgConection() *PgConection {
	database := config.GetStr("DB_NAME", "josephine")
	host := config.GetStr("DB_HOST", "localhost")
	port := config.GetInt("DB_PORT", 5432)
	user := config.GetStr("DB_USER", "test")
	password := config.GetStr("DB_PASSWORD", "test")
	sslmode := config.GetStr("DB_SSLMODE", "disable")
	useCore := config.GetBool("DB_USE_CORE", false)
	appName := config.GetStr("DB_APP_NAME", "josephine")
	recordLimit := config.GetInt("DB_RECORD_LIMIT", 1000)
	return &PgConection{
		Database:    database,
		Host:        host,
		Port:        port,
		User:        user,
		Password:    password,
		Sslmode:     sslmode,
		UseCore:     useCore,
		AppName:     appName,
		RecordLimit: recordLimit,
	}
}

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *PgConection) GetParams() et.Json {
	return et.Json{
		"driver":       DriverPostgres,
		"database":     s.Database,
		"host":         s.Host,
		"port":         s.Port,
		"user":         s.User,
		"password":     s.Password,
		"sslmode":      s.Sslmode,
		"use_core":     s.UseCore,
		"app_name":     s.AppName,
		"record_limit": s.RecordLimit,
	}
}

/**
* SetDatabase: Sets the database name in the connection parameters.
* @param name string
**/
func (s *PgConection) SetDatabase(name string) {
	s.Database = name
}

/**
* GetDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *PgConection) GetDatabase() string {
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

func sqliteConection() *SqliteConection {
	name := config.GetStr("DB_NAME", "josephine.db")
	recordLimit := config.GetInt("DB_RECORD_LIMIT", 1000)
	poolMaxOpen := config.GetInt("DB_POOL_MAX_OPEN", 10)
	poolMaxIdle := config.GetInt("DB_POOL_MAX_IDLE", 10)
	poolLifetime := config.GetInt("DB_POOL_LIFETIME", 10)
	poolIdleTime := config.GetInt("DB_POOL_IDLE_TIME", 10)
	appName := config.GetStr("DB_APP_NAME", "josephine")
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
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (s *SqliteConection) GetParams() et.Json {
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
* SetDatabase: Sets the database name in the connection parameters
* @param name string
**/
func (s *SqliteConection) SetDatabase(name string) {
	s.Name = name
}

/**
* GetDatabase: Returns the database name from the connection parameters.
* @return string
**/
func (s *SqliteConection) GetDatabase() string {
	return s.Name
}
