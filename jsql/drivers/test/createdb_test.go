package test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cgalvisleon/et/jsql"
)

// createDBName is the scratch database TestCreateDropDB creates and drops.
const createDBName = "jsql_createdb_test"

/**
* findTarget: Returns a fresh copy of the named target (its Connection is a pointer, so each call has its own).
* @param name string
* @return target
**/
func findTarget(name string) target {
	for _, tg := range targets() {
		if tg.name == name {
			return tg
		}
	}
	return target{}
}

/**
* TestCreateDropDB: CreateDB creates the database of the params and DropDB drops it, both idempotent, in every
* driver available (B8).
* The existence is checked with jsql.TestConnection, which never creates the database. A timeout of 0
* (or none) never fails by timeout; one that already expired does.
**/
func TestCreateDropDB(t *testing.T) {
	const timeout = 30 * time.Second
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			server := findTarget(tg.name).params.Connection
			if err := jsql.TestConnection(ctx, server, ""); err != nil {
				t.Skipf("%s not available: %v", tg.name, err)
			}

			// Oracle: the service is not created nor dropped from a connection (spec §9): CreateDB only checks that it
			// answers and DropDB fails with MSG_DRIVER_CANNOT_DROP_DB
			if tg.name == jsql.DriverOracle {
				if err := jsql.CreateDB(&tg.params, timeout); err != nil {
					t.Fatalf("CreateDB: %v", err)
				}
				db, err := jsql.NewDB(tg.params)
				if err != nil {
					t.Fatal(err)
				}
				if err := jsql.DropDB(db, timeout); err == nil || err.Error() != fmt.Sprintf(jsql.MSG_DRIVER_CANNOT_DROP_DB, jsql.DriverOracle) {
					t.Fatalf("DropDB: got %v, want %s", err, fmt.Sprintf(jsql.MSG_DRIVER_CANNOT_DROP_DB, jsql.DriverOracle))
				}
				return
			}

			params := tg.params
			params.Name = createDBName
			params.Timeout = timeout
			params.Connection.SetDatabase(createDBName)
			db, err := jsql.NewDB(params)
			if err != nil {
				t.Fatal(err)
			}
			exists := func() bool {
				return jsql.TestConnection(ctx, params.Connection, "") == nil
			}

			if err := jsql.DropDB(db, timeout); err != nil {
				t.Fatalf("DropDB (leftover): %v", err)
			}
			t.Cleanup(func() { jsql.DropDB(db) })
			if exists() {
				t.Fatalf("database %s exists before CreateDB", createDBName)
			}

			if err := jsql.CreateDB(&params, time.Nanosecond); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("CreateDB (expired timeout): got %v, want %v", err, context.DeadlineExceeded)
			}
			if exists() {
				t.Fatalf("database %s exists after a CreateDB that timed out", createDBName)
			}

			if err := jsql.CreateDB(&params, timeout); err != nil {
				t.Fatalf("CreateDB: %v", err)
			}
			if !exists() {
				t.Fatalf("database %s does not exist after CreateDB", createDBName)
			}
			if err := jsql.CreateDB(&params, 0); err != nil {
				t.Fatalf("CreateDB (existing database, timeout 0): %v", err)
			}
			if err := jsql.CreateDB(&params); err != nil {
				t.Fatalf("CreateDB (existing database, no timeout): %v", err)
			}

			if err := db.Init(); err != nil {
				t.Fatalf("Init: %v", err)
			}
			if _, err := db.Sql(rawSelectOne(tg.name)); err != nil {
				t.Fatalf("Sql: %v", err)
			}

			if err := jsql.DropDB(db, time.Nanosecond); !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("DropDB (expired timeout): got %v, want %v", err, context.DeadlineExceeded)
			}
			if !exists() {
				t.Fatalf("database %s does not exist after a DropDB that timed out", createDBName)
			}

			if err := jsql.DropDB(db, 0); err != nil {
				t.Fatalf("DropDB (timeout 0): %v", err)
			}
			if exists() {
				t.Fatalf("database %s exists after DropDB", createDBName)
			}
			if err := jsql.DropDB(db); err != nil {
				t.Fatalf("DropDB (missing database, no timeout): %v", err)
			}
		})
	}
}
