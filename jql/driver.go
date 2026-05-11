package jql

import "database/sql"

const (
	DriverPostgres = "postgres"
	DriverSqlite   = "sqlite"
	DriverMysql    = "mysql"
	DriverMssql    = "mssql"
	DriverOracle   = "oracle"
	DriverJosefina = "josefina"
)

type Driver interface {
	Connect(db *DB) (*sql.DB, error)
	Load(model *Model) (string, error)
	Query(query *Query) (string, error)
	Command(command *Command) (string, error)
}

var drivers map[string]Driver

func init() {
	drivers = make(map[string]Driver)
}

func Register(name string, driver Driver) {
	drivers[name] = driver
}
