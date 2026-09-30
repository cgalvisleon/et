package jsql

import (
	"context"
	"database/sql"
	"time"
)

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
	CreateDB(connection *ConnectParams, timeout ...time.Duration) error
	DropDB(db *DB, timeout ...time.Duration) error
	Connect(db *DB, timeout ...time.Duration) (*sql.DB, error)
	ExistModel(db *sql.DB, model *Model, timeout ...time.Duration) (bool, error)
	Load(model *Model, timeout ...time.Duration) (string, error)
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
