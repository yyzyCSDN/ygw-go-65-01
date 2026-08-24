package exec

import (
	"sync"
	"testing"
	"time"

	"fnexec/internal/model"
)

// TestTimeoutNoRetryCompletedCall verifies a result that arrives at the
// deadline boundary is not re-classified as a timeout and not retried.
func TestTimeoutNoRetryCompletedCall(t *testing.T) {
	ex, _, _, _, retrier, _ := testExecutor(t, 0)
	call := model.NewCall("race-1", "demo", []byte("done"), time.Now().Add(time.Second))
	ex.results.Commit(&model.Result{
		CallID:     call.ID,
		FuncName:   call.FuncName,
		Outcome:    model.OutcomeSuccess,
		Output:     []byte("done"),
		Attempt:    call.Attempt,
		FinishedAt: time.Now(),
	})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := ex.finalizeTimeout(call, func() (handlerOutcome, bool) {
			return handlerOutcome{out: []byte("done"), err: nil}, true
		})
		if !result.Succeeded() {
			t.Errorf("completed call must keep its success, got %+v", result)
		}
	}()
	wg.Wait()
	if len(retrier.Scheduled()) != 0 {
		t.Fatal("completed call must not be scheduled for retry")
	}
	got := ex.results.Get(call.ID)
	if got == nil || got.Outcome != model.OutcomeSuccess {
		t.Fatalf("stored outcome must remain success, got %+v", got)
	}
}
