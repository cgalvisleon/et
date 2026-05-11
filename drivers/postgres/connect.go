package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
)

/**
* Connect: Establishes a PostgreSQL connection using the parameters stored in db.
* Reads DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME and DB_SSL_MODE from db.Params.
* @param db *jsql.DB
* @return *sql.DB, error
**/
func (s *Postgres) Connect(db *jsql.DB) (*sql.DB, error) {
	params := db.Params
	host := params.GetStr("DB_HOST", "localhost")
	port := params.GetInt("DB_PORT", 5432)
	user := params.GetStr("DB_USER", "postgres")
	password := params.GetStr("DB_PASSWORD", "")
	name := params.GetStr("DB_NAME", "postgres")
	sslMode := params.GetStr("DB_SSL_MODE", "disable")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		user, password, host, port, name, sslMode,
	)

	result, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := result.Ping(); err != nil {
		result.Close()
		return nil, err
	}

	maxOpen := params.GetInt("DB_POOL_MAX_OPEN", 25)
	maxIdle := params.GetInt("DB_POOL_MAX_IDLE", 5)
	connLifetime := params.GetInt("DB_POOL_CONN_LIFETIME", 900)
	connIdleTime := params.GetInt("DB_POOL_CONN_IDLE_TIME", 300)

	result.SetMaxOpenConns(maxOpen)
	result.SetMaxIdleConns(maxIdle)
	result.SetConnMaxLifetime(time.Duration(connLifetime) * time.Second)
	result.SetConnMaxIdleTime(time.Duration(connIdleTime) * time.Second)

	logs.Logf("Postgres", "Connected host:%s:%d db:%s", host, port, name)
	return result, nil
}
