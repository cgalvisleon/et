package jsql

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cgalvisleon/et/envar"
	"github.com/cgalvisleon/et/et"
)

/**
* TestSearchConfig: An ISO code is the configuration of the engine, a configuration is accepted by its name, an
* unknown lang is simple and an empty one the default lang (DB_SEARCH_LANG or spanish).
**/
func TestSearchConfig(t *testing.T) {
	cases := map[string]string{
		"es": "spanish", "ES": "spanish", "es-CO": "spanish", "pt_BR": "portuguese", "en": "english", "fr": "french",
		"de": "german", "it": "italian", "nl": "dutch", "ru": "russian", "nb": "norwegian", "spanish": "spanish",
		"simple": "simple", "english": "english", "xx": "simple", "klingon": "simple", "es';drop": "simple",
		"": "spanish", " ": "spanish",
	}
	for lang, want := range cases {
		if got := searchConfig(lang); got != want {
			t.Errorf("searchConfig(%q) = %q, want %q", lang, got, want)
		}
	}
	envar.SetStr("DB_SEARCH_LANG", "en")
	defer envar.SetStr("DB_SEARCH_LANG", "spanish")
	if got := searchConfig(""); got != "english" {
		t.Errorf("searchConfig with DB_SEARCH_LANG=en = %q", got)
	}
}

/**
* searchModel: A model without connection, with a column name, a memo description and the SourceField.
* @return *DB, *Model
**/
func searchModel() (*DB, *Model) {
	db := &DB{Name: "test", Schemas: map[string]*Schema{}}
	model := db.newModel("s", "products", 1, "")
	model.DefineColumn("name", et.TEXT, "")
	model.DefineColumn("description", et.MEMO, "")
	model.DefineSource()
	return db, model
}

/**
* TestDefineFullText: full_text is the search full_text: its lang by default, its langs kept when its fields change,
* the langs of the index without repeating a configuration, and an invalid lang rejected.
**/
func TestDefineFullText(t *testing.T) {
	_, model := searchModel()
	if _, err := model.DefineFullTextLang("en"); err == nil || !strings.Contains(err.Error(), "full_text") {
		t.Errorf("lang without full_text: %v", err)
	}
	search, err := model.DefineFullText("name", "description", "tags")
	if err != nil || search.Lang != "spanish" || search.Name != FULL_TEXT {
		t.Fatalf("DefineFullText: %v %v", search, err)
	}
	if _, err := model.DefineFullTextLang("es", "en", "spanish", "EN"); err != nil {
		t.Fatal(err)
	}
	search, _ = model.DefineFullText("name", "tags")
	if search.Lang != "es" || strings.Join(search.Langs, ",") != "en,spanish" || strings.Join(search.Fields, ",") != "name,tags" {
		t.Errorf("full_text keeps its langs: %+v", search)
	}
	if got := strings.Join(search.indexConfigs(), ","); got != "spanish,english" {
		t.Errorf("indexConfigs = %s", got)
	}
	if _, err := model.DefineFullTextLang("es';drop"); err == nil {
		t.Error("invalid lang: want an error")
	}
	if _, err := model.DefineSearch("fts", "", "created_x;drop"); err == nil {
		t.Error("invalid field: want an error")
	}
}

/**
* TestFullTextJson: full_text goes by its own keys in ToJson (not in searches), comes back with UnmarshalJSON, and
* twice (full_text and an entry full_text of searches) is an error.
**/
func TestFullTextJson(t *testing.T) {
	_, model := searchModel()
	model.DefineFullText("name", "description")
	model.DefineFullTextLang("es", "en")
	model.DefineSearch("fts", "en", "name")
	data := model.ToJson()
	if data["full_text_lang"] != "es" || strings.Join(data["full_text"].([]string), ",") != "name,description" {
		t.Errorf("ToJson full_text: %v %v", data["full_text"], data["full_text_lang"])
	}
	if _, ok := data["searches"].(map[string]*TextSearch)[FULL_TEXT]; ok {
		t.Error("full_text also in searches")
	}

	bt, _ := json.Marshal(data)
	var back Model
	if err := json.Unmarshal(bt, &back); err != nil {
		t.Fatal(err)
	}
	search := back.Searches[FULL_TEXT]
	if search == nil || search.Lang != "es" || strings.Join(search.Langs, ",") != "en" || back.Searches["fts"] == nil {
		t.Errorf("round trip: %+v %v", search, back.Searches)
	}

	twice := `{"name": "m", "full_text": ["name"], "searches": {"full_text": {"name": "full_text", "fields": ["name"]}}}`
	if err := json.Unmarshal([]byte(twice), &back); err == nil {
		t.Error("full_text twice in the json of a model: want an error")
	}
	var define Define
	json.Unmarshal([]byte(`{"full_text": ["name"], "full_text_lang": "en", "searches": [{"name": "full_text", "fields": ["name"]}]}`), &define)
	if err := model.defineFullTextOf(define); err == nil {
		t.Error("full_text twice in a Define: want an error")
	}
	if define.FullTextLang != "en" {
		t.Errorf("Define.FullTextLang = %q", define.FullTextLang)
	}
}

/**
* TestWithFullText: The key "search" goes with AND to each group of the or, sorts by relevance without orders (not in
* a count), leaves the query as it was, and fails without full_text.
**/
func TestWithFullText(t *testing.T) {
	db, model := searchModel()
	db.RecordLimit = 100
	model.DefineFullText("name")
	query, err := newQuery(model).loadQuery(et.Json{
		"where":  []any{et.Json{"name": et.Json{"eq": "a"}}, et.Json{"or": et.Json{"name": et.Json{"eq": "b"}}}},
		"search": et.Json{"text": "bomba", "lang": "en"},
	})
	if err != nil {
		t.Fatal(err)
	}
	restore, err := query.withFullText()
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, cond := range query.Conditions {
		got = append(got, cond.Connector.Str()+" "+cond.Field.String()+" "+cond.Operator.Str())
	}
	want := " A.full_text search|and name eq|or A.full_text search|and name eq"
	if strings.Join(got, "|") != want {
		t.Errorf("conditions = %q", strings.Join(got, "|"))
	}
	if text, lang, ok := SearchOf(query.Conditions[0]); text != "bomba" || lang != "en" || !ok {
		t.Errorf("search value = %q %q", text, lang)
	}
	if len(query.OrdersBy) != 1 || query.OrdersBy[0].Name != "A.full_text" || query.OrdersBy[0].Sorted {
		t.Errorf("relevance order: %v", query.OrdersBy)
	}
	restore()
	if len(query.Conditions) != 2 || len(query.OrdersBy) != 0 {
		t.Errorf("restore: %d conditions, %d orders", len(query.Conditions), len(query.OrdersBy))
	}

	count := newQuery(model).Search("bomba")
	count.IsCount = true
	restore, _ = count.withFullText()
	if len(count.Conditions) != 1 || len(count.OrdersBy) != 0 {
		t.Errorf("count: %d conditions, %d orders", len(count.Conditions), len(count.OrdersBy))
	}
	restore()

	if _, err := newQuery(model).loadQuery(et.Json{"search": 12}); err == nil {
		t.Error("search with a number: want an error")
	}
	_, other := searchModel()
	other.Name = "other"
	if _, err := newQuery(other).Search("bomba").withFullText(); err == nil || !strings.Contains(err.Error(), "full_text") {
		t.Errorf("without full_text: %v", err)
	}
}
