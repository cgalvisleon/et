package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/logs"
	_ "github.com/mattn/go-sqlite3"
)

/**
* Connect: Establishes a SQLite connection using the file path stored in DB_NAME.
* Enables foreign key enforcement via PRAGMA and configures the connection pool.
* SQLite is single-writer; DB_POOL_MAX_OPEN defaults to 1.
* @param ctx context.Context, db *jsql.DB
* @return *sql.DB, error
**/
func (s *Sqlite) Connect(ctx context.Context, db *jsql.DB) (*sql.DB, error) {
	params := db.Params
	name := params.ValStr("", "name")
	if name == "" {
		return nil, fmt.Errorf("name is required for sqlite driver")
	}

	result, err := sql.Open("sqlite3", name)
	if err != nil {
		return nil, err
	}

	if err := result.PingContext(ctx); err != nil {
		result.Close()
		return nil, err
	}

	if _, err := result.ExecContext(ctx, "PRAGMA foreign_keys = ON;"); err != nil {
		result.Close()
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

	logs.Logf("Sqlite", "Connected file:%s", name)
	return result, nil
}
