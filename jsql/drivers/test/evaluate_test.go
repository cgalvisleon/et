package test

import (
	"os"
	"sort"
	"testing"

	"github.com/cgalvisleon/et/et"
)

/**
* TestMemoryMatchesSql: The same where gives the same rows in SQL (every driver) and in memory (et.EvaluateObject,
* the evaluator of jwf scripts and et.From): AND binds tighter than OR, a condition without connector after the
* first one is an AND, like uses % and _ without telling case apart and between takes [min, max] or {min, max}.
**/
func TestMemoryMatchesSql(t *testing.T) {
	rows := []et.Json{
		{"id": "r1", "name": "Ana Maria", "a": 1, "b": 0, "c": 0},
		{"id": "r2", "name": "ANA", "a": 0, "b": 1, "c": 1},
		{"id": "r3", "name": "Mariana", "a": 0, "b": 1, "c": 0},
		{"id": "r4", "name": "bob", "a": 1, "b": 1, "c": 1},
	}
	cases := []struct {
		name  string
		where []et.Json
	}{
		{"a OR b AND c", []et.Json{{"a": et.Json{"eq": 1}}, {"or": et.Json{"b": et.Json{"eq": 1}}}, {"and": et.Json{"c": et.Json{"eq": 1}}}}},
		{"b AND c OR a", []et.Json{{"b": et.Json{"eq": 1}}, {"and": et.Json{"c": et.Json{"eq": 1}}}, {"or": et.Json{"a": et.Json{"eq": 1}}}}},
		{"a AND b OR c", []et.Json{{"a": et.Json{"eq": 1}}, {"and": et.Json{"b": et.Json{"eq": 1}}}, {"or": et.Json{"c": et.Json{"eq": 1}}}}},
		{"without connector", []et.Json{{"a": et.Json{"eq": 1}}, {"b": et.Json{"eq": 1}}}},
		{"like %ana%", []et.Json{{"name": et.Json{"like": "%ana%"}}}},
		{"like ANA%", []et.Json{{"name": et.Json{"like": "ANA%"}}}},
		{"like a_a", []et.Json{{"name": et.Json{"like": "a_a"}}}},
		{"like %ar%a", []et.Json{{"name": et.Json{"like": "%ar%a"}}}},
		{"between list", []et.Json{{"b": et.Json{"between": []any{1, 2}}}}},
		{"not_between object", []et.Json{{"c": et.Json{"not_between": et.Json{"min": 1, "max": 1}}}, {"or": et.Json{"a": et.Json{"eq": 1}}}}},
		{"like OR more", []et.Json{{"name": et.Json{"like": "b%"}}, {"or": et.Json{"c": et.Json{"more": 0}}}, {"and": et.Json{"a": et.Json{"less": 1}}}}},
	}

	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)
			model, err := db.DefineModel(tg.schema, "evaluates", 1, "test")
			if err != nil {
				t.Fatal(err)
			}
			if os.Getenv("JSQL_DEBUG") != "" {
				model.Debug()
			}
			model.DefineColumn("name", et.TEXT, "")
			model.DefineColumn("a", et.INT, 0)
			model.DefineColumn("b", et.INT, 0)
			model.DefineColumn("c", et.INT, 0)
			if err := model.Init(); err != nil {
				t.Fatal("init:", err)
			}
			for _, row := range rows {
				if _, err := model.Insert(row).Exec(); err != nil {
					t.Fatal("insert:", err)
				}
			}

			for _, c := range cases {
				conditions, err := et.ToConditions(c.where)
				if err != nil {
					t.Fatalf("%s: %v", c.name, err)
				}
				memory := []string{}
				for _, row := range rows {
					if et.EvaluateObject(row, conditions) {
						memory = append(memory, row.Str("id"))
					}
				}

				items, err := model.Query(et.Json{"selects": []any{"A.id"}, "where": c.where}).All()
				if err != nil {
					t.Fatalf("%s: %v", c.name, err)
				}
				sql := []string{}
				for _, item := range items.Result {
					sql = append(sql, item.Str("id"))
				}
				sort.Strings(sql)

				if len(sql) != len(memory) {
					t.Errorf("%s: sql %v, memory %v", c.name, sql, memory)
					continue
				}
				for i := range sql {
					if sql[i] != memory[i] {
						t.Errorf("%s: sql %v, memory %v", c.name, sql, memory)
						break
					}
				}
			}

			// "or" at the first level of a query: each condition goes with OR (b = 9 OR a = 1 OR c = 1)
			items, err := model.Query(et.Json{
				"selects": []any{"A.id"},
				"where":   []any{et.Json{"b": et.Json{"eq": 9}}},
				"or":      []any{et.Json{"a": et.Json{"eq": 1}}, et.Json{"c": et.Json{"eq": 1}}},
				"orders":  []any{et.Json{"A.id": true}},
			}).All()
			if err != nil {
				t.Fatal("or group:", err)
			}
			assertRows(t, "or group", []et.Json{{"id": "r1"}, {"id": "r2"}, {"id": "r4"}}, items.Result)
		})
	}
}
