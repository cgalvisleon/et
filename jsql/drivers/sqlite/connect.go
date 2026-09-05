package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
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
* connectTo: Opens the SQLite file at path and applies PRAGMAs needed for
* correctness (foreign keys) and concurrent reads (WAL journal mode).
* @param ctx context.Context, path string
* @return *sql.DB, error
**/
func connectTo(ctx context.Context, path string) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	result, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if err := result.PingContext(ctx); err != nil {
		result.Close()
		return nil, err
	}

	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA journal_mode = WAL;",
		"PRAGMA busy_timeout = 5000;",
	} {
		if _, err := result.ExecContext(ctx, pragma); err != nil {
			result.Close()
			return nil, err
		}
	}

	return result, nil
}

/**
* Connect: Opens the SQLite database file described by db.Params ("name" holds the
* file path) and configures the connection pool. A single writer connection is
* enforced (SQLite allows only one writer at a time); WAL mode still allows
* concurrent readers.
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

	if db.ShowLog() {
		logs.Logf("Sqlite", "Connected db:%s", path)
	}
	return result, nil
}
