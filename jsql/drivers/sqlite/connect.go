package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/cgalvisleon/et/et"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cgalvisleon/et/jsql"
	_ "modernc.org/sqlite"
)

/**
* dbPath: Resolves the SQLite database file path from the connection params.
* @param params et.Json
* @return string, error
**/
func dbPath(params et.Json) (string, error) {
	// "file" es la clave de SqliteConection; "name" era la de antes (un DB guardado con ToJson antes del cambio)
	path := params.ValStr(params.ValStr("", "name"), "file")
	if path == "" {
		return "", fmt.Errorf("database is required")
	}
	return path, nil
}

/**
* pragmas: PRAGMAs every pooled connection needs, for correctness (foreign keys)
* and concurrency (WAL journal mode, waiting on a locked database instead of
* failing). They go in the DSN so the driver runs them on each new connection,
* not only on the first one.
**/
var pragmas = []string{
	"foreign_keys(1)",
	"journal_mode(WAL)",
	"busy_timeout(5000)",
}

/**
* dsn: Returns path with the connection PRAGMAs as "_pragma" query parameters.
* @param path string
* @return string
**/
func dsn(path string) string {
	params := make([]string, len(pragmas))
	for i, pragma := range pragmas {
		params[i] = "_pragma=" + pragma
	}
	return path + "?" + strings.Join(params, "&")
}

/**
* connectTo: Opens the SQLite file at path, with the PRAGMAs applied to every connection.
* @param ctx context.Context, path string
* @return *sql.DB, error
**/
func connectTo(ctx context.Context, path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	result, err := sql.Open("sqlite", dsn(path))
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
* Connect: Opens the SQLite database file described by db.Params ("file" holds the
* file path) and configures the connection pool. SQLite allows one writer at a
* time: WAL mode lets readers run alongside it and busy_timeout makes other
* writers wait for it. With a timeout, opening fails once it expires; none or 0 means no timeout.
* @param db *jsql.DB, timeout ...time.Duration
* @return *sql.DB, error
**/
func (s *Sqlite) Connect(db *jsql.DB, timeout ...time.Duration) (*sql.DB, error) {
	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	params := db.Params
	path, err := dbPath(params)
	if err != nil {
		return nil, err
	}

	result, err := connectTo(ctx, path)
	if err != nil {
		return nil, err
	}

	maxOpen := params.ValInt(1, "pool_max_open")
	if maxOpen < 1 {
		maxOpen = 1
	}
	connLifetime := params.ValInt(10, "pool_lifetime")
	connIdleTime := params.ValInt(10, "pool_idle_time")

	result.SetMaxOpenConns(maxOpen)
	result.SetMaxIdleConns(maxOpen)
	result.SetConnMaxLifetime(time.Duration(connLifetime) * time.Minute)
	result.SetConnMaxIdleTime(time.Duration(connIdleTime) * time.Minute)

	return result, nil
}

/**
* CreateDB: Creates the SQLite database file of the connection ("file") when it does not exist. It is
* idempotent. None or 0 timeout means no timeout.
* @param connection *jsql.ConnectParams, timeout ...time.Duration
* @return error
**/
func (s *Sqlite) CreateDB(connection *jsql.ConnectParams, timeout ...time.Duration) error {
	if connection == nil || connection.Connection == nil {
		return fmt.Errorf(jsql.MSG_ATRIB_REQUIRED, "connection")
	}

	path, err := dbPath(connection.Connection.GetParams())
	if err != nil {
		return err
	}

	ctx, cancel := jsql.TimeoutContext(timeout...)
	defer cancel()

	result, err := connectTo(ctx, path)
	if err != nil {
		return err
	}

	return result.Close()
}

/**
* DropDB: Deletes the SQLite database file of db.Params ("file") with its WAL and shared-memory files.
* It is idempotent: a missing file is not an error. The timeout is not used: nothing waits.
* @param db *jsql.DB, timeout ...time.Duration
* @return error
**/
func (s *Sqlite) DropDB(db *jsql.DB, timeout ...time.Duration) error {
	path, err := dbPath(db.Params)
	if err != nil {
		return err
	}

	for _, suffix := range []string{"", "-wal", "-shm"} {
		err := os.Remove(path + suffix)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}
