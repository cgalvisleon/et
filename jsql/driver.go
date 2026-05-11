package jsql

import "database/sql"

const (
	DriverPostgres = "postgres"
	DriverSqlite   = "sqlite"
	DriverMysql    = "mysql"
	DriverMssql    = "mssql"
	DriverOracle   = "oracle"
	DriverJosefina = "josefina"
)

/**
* Driver: Interface that every database backend must implement to generate SQL and manage connections.
**/
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

/**
* Register: Registers a Driver implementation under the given name so jsql can resolve it by config.
* @param name string
* @param driver Driver
**/
func Register(name string, driver Driver) {
	drivers[name] = driver
}
