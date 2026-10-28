package jsql

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* TestQuoted: Every value becomes a SQL literal: strings escaped, numbers bare, pointers dereferenced,
* lists and unknown values as quoted JSON (never Go's %v), nil as NULL.
**/
func TestQuoted(t *testing.T) {
	now := time.Date(2026, 10, 27, 13, 4, 5, 0, time.UTC)
	var nilTime *time.Time
	name := "O'Neil"
	cases := []struct {
		in   any
		want string
	}{
		{"O'Neil", "'O''Neil'"},
		{&name, "'O''Neil'"},
		{42, "42"},
		{uint16(7), "7"},
		{int8(-3), "-3"},
		{2.5, "2.5"},
		{true, "true"},
		{nil, "NULL"},
		{nilTime, "NULL"},
		{now, "'2026-10-27 13:04:05'"},
		{&now, "'2026-10-27 13:04:05'"},
		{[]byte{0xab}, `'\xab'`},
		{[]int{1, 2}, "'[1,2]'"},
		{[]float64{1.5}, "'[1.5]'"},
		{et.Json{"a": "it's"}, `'{"a":"it''s"}'`},
		{struct{ N int }{3}, `'{"N":3}'`},
	}
	for _, c := range cases {
		if got := fmt.Sprint(quoted(c.in)); got != c.want {
			t.Errorf("quoted(%#v) = %s, want %s", c.in, got, c.want)
		}
	}
}

/**
* TestSqlParse: $N placeholders are replaced once, whole ($10 is not $1 + 0), with quoted values.
**/
func TestSqlParse(t *testing.T) {
	args := []any{"a", 2, 3, 4, 5, 6, 7, 8, 9, "ten"}
	got := sqlParse("SELECT $1, $10, $11, '$2'", args...)
	want := "SELECT 'a', 'ten', $11, '2'"
	if got != want {
		t.Errorf("sqlParse = %s, want %s", got, want)
	}
	if got := sqlParse("SELECT $1", "$2"); got != "SELECT '$2'" {
		t.Errorf("a value with $2 is replaced again: %s", got)
	}
}

/**
* TestRollupConditions: The keys are repeated in every OR term of the where, so an OR never drops them.
**/
func TestRollupConditions(t *testing.T) {
	str := func(conds []*et.Condition) string {
		parts := []string{}
		for _, c := range conds {
			parts = append(parts, fmt.Sprintf("%s %s", c.Connector, c.Field.String()))
		}
		return strings.Join(parts, ", ")
	}
	key := Eq("user_id", "u1")
	cases := []struct {
		where []*et.Condition
		want  string
	}{
		{nil, " user_id"},
		{[]*et.Condition{More("amount", 1)}, " user_id, and amount"},
		{[]*et.Condition{More("amount", 1), Or("status", et.EQ, "x")}, " user_id, and amount, or user_id, and status"},
		{[]*et.Condition{More("a", 1), And("b", et.EQ, 2), Or("c", et.EQ, 3)}, " user_id, and a, and b, or user_id, and c"},
	}
	for _, c := range cases {
		got := str(rollupConditions([]*et.Condition{key}, c.where))
		if got != c.want {
			t.Errorf("rollupConditions = %q, want %q", got, c.want)
		}
	}
	// The definition is not changed
	where := []*et.Condition{More("amount", 1)}
	rollupConditions([]*et.Condition{key}, where)
	if where[0].Connector != et.NaC || key.Connector != et.NaC {
		t.Errorf("rollupConditions changed its input: %s, %s", where[0].Connector, key.Connector)
	}
}

/**
* TestResolveWhere: A {field: "x"} value is read from the row; a row without x does not apply the rollup.
**/
func TestResolveWhere(t *testing.T) {
	rollup := &QueryRollups{Where: []*et.Condition{More("amount", et.Fld("limit")), Eq("status", "paid")}}
	where, ok := rollup.resolveWhere(et.Json{"limit": 30})
	if !ok || len(where) != 2 || where[0].Value.Value != 30 || where[1].Value.Value != "paid" {
		t.Fatalf("resolveWhere = %v, %v", where, ok)
	}
	if rollup.Where[0].Value.Type != et.FIELD {
		t.Errorf("resolveWhere changed the definition")
	}
	if _, ok := rollup.resolveWhere(et.Json{"other": 1}); ok {
		t.Errorf("a row without limit should not apply the rollup")
	}
	if !rollup.hasRowRefs() {
		t.Errorf("hasRowRefs = false")
	}
}

/**
* TestRollupsJson: Rollups and QueryRollups keep their where (as a list) and order through JSON.
**/
func TestRollupsJson(t *testing.T) {
	rollup := &Rollups{Keys: map[string]string{"id": "user_id"}, Operation: RollupObject,
		Where: []*et.Condition{More("amount", et.Fld("age")), Or("status", et.EQ, "x")}}
	rollup.OrderBy("created_at", false)
	bt, err := json.Marshal(rollup)
	if err != nil {
		t.Fatal(err)
	}
	var back Rollups
	if err := json.Unmarshal(bt, &back); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(whereToJson(back.Where)) != fmt.Sprint(whereToJson(rollup.Where)) {
		t.Errorf("where = %v, want %v", whereToJson(back.Where), whereToJson(rollup.Where))
	}
	if len(back.OrdersBy) != 1 || back.OrdersBy[0].Name != "created_at" || back.OrdersBy[0].Sorted {
		t.Errorf("order_by = %v", back.OrdersBy)
	}
	var query QueryRollups
	if err := json.Unmarshal(bt, &query); err != nil || len(query.Where) != 2 || len(query.OrdersBy) != 1 {
		t.Errorf("QueryRollups from json: %v, %v", query, err)
	}
	if fields := back.rowFields(); len(fields) != 2 {
		t.Errorf("rowFields = %v, want id and age", fields)
	}
}

/**
* TestRollupWhereOf: The where of defineRollup in scripts: an object, a list, a condition or null.
**/
func TestRollupWhereOf(t *testing.T) {
	cases := []struct {
		in   any
		want int
	}{
		{map[string]any{"amount": map[string]any{"more": 150}}, 1},
		{[]any{map[string]any{"a": map[string]any{"eq": 1}}, map[string]any{"or": map[string]any{"b": map[string]any{"eq": 2}}}}, 2},
		{Eq("a", 1), 1},
		{nil, 0},
	}
	for _, c := range cases {
		got, err := rollupWhereOf(c.in)
		if err != nil || len(got) != c.want {
			t.Errorf("rollupWhereOf(%v) = %v, %v; want %d conditions", c.in, got, err, c.want)
		}
	}
	if _, err := rollupWhereOf("nope"); err == nil {
		t.Errorf("rollupWhereOf(text) should fail")
	}
}

/**
* TestBatchHelpers: keyValues, groupRows, pageOf and firstOf, used to resolve relations in batches.
**/
func TestBatchHelpers(t *testing.T) {
	items := []et.Json{{"id": 1}, {"id": 1.0}, {"id": nil}, {"other": 2}, {"id": "b"}}
	if got := fmt.Sprint(keyValues(items, "id")); got != "[1 b]" {
		t.Errorf("keyValues = %s", got)
	}
	rows := []et.Json{{"k": 1, "v": "a"}, {"k": 2, "v": "b"}, {"k": 1.0, "v": "c"}}
	groups := groupRows(rows, "k", true)
	if len(groups["1"]) != 2 || len(groups["2"]) != 1 || rows[0]["k"] != nil {
		t.Errorf("groupRows = %v", groups)
	}
	list := []et.Json{{"n": 1}, {"n": 2}, {"n": 3}}
	if got := pageOf(list, 2, 2); len(got) != 1 || got[0]["n"] != 3 {
		t.Errorf("pageOf(2, 2) = %v", got)
	}
	if got := pageOf(list, 3, 2); len(got) != 0 {
		t.Errorf("pageOf beyond the end = %v", got)
	}
	if got := pageOf(list, 1, 0); len(got) != 3 {
		t.Errorf("pageOf without rows = %v", got)
	}
	if firstOf(nil) != nil || firstOf(list)["n"] != 1 {
		t.Errorf("firstOf")
	}
	if k, fk, ok := singleKey(map[string]string{"id": "user_id"}); !ok || k != "id" || fk != "user_id" {
		t.Errorf("singleKey")
	}
	if _, _, ok := singleKey(map[string]string{"a": "b", "c": "d"}); ok {
		t.Errorf("singleKey with two keys")
	}
}

/**
* TestSchemasConcurrent: Models are looked up and defined from many goroutines (run with -race); looking up
* a model of a schema that does not exist does not create the schema.
**/
func TestSchemasConcurrent(t *testing.T) {
	db := &DB{Name: "test", Schemas: map[string]*Schema{}}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			db.newModel(fmt.Sprintf("s%d", i%4), fmt.Sprintf("m%d", i), 1, "")
		}(i)
		go func(i int) {
			defer wg.Done()
			db.getModel(fmt.Sprintf("missing%d", i), "m")
			db.toJson()
		}(i)
	}
	wg.Wait()
	if _, ok := db.lookupSchema("missing0"); ok {
		t.Errorf("getModel created the schema it looked up")
	}
	if len(db.schemaList()) != 4 {
		t.Errorf("schemas = %d, want 4", len(db.schemaList()))
	}
}

/**
* TestDefineConcurrent: Definitions on one model from many goroutines are serialized (run with -race): none is
* lost and the columns slice is never seen half replaced.
**/
func TestDefineConcurrent(t *testing.T) {
	db := &DB{Name: "test", Schemas: map[string]*Schema{}}
	model := db.newModel("s", "m", 1, "")
	to := db.newModel("s", "to", 1, "")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			model.DefineRollup(fmt.Sprintf("r%d", i), to, map[string]string{"id": "m_id"}, nil, RollupCount)
		}(i)
		go func(i int) {
			defer wg.Done()
			model.DefineCalc(fmt.Sprintf("c%d", i), "item.x = 1;")
		}(i)
	}
	wg.Wait()
	if len(model.Rollups) != 20 || len(model.calcScripts) != 20 {
		t.Errorf("rollups = %d, calcs = %d, want 20", len(model.Rollups), len(model.calcScripts))
	}
	for i := 0; i < 20; i++ {
		if model.indexColumn(fmt.Sprintf("r%d", i)) == -1 || model.indexColumn(fmt.Sprintf("c%d", i)) == -1 {
			t.Fatalf("column %d lost", i)
		}
	}
}
