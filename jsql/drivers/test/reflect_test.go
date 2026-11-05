package test

import (
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* TestReflectModel: A second DB on the same database, which defined no model (another process), runs JSON
* queries and commands on the tables of the first one: the models are read from the catalog (Describer),
* attributes of _source included. Also checks that two select fields with the same result name fail.
* @param t *testing.T
**/
func TestReflectModel(t *testing.T) {
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)
			clients := defineModel(t, db, tg.schema, "r_clients", "name")
			orders := defineModel(t, db, tg.schema, "r_orders", "client_id", "total")
			if _, err := clients.Insert(et.Json{"id": "c1", "name": "Ana", "city": "Cali"}).One(); err != nil {
				t.Fatal(err)
			}
			if _, err := orders.Insert(et.Json{"id": "o1", "client_id": "c1", "total": "10"}).One(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				db.RemoveModel(tg.schema, "r_orders")
				db.RemoveModel(tg.schema, "r_clients")
			})

			other, err := jsql.ConnectTo(tg.params)
			if err != nil {
				t.Fatal(err)
			}
			defer other.Close()

			c := tg.schema + ".r_clients:C"
			o := tg.schema + ".r_orders:O"
			query := func(label string, sql et.Json) et.Items {
				t.Helper()
				got, err := other.Query(sql)
				if err != nil {
					t.Fatalf("%s: %v", label, err)
				}
				return got
			}

			got := query("select", et.Json{"select": []any{"C.name", "C.city"}, "from": c})
			if got.Count != 1 || got.Str(0, "name") != "Ana" || got.Str(0, "city") != "Cali" {
				t.Errorf("select: %v", got.Result)
			}

			got = query("join", et.Json{"select": []any{"C.name:client", "O.total"}, "from": c,
				"join": []any{et.Json{"to": o, "on": []any{et.Json{"O.client_id": et.Json{"eq": et.Json{"field": "C.id"}}}}}}})
			if got.Count != 1 || got.Str(0, "client") != "Ana" || got.Str(0, "total") != "10" {
				t.Errorf("join: %v", got.Result)
			}

			from := tg.schema + ".r_orders"
			query("insert", et.Json{"insert": et.Json{"from": from, "data": et.Json{"id": "o2", "client_id": "c1", "total": "20"}}})
			query("update", et.Json{"update": et.Json{"from": from, "data": et.Json{"total": "25"}, "where": []any{et.Json{"id": et.Json{"eq": "o2"}}}}})
			query("upsert", et.Json{"upsert": et.Json{"from": from, "data": et.Json{"id": "o3", "client_id": "c1", "total": "30"}, "where": []any{et.Json{"id": et.Json{"eq": "o3"}}}}})
			query("delete", et.Json{"delete": et.Json{"from": from, "where": []any{et.Json{"id": et.Json{"eq": "o1"}}}}})

			rows, err := orders.Where(jsql.Eq("client_id", "c1")).OrderBy("id").All()
			if err != nil {
				t.Fatal(err)
			}
			ids := []string{}
			for _, row := range rows.Result {
				ids = append(ids, row.Str("id")+"="+row.Str("total"))
			}
			if strings.Join(ids, ",") != "o2=25,o3=30" {
				t.Errorf("commands: %v", ids)
			}

			if _, err := other.Query(et.Json{"from": "r_none"}); err == nil {
				t.Error("from without schema of an undefined model must fail")
			}
			if _, err := other.Query(et.Json{"from": tg.schema + ".r_none"}); err == nil {
				t.Error("from of a missing table must fail")
			}

			_, err = other.Query(et.Json{"select": []any{"C.name", "O.name"}, "from": []any{c, o}})
			if err == nil || !strings.Contains(err.Error(), "name") {
				t.Errorf("duplicated select name must fail: %v", err)
			}
		})
	}
}
