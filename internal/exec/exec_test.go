package exec

import (
	"context"
	"errors"
	"testing"
	"time"

	"fnexec/internal/cold"
	"fnexec/internal/func"
	"fnexec/internal/model"
	"fnexec/internal/queue"
	"fnexec/internal/retry"
	"fnexec/internal/scale"
)

func testExecutor(t *testing.T, bootDelay time.Duration) (*Executor, *funcs.Registry, *queue.Queue, *scale.Scaler, *retry.Manager, *cold.Manager) {
	t.Helper()
	reg := funcs.NewRegistry()
	reg.Register(&model.Function{
		Name:         "demo",
		Timeout:      time.Second,
		MaxRetries:   1,
		MinInstances: 0,
		MaxInstances: 4,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			return payload, nil
		},
	})
	q := queue.New()
	scaler := scale.NewScaler(nil)
	coldMgr := cold.NewManager(reg, cold.NewLocalBooter(bootDelay), scaler)
	scaler.SetOnReclaim(func(id string) { coldMgr.Invalidate(id) })
	retrier := retry.NewManager(retry.Config{Queue: q})
	ex := NewExecutor(Config{
		Queue:          q,
		Cold:           coldMgr,
		Scale:          scaler,
		Retry:          retrier,
		Funcs:          reg,
		BatchSize:      4,
		Workers:        2,
		DefaultTimeout: 500 * time.Millisecond,
		HandleLimit:    16,
	})
	retrier.SetChecker(ex.Results())
	return ex, reg, q, scaler, retrier, coldMgr
}

func TestDispatchSucceeds(t *testing.T) {
	ex, _, _, _, _, _ := testExecutor(t, 0)
	call := model.NewCall("c1", "demo", []byte("ok"), time.Now().Add(time.Second))
	result := ex.Dispatch(context.Background(), call)
	if !result.Succeeded() {
		t.Fatalf("dispatch must succeed, got %+v", result)
	}
	if string(result.Output) != "ok" {
		t.Fatalf("unexpected output %q", result.Output)
	}
}

func TestFinalizeTimeoutRecordsTimeout(t *testing.T) {
	ex, _, _, _, _, _ := testExecutor(t, 0)
	call := model.NewCall("c4", "demo", nil, time.Now().Add(time.Second))
	drain := func() (handlerOutcome, bool) {
		return handlerOutcome{}, false
	}
	result := ex.finalizeTimeout(call, drain)
	if result.Outcome != model.OutcomeTimeout {
		t.Fatalf("finalize must record a timeout, got %+v", result)
	}
}

func TestExecuteTimeoutPropagates(t *testing.T) {
	ex, _, _, _, _, _ := testExecutor(t, 0)
	cancelled := make(chan struct{}, 1)
	ex.funcs.Register(&model.Function{
		Name:    "slow",
		Timeout: 100 * time.Millisecond,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			<-ctx.Done()
			cancelled <- struct{}{}
			return nil, ctx.Err()
		},
	})
	call := model.NewCall("c5", "slow", nil, time.Now().Add(50*time.Millisecond))
	start := time.Now()
	result := ex.Dispatch(context.Background(), call)
	if result.Succeeded() {
		t.Fatalf("slow call must not succeed, got %+v", result)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("timeout must return promptly")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("handler must be cancelled when the call deadline passes")
	}
}

func TestContextCancellationStopsWorkers(t *testing.T) {
	ex, _, q, _, _, _ := testExecutor(t, 0)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		ex.Run(ctx)
		close(done)
	}()
	call := model.NewCall("c6", "demo", nil, time.Now().Add(time.Second))
	if err := q.Enqueue(call); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("workers must stop on cancellation")
	}
}

func TestDispatchUnknownErrorSurfaces(t *testing.T) {
	ex, reg, _, _, _, _ := testExecutor(t, 0)
	reg.Register(&model.Function{
		Name:    "broken",
		Timeout: time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) {
			return nil, errors.New("handler exploded")
		},
	})
	call := model.NewCall("c7", "broken", nil, time.Now().Add(time.Second))
	result := ex.Dispatch(context.Background(), call)
	if result.Succeeded() {
		t.Fatal("broken handler must fail")
	}
	if result.Error != "handler exploded" {
		t.Fatalf("unexpected error text %q", result.Error)
	}
}
