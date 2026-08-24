package trigger

import (
	"context"
	"testing"
	"time"

	"fnexec/internal/func"
	"fnexec/internal/model"
	"fnexec/internal/queue"
)

func TestHandleEnqueuesCall(t *testing.T) {
	reg := funcs.NewRegistry()
	if err := funcs.RegisterBuiltins(reg); err != nil {
		t.Fatal(err)
	}
	q := queue.New()
	tr := New(q, reg, NewThrottle(100))
	ev := model.NewEvent("ev-1", "echo", []byte(`{"text":"hi"}`))
	call, err := tr.Handle(context.Background(), ev)
	if err != nil {
		t.Fatal(err)
	}
	if call.ID == "" || call.FuncName != "echo" {
		t.Fatalf("unexpected call: %+v", call)
	}
	if q.Len() != 1 {
		t.Fatalf("call must be enqueued, got %d", q.Len())
	}
}

func TestHandleThrottled(t *testing.T) {
	reg := funcs.NewRegistry()
	if err := funcs.RegisterBuiltins(reg); err != nil {
		t.Fatal(err)
	}
	q := queue.New()
	tr := New(q, reg, NewThrottle(1))
	ev := model.NewEvent("ev-3", "echo", nil)
	tr.Handle(context.Background(), ev)
	tr.Handle(context.Background(), ev)
	if _, err := tr.Handle(context.Background(), ev); err != ErrThrottled {
		t.Fatalf("third call in the same second must be throttled, got %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := tr.Handle(context.Background(), ev); err != nil {
		t.Fatalf("throttle must refill, got %v", err)
	}
}
