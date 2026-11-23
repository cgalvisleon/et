package cache

import (
	"testing"
	"time"

	"github.com/cgalvisleon/et/et"
)

/**
* TestTextOf: El texto con que se guarda cada tipo; uno desconocido no se guarda
**/
func TestTextOf(t *testing.T) {
	at := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	cases := []struct {
		val  interface{}
		want string
	}{
		{"hola", "hola"},
		{7, "7"},
		{int64(8), "8"},
		{1.5, "1.5"},
		{true, "true"},
		{[]byte("b"), "b"},
		{at, "2026-11-23T10:00:00Z"},
		{time.Second, "1s"},
		{et.Json{"a": 1}, `{"a":1}`},
	}
	for _, c := range cases {
		got, ok := TextOf(c.val)
		if !ok || got != c.want {
			t.Fatalf("TextOf(%v) = %q, %v; quería %q", c.val, got, ok, c.want)
		}
	}

	if _, ok := TextOf(struct{}{}); ok {
		t.Fatal("un tipo desconocido no tiene texto")
	}
}
