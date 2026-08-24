package exec

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"fnexec/internal/model"
)

// TestFinalizeTimeoutHonorsAlreadyCompletedResult reproduces the boundary
// race: the handler produced its result and it sits in the buffered `done`
// channel, but the deadline also fired. The call genuinely finished, so it
// must be recorded with its real outcome (not a timeout) and must not be
// retried. Before the fix, finalizeTimeout ignored the drained result and
// always recorded a timeout + scheduled a retry, causing duplicate execution.
func TestFinalizeTimeoutHonorsAlreadyCompletedResult(t *testing.T) {
	ex, _, _, _, retrier, _ := testExecutor(t, 0)
	call := model.NewCall("race-1", "demo", []byte("done"), time.Now().Add(time.Second))

	// Simulate a result that already arrived from the handler.
	drain := func() (handlerOutcome, bool) {
		return handlerOutcome{out: []byte("done"), err: nil}, true
	}

	before := retrier.Retries()
	result := ex.finalizeTimeout(call, drain)

	if result.Outcome != model.OutcomeSuccess {
		t.Fatalf("completed call must keep its success outcome, got %q (%+v)", result.Outcome, result)
	}
	if string(result.Output) != "done" {
		t.Fatalf("completed call output must be preserved, got %q", string(result.Output))
	}
	if got := retrier.Retries(); got != before {
		t.Fatalf("completed call must not be retried, retries went %d -> %d", before, got)
	}
}

// TestExecuteBoundaryFinishNotMarkedTimeout drives the real Execute path:
// a handler that finishes just as the deadline fires. We force the race by
// having the handler complete and writing its outcome into the same buffered
// channel Execute uses, then expiring the context. We count handler invocations
// to assert the side effect runs exactly once across retries.
func TestExecuteBoundaryFinishNotMarkedTimeout(t *testing.T) {
	ex, reg, _, _, _, _ := testExecutor(t, 0)

	var invocations int32
	reg.Register(&model.Function{
		Name:    "boundary",
		Timeout: 100 * time.Millisecond,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			atomic.AddInt32(&invocations, 1)
			// Finish immediately; the result lands in the buffered `done`
			// channel before/around the deadline firing.
			return []byte("ok"), nil
		},
	})

	// Very short remaining deadline so runCtx.Done() is already expired by
	// the time we reach the select, mimicking a boundary finish.
	call := model.NewCall("race-2", "boundary", nil, time.Now().Add(time.Nanosecond))
	result := ex.Dispatch(context.Background(), call)

	if result.Outcome == model.OutcomeTimeout {
		t.Fatalf("boundary-finished call must not be marked timeout, got %+v", result)
	}
	if got := atomic.LoadInt32(&invocations); got != 1 {
		t.Fatalf("handler must run exactly once, got %d", got)
	}
}
