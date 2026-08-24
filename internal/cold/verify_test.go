package cold_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fnexec/internal/cold"
	"fnexec/internal/exec"
	"fnexec/internal/func"
	"fnexec/internal/model"
	"fnexec/internal/queue"
	"fnexec/internal/retry"
	"fnexec/internal/scale"
)

// TestColdStartNoDuplicateExecution verifies one call is executed exactly once
// even when several dispatchers race through the cold-start path.
func TestColdStartNoDuplicateExecution(t *testing.T) {
	reg := funcs.NewRegistry()
	var sideEffects atomic.Int64
	reg.Register(&model.Function{
		Name:    "once",
		Timeout: time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			sideEffects.Add(1)
			time.Sleep(30 * time.Millisecond)
			return payload, nil
		},
	})
	q := queue.New()
	scaler := scale.NewScaler(nil)
	mgr := cold.NewManager(reg, cold.NewLocalBooter(60*time.Millisecond), scaler)
	scaler.SetOnReclaim(func(id string) { mgr.Invalidate(id) })
	retrier := retry.NewManager(retry.Config{Queue: q, MaxRetries: 0})
	ex := exec.NewExecutor(exec.Config{
		Queue:          q,
		Cold:           mgr,
		Scale:          scaler,
		Retry:          retrier,
		Funcs:          reg,
		BatchSize:      4,
		Workers:        1,
		DefaultTimeout: time.Second,
		HandleLimit:    32,
	})
	retrier.SetChecker(ex.Results())

	call := model.NewCall("cold-dup-1", "once", []byte("x"), time.Now().Add(time.Second))
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ex.Dispatch(context.Background(), call)
		}()
	}
	wg.Wait()
	if got := sideEffects.Load(); got != 1 {
		t.Fatalf("the same call must execute exactly once, executed %d times", got)
	}
}
