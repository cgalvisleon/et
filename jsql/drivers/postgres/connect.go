package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/cgalvisleon/et/utility"
	_ "github.com/lib/pq"
)

/**
* defaultChain: Returns the default connection string for PostgreSQL
* @param params utility.Config
* @return string
**/
func (s *Postgres) defaultChain(params utility.Config) string {
	host := params.GetStr("DB_HOST", "localhost")
	port := params.GetInt("DB_PORT", 5432)
	user := params.GetStr("DB_USER", "postgres")
	password := params.GetStr("DB_PASSWORD", "")
	name := "postgres"
	sslMode := params.GetStr("DB_SSL_MODE", "disable")
	appName := params.GetStr("DB_APP_NAME", "et")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&application_name=%s",
		user, password, host, port, name, sslMode, appName,
	)
	return dsn
}

/**
* chain: Returns the connection string for the named database specified in DB_NAME.
* @param params utility.Config
* @return string
**/
func (s *Postgres) chain(params utility.Config) string {
	host := params.GetStr("DB_HOST", "localhost")
	port := params.GetInt("DB_PORT", 5432)
	user := params.GetStr("DB_USER", "postgres")
	password := params.GetStr("DB_PASSWORD", "")
	name := params.GetStr("DB_NAME", "")
	sslMode := params.GetStr("DB_SSL_MODE", "disable")
	appName := params.GetStr("DB_APP_NAME", "et")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&application_name=%s",
		user, password, host, port, name, sslMode, appName,
	)
	return dsn
}

/**
* connectTo: Establishes a PostgreSQL connection using the provided connection string.
* @param chain string
* @return *sql.DB, error
**/
func (s *Postgres) connectTo(chain string) (*sql.DB, error) {
	result, err := sql.Open("postgres", chain)
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
* Connect: Establishes a PostgreSQL connection using the parameters stored in db.
* Reads DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME and DB_SSL_MODE from db.Params.
* @param db *jsql.DB
* @return *sql.DB, error
**/
func (s *Postgres) Connect(db *jsql.DB) (*sql.DB, error) {
	params := db.Params
	dsn := s.defaultChain(params)
	result, err := s.connectTo(dsn)
	if err != nil {
		return nil, err
	}

	database := params.GetStr("DB_NAME", "")
	err = CreateDatabase(result, database)
	if err != nil {
		return nil, err
	}

	dsn = s.chain(params)
	result, err = s.connectTo(dsn)
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
	port := params.GetInt("DB_PORT", 5432)
	name := params.GetStr("DB_NAME", "")
	logs.Logf("Postgres", "Connected host:%s:%d db:%s", host, port, name)
	return result, nil
}
