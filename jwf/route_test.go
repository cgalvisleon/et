package jwf

import (
	"testing"

	"github.com/cgalvisleon/et/et"
)

/**
* routeFlow: start → check (condition, two outputs) and only check[0] → yes; check[1] has no connection
**/
func routeFlow(t *testing.T) *Flow {
	script := func(code string) []any {
		return []any{et.Json{"code": code, "language": LANGUAGE_JAVASCRIPT}}
	}
	step := func(id string, kind Kind, outputs int, code string) et.Json {
		return et.Json{"id": id, "kind": kind, "tag": id, "version": "1.0.0", "title": id, "definition": script(code), "params": et.Json{}, "inputs": 1, "outputs": outputs}
	}
	conn := func(id, from string, index int, to string) et.Json {
		return et.Json{
			"id":     id,
			"kind":   PortOutput,
			"source": et.Json{"steper_id": from, "port": PortOutput, "index": index},
			"target": et.Json{"steper_id": to, "port": "input", "index": 0},
		}
	}
	flow, err := LoadFlow(et.Json{
		"id":      "flow:route",
		"tag":     "route",
		"name":    "route",
		"version": "1.0.0",
		"steps": et.Json{
			"start": step("start", KindTrigger, 1, ``),
			"check": step("check", KindCondition, 2, `goTo(ctx.Bool("ok") ? 0 : 1)`),
			"yes":   step("yes", KindAction, 1, `ctx.Set("yes", true)`),
		},
		"connections": []any{conn("c1", "start", 0, "check"), conn("c2", "check", 0, "yes")},
		"triggers":    []any{et.Json{"tag": "start", "start_id": "start"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return flow
}

/**
* TestGoToWithoutConnection: an output chosen with goTo that has no connection ends the run at that step; it does not
* fall back to the connection of output 0
**/
func TestGoToWithoutConnection(t *testing.T) {
	for _, tc := range []struct {
		ok      bool
		wantYes bool
	}{
		{true, true},
		{false, false},
	} {
		flow := routeFlow(t)
		instance := flow.NewInstance("", "", flow.Name, et.Json{}, flow.Triggers[0], "")
		if _, err := instance.Run(et.Json{"ok": tc.ok}, true); err != nil {
			t.Fatalf("ok=%v: %v", tc.ok, err)
		}
		if got := instance.Ctx.Bool("yes"); got != tc.wantYes {
			t.Errorf("ok=%v: yes = %v, want %v", tc.ok, got, tc.wantYes)
		}
		if instance.Status != DONE {
			t.Errorf("ok=%v: status = %v, want %v", tc.ok, instance.Status, DONE)
		}
	}
}
