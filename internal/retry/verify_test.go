package retry

import (
	"testing"
	"time"

	"fnexec/internal/model"
	"fnexec/internal/queue"
)

// TestRetryChecksCommittedResult verifies a call whose result is already
// committed is never executed again.
func TestRetryChecksCommittedResult(t *testing.T) {
	q := queue.New()
	m := NewManager(Config{Queue: q, MaxRetries: 2})
	m.SetChecker(&committedChecker{committed: map[string]bool{"c1": true}})
	call := model.NewCall("c1", "demo", nil, time.Now().Add(time.Second))
	if err := m.Schedule(call); err != nil {
		t.Fatal(err)
	}
	if q.Len() != 0 {
		t.Fatalf("committed call must not be re-enqueued, queue length %d", q.Len())
	}
	if m.Retries() != 0 {
		t.Fatalf("committed call must not count as a retry, got %d", m.Retries())
	}
}

type committedChecker struct {
	committed map[string]bool
}

func (c *committedChecker) CommittedSuccess(callID string) bool {
	return c.committed[callID]
}
