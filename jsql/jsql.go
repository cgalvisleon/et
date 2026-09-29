package jsql

import (
	"errors"

	"github.com/cgalvisleon/et/envar"
)

var (
	dbs                    = make(map[string]*DB)
	ErrRecordAlreadyExists = errors.New("record already exists")
	ErrUpsertWhereRequired = errors.New("upsert requires a where")
)

func init() {
	dbs = make(map[string]*DB)
}

type ConnectParams struct {
	ID          string     `json:"id"`
	Driver      string     `json:"driver"`
	Host        string     `json:"host"`
	Name        string     `json:"name"`
	Connection  Connection `json:"connection"`
	RecordLimit int        `json:"record_limit"`
	IsDebug     bool       `json:"is_debug"`
}

/**
* connectTo: Returns an existing DB by name, or creates and initialises a new one from params.
* @param tenantId, host, driver, name string, showLog bool
* @return *DB, error
**/
func connectTo(params ConnectParams) (*DB, error) {
	if _, ok := dbs[params.ID]; ok {
		return dbs[params.ID], nil
	}
	result, err := NewDB(params)
	if err != nil {
		return nil, err
	}

	err = result.Init()
	if err != nil {
		return nil, err
	}

	dbs[params.ID] = result
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
