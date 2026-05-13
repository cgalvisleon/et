package mssql

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	_ "github.com/microsoft/go-mssqldb"
)

/**
* masterDSN: Returns the DSN pointing to the master database, used to create the target DB.
* @param params utility.Config
* @return string
**/
func masterDSN(params utility.Config) string {
	host := params.GetStr("DB_HOST", "localhost")
	port := params.GetInt("DB_PORT", 1433)
	user := params.GetStr("DB_USER", "sa")
	password := params.GetStr("DB_PASSWORD", "")
	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=master",
		user, password, host, port)
}

/**
* dbDSN: Returns the DSN pointing to the target application database.
* @param params utility.Config
* @return string
**/
func dbDSN(params utility.Config) string {
	host := params.GetStr("DB_HOST", "localhost")
	port := params.GetInt("DB_PORT", 1433)
	user := params.GetStr("DB_USER", "sa")
	password := params.GetStr("DB_PASSWORD", "")
	name := params.GetStr("DB_NAME", "")
	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s",
		user, password, host, port, name)
}

/**
* connectTo: Opens and pings a SQL Server connection using the given DSN.
* @param dsn string
* @return *sql.DB, error
**/
func connectTo(dsn string) (*sql.DB, error) {
	result, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return nil, err
	}
	if err := result.Ping(); err != nil {
		result.Close()
		return nil, err
	}
	return result, nil
}

/**
* createDatabase: Creates the named database if it does not already exist.
* @param db *sql.DB
* @param name string
* @return error
**/
func createDatabase(db *sql.DB, name string) error {
	stmt := fmt.Sprintf(`
IF NOT EXISTS (SELECT 1 FROM sys.databases WHERE name = N'%s')
    CREATE DATABASE [%s];`, name, name)
	if _, err := db.Exec(stmt); err != nil {
		return err
	}
	logs.Logf("MSSQL", "Database [%s] ready", name)
	return nil
}

/**
* Connect: Establishes a SQL Server connection using parameters stored in db.
* Reads DB_HOST, DB_PORT, DB_USER, DB_PASSWORD and DB_NAME from db.Params.
* Creates the database if it does not exist, then reconnects with it selected.
* @param db *jsql.DB
* @return *sql.DB, error
**/
func (s *MSSQL) Connect(db *jsql.DB) (*sql.DB, error) {
	params := db.Params

	master, err := connectTo(masterDSN(params))
	if err != nil {
		return nil, err
	}

	name := params.GetStr("DB_NAME", "")
	if err := createDatabase(master, name); err != nil {
		master.Close()
		return nil, err
	}
	master.Close()

	result, err := connectTo(dbDSN(params))
	if err != nil {
		return nil, err
	}

	maxOpen := params.GetInt("DB_POOL_MAX_OPEN", 50)
	maxIdle := params.GetInt("DB_POOL_MAX_IDLE", 5)
	connLifetime := params.GetInt("DB_POOL_CONN_LIFETIME", 900)
	connIdleTime := params.GetInt("DB_POOL_CONN_IDLE_TIME", 300)

	result.SetMaxOpenConns(maxOpen)
	result.SetMaxIdleConns(maxIdle)
	result.SetConnMaxLifetime(time.Duration(connLifetime) * time.Second)
	result.SetConnMaxIdleTime(time.Duration(connIdleTime) * time.Second)

	host := params.GetStr("DB_HOST", "")
	port := params.GetInt("DB_PORT", 1433)
	logs.Logf("MSSQL", "Connected host:%s:%d db:%s", host, port, name)
	return result, nil
}
