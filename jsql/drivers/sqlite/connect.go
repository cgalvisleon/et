package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cgalvisleon/et/jsql"
	_ "modernc.org/sqlite"
)

/**
* dbPath: Resolves the SQLite database file path from the connection params.
* @param db *jsql.DB
* @return string, error
**/
func dbPath(db *jsql.DB) (string, error) {
	params := db.Params
	path := params.ValStr("", "name")
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
* Connect: Opens the SQLite database file described by db.Params ("name" holds the
* file path) and configures the connection pool. SQLite allows one writer at a
* time: WAL mode lets readers run alongside it and busy_timeout makes other
* writers wait for it.
* @param ctx context.Context, db *jsql.DB
* @return *sql.DB, error
**/
func (s *Sqlite) Connect(ctx context.Context, db *jsql.DB) (*sql.DB, error) {
	path, err := dbPath(db)
	if err != nil {
		return nil, err
	}

	result, err := connectTo(ctx, path)
	if err != nil {
		return nil, err
	}

	params := db.Params
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
