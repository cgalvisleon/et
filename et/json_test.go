package et

import (
	"encoding/json"
	"testing"
	"time"
)

func TestToJsonDamagedText(t *testing.T) {
	cases := map[string]string{
		"pretty":              "{\n  \"a\": 1,\n  \"b\": \"x\"\n}",
		"raw newline":         "{\"a\": 1, \"b\": \"l1\nl2\"}",
		"escaped once more":   `{\n  \"a\": 1,\n  \"b\": \"x\"\n}`,
		"encoded twice":       `"{\"a\":1,\"b\":\"x\"}"`,
		"bom and spaces":      "\uFEFF  {\"a\":1,\"b\":\"x\"} \n",
		"escaped with tab":    "{\"a\":1,\"b\":\"x\ty\"}",
		"escaped and newline": "{\\\"a\\\":\n1,\\\"b\\\":\\\"x\\\"}",
	}
	for name, src := range cases {
		j, err := ToJson(src)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if j.Int("a") != 1 || j.Str("b") == "" {
			t.Errorf("%s: got %v", name, j)
		}
	}

	j, _ := ToJson("{\"b\": \"l1\nl2\"}")
	if j.Str("b") != "l1\nl2" {
		t.Errorf("raw newline kept in the value: %q", j.Str("b"))
	}
	if _, err := ToJson(`{"a": `); err == nil {
		t.Error("truncated JSON must fail")
	}
	if _, err := ToJson(`{"a":1} {"b":2}`); err == nil {
		t.Error("trailing data must fail")
	}
}

func TestScanAndNestedText(t *testing.T) {
	var s Json
	if err := s.Scan("{\"a\":\"l1\nl2\"}"); err != nil || s.Str("a") != "l1\nl2" {
		t.Errorf("Scan raw newline: %v %v", s, err)
	}
	if err := s.Scan(nil); err != nil || len(s) != 0 {
		t.Errorf("Scan NULL: %v %v", s, err)
	}

	src := Json{
		"obj":  "{\"a\":\"l1\nl2\"}",
		"list": "[1, 2]",
		"bad":  "nope",
		"rows": []any{"{\"a\":1}", 5, "{\"a\":2}"},
	}
	if src.Json("obj").Str("a") != "l1\nl2" {
		t.Errorf("ValJson raw newline: %v", src.Json("obj"))
	}
	if got := src.ValJson(Json{"def": true}, "bad"); !got.Bool("def") {
		t.Errorf("ValJson invalid text gives the default: %v", got)
	}
	if got := src.ArrayInt("list"); len(got) != 2 || got[1] != 2 {
		t.Errorf("Array from JSON text: %v", got)
	}
	if got := src.ArrayJson("rows"); len(got) != 2 || got[1].Int("a") != 2 {
		t.Errorf("ArrayJson skips non objects: %v", got)
	}
}

func TestIntegersKeepNoDecimals(t *testing.T) {
	j, err := ToJson(`{"id": 9007199254740993, "n": 1, "big": 1000000, "neg": -3, "list": [1, 2, 3]}`)
	if err != nil {
		t.Fatal(err)
	}
	if j.Int64("id") != 9007199254740993 || j.Str("id") != "9007199254740993" {
		t.Errorf("big integer: %v %q", j.Int64("id"), j.Str("id"))
	}
	if _, ok := j["n"].(float64); !ok {
		t.Errorf("small numbers stay float64 for compatibility: %T", j["n"])
	}
	for key, want := range map[string]string{"n": "1", "big": "1000000", "neg": "-3"} {
		if got := j.Str(key); got != want {
			t.Errorf("Str(%s) = %q, want %q", key, got, want)
		}
	}
	if got := j.ArrayInt("list"); len(got) != 3 || got[2] != 3 {
		t.Errorf("ArrayInt: %v", got)
	}
	if got := j.ArrayInt64("list"); len(got) != 3 {
		t.Errorf("ArrayInt64: %v", got)
	}
	if got := j.ArrayStr("list"); got[0] != "1" {
		t.Errorf("ArrayStr: %v", got)
	}
	m := Json{"m": j}
	if m.MapStr("m")["big"] != "1000000" || m.MapInt("m")["big"] != 1000000 {
		t.Errorf("MapStr/MapInt: %v %v", m.MapStr("m"), m.MapInt("m"))
	}
	if got := (Json{"a": []any{1, int64(2), 3.5, "4"}}).ArrayNumber("a"); len(got) != 4 {
		t.Errorf("ArrayNumber: %v", got)
	}

	ints := Json{"s": " 2 ", "f": "1.0", "u": uint(5), "i8": int8(-5), "num": json.Number("7"), "frac": "1.5", "x": "abc", "trunc": 1.9}
	for key, want := range map[string]int{"s": 2, "f": 1, "u": 5, "i8": -5, "num": 7, "frac": -1, "x": -1, "trunc": 1} {
		if got := ints.ValInt(-1, key); got != want {
			t.Errorf("ValInt(%s) = %d, want %d", key, got, want)
		}
	}

	nested := Json{"x": `{"n": 5, "f": 2.5}`}.Json("x")
	if nested.Int("n") != 5 || nested.Num("f") != 2.5 || nested.Str("n") != "5" {
		t.Errorf("numbers of a JSON text: %v", nested)
	}

	item := Item{Ok: true, Result: Json{"id": int64(9007199254740993)}}
	if got := item.ToJson().Json("result").Int64("id"); got != 9007199254740993 {
		t.Errorf("Item.ToJson keeps big integers: %d", got)
	}
}

func TestFloatsKeepTheirDecimals(t *testing.T) {
	j, _ := ToJson(`{"a": 3.10, "b": 0.1234567, "c": 1e21, "d": -0.5}`)
	for key, want := range map[string]string{"a": "3.1", "b": "0.1234567", "c": "1000000000000000000000", "d": "-0.5"} {
		if got := j.Str(key); got != want {
			t.Errorf("Str(%s) = %q, want %q", key, got, want)
		}
	}
	if got := (Json{"x": float32(0.1)}).Str("x"); got != "0.1" {
		t.Errorf("float32: %q", got)
	}
	if got := (Json{"x": " 2.5 "}).Num("x"); got != 2.5 {
		t.Errorf("Num of text: %v", got)
	}
}

func TestBool(t *testing.T) {
	j, _ := ToJson(`{"one": 1, "zero": 0, "two": 2}`)
	values := Json{
		"one": j["one"], "zero": j["zero"], "two": j["two"],
		"i32": int32(1), "u8": uint8(0), "num": json.Number("1"),
		"s1": "1", "s0": "0", "T": "TRUE", "sp": " true ", "yes": "yes", "si": "Sí", "off": "off", "x": "maybe",
	}
	want := map[string]int{ // 1 true, 0 false, -1 the default
		"one": 1, "zero": 0, "two": -1, "i32": 1, "u8": 0, "num": 1,
		"s1": 1, "s0": 0, "T": 1, "sp": 1, "yes": 1, "si": 1, "off": 0, "x": -1,
	}
	for key, w := range want {
		gotTrue, gotFalse := values.ValBool(true, key), values.ValBool(false, key)
		got := -1
		if gotTrue && gotFalse {
			got = 1
		} else if !gotTrue && !gotFalse {
			got = 0
		}
		if got != w {
			t.Errorf("ValBool(%s) = %d, want %d", key, got, w)
		}
	}
}

func TestOtherConversions(t *testing.T) {
	var dst struct{ A int }
	if err := (Json{"A": 7}).ToObject(&dst); err != nil || dst.A != 7 {
		t.Errorf("ToObject: %v %d", err, dst.A)
	}

	b64 := Json{"x": "hola"}.ToBase64("x")
	if got := (Json{"x": b64}).FromBase64("x"); got != "hola" {
		t.Errorf("Base64 round trip: %q", got)
	}
	// Encoded by the former ToBase64 (a quoted JSON string).
	if got := (Json{"x": "ImhvbGEi"}).FromBase64("x"); got != "hola" {
		t.Errorf("former Base64: %q", got)
	}

	if bt, _ := (Json{"x": `{"a":1}`}).Byte("x"); string(bt) != `{"a":1}` {
		t.Errorf("Byte of text: %s", bt)
	}
	if bt, _ := (Json{"x": Json{"a": 1}}).Byte("x"); string(bt) != `{"a":1}` {
		t.Errorf("Byte of object: %s", bt)
	}

	for _, text := range []string{"2026-10-21 10:00:00", "2026-10-21", "2026-10-21T10:00:00", "2026-10-21T10:00:00Z"} {
		if got := (Json{"x": text}).ValTime(time.Time{}, "x"); got.Year() != 2026 {
			t.Errorf("ValTime(%q) = %v", text, got)
		}
	}

	if got := (Json{"x": func() {}}).ToEscapeHTML(); got != "" {
		t.Errorf("ToEscapeHTML of a value JSON can't hold: %q", got)
	}
}
