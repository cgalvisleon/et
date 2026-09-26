package oracle

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	go_ora "github.com/sijms/go-ora/v2"
)

/**
* chain: Returns the go-ora connection URL built from the connection params.
* @param params et.Json
* @return string
**/
func chain(params et.Json) string {
	host := params.ValStr("localhost", "host")
	port := params.ValInt(1521, "port")
	user := params.ValStr("", "username")
	password := params.ValStr("", "password")
	service := params.ValStr("", "service_name")
	options := map[string]string{}
	if params.Bool("ssl") {
		options["SSL"] = "true"
		options["SSL VERIFY"] = strconv.FormatBool(params.Bool("ssl_verify"))
	}

	return go_ora.BuildUrl(host, port, service, user, password, options)
}

/**
* connectTo: Establishes an Oracle connection using the provided connection string.
* @param ctx context.Context
* @param chain string
* @return *sql.DB, error
**/
func connectTo(ctx context.Context, chain string) (*sql.DB, error) {
	result, err := sql.Open("oracle", chain)
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
* Connect: Establishes an Oracle connection using the parameters stored in db.
* Reads host, port, username, password, service_name, ssl and ssl_verify from db.Params.
* Unlike postgres, the service (database) is not created: it must already exist.
* @param ctx context.Context, db *jsql.DB
* @return *sql.DB, error
**/
func (s *Oracle) Connect(ctx context.Context, db *jsql.DB) (*sql.DB, error) {
	params := db.Params
	if params.ValStr("", "service_name") == "" {
		return nil, errors.New("service_name is required")
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
