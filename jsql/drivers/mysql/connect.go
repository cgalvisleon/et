package mysql

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
)

/**
* rootDSN: Returns the DSN without a database name, used to create the target database.
* @param params utility.Config
* @return string
**/
func rootDSN(params utility.Config) string {
	host := params.GetStr("DB_HOST", "localhost")
	port := params.GetInt("DB_PORT", 3306)
	user := params.GetStr("DB_USER", "root")
	password := params.GetStr("DB_PASSWORD", "")
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=true&loc=UTC",
		user, password, host, port)
}

/**
* dbDSN: Returns the DSN with the target database name.
* @param params utility.Config
* @return string
**/
func dbDSN(params utility.Config) string {
	host := params.GetStr("DB_HOST", "localhost")
	port := params.GetInt("DB_PORT", 3306)
	user := params.GetStr("DB_USER", "root")
	password := params.GetStr("DB_PASSWORD", "")
	name := params.GetStr("DB_NAME", "")
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		user, password, host, port, name)
}

/**
* connectTo: Opens and pings a MySQL connection using the given DSN.
* @param dsn string
* @return *sql.DB, error
**/
func connectTo(dsn string) (*sql.DB, error) {
	result, err := sql.Open("mysql", dsn)
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
	stmt := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;",
		name,
	)
	if _, err := db.Exec(stmt); err != nil {
		return err
	}
	logs.Logf("MySQL", "Database `%s` ready", name)
	return nil
}

/**
* Connect: Establishes a MySQL connection using the parameters stored in db.
* Reads DB_HOST, DB_PORT, DB_USER, DB_PASSWORD and DB_NAME from db.Params.
* Creates the database if it does not exist, then reconnects with it selected.
* @param db *jsql.DB
* @return *sql.DB, error
**/
func (s *MySQL) Connect(db *jsql.DB) (*sql.DB, error) {
	params := db.Params

	root, err := connectTo(rootDSN(params))
	if err != nil {
		return nil, err
	}

	name := params.GetStr("DB_NAME", "")
	if err := createDatabase(root, name); err != nil {
		root.Close()
		return nil, err
	}
	root.Close()

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
	port := params.GetInt("DB_PORT", 3306)
	logs.Logf("MySQL", "Connected host:%s:%d db:%s", host, port, name)
	return result, nil
}
