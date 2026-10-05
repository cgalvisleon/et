package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

// wrapperScript exercises the goja bindings (db, jsql, newTx) from a calc script; the %q are the schema
// and a scratch sqlite file.
const wrapperScript = `
const S = %q;
const FILE = %q;
const out = {};
const clients = db.getModel(S, "w_clients");
const ordersModel = clients.detail("orders");
ordersModel.delete().where(jsql.notNull("id")).exec();
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
// Relations of the model
const details = clients.getDetails();
out.detailNames = Object.keys(details).join(",");
out.detailTo = details.orders.ref().to.name;
out.detailKeys = details.orders.toJson().keys.id;
out.detailModel = details.orders.to().model().toJson().name;
ordersModel.insert({id: "o1", client_id: "c1", amount: 10}).exec();
out.detailRows = details.orders.getQuery({id: "c1"}, 1, 30).all().count;
out.rollupOp = clients.getRollups().orders_count.toJson().operation;
out.rollupTo = clients.getRollups().orders_count.to().ref().name;
const tags = clients.getMasters().tags;
out.masterBridge = tags.ref().bridge.name;
out.masterTo = tags.to().ref().name;
out.fromName = clients.getFrom().ref().name;
// Relations resolved for a query
// details are resolved by .detail(...), rollups when the query runs
const q = clients.select("id", "orders_count").detail("orders").where(jsql.eq("id", "c1"));
q.all();
out.queryFroms = q.getFroms().length;
out.queryDetails = Object.keys(q.getDetails()).join(",");
out.queryDetailRows = q.getDetails().orders.getQuery({id: "c1"}).all().count;
out.queryRollup = q.getRollups().orders_count.getQuery({id: "c1"}) !== null;
out.queryRollupMissing = q.getRollups().orders_count.getQuery({}) === null;
const otherParams = {driver: "sqlite", name: "other", connection: {file: FILE}};
jsql.createDB(otherParams, 5000);
const conn = jsql.newConnection("sqlite", {file: "first.db", timeout: 1500});
out.connFile = conn.getParams().file;
out.connTimeout = conn.getParams().timeout;
conn.setDatabase(FILE);
out.connDb = conn.getDatabase() === FILE;
out.connId = typeof conn.id;
const ora = jsql.newConnection("oracle", {service_name: "svc"});
out.oraDb = ora.getDatabase();
out.oraId = typeof ora.id;
const other = jsql.connectTo({driver: "sqlite", connection: conn});
out.other = other.sql("SELECT 7 AS n").result[0].n;
jsql.dropDB(other);
item.out = JSON.stringify(out);
`

/**
* TestWrapper: Runs a JavaScript calc script that uses the goja bindings of jsql (db, jsql, newTx):
* model lookup, insert, update, delete, query builder, count, exists, a rolled back tx, a thrown
* Go error, toJson, db.query, connection objects (jsql.newConnection) and createDB/connectTo/dropDB of a
* scratch sqlite database.
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

			// Tables of the models, children first (a failed run may leave them with an older shape)
			tables := []string{"w_probe", "w_clients_orders", "w_clients_w_tags", "w_tags", "w_clients"}
			for _, table := range tables {
				if tg.name == "postgres" {
					table = tg.schema + "." + table + " CASCADE"
				}
				db.Sql("DROP TABLE IF EXISTS " + table)
			}

			clients, err := db.DefineModel(tg.schema, "w_clients", 1, "test")
			if err != nil {
				t.Fatal(err)
			}
			clients.DefineColumn("name", et.KEY, "")
			orders, err := clients.DefineDetail("orders", map[string]string{"id": "client_id"}, 30, "amount")
			if err != nil {
				t.Fatal(err)
			}
			orders.DefinePrimaryKey("id", et.KEY, "")
			orders.DefineColumn("amount", et.FLOAT, 0)
			if _, err := clients.DefineRollup("orders_count", orders, map[string]string{"id": "client_id"}, nil, jsql.RollupCount); err != nil {
				t.Fatal(err)
			}
			tags, err := db.DefineModel(tg.schema, "w_tags", 1, "test")
			if err != nil {
				t.Fatal(err)
			}
			tags.DefineColumn("name", et.KEY, "")
			if _, err := clients.DefineMaster("tags", tags, map[string]string{"id": "client_id"}, map[string]string{"id": "tag_id"}, []string{"name"}); err != nil {
				t.Fatal(err)
			}
			probe, err := db.DefineModel(tg.schema, "w_probe", 1, "test")
			if err != nil {
				t.Fatal(err)
			}
			file := filepath.Join(t.TempDir(), "other.db")
			probe.DefineCalc("out", fmt.Sprintf(wrapperScript, tg.schema, file))
			for _, m := range []*jsql.Model{clients, tags, probe} {
				if err := m.Init(); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() {
				for _, table := range tables {
					db.RemoveModel(tg.schema, table)
				}
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
				"inserted":           "Luis",
				"one":                "Ana",
				"count":              float64(2),
				"all":                "c1,c2",
				"missing":            false,
				"updated":            "Ana M",
				"rolledBack":         false,
				"thrown":             true,
				"modelName":          "w_clients",
				"query":              "Luis",
				"other":              float64(7),
				"connFile":           "first.db",
				"connTimeout":        float64(1500),
				"connDb":             true,
				"connId":             "undefined",
				"oraDb":              "svc",
				"oraId":              "function",
				"detailNames":        "orders",
				"detailTo":           "w_clients_orders",
				"detailKeys":         "client_id",
				"detailModel":        "w_clients_orders",
				"detailRows":         float64(1),
				"rollupOp":           "count",
				"rollupTo":           "w_clients_orders",
				"masterBridge":       "w_clients_w_tags",
				"masterTo":           "w_tags",
				"fromName":           "w_clients",
				"queryFroms":         float64(1),
				"queryDetails":       "orders",
				"queryDetailRows":    float64(1),
				"queryRollup":        true,
				"queryRollupMissing": true,
			}
			for k, v := range want {
				if got[k] != v {
					t.Errorf("%s = %v (%T); want %v", k, got[k], got[k], v)
				}
			}
			if _, err := os.Stat(file); !os.IsNotExist(err) {
				t.Errorf("jsql.dropDB left %s: %v", file, err)
			}
		})
	}
}
