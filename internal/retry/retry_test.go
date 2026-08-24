package retry

import (
	"testing"
	"time"

	"fnexec/internal/model"
	"fnexec/internal/queue"
)

func TestScheduleRequeuesFailedCall(t *testing.T) {
	q := queue.New()
	m := NewManager(Config{Queue: q, MaxRetries: 2})
	m.SetChecker(&fakeChecker{})
	call := model.NewCall("c2", "demo", nil, time.Now().Add(time.Second))
	if err := m.Schedule(call); err != nil {
		t.Fatal(err)
	}
	if q.Len() != 1 {
		t.Fatalf("failed call must be re-enqueued, got %d", q.Len())
	}
	if call.Attempt != 2 || call.Status != model.StatusRetrying {
		t.Fatalf("call must be marked retrying, got %d/%s", call.Attempt, call.Status)
	}
}

func TestScheduleRespectsBudget(t *testing.T) {
	q := queue.New()
	m := NewManager(Config{Queue: q, MaxRetries: 1})
	m.SetChecker(&fakeChecker{})
	call := model.NewCall("c3", "demo", nil, time.Now().Add(time.Second))
	if err := m.Schedule(call); err != nil {
		t.Fatal(err)
	}
	if err := m.Schedule(call); err != ErrMaxRetries {
		t.Fatalf("second schedule must exceed budget, got %v", err)
	}
}

type fakeChecker struct {
	success map[string]bool
}

func (f *fakeChecker) CommittedSuccess(callID string) bool {
	return f != nil && f.success != nil && f.success[callID]
}
