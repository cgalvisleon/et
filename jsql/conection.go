package jsql

import "github.com/cgalvisleon/et/et"

type Connection interface {
	getParams() et.Json
}

type PgConection struct {
	Database     string
	Host         string
	Port         int
	User         string
	Password     string
	Sslmode      string
	UserCore     bool
	RecordLimit  int
	PoolMaxOpen  int
	PoolMaxIdle  int
	PoolLifetime int
	PoolIdleTime int
	AppName      string
}

func (c *PgConection) getParams() et.Json {
	return et.Json{
		"driver":         DriverPostgres,
		"database":       c.Database,
		"host":           c.Host,
		"port":           c.Port,
		"user":           c.User,
		"password":       c.Password,
		"sslmode":        c.Sslmode,
		"user_core":      c.UserCore,
		"record_limit":   c.RecordLimit,
		"pool_max_open":  c.PoolMaxOpen,
		"pool_max_idle":  c.PoolMaxIdle,
		"pool_lifetime":  c.PoolLifetime,
		"pool_idle_time": c.PoolIdleTime,
		"app_name":       c.AppName,
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

func (c *SqliteConection) getParams() et.Json {
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
