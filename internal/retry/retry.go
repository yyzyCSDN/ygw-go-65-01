package retry

import (
	"errors"
	"sync"
	"sync/atomic"

	"fnexec/internal/model"
	"fnexec/internal/queue"
)

// ErrMaxRetries is returned when a call exceeds its retry budget.
var ErrMaxRetries = errors.New("max retries exceeded")

// SuccessChecker reports whether a call already finished successfully.
type SuccessChecker interface {
	CommittedSuccess(callID string) bool
}

// Config wires the retry manager.
type Config struct {
	Queue      *queue.Queue
	MaxRetries int
}

// Manager decides whether and how failed calls are retried.
type Manager struct {
	mu         sync.Mutex
	queue      *queue.Queue
	checker    SuccessChecker
	maxRetries int
	scheduled  []string
	retries    atomic.Int64
}

// NewManager creates a retry manager.
func NewManager(cfg Config) *Manager {
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 2
	}
	return &Manager{queue: cfg.Queue, maxRetries: cfg.MaxRetries}
}

// SetChecker installs the committed-result checker.
func (m *Manager) SetChecker(checker SuccessChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checker = checker
}

// Schedule re-enqueues a call for another attempt when allowed.
func (m *Manager) Schedule(call *model.Call) error {
	m.mu.Lock()
	checker := m.checker
	m.mu.Unlock()
	if checker != nil && checker.CommittedSuccess(call.ID) {
		return nil
	}
	if call.Attempt > m.maxRetries {
		return ErrMaxRetries
	}
	call.Attempt++
	call.Status = model.StatusRetrying
	if err := m.queue.Enqueue(call); err != nil {
		return err
	}
	m.mu.Lock()
	m.scheduled = append(m.scheduled, call.ID)
	m.mu.Unlock()
	m.retries.Add(1)
	return nil
}

// ShouldRetry reports whether a failed call may be attempted again.
func (m *Manager) ShouldRetry(call *model.Call, result *model.Result) bool {
	if result == nil || result.Succeeded() {
		return false
	}
	if call == nil || call.Attempt > m.maxRetries {
		return false
	}
	return true
}

// Retries returns the number of retries scheduled so far.
func (m *Manager) Retries() int64 {
	return m.retries.Load()
}

// Scheduled returns the call IDs currently scheduled for retry.
func (m *Manager) Scheduled() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.scheduled))
	copy(out, m.scheduled)
	return out
}
