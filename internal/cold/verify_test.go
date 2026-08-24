package cold_test

import (
	"context"
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

// TestColdCacheInvalidatedOnReclaim verifies a reclaimed instance is no longer
// handed out by the cold-start cache.
func TestColdCacheInvalidatedOnReclaim(t *testing.T) {
	reg := funcs.NewRegistry()
	var sideEffects atomic.Int64
	reg.Register(&model.Function{
		Name:    "warm",
		Timeout: time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			sideEffects.Add(1)
			return payload, nil
		},
	})
	q := queue.New()
	scaler := scale.NewScaler(nil)
	mgr := cold.NewManager(reg, cold.NewLocalBooter(5*time.Millisecond), scaler)
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
		HandleLimit:    16,
	})
	retrier.SetChecker(ex.Results())

	first := model.NewCall("warm-1", "warm", nil, time.Now().Add(time.Second))
	if r := ex.Dispatch(context.Background(), first); !r.Succeeded() {
		t.Fatalf("first dispatch must succeed: %+v", r)
	}
	inst, ok := mgr.Cached("warm")
	if !ok {
		t.Fatal("warm instance must be cached")
	}
	if err := scaler.Reclaim(context.Background(), inst.ID); err != nil {
		t.Fatal(err)
	}
	second := model.NewCall("warm-2", "warm", nil, time.Now().Add(time.Second))
	if r := ex.Dispatch(context.Background(), second); !r.Succeeded() {
		t.Fatalf("dispatch after reclaim must succeed, got %+v", r)
	}
	fresh, ok := mgr.Cached("warm")
	if !ok || fresh.ID == inst.ID {
		t.Fatal("cache must hold a fresh instance after reclaim")
	}
	if sideEffects.Load() != 2 {
		t.Fatalf("expected exactly two executions, got %d", sideEffects.Load())
	}
}
