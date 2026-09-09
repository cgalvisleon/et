package jwf

import (
	"fmt"
	"sync"
	"testing"

	"github.com/cgalvisleon/et/et"
)

// TestFlowGetErrorReturnsErrorHandlerStep guards against the getError regression
// where the method returned the step that failed instead of the step wired via
// Flow.Error(...), which made error branches never actually run.
func TestFlowGetErrorReturnsErrorHandlerStep(t *testing.T) {
	flow := &Flow{
		Steps:       make(map[string]*Step),
		Connections: make([]*Connection, 0),
	}
	stepA := &Step{ID: "a"}
	stepB := &Step{ID: "b"}
	flow.Steps[stepA.ID] = stepA
	flow.Steps[stepB.ID] = stepB

	if _, ok := flow.addConnection(stepA.ID, stepB.ID, 0, PortError); !ok {
		t.Fatal("expected the error connection to be created")
	}

	got, exists := flow.getError(stepA.ID, 0)
	if !exists {
		t.Fatal("expected getError to find the error handler step")
	}
	if got.ID != stepB.ID {
		t.Fatalf("getError returned step %q, want the error-handler step %q", got.ID, stepB.ID)
	}
}

// TestStepRunOnPublishNilCallbackDoesNotPanic guards against runOnPublish's
// inverted nil-check, which called a nil Go callback (s.onPublish) whenever no
// callback was registered — i.e. always, since nothing in the package ever sets
// it — making every Flow.Publish() panic.
func TestStepRunOnPublishNilCallbackDoesNotPanic(t *testing.T) {
	wf := &WorkFlow{bindings: make(map[string]any)}
	flow := &Flow{workflow: wf}
	step := &Step{Params: et.Json{}, OnPublish: "true;"}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("runOnPublish panicked: %v", r)
		}
	}()

	if _, err := step.runOnPublish(flow, et.Json{}); err != nil {
		t.Fatalf("runOnPublish returned an unexpected error: %v", err)
	}
}

// TestWorkflowAddStepGetStepConcurrent guards against addStep locking the wrong
// mutex (muFlows instead of muSteps), which left WorkFlow.Steps unsynchronized
// against concurrent getStep/removeStep. Run with -race.
func TestWorkflowAddStepGetStepConcurrent(t *testing.T) {
	wf := &WorkFlow{Steps: make(map[string]*Step)}

	var wg sync.WaitGroup
	const n = 100
	for i := range n {
		id := fmt.Sprintf("step-%d", i)
		wg.Add(2)
		go func() {
			defer wg.Done()
			wf.addStep(&Step{ID: id})
		}()
		go func() {
			defer wg.Done()
			wf.getStep(id)
		}()
	}
	wg.Wait()
}
