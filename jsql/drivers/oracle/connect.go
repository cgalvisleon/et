package oracle

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/sijms/go-ora/v2"

	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
)

/**
* buildDSN: Returns the Oracle DSN in oracle://user:password@host:port/service format.
* DB_NAME is used as the Oracle service name (SID or service name).
* @param params utility.Config
* @return string
**/
func buildDSN(params utility.Config) string {
	host := params.GetStr("DB_HOST", "localhost")
	port := params.GetInt("DB_PORT", 1521)
	user := params.GetStr("DB_USER", "system")
	password := params.GetStr("DB_PASSWORD", "")
	service := params.GetStr("DB_NAME", "XE")
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s", user, password, host, port, service)
}

/**
* connectTo: Opens and pings an Oracle connection using the given DSN.
* @param dsn string
* @return *sql.DB, error
**/
func connectTo(dsn string) (*sql.DB, error) {
	result, err := sql.Open("oracle", dsn)
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
* Connect: Establishes an Oracle connection using parameters stored in db.
* Reads DB_HOST, DB_PORT, DB_USER, DB_PASSWORD and DB_NAME (service name) from db.Params.
* Unlike PostgreSQL/MySQL, Oracle schemas are users — no separate CREATE DATABASE step.
* @param db *jsql.DB
* @return *sql.DB, error
**/
func (s *Oracle) Connect(db *jsql.DB) (*sql.DB, error) {
	params := db.Params

	result, err := connectTo(buildDSN(params))
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
	port := params.GetInt("DB_PORT", 1521)
	service := params.GetStr("DB_NAME", "")
	logs.Logf("Oracle", "Connected host:%s:%d service:%s", host, port, service)
	return result, nil
}
