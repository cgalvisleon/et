package jwf

import (
	"testing"

	"github.com/cgalvisleon/et/et"
)

/**
* TestRun: Run with an et.Json binding changed in place and reassigned, ctx of RunCode, the names of before (str, set,
* omit) and the standard bindings
**/
func TestRun(t *testing.T) {
	record := et.Json{"name": "Ana", "user": et.Json{"email": "a@x.com"}}
	code := `
		NEW.code = "C1";
		const email = NEW.user.Str("email") + "|" + NEW.str("name");
		NEW.set({city: "Bogotá"});
		NEW.omit("name");
		OLD = {replaced: true};
		email + "|" + typeof UUID();
	`
	result, globals, err := Run(code, Bindings(map[string]any{"NEW": record, "OLD": et.Json{}}), "OLD", "NEW")
	if err != nil {
		t.Fatal(err)
	}
	if result != "a@x.com|Ana|string" {
		t.Errorf("result = %v", result)
	}
	if record.Str("code") != "C1" || record.Str("city") != "Bogotá" || record.IsExist("name") {
		t.Errorf("NEW not changed in place: %v", record)
	}
	if !globals.Json("OLD").Bool("replaced") {
		t.Errorf("OLD reassigned not read back: %v", globals)
	}

	value, err := RunCode(`({total: ctx.Int("a") + ctx.b})`, et.Json{"a": 2, "b": 3}, nil)
	if err != nil {
		t.Fatal(err)
	}
	object, ok := value.(et.Json)
	if !ok || object.Int("total") != 5 {
		t.Errorf("RunCode = %v (%T)", value, value)
	}

	// if:condition: a list of conditions (WheresField) bound with ctx, in order
	ifCode := `
		const wheres = bind(params.condition, ctx);
		evaluateObject(ctx, wheres);
	`
	condition := []any{et.Json{"age": et.Json{"more_eq": "{{min}}"}}, et.Json{"and": et.Json{"status": et.Json{"eq": "active"}}}}
	for _, tc := range []struct {
		ctx  et.Json
		want bool
	}{
		{et.Json{"min": 18, "age": 20, "status": "active"}, true},
		{et.Json{"min": 18, "age": 20, "status": "archived"}, false},
		{et.Json{"min": 18, "age": 10, "status": "active"}, false},
	} {
		bindings := Bindings(map[string]any{"params": et.Json{"condition": condition}, "bind": bindLike})
		got, err := RunCode(ifCode, tc.ctx, bindings)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("evaluateObject(%v) = %v, want %v", tc.ctx, got, tc.want)
		}
	}
}

// bindLike: {{key}} of ctx in the string values of a list of conditions (bind of jsql, which jwf cannot import)
func bindLike(source any, vars et.Json) any {
	var walk func(value any) any
	walk = func(value any) any {
		switch v := value.(type) {
		case et.Json:
			result := et.Json{}
			for key, item := range v {
				result[key] = walk(item)
			}
			return result
		case []any:
			result := make([]any, len(v))
			for i, item := range v {
				result[i] = walk(item)
			}
			return result
		case string:
			if len(v) > 4 && v[:2] == "{{" && v[len(v)-2:] == "}}" {
				return vars[v[2:len(v)-2]]
			}
		}
		return value
	}
	return walk(source)
}
