package et

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

/**
* TestEvaluatePrecedence: AND before OR, the same in EvaluateObject, EvaluateValue and Wheres.
**/
func TestEvaluatePrecedence(t *testing.T) {
	obj := Json{"a": 1, "b": 0, "c": 0}
	cases := []struct {
		name  string
		where []*Condition
		want  bool
	}{
		{"a OR b AND c", []*Condition{Eq("a", 1), Or("b", EQ, 1), And("c", EQ, 1)}, true},
		{"b AND c OR a", []*Condition{Eq("b", 1), And("c", EQ, 1), Or("a", EQ, 1)}, true},
		{"b OR a AND c", []*Condition{Eq("b", 1), Or("a", EQ, 1), And("c", EQ, 1)}, false},
		{"a AND b OR c", []*Condition{Eq("a", 1), And("b", EQ, 1), Or("c", EQ, 1)}, false},
		{"a, b without connector", []*Condition{Eq("a", 1), Eq("b", 1)}, false},
		{"a, a without connector", []*Condition{Eq("a", 1), Eq("a", 1)}, true},
		{"empty", nil, true},
	}
	for _, c := range cases {
		if got := EvaluateObject(obj, c.where); got != c.want {
			t.Errorf("EvaluateObject %s = %v, want %v", c.name, got, c.want)
		}
		if got := len(From([]Json{obj}, "").whereAll(c.where)) == 1; got != c.want {
			t.Errorf("Wheres %s = %v, want %v", c.name, got, c.want)
		}
	}

	value := []*Condition{More("v", 10), Or("v", EQ, 1), And("v", LESS, 0)}
	if EvaluateValue(1, value) {
		t.Errorf("EvaluateValue 1 > 10 OR 1 = 1 AND 1 < 0 = true, want false")
	}
	if !EvaluateValue(11, value) {
		t.Errorf("EvaluateValue 11 > 10 OR … = false, want true")
	}
}

/**
* whereAll: Runs the conditions on a Wheres (test helper)
* @param conditions []*Condition
* @return []Json
**/
func (s *Wheres) whereAll(conditions []*Condition) []Json {
	s.Conditions = conditions
	return s.All()
}

/**
* TestCompareNumbers: Exact integers, floats with decimals and NaN.
**/
func TestCompareNumbers(t *testing.T) {
	cases := []struct {
		a, b any
		want int
		ok   bool
	}{
		{int64(9007199254740993), int64(9007199254740992), 1, true},
		{int64(9007199254740993), float64(9007199254740992), 1, true},
		{uint64(18446744073709551615), int64(-1), 1, true},
		{int64(-1), uint64(0), -1, true},
		{json.Number("9007199254740993"), int64(9007199254740993), 0, true},
		{json.Number("18446744073709551615"), uint64(18446744073709551615), 0, true},
		{1, 1.0, 0, true},
		{2, 2.5, -1, true},
		{2.5, 2.4, 1, true},
		{int8(3), uint16(3), 0, true},
		{"1", 1, 0, false},
	}
	for _, c := range cases {
		got, ok := compareNumbers(c.a, c.b)
		if ok != c.ok || (ok && got != c.want) {
			t.Errorf("compareNumbers(%v, %v) = %d, %v; want %d, %v", c.a, c.b, got, ok, c.want, c.ok)
		}
	}

	big := Json{"id": int64(9007199254740993)}
	if EvaluateObject(big, []*Condition{Eq("id", int64(9007199254740992))}) {
		t.Errorf("two different ids beyond 2^53 are equal")
	}
	if !EvaluateObject(big, []*Condition{Eq("id", int64(9007199254740993))}) {
		t.Errorf("the same id beyond 2^53 is not equal")
	}
	if !EvaluateObject(big, []*Condition{More("id", int64(9007199254740992))}) {
		t.Errorf("more on ids beyond 2^53")
	}
}

/**
* TestMatchLike: like as SQL does it (% and _, case insensitive, wildcards anywhere), with * kept as %.
**/
func TestMatchLike(t *testing.T) {
	cases := []struct {
		value, pattern string
		want           bool
	}{
		{"Ana Maria", "%ana%", true},
		{"Ana Maria", "%MARIA", true},
		{"Ana Maria", "ana%", true},
		{"Ana Maria", "*maria", true},
		{"Ana Maria", "ana*", true},
		{"Ana Maria", "a%m%a", true},
		{"Ana Maria", "a%x%a", false},
		{"Ana Maria", "ana", false},
		{"Ana", "ana", true},
		{"Ana", "a_a", true},
		{"Ana", "a_", false},
		{"Ana", "___", true},
		{"", "%", true},
		{"", "_", false},
		{"50%", `50\%`, true},
		{"500", `50\%`, false},
		{"a_b", `a\_b`, true},
		{"axb", `a\_b`, false},
		{"aaab", "%ab", true},
		{"mississippi", "%iss%ppi", true},
		{"Ñandú", "ñan%", true},
	}
	for _, c := range cases {
		if got := MatchLike(c.value, c.pattern); got != c.want {
			t.Errorf("MatchLike(%q, %q) = %v, want %v", c.value, c.pattern, got, c.want)
		}
	}
	if !EvaluateObject(Json{"name": "Ana Maria"}, []*Condition{Like("name", "%maria")}) {
		t.Errorf("Like %%maria")
	}
}

/**
* TestOperators: Every operator applied to a value.
**/
func TestOperators(t *testing.T) {
	now := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		cond *Condition
		val  any
		want bool
	}{
		{Eq("v", "a"), "a", true},
		{Eq("v", "a"), "b", false},
		{Eq("v", 2), 2.0, true},
		{Eq("v", "2026-10-27T10:00:00Z"), now, true},
		{Neg("v", 1), 2, true},
		{Neg("v", 1), 1, false},
		{Less("v", 5), 4, true},
		{Less("v", 5), 5, false},
		{condition("v", 5, LESS_EQ), 5, true},
		{More("v", "b"), "c", true},
		{More("v", now), now.Add(time.Second), true},
		{condition("v", 5, MORE_EQ), 5, true},
		{condition("v", 5, MORE_EQ), "5", false},
		{In("v", []any{1, "x", 3}), "x", true},
		{In("v", []any{1, 2}), 3, false},
		{condition("v", []any{1, 2}, NOT_IN), 3, true},
		{Null("v"), nil, true},
		{Null("v"), 0, false},
		{condition("v", nil, NOT_NULL), 0, true},
		{Between("v", 1, 10), 10, true},
		{Between("v", 1, 10), 11, false},
		{NotBetween("v", 1, 10), 11, true},
		{Between("v", "2026-01-01T00:00:00Z", "2026-12-31T00:00:00Z"), "2026-10-27T00:00:00Z", true},
		{Less("v", 5), nil, false},
	}
	for _, c := range cases {
		if got := c.cond.ApplyToValue(c.val); got != c.want {
			t.Errorf("%s %v on %v = %v, want %v", c.cond.Operator, c.cond.Value.Value, c.val, got, c.want)
		}
	}
}

/**
* TestWheres: Joins, order, page and limit, select and hidden over lists in memory.
**/
func TestWheres(t *testing.T) {
	users := []Json{{"id": 1, "name": "ana", "age": 30}, {"id": 2, "name": "bob", "age": 20}, {"id": 3, "name": "eva", "age": 40}}
	orders := []Json{{"user_id": 1, "total": 10}, {"user_id": 1, "total": 5}, {"user_id": 3, "total": 7}}

	got := From(users, "").Where(More("age", 25)).Desc("age").All()
	if len(got) != 2 || got[0]["name"] != "eva" || got[1]["name"] != "ana" {
		t.Errorf("where + desc = %v", got)
	}

	got = From(users, "").Asc("age").Limit(2, 1).All()
	if len(got) != 1 || got[0]["name"] != "ana" {
		t.Errorf("asc + page 2 = %v", got)
	}

	got = From(users, "").Limit(1, 2).All()
	if len(got) != 2 {
		t.Errorf("limit without order = %v", got)
	}

	got = From(users, "").Hidden("age").Select("name").All()
	if len(got) != 3 || len(got[0]) != 1 || got[0]["name"] == nil {
		t.Errorf("select + hidden = %v", got)
	}

	keys := map[string]string{"u.id": "o.user_id"}
	join := From(users, "u").Join(orders, "o", keys).Where(Eq("o.total", 7)).All()
	if len(join) != 1 || join[0]["u.name"] != "eva" {
		t.Errorf("join = %v", join)
	}

	left := From(users, "u").LeftJoin(orders, "o", keys).All()
	if len(left) != 4 {
		t.Errorf("left join = %d rows, want 4", len(left))
	}
}

/**
* TestSafeData: Trim keeps the newest items, Set ignores an index out of range, SafeMap works without NewSafeMap.
**/
func TestSafeData(t *testing.T) {
	data := NewSafeData()
	for i := 0; i < 5; i++ {
		data.Add(Json{"n": i})
	}
	if removed := data.Trim(3); removed != 2 || data.Len() != 3 {
		t.Fatalf("Trim(3) = %d, len %d", removed, data.Len())
	}
	if first, _ := data.Get(0); first["n"] != 2 {
		t.Errorf("Trim kept %v first, want the newest (n = 2)", first)
	}
	if data.Trim(10) != 0 || data.Len() != 3 {
		t.Errorf("Trim above the length removed items")
	}
	if data.Set(7, Json{}) || !data.Set(0, Json{"n": 9}) {
		t.Errorf("Set out of range")
	}
	if data.Trim(0); data.Len() != 0 {
		t.Errorf("Trim(0) left %d", data.Len())
	}

	var m SafeMap
	m.Set("a", 1)
	if v, ok := m.Get("a"); !ok || v != 1 {
		t.Errorf("SafeMap without constructor = %v, %v", v, ok)
	}
	if v, ok := NewSafeMap().Get("x"); ok || v != nil {
		t.Errorf("NewSafeMap")
	}
}

/**
* TestJsonSafe: Put into itself, SetNested without keys and DeepClone.
**/
func TestJsonSafe(t *testing.T) {
	j := Json{"a": 1}
	j.Put(j)
	if j["a"] != 1 {
		t.Errorf("Put into itself emptied the map: %v", j)
	}
	var nilJson Json
	nilJson.Put(Json{"a": 1})

	j.SetNested(nil, 2)
	if len(j) != 1 {
		t.Errorf("SetNested without keys = %v", j)
	}

	src := Json{"user": Json{"name": "ana"}, "tags": []any{Json{"t": 1}}, "list": []Json{{"x": 1}}, "m": map[string]any{"k": 1}}
	deep := src.DeepClone()
	deep.Json("user")["name"] = "bob"
	deep["tags"].([]any)[0].(Json)["t"] = 2
	deep["list"].([]Json)[0]["x"] = 2
	deep["m"].(map[string]any)["k"] = 2
	if src.Json("user")["name"] != "ana" || src["tags"].([]any)[0].(Json)["t"] != 1 ||
		src["list"].([]Json)[0]["x"] != 1 || src["m"].(map[string]any)["k"] != 1 {
		t.Errorf("DeepClone shares nested values: %v", src)
	}
	shallow := src.Clone()
	shallow.Json("user")["name"] = "eva"
	if src.Json("user")["name"] != "eva" {
		t.Errorf("Clone is expected to be shallow")
	}
}

/**
* TestDeepEqualValue: Scalars without JSON encoding, numbers by value, objects and lists through JSON.
**/
func TestDeepEqualValue(t *testing.T) {
	cases := []struct {
		a, b any
		want bool
	}{
		{nil, nil, true},
		{nil, 0, false},
		{"a", "a", true},
		{"1", 1, false},
		{1, 1.0, true},
		{int64(9007199254740993), int64(9007199254740992), false},
		{true, true, true},
		{true, 1, false},
		{Json{"a": 1}, map[string]any{"a": 1}, true},
		{[]any{1, 2}, []any{1, 2}, true},
		{[]any{1, 2}, []any{2, 1}, false},
	}
	for _, c := range cases {
		if got := deepEqualValue(c.a, c.b); got != c.want {
			t.Errorf("deepEqualValue(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
	if (Json{"a": 1, "b": "x"}).IsChanged(Json{"a": 1.0}) {
		t.Errorf("IsChanged 1 vs 1.0")
	}
}

/**
* TestParseJSONFresh: A repaired text does not keep the keys of a failed attempt.
**/
func TestParseJSONFresh(t *testing.T) {
	result := map[string]any{}
	// The first attempt reads "a" before failing at the raw line break inside "b"
	if err := parseJSON("{\"a\": 1, \"b\": \"x\ny\"}", &result); err != nil {
		t.Fatal(err)
	}
	if result["b"] != "x\ny" || len(result) != 2 {
		t.Errorf("parseJSON = %v", result)
	}

	var list []Json
	if err := parseJSON(`"[{\"a\":1}]"`, &list); err != nil || len(list) != 1 || fmt.Sprint(list[0]["a"]) != "1" {
		t.Errorf("parseJSON double encoded = %v, %v", list, err)
	}
}

/**
* TestToWhere: Un where de un script conserva el nombre del campo (B14 de jsql): un
* objeto {campo: {op: valor}}, un *Condition, una lista y un objeto con where/and
**/
func TestToWhere(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{map[string]any{"status": map[string]any{"eq": "active"}}, "[map[status:map[eq:active]]]"},
		{Eq("id", "c1"), "[map[id:map[eq:c1]]]"},
		{[]any{map[string]any{"a": map[string]any{"eq": 1}}, map[string]any{"and": map[string]any{"b": map[string]any{"eq": 2}}}}, "[map[a:map[eq:1]] map[and:map[b:map[eq:2]]]]"},
		{map[string]any{"and": map[string]any{"b": map[string]any{"eq": 2}}, "where": map[string]any{"a": map[string]any{"eq": 1}}}, "[map[where:map[a:map[eq:1]]] map[and:map[b:map[eq:2]]]]"},
		{"nope", "[]"},
	}
	for _, c := range cases {
		if got := fmt.Sprint(ToWhere(c.in)); got != c.want {
			t.Errorf("ToWhere(%v) = %s, want %s", c.in, got, c.want)
		}
	}
	// Every result is a condition ToWheRes reads
	for _, c := range cases[:4] {
		if _, err := ToWheRes(ToWhere(c.in)); err != nil {
			t.Errorf("ToWheRes(ToWhere(%v)): %v", c.in, err)
		}
	}
}

/**
* TestItemsOutOfRange: The accessors of Items give their default value with an index out of range, not a panic.
**/
func TestItemsOutOfRange(t *testing.T) {
	items := Items{Result: []Json{{"a": "x"}}, Count: 3}
	if items.Str(0, "a") != "x" || items.Str(5, "a") != "" || items.Str(-1, "a") != "" {
		t.Errorf("Str out of range")
	}
	if items.ValInt(9, 7, "a") != 7 || items.Exist(9, "a") || items.Get(9, "a") != nil {
		t.Errorf("ValInt/Exist/Get out of range")
	}
	if _, err := items.One(3); err == nil {
		t.Errorf("One beyond the result with a wrong Count")
	}
}
