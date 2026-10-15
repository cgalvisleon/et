package et

import (
	"testing"

	"github.com/dop251/goja"
)

/**
* TestEvaluateField: Conditions whose value is a field reference ({field: "name"}, FIELD) or an aggregate, evaluated
* on an object, from Go and from a script (the wrappers set as goja globals).
**/
func TestEvaluateField(t *testing.T) {
	obj := Json{
		"age": 20, "min_age": 18, "max_age": 19, "total": 10, "c": 7,
		"prices":     []any{2, 3, 5},
		"items":      []any{Json{"price": 2}, map[string]any{"price": 3.5}},
		"a":          Json{"b": 7},
		"created_at": "2026-10-15T10:00:00Z",
	}

	cases := []struct {
		where []Json
		want  bool
	}{
		{[]Json{{"age": Json{"more_eq": Json{"field": "min_age"}}}}, true},
		{[]Json{{"age": Json{"less": Json{"field": "max_age"}}}}, false},
		{[]Json{{"c": Json{"eq": Json{"field": "a->b"}}}}, true},
		{[]Json{{"total": Json{"eq": Json{"field": "sum(prices)"}}}}, true},
		{[]Json{{"total": Json{"less": Json{"field": "sum(items->price)"}}}}, false},
		{[]Json{{"max_age": Json{"more": Json{"field": "count(prices)"}}}}, true},
		{[]Json{{"extract_year(created_at)": Json{"eq": 2026}}}, true},
		{[]Json{{"sum(prices)": Json{"eq": Json{"field": "total"}}}}, true},
		{[]Json{{"age": Json{"eq": Json{"field": "missing"}}}}, false},
		{[]Json{{"age": Json{"eq": 1}}, {"or": Json{"c": Json{"eq": Json{"field": "a->b"}}}}}, true},
	}
	for _, c := range cases {
		conditions, err := ToConditions(c.where)
		if err != nil {
			t.Fatal(err)
		}
		if got := EvaluateObject(obj, conditions); got != c.want {
			t.Errorf("%v = %v, want %v", c.where, got, c.want)
		}
	}

	// ToJson writes a reference back as {field: "…"}
	cond, _ := ToCondition(Json{"age": Json{"more": Json{"field": "sum(A.prices):total"}}})
	if got := cond.ToJson().ToString(); got != `{"age":{"more":{"field":"sum(A.prices):total"}}}` {
		t.Errorf("ToJson = %s", got)
	}

	vm := goja.New()
	vm.Set("evaluateObject", EvaluateObjectWrapper())
	vm.Set("evaluateValue", EvaluateValueWrapper())
	for name, fn := range FieldWrapper() {
		vm.Set(name, fn)
	}
	vm.Set("o", obj)
	scripts := map[string]any{
		`evaluateObject(o, [{age: {more_eq: field("min_age")}}, {and: {total: {eq: sum("prices")}}}])`: true,
		`evaluateObject(o, [{max_age: {eq: max("prices")}}])`:                                          false,
		`evaluateValue(20, [{value: {more_eq: field("min_age")}}], o)`:                                 true,
		`evaluateValue(20, [{value: {more_eq: field("min_age")}}])`:                                    false,
		`evaluateValue(10, [{value: {eq: month("created_at")}}], o)`:                                   true,
		`JSON.stringify([field("x"), avg("items->price"), year("created_at")])`:                        `[{"field":"x"},{"field":"avg(items->price)"},{"field":"extract_year(created_at)"}]`,
		`try { evaluateObject(o, [{}]); "no" } catch (e) { "throws" }`:                                 "throws",
	}
	for code, want := range scripts {
		value, err := vm.RunString(code)
		if err != nil {
			t.Errorf("%s: %v", code, err)
			continue
		}
		if got := value.Export(); got != want {
			t.Errorf("%s = %v, want %v", code, got, want)
		}
	}
}
