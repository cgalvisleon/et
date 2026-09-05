package jsql

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
)

const (
	storeDb     = "dbs"
	storeModels = "models"
)

var (
	ErrRecordAlreadyExists = errors.New("record already exists")
)

/**
* GetConnection: Returns a Connection object based on the specified driver and environment variables.
* @param driver, host string
* @return Connection, error
**/
func GetConnection(driver, host string) (Connection, error) {
	switch driver {
	case DriverPostgres:
		config := pgConection(host)
		return config, nil
	case DriverSqlite:
		config := sqliteConection(host)
		return config, nil
	default:
		return nil, fmt.Errorf(MSG_UNSUPPORTED_DRIVER, driver)
	}
}

/**
* ConnectTo: Returns an existing DB by name, or creates and initialises a new one from params.
* @param tenantId, host, driver, name string, showLog bool
* @return *DB, error
**/
func ConnectTo(tenantId, host, driver, name string, showLog ...bool) (*DB, error) {
	result, err := NewDB(tenantId, host, name, driver, showLog...)
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
* LoadTo: Returns an existing DB by name.
* @param name string
* @return *DB, error
**/
func LoadTo(name string) (*DB, error) {
	tenantId := envar.GetStr("DB_TENANT_ID", "tenant:root")
	driver := envar.GetStr("DB_DRIVER", DriverPostgres)
	host := envar.GetStr("DB_HOST", "localhost")
	result, err := ConnectTo(tenantId, host, driver, name, true)
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* Load: Connects to the default database reading configuration from environment variables.
* @return *DB, error
**/
func Load() (*DB, error) {
	name := envar.GetStr("DB_NAME", "josephine")
	return LoadTo(name)
}
