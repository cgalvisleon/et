package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	_ "github.com/cgalvisleon/et/jsql/drivers/oracle"
	_ "github.com/cgalvisleon/et/jsql/drivers/postgres"
	_ "github.com/cgalvisleon/et/jsql/drivers/sqlite"
	"github.com/joho/godotenv"
)

/**
* target: A database the insert/update test runs against.
* Params are passed to jsql.ConnectTo directly, because envar caches the environment on first read;
* schema is the model schema (in Oracle, an existing user).
**/
type target struct {
	name    string
	params  jsql.ConnectParams
	schema  string
	cleanup []string
	file    string
}

/**
* getEnv: Returns the environment variable key, or def when it is empty.
* @param key, def string
* @return string
**/
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

/**
* getEnvInt: Returns the environment variable key as int, or def when it is empty or invalid.
* @param key string, def int
* @return int
**/
func getEnvInt(key string, def int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return v
}

/**
* targets: Postgres uses the DB_* variables (the repository .env is loaded); Oracle uses ORACLE_*,
* with defaults for the local gvenzl/oracle-free container (user jsql, service FREEPDB1).
* @return []target
**/
func targets() []target {
	_ = godotenv.Load("../../../.env")
	pgHost := getEnv("DB_HOST", "localhost")
	pgName := getEnv("DB_NAME", "test")
	oraHost := getEnv("ORACLE_HOST", "localhost")
	oraService := getEnv("ORACLE_SERVICE", "FREEPDB1")
	oraUser := getEnv("ORACLE_USER", "jsql")
	sqliteFile := filepath.Join(os.TempDir(), "jsql_test.db")
	return []target{
		{
			name: "sqlite",
			params: jsql.ConnectParams{
				Driver: jsql.DriverSqlite,
				Host:   "local",
				Name:   sqliteFile,
				Connection: &jsql.SqliteConection{
					Name:        sqliteFile,
					RecordLimit: 1000,
				},
				RecordLimit: 1000,
			},
			schema: "jsql_test",
			file:   sqliteFile,
		},
		{
			name: "postgres",
			params: jsql.ConnectParams{
				Driver: jsql.DriverPostgres,
				Host:   pgHost,
				Name:   pgName,
				Connection: &jsql.PgConection{
					Host:        pgHost,
					Port:        getEnvInt("DB_PORT", 5432),
					Database:    pgName,
					User:        getEnv("DB_USER", "test"),
					Password:    getEnv("DB_PASSWORD", "test"),
					Sslmode:     getEnv("DB_SSLMODE", "disable"),
					AppName:     "jsql_test",
					RecordLimit: 1000,
				},
				RecordLimit: 1000,
			},
			schema:  "jsql_test",
			cleanup: []string{`DROP SCHEMA IF EXISTS jsql_test CASCADE`},
		},
		{
			name: "oracle",
			params: jsql.ConnectParams{
				Driver: jsql.DriverOracle,
				Host:   oraHost,
				Name:   oraService,
				Connection: &jsql.OracleConection{
					Host:        oraHost,
					Port:        getEnvInt("ORACLE_PORT", 1521),
					Username:    oraUser,
					Password:    getEnv("ORACLE_PASSWORD", "jsqlTest1"),
					ServiceName: oraService,
				},
				RecordLimit: 1000,
			},
			schema:  oraUser,
			cleanup: oracleDrops(oraUser, "transfers", "subscriptions", "clients", "plans"),
		},
	}
}

/**
* oracleDrops: Returns one DROP TABLE block per table, ignoring tables that do not exist.
* @param schema string, tables ...string
* @return []string
**/
func oracleDrops(schema string, tables ...string) []string {
	result := make([]string, len(tables))
	for i, table := range tables {
		result[i] = fmt.Sprintf(`BEGIN EXECUTE IMMEDIATE 'DROP TABLE %s."%s" CASCADE CONSTRAINTS'; EXCEPTION WHEN OTHERS THEN NULL; END;`, schema, table)
	}
	return result
}

/**
* connect: Opens the target database with a clean slate (the SQLite file is recreated and the test
* tables dropped) and registers the cleanup; the subtest is skipped when the database is not available.
* @param t *testing.T, tg target
* @return *jsql.DB
**/
func connect(t *testing.T, tg target) *jsql.DB {
	t.Helper()
	if tg.file != "" {
		removeFile(tg.file)
		t.Cleanup(func() { removeFile(tg.file) })
	}
	db, err := jsql.ConnectTo(tg.params)
	if err != nil {
		t.Skipf("%s not available: %v", tg.name, err)
	}
	drop := func() {
		for _, stmt := range tg.cleanup {
			db.Sql(stmt)
		}
	}
	drop()
	t.Cleanup(func() {
		drop()
		db.Close()
	})
	return db
}

/**
* removeFile: Deletes a SQLite database file together with its WAL and shared-memory files.
* @param path string
**/
func removeFile(path string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		os.Remove(path + suffix)
	}
}

/**
* loadFixture: Reads test.json as et.Json.
* @param t *testing.T
* @return et.Json
**/
func loadFixture(t *testing.T) et.Json {
	t.Helper()
	bt, err := os.ReadFile("test.json")
	if err != nil {
		t.Fatal(err)
	}
	var result et.Json
	if err := json.Unmarshal(bt, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

/**
* normalize: Round-trips v through JSON so values of different Go types compare by content.
* When dropNulls is true, null values are removed at every level.
* @param v any, dropNulls bool
* @return any
**/
func normalize(v any, dropNulls bool) any {
	bt, _ := json.Marshal(v)
	var result any
	_ = json.Unmarshal(bt, &result)
	if dropNulls {
		result = stripNulls(result)
	}
	return result
}

/**
* stripNulls: Removes null values from maps, recursively.
* @param v any
* @return any
**/
func stripNulls(v any) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if val == nil {
				delete(t, k)
				continue
			}
			t[k] = stripNulls(val)
		}
	case []any:
		for i, val := range t {
			t[i] = stripNulls(val)
		}
	}
	return v
}

/**
* assertFields: Checks that every key of expected has the same content in got.
* @param t *testing.T, label string, expected, got et.Json, dropNulls bool
**/
func assertFields(t *testing.T, label string, expected, got et.Json, dropNulls bool) {
	t.Helper()
	exp := normalize(expected, dropNulls).(map[string]any)
	res := normalize(got, dropNulls).(map[string]any)
	for k, want := range exp {
		if !reflect.DeepEqual(want, res[k]) {
			a, _ := json.Marshal(want)
			b, _ := json.Marshal(res[k])
			t.Errorf("%s: field %q\n  want: %.300s\n  got:  %.300s", label, k, a, b)
		}
	}
}

/**
* assertRows: Checks that got has the same rows, in order, with the same content as expected.
* @param t *testing.T, label string, expected, got []et.Json
**/
func assertRows(t *testing.T, label string, expected, got []et.Json) {
	t.Helper()
	exp := normalize(expected, false)
	res := normalize(got, false)
	if !reflect.DeepEqual(exp, res) {
		a, _ := json.Marshal(exp)
		b, _ := json.Marshal(res)
		t.Errorf("%s\n  want: %s\n  got:  %s", label, a, b)
	}
}

/**
* defineModel: Defines a model with the standard columns (id, status, dates, _source) plus the given
* KEY columns; any other value is stored in _source. With JSQL_DEBUG set, the model logs its SQL.
* @param t *testing.T, db *jsql.DB, schema, name string, columns ...string
* @return *jsql.Model
**/
func defineModel(t *testing.T, db *jsql.DB, schema, name string, columns ...string) *jsql.Model {
	t.Helper()
	model, err := db.DefineModel(schema, name, 1, "test")
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("JSQL_DEBUG") != "" {
		model.Debug()
	}
	for _, column := range columns {
		model.DefineColumn(column, et.KEY, "")
	}
	if err := model.Init(); err != nil {
		t.Fatalf("init %s: %v", name, err)
	}
	return model
}

/**
* defineTransfers: Defines the transfers model: id/status/created_at/updated_at from DefineModel,
* kind/code/client_id as columns, and everything else (nested data, extended attributes) in _source.
* With JSQL_DEBUG set, the model logs every generated SQL statement (DDL, queries and commands).
* @param t *testing.T, db *jsql.DB, schema string
* @return *jsql.Model
**/
func defineTransfers(t *testing.T, db *jsql.DB, schema string) *jsql.Model {
	t.Helper()
	return defineModel(t, db, schema, "transfers", "kind", "code", "client_id")
}

/**
* TestInsertUpdate: Inserts test.json and updates columns and nested attributes, checking that the
* RETURNING data and a later read keep every value (quotes, backslashes, regex, nested objects).
**/
func TestInsertUpdate(t *testing.T) {
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)

			// Oracle's JSON_MERGEPATCH removes keys set to null, so nulls are ignored there.
			dropNulls := tg.name == "oracle"
			model := defineTransfers(t, db, tg.schema)
			fixture := loadFixture(t)
			id := fixture.Str("id")

			inserted, err := model.Insert(fixture.Clone()).One()
			if err != nil {
				t.Fatal("insert:", err)
			}
			if !inserted.Ok {
				t.Fatal("insert returned no row")
			}
			assertFields(t, "insert", fixture, inserted.Result, false)

			read, err := model.Where(jsql.Eq("id", id)).One()
			if err != nil {
				t.Fatal("read after insert:", err)
			}
			assertFields(t, "read after insert", fixture, read.Result, false)

			// The inserted record can be queried by columns and by nested attributes of _source.
			found, err := model.Query(et.Json{
				"selects": []any{
					"id",
					"code",
					"data->transfer_address->municipality_name:municipality",
					"extendedAttributeValues->plan_comercial:plan",
					"step",
				},
				"where": []any{
					et.Json{"kind": et.Json{"eq": "transfers"}},
					et.Json{"and": et.Json{"data->transfer_address->stratum": et.Json{"eq": "4"}}},
					et.Json{"and": et.Json{"step": et.Json{"eq": 3}}},
				},
			}).All()
			if err != nil {
				t.Fatal("query:", err)
			}
			assertRows(t, "query inserted", []et.Json{{
				"id":           id,
				"code":         "00000009",
				"municipality": "PALMIRA",
				"plan":         "PLAN 200 MEGAS 2026 F",
				"step":         3,
			}}, found.Result)

			changes := et.Json{
				"status":                                "done",
				"code":                                  "00000010",
				"step":                                  4,
				"caption":                               `Traslado "urgente" de O'Brien`,
				"data->selected_service->status":        "inactivo",
				"data->transfer_address->stratum":       "5",
				"data->transfer_address->notes":         `C:\ruta\nueva`,
				"extendedAttributeValues->tipo_cliente": "INQUILINO",
			}
			updated, err := model.Update(changes).Where(jsql.Eq("id", id)).One()
			if err != nil {
				t.Fatal("update:", err)
			}

			expected := normalize(fixture, false).(map[string]any)
			expected["status"] = "done"
			expected["code"] = "00000010"
			expected["step"] = float64(4)
			expected["caption"] = `Traslado "urgente" de O'Brien`
			data := expected["data"].(map[string]any)
			data["selected_service"].(map[string]any)["status"] = "inactivo"
			address := data["transfer_address"].(map[string]any)
			address["stratum"] = "5"
			address["notes"] = `C:\ruta\nueva`
			expected["extendedAttributeValues"].(map[string]any)["tipo_cliente"] = "INQUILINO"

			assertFields(t, "update", expected, updated.Result, dropNulls)

			read, err = model.Where(jsql.Eq("id", id)).One()
			if err != nil {
				t.Fatal("read after update:", err)
			}
			assertFields(t, "read after update", expected, read.Result, dropNulls)

			if city := read.Result.Str("data", "selected_service", "address", "city"); city != "PALMIRA" {
				t.Errorf("nested sibling lost: data.selected_service.address.city = %q", city)
			}
			if lat := read.Result.Num("data", "transfer_address", "coordinates", "lat"); lat != 3.1234567 {
				t.Errorf("nested sibling lost: data.transfer_address.coordinates.lat = %v", lat)
			}
		})
	}
}

/**
* TestJoinGroupHaving: Uses three related tables (clients, plans and subscriptions) to query with
* JSON descriptors: a JOIN of the three, a GROUP BY with aggregates, and the same grouping filtered
* with HAVING. city and price are attributes stored in _source.
**/
func TestJoinGroupHaving(t *testing.T) {
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)
			clients := defineModel(t, db, tg.schema, "clients", "name")
			plans := defineModel(t, db, tg.schema, "plans", "name")
			subscriptions := defineModel(t, db, tg.schema, "subscriptions", "client_id", "plan_id")

			insert := func(model *jsql.Model, rows ...et.Json) {
				for _, row := range rows {
					if _, err := model.Insert(row).Exec(); err != nil {
						t.Fatalf("insert %s: %v", model.Name, err)
					}
				}
			}
			insert(clients,
				et.Json{"id": "c1", "name": "Ana", "city": "PALMIRA"},
				et.Json{"id": "c2", "name": "Luis", "city": "CALI"},
				et.Json{"id": "c3", "name": "Marta O'Neil", "city": "PALMIRA"},
			)
			insert(plans,
				et.Json{"id": "p1", "name": "PLAN 200", "price": 90000},
				et.Json{"id": "p2", "name": "PLAN 500", "price": 150000},
			)
			insert(subscriptions,
				et.Json{"id": "s1", "client_id": "c1", "plan_id": "p1"},
				et.Json{"id": "s2", "client_id": "c1", "plan_id": "p2"},
				et.Json{"id": "s3", "client_id": "c2", "plan_id": "p1"},
				et.Json{"id": "s4", "client_id": "c3", "plan_id": "p2"},
			)

			joins := []any{
				et.Json{"to": tg.schema + ".clients:C", "on": []any{et.Json{"A.client_id": et.Json{"eq": "C.id"}}}},
				et.Json{"to": tg.schema + ".plans:P", "on": []any{et.Json{"A.plan_id": et.Json{"eq": "P.id"}}}},
			}

			// JOIN: each subscription with its client and plan.
			joined, err := subscriptions.Query(et.Json{
				"selects": []any{"A.id", "C.name:client", "C.city:city", "P.name:plan", "P.price:price"},
				"join":    joins,
				"orders":  []any{et.Json{"A.id": true}},
			}).All()
			if err != nil {
				t.Fatal("join:", err)
			}
			assertRows(t, "join", []et.Json{
				{"id": "s1", "client": "Ana", "city": "PALMIRA", "plan": "PLAN 200", "price": 90000},
				{"id": "s2", "client": "Ana", "city": "PALMIRA", "plan": "PLAN 500", "price": 150000},
				{"id": "s3", "client": "Luis", "city": "CALI", "plan": "PLAN 200", "price": 90000},
				{"id": "s4", "client": "Marta O'Neil", "city": "PALMIRA", "plan": "PLAN 500", "price": 150000},
			}, joined.Result)

			// GROUP BY: subscriptions and amount per city.
			grouped, err := subscriptions.Query(et.Json{
				"selects": []any{"C.city:city", "count(A.id):total", "sum(P.price):amount"},
				"join":    joins,
				"groups":  []any{"C.city"},
				"orders":  []any{et.Json{"C.city": true}},
			}).All()
			if err != nil {
				t.Fatal("group by:", err)
			}
			assertRows(t, "group by", []et.Json{
				{"city": "CALI", "total": 1, "amount": 90000},
				{"city": "PALMIRA", "total": 3, "amount": 390000},
			}, grouped.Result)

			// HAVING: only the cities with more than one subscription.
			having, err := subscriptions.Query(et.Json{
				"selects": []any{"C.city:city", "count(A.id):total", "sum(P.price):amount"},
				"join":    joins,
				"groups":  []any{"C.city"},
				"havings": []any{et.Json{"count(A.id)": et.Json{"more": 1}}},
				"orders":  []any{et.Json{"C.city": true}},
			}).All()
			if err != nil {
				t.Fatal("having:", err)
			}
			assertRows(t, "having", []et.Json{
				{"city": "PALMIRA", "total": 3, "amount": 390000},
			}, having.Result)
		})
	}
}
