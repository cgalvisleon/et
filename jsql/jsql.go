package jsql

import (
	"errors"
	"time"

	"github.com/cgalvisleon/et/envar"
)

var (
	ErrRecordAlreadyExists = errors.New("record already exists")
	ErrUpsertWhereRequired = errors.New("upsert requires a where")
)

type ConnectParams struct {
	ID          string        `json:"id"`
	Driver      string        `json:"driver"`
	Host        string        `json:"host"`
	Name        string        `json:"name"`
	Connection  Connection    `json:"connection"`
	RecordLimit int           `json:"record_limit"`
	Timeout     time.Duration `json:"timeout"`
	IsDebug     bool          `json:"is_debug"`
}

/**
* connectTo: Returns an existing DB by name, or creates and initialises a new one from params.
* @param tenantId, host, driver, name string, showLog bool
* @return *DB, error
**/
func connectTo(params ConnectParams) (*DB, error) {
	result, err := NewDB(params)
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
* loadTo: Returns an existing DB by name.
* @param name, hostName string
* @return *DB, error
**/
func loadTo(dbName string, hostName ...string) (*DB, error) {
	driver := envar.GetStr("DB_DRIVER", DriverPostgres)
	host := envar.GetStr("DB_HOST", "localhost")
	if len(hostName) > 0 {
		host = hostName[0]
	}

	connection, err := getConnection(driver, host)
	if err != nil {
		return nil, err
	}

	connection.SetDatabase(dbName)
	recordLimit := envar.GetInt("DB_RECORD_LIMIT", 1000)
	isDebug := envar.GetBool("DB_IS_DEBUG", false)
	result, err := ConnectTo(ConnectParams{
		ID:          connection.ID(),
		Driver:      driver,
		Host:        host,
		Name:        dbName,
		Connection:  connection,
		RecordLimit: recordLimit,
		IsDebug:     isDebug,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

/**
* load: Connects to the default database reading configuration from environment variables.
* @return *DB, error
**/
func load() (*DB, error) {
	name := envar.GetStr("DB_NAME", "josephine")
	return LoadTo(name)
}

/**
* createDB: Creates the database of the connection params with the driver, when it does not exist.
* @param connection *ConnectParams, timeout ...time.Duration (none or 0: no timeout)
* @return error
**/
func createDB(connection *ConnectParams, timeout ...time.Duration) error {
	if drivers[connection.Driver] == nil {
		return errors.New(MSG_DRIVER_NOT_FOUND)
	}

	driver := drivers[connection.Driver]
	return driver.CreateDB(connection, timeout...)
}

/**
* dropDB: Closes the connection pool (if open) and drops the database of the connection params with the driver.
* @param db *DB, timeout ...time.Duration (none or 0: no timeout)
* @return error
**/
func dropDB(db *DB, timeout ...time.Duration) error {
	if db.driver == nil {
		return errors.New(MSG_DRIVER_NOT_FOUND)
	}

	if db.db != nil {
		db.db.Close()
		db.db = nil
		db.isInit = false
	}

	return db.driver.DropDB(db, timeout...)
}
