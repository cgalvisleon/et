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
* @param ctx context.Context
* @param db *jsql.DB
* @return *sql.DB, error
**/
func (s *Sqlite) Connect(ctx context.Context, db *jsql.DB) (*sql.DB, error) {
	params := db.Params
	name := params.GetStr("DB_NAME", "")
	if name == "" {
		return nil, fmt.Errorf("DB_NAME is required for sqlite driver")
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

	maxOpen := params.GetInt("DB_POOL_MAX_OPEN", 1)
	maxIdle := params.GetInt("DB_POOL_MAX_IDLE", 1)
	connLifetime := params.GetInt("DB_POOL_CONN_LIFETIME", 900)
	connIdleTime := params.GetInt("DB_POOL_CONN_IDLE_TIME", 300)

	result.SetMaxOpenConns(maxOpen)
	result.SetMaxIdleConns(maxIdle)
	result.SetConnMaxLifetime(time.Duration(connLifetime) * time.Second)
	result.SetConnMaxIdleTime(time.Duration(connIdleTime) * time.Second)

	logs.Logf("Sqlite", "Connected file:%s", name)
	return result, nil
}
