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
	user := params.ValStr("", "user")
	password := params.ValStr("", "password")
	database := "postgres"
	sslMode := params.ValStr("disable", "sslmode")
	if sslMode == "" {
		sslMode = "disable"
	}
	appName := params.ValStr("et", "app_name")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&application_name=%s",
		user, password, host, port, database, sslMode, appName,
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
	database := params.ValStr("", "database")
	sslMode := params.ValStr("disable", "sslmode")
	if sslMode == "" {
		sslMode = "disable"
	}
	appName := params.ValStr("et", "app_name")

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&application_name=%s",
		user, password, host, port, database, sslMode, appName,
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
* Connect: Establishes a PostgreSQL connection using the parameters stored in db; the database is created
* when it does not exist. With a timeout, connecting fails once it expires; none or 0 means no timeout.
* @param db *jsql.DB, timeout ...time.Duration
* @return *sql.DB, error
**/
func (s *Postgres) Connect(db *jsql.DB, timeout ...time.Duration) (*sql.DB, error) {
	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	params := db.Params
	err := createDB(ctx, params)
	if err != nil {
		return nil, err
	}

	result, err := connectWithRetry(ctx, chain(params), 5)
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

	return result, nil
}

/**
* createDB: Creates the database of the params ("database") when it does not exist, through the default
* "postgres" database of the server.
* @param ctx context.Context, params et.Json
* @return error
**/
func createDB(ctx context.Context, params et.Json) error {
	database := params.ValStr("", "database")
	if database == "" {
		return fmt.Errorf("database is required")
	}

	server, err := connectWithRetry(ctx, defaultChain(params), 5)
	if err != nil {
		return err
	}
	defer server.Close()

	return CreateDatabase(ctx, server, database)
}

/**
* CreateDB: Creates the database of the connection ("database") when it does not exist. It is idempotent: an
* existing database is left untouched. None or 0 timeout means no timeout.
* @param connection *jsql.ConnectParams, timeout ...time.Duration
* @return error
**/
func (s *Postgres) CreateDB(connection *jsql.ConnectParams, timeout ...time.Duration) error {
	if connection == nil || connection.Connection == nil {
		return fmt.Errorf(jsql.MSG_ATRIB_REQUIRED, "connection")
	}

	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	return createDB(ctx, connection.Connection.GetParams())
}

/**
* DropDB: Drops the database of db.Params ("database") when it exists, through the default "postgres"
* database of the server. It is idempotent, and it fails while other sessions are connected to the database.
* None or 0 timeout means no timeout.
* @param db *jsql.DB, timeout ...time.Duration
* @return error
**/
func (s *Postgres) DropDB(db *jsql.DB, timeout ...time.Duration) error {
	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	params := db.Params
	database := params.ValStr("", "database")
	if database == "" {
		return fmt.Errorf("database is required")
	}

	server, err := connectWithRetry(ctx, defaultChain(params), 5)
	if err != nil {
		return err
	}
	defer server.Close()

	exist, err := ExistDatabase(ctx, server, database)
	if err != nil {
		return err
	}

	if !exist {
		return nil
	}

	return DropDatabase(ctx, server, database)
}

/**
* ExistDatabase: Returns true when a database with the given name exists in the PostgreSQL instance.
* @param ctx context.Context, db *sql.DB, name string
* @return bool, error
**/
func ExistDatabase(ctx context.Context, db *sql.DB, name string) (bool, error) {
	query := `
	SELECT EXISTS(
	SELECT 1
	FROM pg_database
	WHERE UPPER(datname) = UPPER($1));`
	rows, err := db.QueryContext(ctx, query, name)
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
* @param ctx context.Context, db *sql.DB, name string
* @return error
**/
func CreateDatabase(ctx context.Context, db *sql.DB, name string) error {
	exist, err := ExistDatabase(ctx, db, name)
	if err != nil {
		return err
	}

	if exist {
		return nil
	}

	sql := fmt.Sprintf(`CREATE DATABASE %s;`, pq.QuoteIdentifier(name))
	_, err = db.ExecContext(ctx, sql)
	if err != nil {
		return err
	}

	logs.Logf("Postgres", `Database %s created`, name)

	return nil
}

/**
* DropDatabase: Drops the PostgreSQL database with the given name.
* @param ctx context.Context, db *sql.DB, name string
* @return error
**/
func DropDatabase(ctx context.Context, db *sql.DB, name string) error {
	sql := fmt.Sprintf(`DROP DATABASE %s;`, pq.QuoteIdentifier(name))
	_, err := db.ExecContext(ctx, sql)
	if err != nil {
		return err
	}

	logs.Logf("Postgres", `Database %s dropped`, name)

	return nil
}
