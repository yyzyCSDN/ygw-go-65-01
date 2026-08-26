package funcs_test

import (
	"context"
	"testing"

	"fnexec/internal/func"
	"fnexec/internal/model"
	"fnexec/internal/queue"
	"fnexec/internal/trigger"
)

// TestMissingFuncNoNilPanic verifies triggering an unregistered function
// returns a clear error instead of panicking.
func TestMissingFuncNoNilPanic(t *testing.T) {
	reg := funcs.NewRegistry()
	q := queue.New()
	tr := trigger.New(q, reg, trigger.NewThrottle(100))
	ev := model.NewEvent("missing-1", "ghost", nil)
	call, err := tr.Handle(context.Background(), ev)
	if err == nil {
		t.Fatalf("unregistered function must be rejected, got call %+v", call)
	}
	if q.Len() != 0 {
		t.Fatalf("no call may be enqueued for an unregistered function, got %d", q.Len())
	}
	if fn, err := reg.Lookup("ghost"); err == nil || fn != nil {
		t.Fatalf("lookup of an unregistered function must fail, got fn=%v err=%v", fn, err)
	}
}
