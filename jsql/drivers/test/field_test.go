package test

import (
	"os"
	"testing"

	"github.com/cgalvisleon/et/et"
)

/**
* TestFieldValue: A condition whose value is a field reference ({"field": "name"}, et.FIELD) compares two columns of
* the query, also as aggregates in a HAVING. A name that is not a column is an attribute of _source (cast to the
* other side's type; missing, it is NULL); a reference that does not resolve (an unknown origin) makes the condition
* false (no rows, nothing updated), never a text literal.
**/
func TestFieldValue(t *testing.T) {
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)
			budgets, err := db.DefineModel(tg.schema, "budgets", 1, "test")
			if err != nil {
				t.Fatal(err)
			}
			if os.Getenv("JSQL_DEBUG") != "" {
				budgets.Debug()
			}
			budgets.DefineColumn("area", et.KEY, "")
			budgets.DefineColumn("spent", et.INT, 0)
			budgets.DefineColumn("budget", et.INT, 0)
			if err := budgets.Init(); err != nil {
				t.Fatal("init budgets:", err)
			}

			for _, row := range []et.Json{
				{"id": "b1", "area": "IT", "spent": 100, "budget": 90, "cap": 200},
				{"id": "b2", "area": "IT", "spent": 50, "budget": 80, "cap": 40},
				{"id": "b3", "area": "HR", "spent": 80, "budget": 80},
			} {
				if _, err := budgets.Insert(row).Exec(); err != nil {
					t.Fatal("insert:", err)
				}
			}

			query := func(label string, desc et.Json, want []et.Json) {
				t.Helper()
				desc["orders"] = []any{et.Json{"A.id": true}}
				items, err := budgets.Query(desc).All()
				if err != nil {
					t.Fatalf("%s: %v", label, err)
				}
				assertRows(t, label, want, items.Result)
			}

			query("more than another column", et.Json{
				"selects": []any{"A.id"},
				"where":   []any{et.Json{"A.spent": et.Json{"more": et.Json{"field": "A.budget"}}}},
			}, []et.Json{{"id": "b1"}})

			query("equal to another column", et.Json{
				"selects": []any{"A.id"},
				"where":   []any{et.Json{"spent": et.Json{"eq": et.Json{"field": "budget"}}}},
			}, []et.Json{{"id": "b3"}})

			query("column against an attribute", et.Json{
				"selects": []any{"A.id"},
				"where":   []any{et.Json{"spent": et.Json{"more": et.Json{"field": "cap"}}}},
			}, []et.Json{{"id": "b2"}})

			query("attribute against a column", et.Json{
				"selects": []any{"A.id"},
				"where":   []any{et.Json{"cap": et.Json{"more": et.Json{"field": "spent"}}}},
			}, []et.Json{{"id": "b1"}})

			query("missing attribute", et.Json{
				"selects": []any{"A.id"},
				"where":   []any{et.Json{"spent": et.Json{"more": et.Json{"field": "nope"}}}},
			}, []et.Json{})

			query("unknown origin", et.Json{
				"selects": []any{"A.id"},
				"where":   []any{et.Json{"spent": et.Json{"more": et.Json{"field": "X.spent"}}}},
			}, []et.Json{})

			having, err := budgets.Query(et.Json{
				"selects": []any{"A.area:area", "sum(A.spent):spent"},
				"groups":  []any{"A.area"},
				"havings": []any{et.Json{"sum(A.spent)": et.Json{"less": et.Json{"field": "sum(A.budget)"}}}},
				"orders":  []any{et.Json{"A.area": true}},
			}).All()
			if err != nil {
				t.Fatal("having:", err)
			}
			assertRows(t, "having", []et.Json{{"area": "IT", "spent": 150}}, having.Result)

			updated, err := budgets.Update(et.Json{"area": "OVER"}).
				WhereByJson([]et.Json{{"spent": et.Json{"more": et.Json{"field": "X.spent"}}}}).
				Exec()
			if err != nil {
				t.Fatal("update unknown origin:", err)
			}
			if updated.Count != 0 {
				t.Errorf("update with an unknown origin changed %d rows, want 0", updated.Count)
			}

			updated, err = budgets.Update(et.Json{"area": "OVER"}).
				WhereByJson([]et.Json{{"spent": et.Json{"more": et.Json{"field": "budget"}}}}).
				Exec()
			if err != nil {
				t.Fatal("update:", err)
			}
			if updated.Count != 1 {
				t.Errorf("update changed %d rows, want 1", updated.Count)
			}
		})
	}
}
