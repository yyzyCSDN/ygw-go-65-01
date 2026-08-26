package exec

import (
	"context"
	"testing"
	"time"

	"fnexec/internal/cold"
	"fnexec/internal/func"
	"fnexec/internal/model"
	"fnexec/internal/queue"
	"fnexec/internal/retry"
	"fnexec/internal/scale"
)

// TestTimeoutPropagatedToExecutor verifies the call deadline reaches the
// function body so a slow handler is cancelled instead of occupying the
// instance forever.
func TestTimeoutPropagatedToExecutor(t *testing.T) {
	ex, _, _, _, _, _ := newSlowTimeoutExecutor(t)
	cancelled := make(chan struct{}, 1)
	ex.funcs.Register(&model.Function{
		Name:    "hanging",
		Timeout: 100 * time.Millisecond,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			select {
			case <-ctx.Done():
				cancelled <- struct{}{}
				return nil, ctx.Err()
			case <-time.After(10 * time.Second):
				return payload, nil
			}
		},
	})
	call := model.NewCall("hang-1", "hanging", nil, time.Now().Add(50*time.Millisecond))
	start := time.Now()
	result := ex.Dispatch(context.Background(), call)
	if time.Since(start) > 2*time.Second {
		t.Fatal("dispatch must return promptly when the call deadline passes")
	}
	select {
	case <-cancelled:
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("the function body must be cancelled at the call deadline")
	}
	if result.Succeeded() {
		t.Fatalf("hanging call must not succeed, got %+v", result)
	}
}

func newSlowTimeoutExecutor(t *testing.T) (*Executor, *funcs.Registry, *queue.Queue, *scale.Scaler, *retry.Manager, *cold.Manager) {
	t.Helper()
	reg := funcs.NewRegistry()
	reg.Register(&model.Function{
		Name:         "demo",
		Timeout:      time.Second,
		MaxRetries:   0,
		MinInstances: 0,
		MaxInstances: 4,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			return payload, nil
		},
	})
	q := queue.New()
	scaler := scale.NewScaler(nil)
	coldMgr := cold.NewManager(reg, cold.NewLocalBooter(0), scaler)
	scaler.SetOnReclaim(func(id string) { coldMgr.Invalidate(id) })
	retrier := retry.NewManager(retry.Config{Queue: q, MaxRetries: 0})
	ex := NewExecutor(Config{
		Queue:          q,
		Cold:           coldMgr,
		Scale:          scaler,
		Retry:          retrier,
		Funcs:          reg,
		BatchSize:      4,
		Workers:        1,
		DefaultTimeout: 5 * time.Second,
		HandleLimit:    16,
	})
	retrier.SetChecker(ex.Results())
	return ex, reg, q, scaler, retrier, coldMgr
}
