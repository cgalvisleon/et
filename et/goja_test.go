package et

import (
	"testing"

	"github.com/dop251/goja"
)

/**
* TestJsonGoja: A Json goes to a script and back as the same map: keys as properties, methods with their Go names,
* changes made in JavaScript seen in Go, and objects made in JavaScript read as Json
**/
func TestJsonGoja(t *testing.T) {
	vm := goja.New()
	data := Json{"name": "Ana", "user": Json{"email": "a@x.com", "tags": []any{"x"}}, "items": []Json{{"n": 1}, {"n": 2}}}
	vm.Set("data", ToJS(vm, data))
	vm.Set("take", func(call goja.FunctionCall) goja.Value {
		got, ok := JsonOf(call.Argument(0))
		if !ok {
			t.Fatal("take: not a Json")
		}
		return ToJS(vm, got.ValStr("", "a", "b"))
	})

	cases := map[string]any{
		`data.name`:                        "Ana",
		`data.Str("name")`:                 "Ana",
		`data.ValStr("", "user", "email")`: "a@x.com",
		`data.user.Str("email")`:           "a@x.com",
		`data.user === data.user`:          true,
		`data.items[1].Int("n")`:           int64(2),
		`data.ArrayJson("items").length`:   int64(2),
		`Object.keys(data).join(",")`:      "items,name,user",
		`JSON.stringify(data.user.tags)`:   `["x"]`,
		`take({a: {b: "deep"}})`:           "deep",
		`take(data.user) === ""`:           true,
		`(() => { try { data.Str.call(5, "x"); return "no" } catch (e) { return "throw" } })()`: "throw",
	}
	for script, want := range cases {
		got, err := vm.RunString(script)
		if err != nil {
			t.Fatalf("%s: %v", script, err)
		}
		if got.Export() != want {
			t.Errorf("%s = %v (%T), want %v", script, got.Export(), got.Export(), want)
		}
	}

	if _, err := vm.RunString(`data.city = "Bogotá"; data.user.phone = {code: 57}; data.Set("age", 30); delete data.items`); err != nil {
		t.Fatal(err)
	}
	if data.Str("city") != "Bogotá" || data.Int("age") != 30 || data.IsExist("items") {
		t.Errorf("changes in JS not in Go: %v", data)
	}
	if data.ValInt(0, "user", "phone", "code") != 57 {
		t.Errorf("nested object from JS: %v", data.Json("user"))
	}

	// goja exporting as is (a Go func with a Json param): the nested object is read by Json too
	var got Json
	vm.Set("plain", func(value Json) { got = value })
	if _, err := vm.RunString(`plain({wrap: data.user})`); err != nil {
		t.Fatal(err)
	}
	if got.ValStr("", "wrap", "email") != "a@x.com" || got.Json("wrap").Str("email") != "a@x.com" {
		t.Errorf("exported as is: %v", got)
	}
}
