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
	cleanup string
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
			cleanup: `DROP SCHEMA IF EXISTS jsql_test CASCADE`,
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
			cleanup: fmt.Sprintf(`BEGIN EXECUTE IMMEDIATE 'DROP TABLE %s."transfers"'; EXCEPTION WHEN OTHERS THEN NULL; END;`, oraUser),
		},
	}
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
* defineTransfers: Defines the transfers model: id/status/created_at/updated_at from DefineModel,
* kind/code/client_id as columns, and everything else (nested data, extended attributes) in _source.
* @param t *testing.T, db *jsql.DB, schema string
* @return *jsql.Model
**/
func defineTransfers(t *testing.T, db *jsql.DB, schema string) *jsql.Model {
	t.Helper()
	model, err := db.DefineModel(schema, "transfers", 1, "test")
	if err != nil {
		t.Fatal(err)
	}
	model.DefineColumn("kind", et.KEY, "")
	model.DefineColumn("code", et.KEY, "")
	model.DefineColumn("client_id", et.KEY, "")
	if err := model.Init(); err != nil {
		t.Fatal("init:", err)
	}
	return model
}

/**
* TestInsertUpdate: Inserts test.json and updates columns and nested attributes, checking that the
* RETURNING data and a later read keep every value (quotes, backslashes, regex, nested objects).
**/
func TestInsertUpdate(t *testing.T) {
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			if tg.file != "" {
				removeFile(tg.file)
				defer removeFile(tg.file)
			}
			db, err := jsql.ConnectTo(tg.params)
			if err != nil {
				t.Skipf("%s not available: %v", tg.name, err)
			}
			defer db.Close()
			if tg.cleanup != "" {
				db.Sql(tg.cleanup)
				defer db.Sql(tg.cleanup)
			}

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
