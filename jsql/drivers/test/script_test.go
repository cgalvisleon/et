package test

import (
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/jwf"
)

/**
* TestStepScript: A flow step that updates a record from its params (as the engine of core-studio runs it): bind of an
* object and of a where list with ctx, the old names of params (arrayStr, json) and ctx (put, view), and only the row of
* the where updated (sqlite)
**/
func TestStepScript(t *testing.T) {
	tg := findTarget("sqlite")
	db := connect(t, tg)
	model := defineModel(t, db, tg.schema, "u_people", "name", "city")
	for _, row := range []et.Json{{"id": "p1", "name": "Ana", "city": "A"}, {"id": "p2", "name": "Luis", "city": "A"}} {
		if _, err := model.Insert(row).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	code := `
const jsql = params.jsql;
const data = bind(jsql.data, ctx);
const wheres = bind(jsql.where, ctx);
const returning = params.arrayStr("returning");
const m = db.getModel(jsql.schema, jsql.model);
const item = m.update(data).where(wheres).returning(...returning).one();
if (item.ok) {
  ctx.put(item.result);
  const view = params.json("view");
  if (Object.keys(view).length > 0) ctx.view(view);
}
item.ok;
`
	params := et.Json{
		"jsql": et.Json{
			"schema": tg.schema, "model": "u_people",
			"data":  et.Json{"city": "{{city}}"},
			"where": []any{et.Json{"id": et.Json{"eq": "{{id}}"}}},
		},
		"returning": []any{"id", "name", "city"},
		"view":      et.Json{"name": "person"},
	}
	ctx := et.Json{"id": "p1", "city": "B"}
	bindings := jsql.Wrapper(db)
	bindings["params"] = params
	result, err := jwf.RunCode(code, ctx, bindings)
	if err != nil {
		t.Fatal(err)
	}
	if result != true {
		t.Errorf("result = %v", result)
	}
	rows, err := model.Where(jsql.Eq("city", "B")).All()
	if err != nil {
		t.Fatal(err)
	}
	if rows.Count != 1 {
		t.Errorf("rows updated = %d, want 1: %v", rows.Count, rows.Result)
	}
	if ctx.Str("person") != "Ana" {
		t.Errorf("ctx = %v", ctx)
	}
}
