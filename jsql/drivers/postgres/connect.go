package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	"github.com/lib/pq"
)

/**
* defaultChain: Returns the default connection string for PostgreSQL
* @param params et.Json
* @return string
**/
func defaultChain(params et.Json) string {
	host := params.ValStr("localhost", "host")
	port := params.ValInt(5432, "port")
	user := params.ValStr("postgres", "user")
	password := params.ValStr("", "password")
	name := "postgres"
	sslMode := params.ValStr("disable", "sslmode")
	appName := params.ValStr("et", "app_name")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&application_name=%s",
		user, password, host, port, name, sslMode, appName,
	)
	return dsn
}

/**
* chain: Returns the connection string for the named database specified in DB_NAME.
* @param params et.Json
* @return string
**/
func chain(params et.Json) string {
	host := params.ValStr("localhost", "host")
	port := params.ValInt(5432, "port")
	user := params.ValStr("postgres", "user")
	password := params.ValStr("", "password")
	name := params.ValStr("", "name")
	sslMode := params.ValStr("disable", "sslmode")
	appName := params.ValStr("et", "app_name")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&application_name=%s",
		user, password, host, port, name, sslMode, appName,
	)
	return dsn
}

/**
* connectTo: Establishes a PostgreSQL connection using the provided connection string.
* @param ctx context.Context
* @param chain string
* @return *sql.DB, error
**/
func connectTo(ctx context.Context, chain string) (*sql.DB, error) {
	result, err := sql.Open("postgres", chain)
	if err != nil {
		return nil, err
	}

	if err := result.PingContext(ctx); err != nil {
		result.Close()
		return nil, err
	}

	return result, nil
}

/**
* connectWithRetry: Retries connectTo up to maxRetries times with exponential backoff.
* @param ctx context.Context
* @param dsn string
* @param maxRetries int
* @return *sql.DB, error
**/
func connectWithRetry(ctx context.Context, dsn string, maxRetries int) (*sql.DB, error) {
	delay := 500 * time.Millisecond
	var err error
	for i := 0; i <= maxRetries; i++ {
		var db *sql.DB
		db, err = connectTo(ctx, dsn)
		if err == nil {
			return db, nil
		}
		if i == maxRetries {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > 16*time.Second {
			delay = 16 * time.Second
		}
	}
	return nil, err
}

/**
* Connect: Establishes a PostgreSQL connection using the parameters stored in db.
* Reads DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME and DB_SSL_MODE from db.Params.
* @param ctx context.Context
* @param db *jsql.DB
* @return *sql.DB, error
**/
func (s *Postgres) Connect(ctx context.Context, db *jsql.DB) (*sql.DB, error) {
	params := db.Params
	dsn := defaultChain(params)
	result, err := connectWithRetry(ctx, dsn, 5)
	if err != nil {
		return nil, err
	}

	database := params.ValStr("", "database")
	if database == "" {
		result.Close()
		return nil, fmt.Errorf("database is required")
	}

	err = CreateDatabase(result, database)
	result.Close()
	if err != nil {
		return nil, err
	}

	dsn = chain(params)
	result, err = connectWithRetry(ctx, dsn, 5)
	if err != nil {
		return nil, err
	}

	maxOpen := params.ValInt(3, "pool_max_open")
	maxIdle := params.ValInt(1, "pool_max_idle")
	connLifetime := params.ValInt(30, "pool_lifetime")
	connIdleTime := params.ValInt(2, "pool_idle_time")

	result.SetMaxOpenConns(maxOpen)
	result.SetMaxIdleConns(maxIdle)
	result.SetConnMaxLifetime(time.Duration(connLifetime) * time.Minute)
	result.SetConnMaxIdleTime(time.Duration(connIdleTime) * time.Minute)

	host := params.ValStr("", "host")
	port := params.ValInt(5432, "port")
	logs.Logf("Postgres", "Connected host:%s:%d db:%s", host, port, database)
	return result, nil
}

/**
* ExistDatabase: Returns true when a database with the given name exists in the PostgreSQL instance.
* @param db *sql.DB
* @param name string
* @return bool, error
**/
func ExistDatabase(db *sql.DB, name string) (bool, error) {
	query := `
	SELECT EXISTS(
	SELECT 1
	FROM pg_database
	WHERE UPPER(datname) = UPPER($1));`
	rows, err := db.Query(query, name)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	items := jsql.RowsToItems(rows)
	if items.Count == 0 {
		return false, nil
	}

	return items.Bool(0, "exists"), nil
}

/**
* CreateDatabase: Creates a PostgreSQL database with the given name if it does not already exist.
* @param db *sql.DB
* @param name string
* @return error
**/
func CreateDatabase(db *sql.DB, name string) error {
	exist, err := ExistDatabase(db, name)
	if err != nil {
		return err
	}

	if exist {
		return nil
	}

	sql := fmt.Sprintf(`CREATE DATABASE %s;`, pq.QuoteIdentifier(name))
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	logs.Logf("Postgres", `Database %s created`, name)

	return nil
}

/**
* DropDatabase: Drops the PostgreSQL database with the given name.
* @param db *sql.DB
* @param name string
* @return error
**/
func DropDatabase(db *sql.DB, name string) error {
	sql := fmt.Sprintf(`DROP DATABASE %s;`, name)
	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	logs.Logf("Postgres", `Database %s dropped`, name)

	return nil
}
