package test

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
	"github.com/cgalvisleon/et/jsql"
	"github.com/cgalvisleon/et/jsql/drivers/mysql"
	"github.com/cgalvisleon/et/jsql/drivers/postgres"
)

// searchRows: the products every engine searches in (tags is an attribute of _source)
var searchRows = []et.Json{
	{"id": "p1", "name": "Bomba de agua", "description": "Bomba sumergible para pozo", "tags": "riego"},
	{"id": "p2", "name": "Bomba de agua superficial", "description": "Agua para tanques de agua", "tags": "tanque"},
	{"id": "p3", "name": "Motor eléctrico", "description": "Motor trifásico", "tags": "industria"},
	{"id": "p4", "name": "Manguera", "description": "Manguera de riego", "tags": "jardin"},
}

/**
* searchDefine: The products model as the JSON of a Define, with three searches (fts with an attribute, fts_name only
* columns, fts_es in spanish) and a trgm index on name.
* @param t *testing.T, schema string
* @return jsql.Define
**/
func searchDefine(t *testing.T, schema string) jsql.Define {
	t.Helper()
	text := fmt.Sprintf(`{
		"schema": %q, "name": "products", "version": 1, "source_field": "_source",
		"columns": [
			{"name": "id", "type_column": "column", "type_data": "key", "default": ""},
			{"name": "name", "type_column": "column", "type_data": "text", "default": ""},
			{"name": "description", "type_column": "column", "type_data": "memo", "default": ""}
		],
		"primary_keys": [{"name": "id"}],
		"indexes": [{"name": "name", "method": "trgm"}],
		"searches": [
			{"name": "fts", "fields": ["name", "description", "tags"]},
			{"name": "fts_name", "fields": ["name", "description"]},
			{"name": "fts_es", "fields": ["name"], "lang": "spanish"}
		]
	}`, schema)
	var result jsql.Define
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

/**
* searchIds: The ids of the rows of a search, sorted unless keep (the order of the relevance).
* @param t *testing.T, model *jsql.Model, desc et.Json, keep bool
* @return []string, error
**/
func searchIds(model *jsql.Model, desc et.Json, keep bool) ([]string, error) {
	desc["selects"] = []any{"A.id"}
	items, err := model.Query(desc).All()
	if err != nil {
		return nil, err
	}
	result := []string{}
	for _, item := range items.Result {
		result = append(result, item.Str("id"))
	}
	if !keep {
		sort.Strings(result)
	}
	return result, nil
}

/**
* TestSearch: The operator search on every engine (B16): a search of the model over columns and an attribute, the web
* syntax (words, -excluded, "phrase", or), a field alone, relevance where the engine has it (postgres ts_rank, mysql
* MATCH), the searches only in where and orders, and the index created on a table that already exists (postgres GIN
* used by the plan, mysql FULLTEXT).
**/
func TestSearch(t *testing.T) {
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)
			model, err := db.Define(searchDefine(t, tg.schema))
			if err != nil {
				t.Fatal(err)
			}
			if os.Getenv("JSQL_DEBUG") != "" {
				model.Debug()
			}
			if err := model.Init(); err != nil {
				t.Fatal("init:", err)
			}
			for _, row := range searchRows {
				if _, err := model.Insert(row).Exec(); err != nil {
					t.Fatal("insert:", err)
				}
			}
			ranked := tg.name == "postgres" || tg.name == "mysql"

			cases := []struct {
				name  string
				where []any
				want  []string
			}{
				{"word", []any{et.Json{"fts": et.Json{"search": "bomba"}}}, []string{"p1", "p2"}},
				{"excluded", []any{et.Json{"fts": et.Json{"search": "bomba -sumergible"}}}, []string{"p2"}},
				{"attribute", []any{et.Json{"fts": et.Json{"search": "riego"}}}, []string{"p1", "p4"}},
				{"or", []any{et.Json{"fts": et.Json{"search": "motor or manguera"}}}, []string{"p3", "p4"}},
				{"phrase", []any{et.Json{"fts": et.Json{"search": `"de agua"`}}}, []string{"p1", "p2"}},
				{"no match", []any{et.Json{"fts": et.Json{"search": "inexistente"}}}, []string{}},
				{"empty", []any{et.Json{"fts": et.Json{"search": ""}}}, []string{}},
				{"quote", []any{et.Json{"fts": et.Json{"search": "bomba'; drop table x; --"}}}, []string{}},
				{"columns", []any{et.Json{"fts_name": et.Json{"search": "bomba -sumergible"}}}, []string{"p2"}},
				{"field alone", []any{et.Json{"name": et.Json{"search": "motor"}}}, []string{"p3"}},
				{"field with alias", []any{et.Json{"A.name": et.Json{"search": "manguera"}}}, []string{"p4"}},
				{"and status", []any{et.Json{"fts": et.Json{"search": "agua"}}, et.Json{"and": et.Json{"A.id": et.Json{"neg": "p1"}}}}, []string{"p2"}},
			}
			for _, c := range cases {
				got, err := searchIds(model, et.Json{"where": c.where}, false)
				if err != nil {
					t.Errorf("%s: %v", c.name, err)
					continue
				}
				if strings.Join(got, ",") != strings.Join(c.want, ",") {
					t.Errorf("%s: got %v, want %v", c.name, got, c.want)
				}
			}

			// spanish: only postgres stems (bombas → bomba); LIKE and MATCH look for the word as it is
			spanish, err := searchIds(model, et.Json{"where": []any{et.Json{"fts_es": et.Json{"search": "bombas"}}}}, false)
			want := []string{}
			if tg.name == "postgres" {
				want = []string{"p1", "p2"}
			}
			if err != nil || strings.Join(spanish, ",") != strings.Join(want, ",") {
				t.Errorf("spanish: got %v %v, want %v", spanish, err, want)
			}

			// relevance: p2 has agua three times
			order, err := searchIds(model, et.Json{
				"where":  []any{et.Json{"fts_name": et.Json{"search": "agua"}}},
				"orders": []any{et.Json{"fts_name": false}},
			}, ranked)
			want = []string{"p1", "p2"}
			if ranked {
				want = []string{"p2", "p1"}
			}
			if err != nil || strings.Join(order, ",") != strings.Join(want, ",") {
				t.Errorf("relevance: got %v %v, want %v", order, err, want)
			}

			// an order by a search without its condition is ignored
			if _, err := searchIds(model, et.Json{"where": []any{et.Json{"A.id": et.Json{"eq": "p1"}}}, "orders": []any{et.Json{"fts": false}}}, true); err != nil {
				t.Errorf("order without search: %v", err)
			}
			count, err := model.Query(et.Json{"where": []any{et.Json{"fts": et.Json{"search": "bomba"}}}}).Count()
			if err != nil || count != 2 {
				t.Errorf("count: %d %v, want 2", count, err)
			}
			if _, err := model.Query(et.Json{"selects": []any{"fts"}}).All(); err == nil {
				t.Error("select of a search: want an error")
			}
			if _, err := model.Query(et.Json{"where": []any{et.Json{"fts": et.Json{"eq": "bomba"}}}}).All(); err == nil {
				t.Error("eq over a search: want an error")
			}
			if _, err := model.DefineSearch("bad", "", "created_x;drop"); err == nil {
				t.Error("search over an invalid field: want an error")
			}
			if searches, ok := model.ToJson()["searches"].(map[string]*jsql.TextSearch); !ok || len(searches) != 3 || searches["fts"].Lang != "spanish" {
				t.Errorf("searches in the json of the model: %v", model.ToJson()["searches"])
			}

			searchExisting(t, tg, db)
			if tg.name == "postgres" {
				searchTrgm(t, db, model)
			}
		})
	}
}

/**
* searchExisting: A search defined on a model whose table already exists (as after a restart with a new definition):
* Init creates its index, and the engine uses it.
* @param t *testing.T, tg target, db *jsql.DB
**/
func searchExisting(t *testing.T, tg target, db *jsql.DB) {
	t.Helper()
	legacy, err := db.DefineModel(tg.schema, "legacy", 1, "test")
	if err != nil {
		t.Fatal(err)
	}
	legacy.DefineColumn("name", et.TEXT, "")
	legacy.DefineColumn("description", et.MEMO, "")
	if err := legacy.Init(); err != nil {
		t.Fatal("init legacy:", err)
	}
	for _, row := range searchRows {
		if _, err := legacy.Insert(et.Json{"id": row["id"], "name": row["name"], "description": row["description"]}).Exec(); err != nil {
			t.Fatal("insert legacy:", err)
		}
	}

	indexes := func() int {
		var query string
		switch tg.name {
		case "postgres":
			query = fmt.Sprintf("SELECT COUNT(*) AS n FROM pg_indexes WHERE schemaname = '%s' AND tablename = 'legacy' AND indexname LIKE '%%_fts_%%'", tg.schema)
		case "mysql":
			query = fmt.Sprintf("SELECT COUNT(DISTINCT index_name) AS n FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = '%s_legacy' AND index_type = 'FULLTEXT'", tg.schema)
		default:
			return -1
		}
		items, err := db.Sql(query)
		if err != nil {
			t.Fatal("indexes:", err)
		}
		return items.Int(0, "n")
	}
	if n := indexes(); n > 0 {
		t.Fatalf("legacy has %d search indexes before its search", n)
	}

	if _, err := legacy.DefineSearch("fts", "", "name", "description"); err != nil {
		t.Fatal(err)
	}
	legacy.SetDb(db)
	if err := legacy.Init(); err != nil {
		t.Fatal("init legacy with search:", err)
	}
	if n := indexes(); n == 0 {
		t.Error("the search index was not created on the existing table")
	} else if n > 0 {
		t.Logf("%s: search index created on the existing table", tg.name)
	}
	got, err := searchIds(legacy, et.Json{"where": []any{et.Json{"fts": et.Json{"search": "bomba -sumergible"}}}}, false)
	if err != nil || strings.Join(got, ",") != "p2" {
		t.Errorf("legacy search: %v %v", got, err)
	}

	query := legacy.Query(et.Json{"selects": []any{"A.id"}, "where": []any{et.Json{"fts": et.Json{"search": "bomba"}}}})
	switch tg.name {
	case "postgres":
		sql, err := (&postgres.Postgres{}).Query(query)
		if err != nil {
			t.Fatal(err)
		}
		plan := explain(t, db, "SET LOCAL enable_seqscan = off", "EXPLAIN "+sql)
		if !strings.Contains(plan, "_fts_") {
			t.Errorf("the plan does not use the GIN index:\n%s", plan)
		} else {
			t.Logf("postgres plan:\n%s", plan)
		}
	case "mysql":
		sql, err := (&mysql.Mysql{}).Query(query)
		if err != nil {
			t.Fatal(err)
		}
		plan := explain(t, db, "", "EXPLAIN "+sql)
		if !strings.Contains(plan, "fulltext") {
			t.Errorf("the plan does not use the FULLTEXT index:\n%s", plan)
		} else {
			t.Logf("mysql plan: %s", plan)
		}
	}
}

/**
* searchTrgm: An index with method trgm is a pg_trgm GIN index and like '%x%' uses it.
* @param t *testing.T, db *jsql.DB, model *jsql.Model
**/
func searchTrgm(t *testing.T, db *jsql.DB, model *jsql.Model) {
	t.Helper()
	items, err := db.Sql(fmt.Sprintf("SELECT indexdef FROM pg_indexes WHERE schemaname = '%s' AND tablename = 'products' AND indexname LIKE '%%_trgm'", model.Schema))
	if err != nil || items.Count != 1 || !strings.Contains(items.Str(0, "indexdef"), "gin_trgm_ops") {
		t.Fatalf("trgm index: %v %v", items.Result, err)
	}
	got, err := searchIds(model, et.Json{"where": []any{et.Json{"name": et.Json{"like": "%omba%"}}}}, false)
	if err != nil || strings.Join(got, ",") != "p1,p2" {
		t.Errorf("like with trgm: %v %v", got, err)
	}
	sql, err := (&postgres.Postgres{}).Query(model.Query(et.Json{"selects": []any{"A.id"}, "where": []any{et.Json{"name": et.Json{"like": "%omba%"}}}}))
	if err != nil {
		t.Fatal(err)
	}
	plan := explain(t, db, "SET LOCAL enable_seqscan = off", "EXPLAIN "+sql)
	if !strings.Contains(plan, "_trgm") {
		t.Errorf("the plan does not use the trgm index:\n%s", plan)
	} else {
		t.Logf("trgm plan:\n%s", plan)
	}
}

/**
* explain: The plan of a query as text, after a setting of the same transaction (rolled back).
* @param t *testing.T, db *jsql.DB, setting, sql string
* @return string
**/
func explain(t *testing.T, db *jsql.DB, setting, sql string) string {
	t.Helper()
	tx := jsql.NewTx()
	defer tx.Rollback()
	if setting != "" {
		if _, err := db.SqlTx(tx, setting); err != nil {
			t.Fatal(setting, err)
		}
	}
	items, err := db.SqlTx(tx, sql)
	if err != nil {
		t.Fatal("explain:", err)
	}
	lines := []string{}
	for _, item := range items.Result {
		lines = append(lines, item.ToString())
	}
	return strings.Join(lines, "\n")
}

// fullTextRows: the articles of TestFullText, in spanish and english
var fullTextRows = []et.Json{
	{"id": "a1", "code": "A-100", "name": "Bombas de agua", "description": "Bomba sumergible", "tags": "riego"},
	{"id": "a2", "code": "B-200", "name": "Water pumps", "description": "Pumping water for tanks", "tags": "tank"},
	{"id": "a3", "code": "C-300", "name": "Motores eléctricos", "description": "Trifásicos", "tags": "industria"},
	{"id": "a4", "code": "D-400", "name": "Tanque de agua", "description": "Agua potable, agua lluvia", "tags": "agua"},
}

/**
* fullTextDefine: The articles model as the JSON of a Define, with full_text over four text columns in es and indexed
* also in langs.
* @param t *testing.T, schema string, fields []string, langs []string
* @return jsql.Define
**/
func fullTextDefine(t *testing.T, schema string, fields, langs []string) jsql.Define {
	t.Helper()
	text := fmt.Sprintf(`{
		"schema": %q, "name": "articles", "version": 1, "source_field": "_source",
		"columns": [
			{"name": "id", "type_column": "column", "type_data": "key", "default": ""},
			{"name": "code", "type_column": "column", "type_data": "text", "default": ""},
			{"name": "name", "type_column": "column", "type_data": "text", "default": ""},
			{"name": "description", "type_column": "column", "type_data": "memo", "default": ""},
			{"name": "tags", "type_column": "column", "type_data": "text", "default": ""}
		],
		"primary_keys": [{"name": "id"}],
		"full_text": %s, "full_text_lang": "es", "full_text_langs": %s
	}`, schema, toJsonText(fields), toJsonText(langs))
	var result jsql.Define
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatal(err)
	}
	return result
}

/**
* toJsonText: A value as JSON text.
* @param v any
* @return string
**/
func toJsonText(v any) string {
	bt, _ := json.Marshal(v)
	return string(bt)
}

/**
* TestFullText: The attribute full_text of a model on every engine: its definition and its json, the key "search" of a
* JSQL query as a text and as {text, lang}, Query.Search and DB.Query, AND over a where with or, relevance without orders,
* the lang of the engine (postgres), MSG_FULLTEXT_UNDEFINED; and in postgres and mysql its indexes (fullTextIndexes).
**/
func TestFullText(t *testing.T) {
	for _, tg := range targets() {
		t.Run(tg.name, func(t *testing.T) {
			db := connect(t, tg)
			fields := []string{"code", "name", "description", "tags"}
			model, err := db.Define(fullTextDefine(t, tg.schema, fields, []string{"en"}))
			if err != nil {
				t.Fatal(err)
			}
			if os.Getenv("JSQL_DEBUG") != "" {
				model.Debug()
			}
			if err := model.Init(); err != nil {
				t.Fatal("init:", err)
			}
			for _, row := range fullTextRows {
				if _, err := model.Insert(row).Exec(); err != nil {
					t.Fatal("insert:", err)
				}
			}

			// The definition and its json
			data := model.ToJson()
			if data["full_text_lang"] != "es" || toJsonText(data["full_text"]) != toJsonText(fields) || toJsonText(data["full_text_langs"]) != `["en"]` {
				t.Errorf("full_text in ToJson: %v %v %v", data["full_text"], data["full_text_lang"], data["full_text_langs"])
			}
			var back jsql.Model
			if err := json.Unmarshal([]byte(toJsonText(data)), &back); err != nil || back.Searches[jsql.FULL_TEXT] == nil ||
				back.Searches[jsql.FULL_TEXT].Lang != "es" || toJsonText(back.Searches[jsql.FULL_TEXT].Langs) != `["en"]` {
				t.Errorf("full_text round trip: %v %v", back.Searches, err)
			}

			ranked := tg.name == "postgres" || tg.name == "mysql"
			cases := []struct {
				name string
				desc et.Json
				want []string
				keep bool
			}{
				{"text", et.Json{"search": "bomba"}, []string{"a1"}, false},
				{"with lang", et.Json{"search": et.Json{"text": "pumps", "lang": "en"}}, []string{"a2"}, false},
				{"empty", et.Json{"search": ""}, []string{}, false},
				{"quote", et.Json{"search": "agua'; drop table x; --"}, []string{}, false},
				{"and over or", et.Json{"search": "agua", "where": []any{et.Json{"A.id": et.Json{"eq": "a1"}}, et.Json{"or": et.Json{"A.id": et.Json{"eq": "a3"}}}}}, []string{"a1"}, false},
				{"with orders", et.Json{"search": "agua", "orders": []any{et.Json{"A.id": true}}}, []string{"a1", "a4"}, true},
				{"operator with lang", et.Json{"where": []any{et.Json{"full_text": et.Json{"search": et.Json{"text": "pumps", "lang": "en"}}}}}, []string{"a2"}, false},
			}
			relevance := []string{"a1", "a4"}
			if ranked {
				relevance = []string{"a4", "a1"} // a4 has agua four times
			}
			cases = append(cases, struct {
				name string
				desc et.Json
				want []string
				keep bool
			}{"relevance", et.Json{"search": "agua"}, relevance, ranked})
			for _, c := range cases {
				got, err := searchIds(model, c.desc, c.keep)
				if err != nil {
					t.Errorf("%s: %v", c.name, err)
					continue
				}
				if strings.Join(got, ",") != strings.Join(c.want, ",") {
					t.Errorf("%s: got %v, want %v", c.name, got, c.want)
				}
			}

			// Query.Search, Count and DB.Query (a JSQL query of the DB)
			items, err := model.Select("A.id").Search("pumps", "en").All()
			if err != nil || items.Count != 1 || items.Str(0, "id") != "a2" {
				t.Errorf("Query.Search: %v %v", items.Result, err)
			}
			if count, err := model.Query(et.Json{"search": "agua"}).Count(); err != nil || count != 2 {
				t.Errorf("count: %d %v, want 2", count, err)
			}
			items, err = db.Query(et.Json{"from": tg.schema + ".articles", "selects": []any{"A.id"}, "search": "bomba"})
			if err != nil || items.Count != 1 || items.Str(0, "id") != "a1" {
				t.Errorf("DB.Query with search: %v %v", items.Result, err)
			}

			// The lang of the engine: only postgres stems, so only there the lang changes what is found
			if tg.name == "postgres" {
				for _, c := range []struct {
					text, lang string
					want       []string
				}{
					{"motor", "es", []string{"a3"}},      // spanish: motores → motor
					{"motor", "xx", []string{}},          // an unknown lang is simple: motor is not motores
					{"pump", "en", []string{"a2"}},       // english: pumps, pumping → pump
					{"motor", "spanish", []string{"a3"}}, // the name of the configuration still works
				} {
					got, err := searchIds(model, et.Json{"search": et.Json{"text": c.text, "lang": c.lang}}, false)
					if err != nil || strings.Join(got, ",") != strings.Join(c.want, ",") {
						t.Errorf("lang %s %q: got %v %v, want %v", c.lang, c.text, got, err, c.want)
					}
				}
				sql, err := (&postgres.Postgres{}).Query(model.Query(et.Json{"where": []any{et.Json{"full_text": et.Json{"search": et.Json{"text": "x", "lang": "es';drop"}}}}}))
				if err != nil || !strings.Contains(sql, "'simple'::regconfig") || strings.Contains(sql, "drop") {
					t.Errorf("an invalid lang is simple and never reaches the SQL: %v\n%s", err, sql)
				}
			}

			// Without full_text
			notext, err := db.DefineModel(tg.schema, "notext", 1, "test")
			if err != nil {
				t.Fatal(err)
			}
			notext.DefineColumn("name", et.TEXT, "")
			if err := notext.Init(); err != nil {
				t.Fatal("init notext:", err)
			}
			_, err = notext.Query(et.Json{"search": "agua"}).All()
			if err == nil || err.Error() != fmt.Sprintf(jsql.MSG_FULLTEXT_UNDEFINED, "notext") {
				t.Errorf("search without full_text: %v", err)
			}

			switch tg.name {
			case "postgres":
				fullTextPostgres(t, tg, db, model)
			case "mysql":
				fullTextMysql(t, tg, db)
			}
		})
	}
}

/**
* pgFtsIndex: A search index of a table in the catalog of postgres.
**/
type pgFtsIndex struct {
	name, def string
	valid     bool
	oid       int
}

/**
* pgFtsIndexes: The indexes with _fts_ in the name of a table, by name.
* @param t *testing.T, db *jsql.DB, schema, table string
* @return map[string]pgFtsIndex
**/
func pgFtsIndexes(t *testing.T, db *jsql.DB, schema, table string) map[string]pgFtsIndex {
	t.Helper()
	items, err := db.Sql(`SELECT c.relname AS name, i.indisvalid AS valid, pg_get_indexdef(i.indexrelid) AS def, c.oid::int AS oid
FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid JOIN pg_class r ON r.oid = i.indrelid JOIN pg_namespace n ON n.oid = r.relnamespace
WHERE n.nspname = $1 AND r.relname = $2 AND c.relname LIKE '%\_fts\_%'`, schema, table)
	if err != nil {
		t.Fatal("indexes:", err)
	}
	result := map[string]pgFtsIndex{}
	for _, item := range items.Result {
		result[item.Str("name")] = pgFtsIndex{name: item.Str("name"), def: item.Str("def"), valid: item.Bool("valid"), oid: item.Int("oid")}
	}
	return result
}

/**
* pgIndexWith: The name of the index whose definition has text (e.g. 'english'), or "".
* @param indexes map[string]pgFtsIndex, text string
* @return string
**/
func pgIndexWith(indexes map[string]pgFtsIndex, text string) string {
	for name, idx := range indexes {
		if strings.Contains(idx.def, text) {
			return name
		}
	}
	return ""
}

/**
* redefine: Defines the articles model again (as a service that restarts with another definition) and starts it.
* @param t *testing.T, tg target, db *jsql.DB, fields, langs []string
* @return *jsql.Model
**/
func redefine(t *testing.T, tg target, db *jsql.DB, fields, langs []string) *jsql.Model {
	t.Helper()
	model, err := db.Define(fullTextDefine(t, tg.schema, fields, langs))
	if err != nil {
		t.Fatal(err)
	}
	if err := model.Init(); err != nil {
		t.Fatal("init:", err)
	}
	return model
}

/**
* fullTextPostgres: The indexes of full_text in postgres: one GIN per lang used by the plan of its lang (a lang without
* index scans the table), the orphans dropped when the fields change and when a lang is removed while an index of
* another origin with a similar name stays, an invalid index created again, DB_SEARCH_INDEXES=off, and the build done
* CONCURRENTLY (pg_stat_progress_create_index) while the table takes writes.
* @param t *testing.T, tg target, db *jsql.DB, model *jsql.Model
**/
func fullTextPostgres(t *testing.T, tg target, db *jsql.DB, model *jsql.Model) {
	t.Helper()
	indexes := pgFtsIndexes(t, db, tg.schema, "articles")
	spanish, english := pgIndexWith(indexes, "'spanish'"), pgIndexWith(indexes, "'english'")
	if len(indexes) != 2 || spanish == "" || english == "" || !indexes[spanish].valid || !indexes[english].valid {
		t.Fatalf("one GIN per lang: %v", indexes)
	}

	plan := func(lang, text string) string {
		var value any = text
		if lang != "" {
			value = et.Json{"text": text, "lang": lang}
		}
		sql, err := (&postgres.Postgres{}).Query(model.Query(et.Json{"selects": []any{"A.id"}, "where": []any{et.Json{"full_text": et.Json{"search": value}}}}))
		if err != nil {
			t.Fatal(err)
		}
		return explain(t, db, "SET LOCAL enable_seqscan = off", "EXPLAIN "+sql)
	}
	if p := plan("en", "pumps"); !strings.Contains(p, english) {
		t.Errorf("the plan in english does not use %s:\n%s", english, p)
	}
	if p := plan("", "bomba"); !strings.Contains(p, spanish) {
		t.Errorf("the plan in spanish does not use %s:\n%s", spanish, p)
	}
	if p := plan("fr", "pompe"); strings.Contains(p, "_fts_") {
		t.Errorf("a lang without index uses one:\n%s", p)
	} else {
		t.Logf("french, without index:\n%s", p)
	}

	// Indexes of another origin with a similar name: not a hash, or not a GIN over to_tsvector
	base := tg.schema + "_articles"
	for _, stmt := range []string{
		fmt.Sprintf("CREATE INDEX %s_full_text_fts_manual ON %s.articles USING GIN (to_tsvector('simple', name))", base, tg.schema),
		fmt.Sprintf("CREATE INDEX %s_other_fts_deadbeef ON %s.articles (code)", base, tg.schema),
	} {
		if _, err := db.Sql(stmt); err != nil {
			t.Fatal(stmt, err)
		}
	}
	foreign := []string{base + "_full_text_fts_manual", base + "_other_fts_deadbeef"}

	// Other fields: two new indexes, the two before dropped, the foreign ones kept
	redefine(t, tg, db, []string{"code", "name", "description"}, []string{"en"})
	after := pgFtsIndexes(t, db, tg.schema, "articles")
	if _, ok := after[spanish]; ok || len(after) != 4 || pgIndexWith(after, "'english'") == english {
		t.Errorf("orphans after changing the fields: %v", after)
	}
	for _, name := range foreign {
		if _, ok := after[name]; !ok {
			t.Errorf("the index of another origin %s was dropped", name)
		}
	}

	// Without the lang en: its index is dropped
	redefine(t, tg, db, []string{"code", "name", "description"}, nil)
	after = pgFtsIndexes(t, db, tg.schema, "articles")
	if len(after) != 3 || pgIndexWith(after, "'english'") != "" || pgIndexWith(after, "'spanish'") == "" {
		t.Errorf("orphan after removing a lang: %v", after)
	}

	// An index left invalid (as by an interrupted concurrent build) is dropped and created again
	current := pgIndexWith(after, "'spanish'")
	if _, err := db.Sql(fmt.Sprintf("UPDATE pg_index SET indisvalid = false WHERE indexrelid = '%s.%s'::regclass", tg.schema, current)); err != nil {
		t.Fatal("invalidate:", err)
	}
	model = redefine(t, tg, db, []string{"code", "name", "description"}, nil)
	again := pgFtsIndexes(t, db, tg.schema, "articles")[current]
	if !again.valid || again.oid == after[current].oid {
		t.Errorf("the invalid index was not created again: %+v (before oid %d)", again, after[current].oid)
	}

	// DB_SEARCH_INDEXES=off: Init does not create the index of en; ApplySearchIndexes does, CONCURRENTLY
	envar.SetStr("DB_SEARCH_INDEXES", "off")
	model = redefine(t, tg, db, []string{"code", "name", "description"}, []string{"en"})
	envar.SetStr("DB_SEARCH_INDEXES", "on")
	if got := pgFtsIndexes(t, db, tg.schema, "articles"); pgIndexWith(got, "'english'") != "" {
		t.Errorf("with DB_SEARCH_INDEXES=off Init created an index: %v", got)
	}

	// A transaction that writes holds the build in its wait: it is CONCURRENTLY, and another session still writes
	tx := jsql.NewTx()
	if _, err := db.SqlTx(tx, fmt.Sprintf("LOCK TABLE %s.articles IN ROW EXCLUSIVE MODE", tg.schema)); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- db.ApplySearchIndexes(model) }()
	command := ""
	for range 100 {
		items, err := db.Sql("SELECT command FROM pg_stat_progress_create_index WHERE relid = $1::regclass", tg.schema+".articles")
		if err == nil && items.Count > 0 {
			command = items.Str(0, "command")
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if command != "CREATE INDEX CONCURRENTLY" {
		t.Errorf("the index build is %q, want CREATE INDEX CONCURRENTLY", command)
	} else {
		t.Logf("build in progress: %s", command)
	}
	written := make(chan error, 1)
	go func() {
		_, err := model.Insert(et.Json{"id": "a5", "code": "E-500", "name": "Pump", "description": "", "tags": ""}).Exec()
		written <- err
	}()
	select {
	case err := <-written:
		if err != nil {
			t.Errorf("insert during the build: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("the insert waited for the index build")
	}
	tx.Commit()
	if err := <-done; err != nil {
		t.Fatal("ApplySearchIndexes:", err)
	}
	got := pgFtsIndexes(t, db, tg.schema, "articles")
	if name := pgIndexWith(got, "'english'"); name == "" || !got[name].valid {
		t.Errorf("ApplySearchIndexes did not create the index of en: %v", got)
	}
}

/**
* fullTextMysql: The FULLTEXT of full_text in mysql: one for all the langs, the orphan dropped when the fields change
* (an index of another origin stays), DB_SEARCH_INDEXES=off and ApplySearchIndexes.
* @param t *testing.T, tg target, db *jsql.DB
**/
func fullTextMysql(t *testing.T, tg target, db *jsql.DB) {
	t.Helper()
	table := tg.schema + "_articles"
	indexes := func() []string {
		items, err := db.Sql(fmt.Sprintf("SELECT DISTINCT index_name AS name FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = '%s' AND index_type = 'FULLTEXT' ORDER BY index_name", table))
		if err != nil {
			t.Fatal("indexes:", err)
		}
		result := []string{}
		for _, item := range items.Result {
			result = append(result, item.Str("name"))
		}
		return result
	}
	first := indexes()
	if len(first) != 1 {
		t.Fatalf("one FULLTEXT for every lang: %v", first)
	}
	if _, err := db.Sql(fmt.Sprintf("ALTER TABLE %s ADD FULLTEXT INDEX articles_full_text_fts_manual (name)", table)); err != nil {
		t.Fatal(err)
	}

	model := redefine(t, tg, db, []string{"code", "name", "description"}, []string{"en"})
	after := indexes()
	if len(after) != 2 || slices.Contains(after, first[0]) || !slices.Contains(after, "articles_full_text_fts_manual") {
		t.Errorf("orphan after changing the fields: %v (before %v)", after, first)
	}
	got, err := searchIds(model, et.Json{"search": "bomba"}, false)
	if err != nil || strings.Join(got, ",") != "a1" {
		t.Errorf("search after the change: %v %v", got, err)
	}

	envar.SetStr("DB_SEARCH_INDEXES", "off")
	model = redefine(t, tg, db, []string{"name", "description"}, nil)
	envar.SetStr("DB_SEARCH_INDEXES", "on")
	if now := indexes(); strings.Join(now, ",") != strings.Join(after, ",") {
		t.Errorf("with DB_SEARCH_INDEXES=off Init changed the indexes: %v", now)
	}
	if err := db.ApplySearchIndexes(model); err != nil {
		t.Fatal("ApplySearchIndexes:", err)
	}
	if now := indexes(); len(now) != 2 || slices.Equal(now, after) {
		t.Errorf("ApplySearchIndexes: %v (before %v)", now, after)
	}
}
