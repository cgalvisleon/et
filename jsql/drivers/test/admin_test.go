package test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* TestAdmin: The administration descriptors of DB.Query (jsql spec §8.3) in every driver available:
* truncate and drop of a model; on postgres and josefina also truncate and drop of a schema (cascade) and
* create, alter and drop of a user; a driver without a statement answers MSG_ADMIN_UNSUPPORTED.
**/
func TestAdmin(t *testing.T) {
	for _, tg := range targets() {
		if tg.name != jsql.DriverSqlite && tg.name != jsql.DriverPostgres && tg.name != jsql.DriverJosefina {
			continue
		}
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)
			schema := tg.schema
			if tg.name == jsql.DriverJosefina {
				schema = tg.schema + "a"
			}
			model := defineModel(t, db, schema, "admin_items", "name")
			ref := schema + "." + model.Name
			run := func(desc et.Json) error {
				t.Helper()
				_, err := db.Query(desc)
				return err
			}
			count := func() int {
				t.Helper()
				n, err := model.Count()
				if err != nil {
					t.Fatal(err)
				}
				return n
			}
			for i := 0; i < 3; i++ {
				if _, err := model.Insert(et.Json{"id": fmt.Sprint("a", i), "name": "x"}).Exec(); err != nil {
					t.Fatal(err)
				}
			}

			if err := run(et.Json{"truncate": et.Json{"model": ref}}); err != nil || count() != 0 {
				t.Fatalf("truncate model: %v, %d rows left", err, count())
			}
			if _, err := model.Insert(et.Json{"id": "b1", "name": "y"}).Exec(); err != nil || count() != 1 {
				t.Fatalf("insert after the truncate: %v", err)
			}

			if tg.name == jsql.DriverSqlite {
				if err := run(et.Json{"drop": et.Json{"schema": "main"}}); err == nil || !strings.Contains(err.Error(), "does not support") {
					t.Fatalf("drop schema on sqlite: %v", err)
				}
			} else {
				if err := run(et.Json{"truncate": et.Json{"schema": schema}}); err != nil || count() != 0 {
					t.Fatalf("truncate schema: %v, %d rows left", err, count())
				}
				user := strings.ReplaceAll(tg.schema, "jsql_", "u_")
				if err := run(et.Json{"create": et.Json{"user": user, "password": "first"}}); err != nil {
					t.Fatalf("create user: %v", err)
				}
				if err := run(et.Json{"alter": et.Json{"user": user, "password": "second"}}); err != nil {
					t.Fatalf("alter user: %v", err)
				}
				if err := run(et.Json{"drop": et.Json{"user": user}}); err != nil {
					t.Fatalf("drop user: %v", err)
				}
				if err := run(et.Json{"drop": et.Json{"user": user, "if_exists": true}}); err != nil {
					t.Fatalf("drop user if exists: %v", err)
				}
			}

			if err := run(et.Json{"drop": et.Json{"model": ref}}); err != nil {
				t.Fatalf("drop model: %v", err)
			}
			if _, err := db.GetModel(schema, model.Name); err == nil {
				t.Fatal("the dropped model is still in the DB")
			}
			if err := run(et.Json{"drop": et.Json{"model": ref, "if_exists": true}}); err != nil {
				t.Fatalf("drop model if exists: %v", err)
			}
			if tg.name != jsql.DriverSqlite {
				if err := run(et.Json{"drop": et.Json{"schema": schema, "cascade": true}}); err != nil {
					t.Fatalf("drop schema: %v", err)
				}
			}
		})
	}
}
