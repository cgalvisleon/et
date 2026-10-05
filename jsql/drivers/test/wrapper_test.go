package test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

// wrapperScript exercises the goja bindings (db, jsql, newTx) from a calc script; %q is the schema.
const wrapperScript = `
const S = %q;
const out = {};
const clients = db.getModel(S, "w_clients");
clients.delete().where(jsql.notNull("id")).exec();
clients.insert({id: "c1", name: "Ana"}).exec();
out.inserted = clients.insert({id: "c2", name: "Luis"}).one().result.name;
out.one = clients.where(jsql.eq("id", "c1")).one().result.name;
out.count = clients.where(jsql.notNull("id")).count();
out.all = clients.select("id", "name").where(jsql.notNull("id")).orderBy("id").all().result.map(r => r.id).join(",");
out.missing = clients.where(jsql.eq("id", "zz")).one().ok;
clients.update({name: "Ana M"}).where(jsql.eq("id", "c1")).exec();
out.updated = clients.where(jsql.eq("id", "c1")).one().result.name;
const tx = newTx();
clients.insert({id: "c3", name: "Rollback"}).execTx(tx);
tx.rollback();
out.rolledBack = clients.where(jsql.eq("id", "c3")).exists();
try { db.getModel(S, "nope"); out.thrown = false; } catch (e) { out.thrown = true; }
out.modelName = clients.toJson().name;
out.query = db.query({from: S + ".w_clients", where: [{"id": {"eq": "c2"}}]}).result[0].name;
item.out = JSON.stringify(out);
`

/**
* TestWrapper: Runs a JavaScript calc script that uses the goja bindings of jsql (db, jsql, newTx):
* model lookup, insert, update, delete, query builder, count, exists, a rolled back tx, a thrown
* Go error, toJson and db.query.
**/
func TestWrapper(t *testing.T) {
	for _, tg := range targets() {
		if tg.name != "sqlite" && tg.name != "postgres" {
			continue
		}
		t.Run(tg.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if tg.name != "sqlite" {
				if err := jsql.TestConnection(ctx, tg.params.Connection, ""); err != nil {
					t.Skipf("%s not available: %v", tg.name, err)
				}
			}

			db, err := jsql.NewDB(tg.params)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Init(); err != nil {
				t.Fatal(err)
			}

			clients, err := db.DefineModel(tg.schema, "w_clients", 1, "test")
			if err != nil {
				t.Fatal(err)
			}
			clients.DefineColumn("name", et.KEY, "")
			probe, err := db.DefineModel(tg.schema, "w_probe", 1, "test")
			if err != nil {
				t.Fatal(err)
			}
			probe.DefineCalc("out", fmt.Sprintf(wrapperScript, tg.schema))
			for _, m := range []*jsql.Model{clients, probe} {
				if err := m.Init(); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() {
				db.RemoveModel(tg.schema, "w_probe")
				db.RemoveModel(tg.schema, "w_clients")
			})

			if _, err := probe.Upsert(et.Json{"id": "p1"}).Where(jsql.Eq("id", "p1")).Exec(); err != nil {
				t.Fatal(err)
			}
			row, err := probe.Where(jsql.Eq("id", "p1")).Calc("out").One()
			if err != nil {
				t.Fatalf("script: %v", err)
			}

			var got map[string]any
			if err := json.Unmarshal([]byte(row.Result.Str("out")), &got); err != nil {
				t.Fatalf("out = %q: %v", row.Result.Str("out"), err)
			}
			want := map[string]any{
				"inserted":   "Luis",
				"one":        "Ana",
				"count":      float64(2),
				"all":        "c1,c2",
				"missing":    false,
				"updated":    "Ana M",
				"rolledBack": false,
				"thrown":     true,
				"modelName":  "w_clients",
				"query":      "Luis",
			}
			for k, v := range want {
				if got[k] != v {
					t.Errorf("%s = %v (%T); want %v", k, got[k], got[k], v)
				}
			}
		})
	}
}
