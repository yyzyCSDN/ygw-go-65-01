package exec

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"fnexec/internal/model"
)

// TestDispatchSkipsRetryWhenAlreadyCommitted reproduces the production bug: a
// network blip triggers a retry after a call already finished successfully.
// Before the fix the retried call was executed again, replaying its side
// effects downstream. After the fix the committed result short-circuits both
// the retry decision and the dispatch, so the handler runs exactly once.
func TestDispatchSkipsRetryWhenAlreadyCommitted(t *testing.T) {
	ex, reg, q, _, _, _ := testExecutor(t, 0)

	var runs atomic.Int64
	reg.Register(&model.Function{
		Name:    "once",
		Timeout: time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			runs.Add(1)
			return payload, nil
		},
	})

	call := model.NewCall("c-retry", "once", []byte("ok"), time.Now().Add(time.Second))

	// First dispatch succeeds and commits a successful outcome.
	first := ex.Dispatch(context.Background(), call)
	if !first.Succeeded() {
		t.Fatalf("first dispatch must succeed, got %+v", first)
	}
	if runs.Load() != 1 {
		t.Fatalf("handler must run once after first dispatch, got %d", runs.Load())
	}

	// Simulate a retry triggered by a network blip: the same call is re-enqueued
	// and dispatched again. It must NOT execute the handler a second time.
	if err := q.Enqueue(call); err != nil {
		t.Fatal(err)
	}
	second := ex.Dispatch(context.Background(), call)
	if !second.Succeeded() {
		t.Fatalf("retried dispatch must still report success, got %+v", second)
	}
	if runs.Load() != 1 {
		t.Fatalf("handler must NOT run again after committed success, got %d", runs.Load())
	}
}

// TestScheduleSkipsCallWithCommittedSuccess verifies the retry manager itself
// refuses to re-enqueue a call whose success was already committed, so the
// retry path never puts duplicate work back on the queue.
func TestScheduleSkipsCallWithCommittedSuccess(t *testing.T) {
	ex, _, q, _, retrier, _ := testExecutor(t, 0)

	// Commit a successful result directly, as a finished call would.
	ex.Results().Commit(&model.Result{
		CallID:     "c-committed",
		FuncName:   "demo",
		Outcome:    model.OutcomeSuccess,
		Output:     []byte("done"),
		Attempt:    1,
		FinishedAt: time.Now(),
	})

	call := model.NewCall("c-committed", "demo", nil, time.Now().Add(time.Second))
	if err := retrier.Schedule(call); err != nil {
		t.Fatalf("schedule must not error on committed success, got %v", err)
	}
	if q.Len() != 0 {
		t.Fatalf("committed-success call must not be re-enqueued, got queue len %d", q.Len())
	}
	if call.Status == model.StatusRetrying {
		t.Fatal("committed-success call must not be marked retrying")
	}
}
