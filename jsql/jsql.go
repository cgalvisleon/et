package jsql

import (
	"errors"
	"fmt"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

var (
	packageName            = "jsql"
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
* @param tenantId, host, driver, name string
* @return *DB, error
**/
func ConnectTo(tenantId, host, driver, name string) (*DB, error) {
	connect, err := GetConnection(driver, host)
	if err != nil {
		return nil, err
	}

	connect.SetDatabase(name)
	params := connect.GetParams()
	result, err := newDB(tenantId, name, params, nil)
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
	driver := envar.GetStr("DB_DRIVER", DriverPostgres)
	host := envar.GetStr("DB_HOST", "localhost")
	result, err := ConnectTo("tenant:root", host, driver, name)
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

/**
* NewDB: Constructs a DB instance from the given config, resolving the driver by name.
* @param params et.Json, store Store, userId string
* @return *DB, error
**/
func NewDB(tenantId, name string, params et.Json, store Store, userId string) (*DB, error) {
	result, err := newDB(tenantId, name, params, store)
	if err != nil {
		return nil, err
	}
	result.addAuditLog(userId, "new_db")
	err = result.Init()
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* LoadDB: Loads a DB instance from the given id and store.
* @param id string, store Store
* @return *DB, error
**/
func LoadDB(store Store, id string) (*DB, error) {
	result := &DB{
		ID:    id,
		store: store,
	}

	err := result.Load()
	if err != nil {
		return nil, err
	}

	err = result.Init()
	if err != nil {
		return nil, err
	}

	return result, nil
}
