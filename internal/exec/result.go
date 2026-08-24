package exec

import (
	"sync"

	"fnexec/internal/model"
)

// ResultStore keeps the outcome of every finished call.
type ResultStore struct {
	mu      sync.Mutex
	results map[string]*model.Result
}

// NewResultStore creates an empty result store.
func NewResultStore() *ResultStore {
	return &ResultStore{results: make(map[string]*model.Result)}
}

// Commit stores a call outcome, replacing any earlier outcome.
func (s *ResultStore) Commit(result *model.Result) {
	if result == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.results[result.CallID] = result
}

// Get returns the outcome of a call, or nil when unknown.
func (s *ResultStore) Get(callID string) *model.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.results[callID]
}

// Committed reports whether any outcome is stored for the call.
func (s *ResultStore) Committed(callID string) bool {
	return s.Get(callID) != nil
}

// CommittedSuccess reports whether the stored outcome is a success.
func (s *ResultStore) CommittedSuccess(callID string) bool {
	result := s.Get(callID)
	return result != nil && result.Outcome == model.OutcomeSuccess
}

// Count returns the number of stored outcomes.
func (s *ResultStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.results)
}

// Snapshot returns all stored outcomes in a stable order.
func (s *ResultStore) Snapshot() []*model.Result {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*model.Result, 0, len(s.results))
	for _, result := range s.results {
		out = append(out, result)
	}
	return out
}
