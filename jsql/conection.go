package jsql

import "github.com/cgalvisleon/et/et"

type Connection interface {
	GetParams() et.Json
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

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (c *PgConection) GetParams() et.Json {
	return et.Json{
		"driver":       DriverPostgres,
		"database":     c.Database,
		"host":         c.Host,
		"port":         c.Port,
		"user":         c.User,
		"password":     c.Password,
		"sslmode":      c.Sslmode,
		"use_core":     c.UseCore,
		"app_name":     c.AppName,
		"record_limit": c.RecordLimit,
	}
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

/**
* GetParams: Returns the connection parameters as a JSON object.
* @return et.Json
**/
func (c *SqliteConection) GetParams() et.Json {
	return et.Json{
		"driver":         DriverSqlite,
		"name":           c.Name,
		"record_limit":   c.RecordLimit,
		"pool_max_open":  c.PoolMaxOpen,
		"pool_max_idle":  c.PoolMaxIdle,
		"pool_lifetime":  c.PoolLifetime,
		"pool_idle_time": c.PoolIdleTime,
		"app_name":       c.AppName,
	}
}
