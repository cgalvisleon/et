package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	driver "github.com/go-sql-driver/mysql"
)

/**
* dsn: Builds the connection string from the params; database may be empty to connect to the server.
* multiStatements lets a command run as one batch (DML + SELECT).
* @param params et.Json, database string
* @return string
**/
func dsn(params et.Json, database string) string {
	cfg := driver.NewConfig()
	cfg.User = params.ValStr("root", "user")
	cfg.Passwd = params.ValStr("", "password")
	cfg.Net = "tcp"
	cfg.Addr = fmt.Sprintf("%s:%d", params.ValStr("localhost", "host"), params.ValInt(3306, "port"))
	cfg.DBName = database
	cfg.MultiStatements = true
	cfg.ParseTime = true
	cfg.Params = map[string]string{"charset": "utf8mb4"}
	return cfg.FormatDSN()
}

/**
* connectTo: Opens a MySQL connection and checks it with a ping.
* @param ctx context.Context, dsn string
* @return *sql.DB, error
**/
func connectTo(ctx context.Context, dsn string) (*sql.DB, error) {
	result, err := sql.Open("mysql", dsn)
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
* @param ctx context.Context, dsn string, maxRetries int
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
* Connect: Connects to the server, creates the database when it does not exist and returns a
* connection to it, with the pool settings of db.Params.
* @param ctx context.Context, db *jsql.DB
* @return *sql.DB, error
**/
func (s *Mysql) Connect(ctx context.Context, db *jsql.DB) (*sql.DB, error) {
	params := db.Params
	database := params.ValStr("", "database")
	if database == "" {
		return nil, errors.New("database is required")
	}

	server, err := connectWithRetry(ctx, dsn(params, ""), 5)
	if err != nil {
		return nil, err
	}
	_, err = server.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", myIdent(database)))
	server.Close()
	if err != nil {
		return nil, err
	}

	result, err := connectWithRetry(ctx, dsn(params, database), 5)
	if err != nil {
		return nil, err
	}

	result.SetMaxOpenConns(params.ValInt(3, "pool_max_open"))
	result.SetMaxIdleConns(params.ValInt(1, "pool_max_idle"))
	result.SetConnMaxLifetime(time.Duration(params.ValInt(30, "pool_lifetime")) * time.Minute)
	result.SetConnMaxIdleTime(time.Duration(params.ValInt(2, "pool_idle_time")) * time.Minute)
	return result, nil
}
