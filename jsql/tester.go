package jsql

import (
	"context"
	"database/sql"
	"fmt"
)

/**
* Tester: A driver that can test a connection without changing anything: it opens it, pings it, runs the query (if
* any) and closes it. Unlike Connect, it never creates the database (postgres, mysql and mssql create it if missing).
**/
type Tester interface {
	Test(ctx context.Context, connection Connection, query string) error
}

/**
* TestConnection: Tests a connection with its driver (the "driver" of its params), without creating anything; the
* context bounds how long it may take. A driver that does not implement Tester answers MSG_DRIVER_CANNOT_TEST.
* @param ctx context.Context, connection Connection, query string
* @return error
**/
func TestConnection(ctx context.Context, connection Connection, query string) error {
	name := connection.GetParams().Str("driver")
	driver, ok := drivers[name]
	if !ok {
		return fmt.Errorf("%s: %s", MSG_DRIVER_NOT_FOUND, name)
	}

	tester, ok := driver.(Tester)
	if !ok {
		return fmt.Errorf(MSG_DRIVER_CANNOT_TEST, name)
	}

	return tester.Test(ctx, connection, query)
}

/**
* Probe: What every Tester does once the connection is open: runs the query (if any), reads nothing and closes it.
* @param ctx context.Context, db *sql.DB, query string
* @return error
**/
func Probe(ctx context.Context, db *sql.DB, query string) error {
	defer db.Close()
	if query == "" {
		return nil
	}

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return err
	}

	return rows.Close()
}
