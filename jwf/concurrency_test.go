package jwf

import (
	"fmt"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/et"
)

/**
* TestConcurrentInstances: the steps of a flow are shared by its runs; each run must see its own bindings (UserId) and
* its own ctx, also when many run at once
**/
func TestConcurrentInstances(t *testing.T) {
	step := func(id string, kind Kind, code string) et.Json {
		definition := []any{et.Json{"code": code, "language": LANGUAGE_JAVASCRIPT}}
		return et.Json{"id": id, "kind": kind, "tag": id, "version": "1.0.0", "title": id, "definition": definition, "params": et.Json{}, "inputs": 1, "outputs": 1}
	}
	flow, err := LoadFlow(et.Json{
		"id":      "flow:concurrent",
		"tag":     "concurrent",
		"name":    "concurrent",
		"version": "1.0.0",
		"steps": et.Json{
			"start": step("start", KindTrigger, ``),
			// Lee el binding de su ejecución, espera (otras escriben entretanto) y lo vuelve a leer
			"who": step("who", KindAction, `
				const first = UserId;
				let n = 0;
				for (let i = 0; i < 20000; i++) { n += i; }
				ctx.Set("first", first);
				ctx.Set("second", UserId);`),
		},
		"connections": []any{et.Json{
			"id":     "c1",
			"kind":   PortOutput,
			"source": et.Json{"steper_id": "start", "port": PortOutput, "index": 0},
			"target": et.Json{"steper_id": "who", "port": "input", "index": 0},
		}},
		"triggers": []any{et.Json{"tag": "start", "start_id": "start"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	const runs = 50
	var wg sync.WaitGroup
	errs := make(chan string, runs)
	for i := range runs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			user := fmt.Sprintf("user:%d", i)
			instance := flow.NewInstance("", "", flow.Name, et.Json{}, flow.Triggers[0], user)
			instance.SetBinding("UserId", user)
			if _, err := instance.Run(et.Json{"run": i}, true); err != nil {
				errs <- fmt.Sprintf("run %d: %v", i, err)
				return
			}
			if got := instance.Ctx.Str("first"); got != user {
				errs <- fmt.Sprintf("run %d: first = %s, want %s", i, got, user)
			}
			if got := instance.Ctx.Str("second"); got != user {
				errs <- fmt.Sprintf("run %d: second = %s, want %s", i, got, user)
			}
			if got := instance.Ctx.Int("run"); got != i {
				errs <- fmt.Sprintf("run %d: ctx.run = %d", i, got)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for msg := range errs {
		t.Error(msg)
	}
}
