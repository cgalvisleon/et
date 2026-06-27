package jsql

import (
	"encoding/json"
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
* getConnection: Returns a Connection object based on the specified driver and environment variables.
* @param tenantId string
* @return Connection, error
**/
func getConnection(tenantId string) (Connection, error) {
	driver := envar.GetStr("DB_DRIVER", DriverPostgres)
	switch driver {
	case DriverPostgres:
		config := pgConection(tenantId)
		return config, nil
	case DriverSqlite:
		config := sqliteConection(tenantId)
		return config, nil
	default:
		return nil, fmt.Errorf(MSG_UNSUPPORTED_DRIVER, driver)
	}
}

/**
* ConnectTo: Returns an existing DB by name, or creates and initialises a new one from params.
* @param connect Connection
* @return *DB, error
**/
func ConnectTo(tenantId, name string, connect Connection) (*DB, error) {
	params := connect.GetParams()
	driver := params.Str("driver")
	result, err := newDB(tenantId, driver, name, params)
	if err != nil {
		return nil, err
	}

	err = result.init()
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
func LoadTo(tenantId, name string) (*DB, error) {
	conn, err := getConnection(tenantId)
	if err != nil {
		return nil, err
	}
	conn.SetDatabase(name)
	return ConnectTo(tenantId, name, conn)
}

/**
* Load: Connects to the default database reading configuration from environment variables.
* @return *DB, error
**/
func Load(tenantId string) (*DB, error) {
	conn, err := getConnection(tenantId)
	if err != nil {
		return nil, err
	}
	name := conn.GetDatabase()
	return ConnectTo(tenantId, name, conn)
}

/**
* NewDB: Constructs a DB instance from the given config, resolving the driver by name.
* @param params et.Json, store Store, userId string
* @return *DB, error
**/
func NewDB(tenantId, driver, name string, params et.Json, store Store, userId string) (*DB, error) {
	result, err := newDB(tenantId, driver, name, params)
	if err != nil {
		return nil, err
	}
	result.addAuditLog(userId, "new_db")
	return result.up(store)
}

/**
* LoadDB: Loads a DB instance from the given id and store.
* @param id string, store Store
* @return *DB, error
**/
func LoadDB(id string, store Store) (*DB, error) {
	if store == nil {
		return nil, errors.New(MSG_DB_STORE_IS_NIL)
	}

	var def et.Json
	exists, err := store.Get("db", id, &def)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, errors.New(MSG_DB_NOT_FOUND)
	}

	result := &DB{}
	err = json.Unmarshal([]byte(def.ToString()), &result)
	if err != nil {
		return nil, err
	}

	return result.up(store)
}
