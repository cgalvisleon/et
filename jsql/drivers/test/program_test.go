package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* TestProgram: The steps descriptor (spec §8.4) on every engine: variables and constants, as/one, set, return,
* error with rollback, if (value, data, query, shortcut), for (range, counter, while, break, continue, max),
* subqueries as values and the script step.
* @param t *testing.T
**/
func TestProgram(t *testing.T) {
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)
			clients := defineModel(t, db, tg.schema, "p_clients", "name")
			orders := defineModel(t, db, tg.schema, "p_orders", "client_id", "total", "state")
			t.Cleanup(func() {
				db.RemoveModel(tg.schema, "p_orders")
				db.RemoveModel(tg.schema, "p_clients")
			})
			for _, row := range []et.Json{{"id": "c1", "name": "Ana"}, {"id": "c2", "name": "Luis"}} {
				if _, err := clients.Insert(row).One(); err != nil {
					t.Fatal(err)
				}
			}
			for _, row := range []et.Json{
				{"id": "o1", "client_id": "c1", "total": "0", "state": "open"},
				{"id": "o2", "client_id": "c1", "total": "10", "state": "open"},
				{"id": "o3", "client_id": "c1", "total": "20", "state": "open"},
				{"id": "o4", "client_id": "c2", "total": "30", "state": "open"},
			} {
				if _, err := orders.Insert(row).One(); err != nil {
					t.Fatal(err)
				}
			}

			c := tg.schema + ".p_clients"
			o := tg.schema + ".p_orders"
			run := func(label string, steps et.Json) et.Items {
				t.Helper()
				got, err := db.Query(et.Json{"steps": steps})
				if err != nil {
					t.Fatalf("%s: %v", label, err)
				}
				return got
			}
			fails := func(label string, steps et.Json, part string) {
				t.Helper()
				_, err := db.Query(et.Json{"steps": steps})
				if err == nil || !strings.Contains(err.Error(), part) {
					t.Errorf("%s: want an error with %q, got %v", label, part, err)
				}
			}
			states := func() string {
				t.Helper()
				rows, err := orders.Where(jsql.Eq("client_id", "c1")).OrderBy("id").All()
				if err != nil {
					t.Fatal(err)
				}
				result := []string{}
				for _, row := range rows.Result {
					result = append(result, row.Str("id")+"="+row.Str("state"))
				}
				return strings.Join(result, ",")
			}
			reset := func() {
				t.Helper()
				if _, err := db.Query(et.Json{"update": et.Json{"from": o, "data": et.Json{"state": "open"}, "limit": 0}}); err != nil {
					t.Fatal(err)
				}
			}

			// Phase 1: const, vars, as/one, set and return.
			got := run("as/one/return", et.Json{
				"const": et.Json{"ID": "c1"},
				"vars":  et.Json{"suffix": "!"},
				"do": []any{
					et.Json{"as": "client", "one": true, "select": []any{"id", "name"}, "from": c, "where": []any{et.Json{"id": et.Json{"eq": "{{ID}}"}}}},
					et.Json{"set": et.Json{"greeting": "hola {{client.name}}{{suffix}}"}},
					et.Json{"return": et.Json{"greeting": "{{greeting}}", "id": "{{client.id}}"}},
				},
			})
			if got.Str(0, "greeting") != "hola Ana!" || got.Str(0, "id") != "c1" {
				t.Errorf("return: %v", got.Result)
			}

			// Phase 1: an error rolls back what the program wrote; a constant cannot be assigned.
			fails("error", et.Json{"do": []any{
				et.Json{"update": et.Json{"from": o, "data": et.Json{"state": "lost"}, "where": []any{et.Json{"id": et.Json{"eq": "o1"}}}}},
				et.Json{"error": "stop {{reason}}"},
			}, "vars": et.Json{"reason": "here"}}, "stop here")
			if s := states(); s != "o1=open,o2=open,o3=open" {
				t.Errorf("rollback: %s", s)
			}
			fails("const", et.Json{"const": et.Json{"X": 1}, "do": []any{et.Json{"set": et.Json{"X": 2}}}}, "X")
			fails("max_steps", et.Json{"max_steps": 2, "do": []any{et.Json{"set": et.Json{"a": 1}}, et.Json{"set": et.Json{"a": 2}}, et.Json{"set": et.Json{"a": 3}}}}, "2")
			fails("break outside for", et.Json{"do": []any{et.Json{"break": true}}}, "break")

			// Phase 2: if over value, data and query, with then/else and the shortcut.
			got = run("if value", et.Json{"vars": et.Json{"n": 150}, "do": []any{
				et.Json{"if": et.Json{"value": "{{n}}", "where": []any{et.Json{"value": et.Json{"more": 100}}}},
					"then": []any{et.Json{"return": "big"}}, "else": []any{et.Json{"return": "small"}}},
			}})
			if got.Str(0, "value") != "big" {
				t.Errorf("if value: %v", got.Result)
			}
			got = run("if data", et.Json{"vars": et.Json{"client": et.Json{"name": "Luis"}}, "do": []any{
				et.Json{"if": et.Json{"data": "{{client}}", "where": []any{et.Json{"name": et.Json{"eq": "Ana"}}}},
					"then": []any{et.Json{"return": "ana"}}, "else": []any{et.Json{"return": "other"}}},
			}})
			if got.Str(0, "value") != "other" {
				t.Errorf("if data: %v", got.Result)
			}
			query := et.Json{"select": []any{"id", "total"}, "from": o, "where": []any{et.Json{"client_id": et.Json{"eq": "c1"}}}}
			for _, tc := range []struct {
				label string
				head  et.Json
				want  string
			}{
				{"any", et.Json{"query": query, "where": []any{et.Json{"total": et.Json{"eq": "20"}}}}, "yes"},
				{"all", et.Json{"query": query, "where": []any{et.Json{"total": et.Json{"eq": "20"}}}, "match": "all"}, "no"},
				{"exists", et.Json{"query": query, "exists": true}, "yes"},
				{"not exists", et.Json{"query": query, "where": []any{et.Json{"total": et.Json{"eq": "99"}}}, "exists": false}, "yes"},
			} {
				got = run("if query "+tc.label, et.Json{"do": []any{
					et.Json{"if": tc.head, "then": []any{et.Json{"return": "yes"}}, "else": []any{et.Json{"return": "no"}}},
				}})
				if got.Str(0, "value") != tc.want {
					t.Errorf("if query %s: %v", tc.label, got.Result)
				}
			}
			run("if shortcut", et.Json{"do": []any{
				et.Json{"if": et.Json{"value": 1, "where": []any{et.Json{"value": et.Json{"eq": 1}}},
					"update": et.Json{"from": o, "data": et.Json{"state": "short"}, "where": []any{et.Json{"id": et.Json{"eq": "o1"}}}}}},
			}})
			if s := states(); s != "o1=short,o2=open,o3=open" {
				t.Errorf("if shortcut: %s", s)
			}
			reset()

			// Phase 3: for range over rows with continue, over an object and a number; counter; while; break; max.
			run("for range", et.Json{"do": []any{
				et.Json{"as": "list", "select": []any{"id", "total"}, "from": o, "where": []any{et.Json{"client_id": et.Json{"eq": "c1"}}}},
				et.Json{"for": et.Json{"range": "{{list}}", "value": "p"}, "do": []any{
					et.Json{"if": et.Json{"data": "{{p}}", "where": []any{et.Json{"total": et.Json{"eq": "0"}}}}, "then": []any{et.Json{"continue": true}}},
					et.Json{"update": et.Json{"from": o, "data": et.Json{"state": "billed"}, "where": []any{et.Json{"id": et.Json{"eq": "{{p.id}}"}}}}},
				}},
			}})
			if s := states(); s != "o1=open,o2=billed,o3=billed" {
				t.Errorf("for range: %s", s)
			}
			reset()

			got = run("for object", et.Json{"vars": et.Json{"keys": "", "obj": et.Json{"b": 2, "a": 1}}, "do": []any{
				et.Json{"for": et.Json{"range": "{{obj}}", "key": "k"}, "do": []any{et.Json{"set": et.Json{"keys": "{{keys}}{{k}}"}}}},
				et.Json{"return": "{{keys}}"},
			}})
			if got.Str(0, "value") != "ab" {
				t.Errorf("for object: %v", got.Result)
			}
			got = run("for number and counter", et.Json{"vars": et.Json{"s": ""}, "do": []any{
				et.Json{"for": et.Json{"range": 3, "key": "i"}, "do": []any{et.Json{"set": et.Json{"s": "{{s}}{{i}}"}}}},
				et.Json{"for": et.Json{"var": "i", "from": 10, "to": 4, "step": -3}, "do": []any{et.Json{"set": et.Json{"s": "{{s}}-{{i}}"}}}},
				et.Json{"return": "{{s}}"},
			}})
			if got.Str(0, "value") != "012-10-7" {
				t.Errorf("for number and counter: %v", got.Result)
			}

			pending := et.Json{"query": et.Json{"select": []any{"count(id):n"}, "from": o,
				"where": []any{et.Json{"client_id": et.Json{"eq": "c1"}}, et.Json{"and": et.Json{"state": et.Json{"eq": "open"}}}}}, "one": true}
			got = run("for while", et.Json{"vars": et.Json{"turns": 0}, "do": []any{
				et.Json{"set": et.Json{"pending": pending}},
				et.Json{"for": et.Json{"while": []any{et.Json{"pending": et.Json{"more": 0}}}, "key": "i"}, "do": []any{
					et.Json{"update": et.Json{"from": o, "data": et.Json{"state": "done"}, "limit": 1,
						"where": []any{et.Json{"client_id": et.Json{"eq": "c1"}}, et.Json{"and": et.Json{"state": et.Json{"eq": "open"}}}}}},
					et.Json{"set": et.Json{"pending": pending, "turns": "{{i}}"}},
				}},
				et.Json{"return": et.Json{"turns": "{{turns}}"}},
			}})
			if s := states(); s != "o1=done,o2=done,o3=done" || got.Int(0, "turns") != 2 {
				t.Errorf("for while: %s %v", s, got.Result)
			}
			reset()

			got = run("for break", et.Json{"vars": et.Json{"last": -1}, "do": []any{
				et.Json{"for": et.Json{"range": 10, "key": "i"}, "do": []any{
					et.Json{"if": et.Json{"value": "{{i}}", "where": []any{et.Json{"value": et.Json{"eq": 2}}}}, "then": []any{et.Json{"break": true}}},
					et.Json{"set": et.Json{"last": "{{i}}"}},
				}},
				et.Json{"return": et.Json{"last": "{{last}}"}},
			}})
			if got.Int(0, "last") != 1 {
				t.Errorf("for break: %v", got.Result)
			}
			fails("for max", et.Json{"do": []any{et.Json{"for": et.Json{"range": 5, "max": 3}, "do": []any{}}}}, "3")
			fails("for while max", et.Json{"vars": et.Json{"x": 1}, "do": []any{
				et.Json{"for": et.Json{"while": []any{et.Json{"x": et.Json{"eq": 1}}}, "max": 4}, "do": []any{}},
			}}, "4")

			// Phase 4: subqueries as values (a list for in, a value with one).
			run("subquery in", et.Json{"do": []any{
				et.Json{"update": et.Json{"from": o, "data": et.Json{"state": "vip"},
					"where": []any{et.Json{"client_id": et.Json{"in": et.Json{"query": et.Json{"select": []any{"id"}, "from": c,
						"where": []any{et.Json{"name": et.Json{"eq": "Ana"}}}}}}}}}},
			}})
			if s := states(); s != "o1=vip,o2=vip,o3=vip" {
				t.Errorf("subquery in: %s", s)
			}
			reset()
			got = run("subquery one", et.Json{"do": []any{
				et.Json{"return": et.Json{"name": et.Json{"query": et.Json{"select": []any{"name"}, "from": c,
					"where": []any{et.Json{"id": et.Json{"eq": "c2"}}}}, "one": true}}},
			}})
			if got.Str(0, "name") != "Luis" {
				t.Errorf("subquery one: %v", got.Result)
			}

			// Phase 5: script step with vars, consts and the program's transaction.
			got = run("script", et.Json{"const": et.Json{"BASE": 5}, "vars": et.Json{"total": 0}, "do": []any{
				et.Json{"as": "doubled", "script": "vars.total = consts.BASE + 1; ({value: vars.total * 2})"},
				et.Json{"return": et.Json{"total": "{{total}}", "doubled": "{{doubled.value}}"}},
			}})
			if got.Int(0, "total") != 6 || got.Int(0, "doubled") != 12 {
				t.Errorf("script: %v", got.Result)
			}
		})
	}
}

/**
* TestProgramSubqueryLimit: A subquery without its own limit that gives more than DB_RECORD_LIMIT rows is an error
* (instead of being cut short); exactly the limit, or a limit of its own, passes (sqlite, RecordLimit 2).
* @param t *testing.T
**/
func TestProgramSubqueryLimit(t *testing.T) {
	file := filepath.Join(os.TempDir(), "jsql_program_limit.db")
	tg := target{name: "sqlite", file: file, schema: "lim", params: jsql.ConnectParams{
		Driver: jsql.DriverSqlite, Host: "local", Name: file, RecordLimit: 2,
		Connection: &jsql.SqliteConection{File: file, RecordLimit: 2},
	}}
	db := connect(t, tg)
	items := defineModel(t, db, tg.schema, "items", "name")
	add := func(id string) {
		if _, err := items.Insert(et.Json{"id": id, "name": id}).One(); err != nil {
			t.Fatal(err)
		}
	}
	sub := func(limit any) et.Json {
		query := et.Json{"select": []any{"id"}, "from": "lim.items"}
		if limit != nil {
			query["limit"] = limit
		}
		return et.Json{"steps": et.Json{"do": []any{et.Json{"return": et.Json{"ids": et.Json{"query": query}}}}}}
	}

	add("a")
	add("b")
	if _, err := db.Query(sub(nil)); err != nil {
		t.Errorf("exactly the limit: %v", err)
	}
	add("c")
	if _, err := db.Query(sub(nil)); err == nil || !strings.Contains(err.Error(), "subquery") {
		t.Errorf("over the limit: want an error, got %v", err)
	}
	if _, err := db.Query(sub(2)); err != nil {
		t.Errorf("own limit: %v", err)
	}
}
