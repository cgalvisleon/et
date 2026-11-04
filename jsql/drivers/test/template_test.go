package test

import (
	"strings"
	"testing"

	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
)

/**
* TestTemplate: a TEMPLATE column defined by JSON builds its text from columns, an attribute and a rollup when it is
* read (all, one, a JSON query, only it in selects), is not stored on insert, and cannot be filtered or ordered by
**/
func TestTemplate(t *testing.T) {
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)
			classes := defineModel(t, db, tg.schema, "tpl_classes", "name")
			types, err := db.Define(jsql.Define{
				Schema:      tg.schema,
				Name:        "tpl_types",
				Version:     1,
				SourceField: jsql.SOURCE,
				Columns: []jsql.Column{
					{Name: jsql.ID, TypeColumn: jsql.COLUMN, TypeData: et.KEY},
					{Name: "descripcion", TypeColumn: jsql.TEMPLATE, TypeData: et.TEXT, Template: `Código: {{code|upper}}, {{name}}; Clase: {{clase.name|default:"Sin clase"}}; {{power|number:1}} W`},
					{Name: "code", TypeColumn: jsql.COLUMN, TypeData: et.KEY},
					{Name: "name", TypeColumn: jsql.COLUMN, TypeData: et.TEXT},
					{Name: "class_id", TypeColumn: jsql.COLUMN, TypeData: et.KEY},
					{Name: jsql.SOURCE, TypeColumn: jsql.COLUMN, TypeData: et.JSON},
				},
				PrimaryKeys: []jsql.DefIndex{{Name: jsql.ID}},
				Rollups: []jsql.DefRollup{{
					Name:      "clase",
					To:        jsql.DefTo{Schema: tg.schema, Name: "tpl_classes"},
					Keys:      map[string]string{"class_id": "id"},
					Select:    []string{"name"},
					Operation: jsql.RollupObject,
				}},
			})
			if err != nil {
				t.Fatal("define:", err)
			}
			if err := types.Init(); err != nil {
				t.Fatal("init:", err)
			}

			if _, err := classes.Insert(et.Json{"id": "c1", "name": "Luminaria"}).Exec(); err != nil {
				t.Fatal(err)
			}
			for _, row := range []et.Json{
				{"id": "t1", "code": "lum-1", "name": "LED 60 W", "class_id": "c1", "power": 1234.5, "descripcion": "no se guarda"},
				{"id": "t2", "code": "pos-1", "name": "Poste", "class_id": "", "power": 0},
			} {
				if _, err := types.Insert(row).Exec(); err != nil {
					t.Fatal("insert:", err)
				}
			}

			want := map[string]string{
				"t1": "Código: LUM-1, LED 60 W; Clase: Luminaria; 1.234,5 W",
				"t2": "Código: POS-1, Poste; Clase: Sin clase; 0,0 W",
			}
			check := func(label string, items []et.Json, only bool) {
				t.Helper()
				if len(items) != 2 {
					t.Fatalf("%s: %d rows", label, len(items))
				}
				for _, item := range items {
					if got := item.Str("descripcion"); got != want[item.Str("id")] {
						t.Errorf("%s %s: %q, want %q", label, item.Str("id"), got, want[item.Str("id")])
					}
					if _, has := item["clase"]; has {
						t.Errorf("%s: the rollup added for the template came back: %v", label, item)
					}
					if only && (item["code"] != nil || item["power"] != nil) {
						t.Errorf("%s: fields not asked for came back: %v", label, item)
					}
				}
			}

			all, err := types.Select().OrderBy("id", true).All()
			if err != nil {
				t.Fatal("all:", err)
			}
			check("all", all.Result, false)

			one, err := types.Where(jsql.Eq("id", "t1")).One()
			if err != nil || !one.Ok {
				t.Fatal("one:", err)
			}
			if got := one.Result.Str("descripcion"); got != want["t1"] {
				t.Errorf("one: %q", got)
			}

			only, err := types.Select("id", "descripcion").OrderBy("id", true).All()
			if err != nil {
				t.Fatal("only:", err)
			}
			check("only the template", only.Result, true)

			query, err := db.Query(et.Json{"from": tg.schema + ".tpl_types", "selects": []any{"id", "descripcion"}, "orders": []any{et.Json{"id": true}}})
			if err != nil {
				t.Fatal("db.Query:", err)
			}
			check("db.Query", query.Result, true)

			again, err := types.Select("id", "descripcion").OrderBy("id", true).All()
			if err == nil {
				check("again", again.Result, true)
			}

			stored, err := db.Sql("SELECT * FROM " + types.Table + " WHERE id = 't1'")
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(textOf(stored.Result), "no se guarda") {
				t.Errorf("the template was stored: %v", stored.Result)
			}

			if _, err := types.Where(jsql.Eq("descripcion", "x")).All(); err == nil || !strings.Contains(err.Error(), "descripcion") {
				t.Errorf("where on the template: %v", err)
			}
			if _, err := types.Select().OrderBy("descripcion", true).All(); err == nil {
				t.Error("order by the template accepted")
			}
			// With a source field an unknown name is an attribute; without it, it is not a field of the model
			if _, err := types.DefineTemplate("bad", "{{descripcion}}"); err == nil {
				t.Error("a template inside a template accepted")
			}
			strict, err := db.Define(jsql.Define{
				Schema:      tg.schema,
				Name:        "tpl_strict",
				Version:     1,
				Columns:     []jsql.Column{{Name: jsql.ID, TypeColumn: jsql.COLUMN, TypeData: et.KEY}},
				PrimaryKeys: []jsql.DefIndex{{Name: jsql.ID}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := strict.DefineTemplate("bad", "{{nope}}"); err == nil {
				t.Error("unknown field accepted")
			}
			if _, err := types.DefineTemplate("bad", "{{code|nope}}"); err == nil {
				t.Error("unknown format accepted")
			}
		})
	}
}

// textOf: rows as one text, to look for a value in them
func textOf(rows []et.Json) string {
	result := []string{}
	for _, row := range rows {
		result = append(result, row.ToString())
	}
	return strings.Join(result, "\n")
}
