package et

import (
	"fmt"
	"testing"
)

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
