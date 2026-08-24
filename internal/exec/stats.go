package exec

import "sync/atomic"

// StatsSnapshot is a serializable view of executor counters.
type StatsSnapshot struct {
	Executions  int64 `json:"executions"`
	Successes   int64 `json:"successes"`
	Failures    int64 `json:"failures"`
	Timeouts    int64 `json:"timeouts"`
	Claims      int   `json:"claims"`
	Results     int   `json:"results"`
	Handles     int   `json:"handles"`
	BatchSize   int   `json:"batch_size"`
	Workers     int   `json:"workers"`
	HandleLimit int   `json:"handle_limit"`
}

// Stats holds the executor's atomic counters.
type Stats struct {
	executions atomic.Int64
	successes  atomic.Int64
	failures   atomic.Int64
	timeouts   atomic.Int64
}

// RecordSuccess increments the success counter.
func (s *Stats) RecordSuccess() {
	s.executions.Add(1)
	s.successes.Add(1)
}

// RecordFailure increments the failure counter.
func (s *Stats) RecordFailure() {
	s.executions.Add(1)
	s.failures.Add(1)
}

// RecordTimeout increments the timeout counter.
func (s *Stats) RecordTimeout() {
	s.executions.Add(1)
	s.timeouts.Add(1)
}

// Snapshot returns the current counter values.
func (s *Stats) Snapshot() (int64, int64, int64, int64) {
	return s.executions.Load(), s.successes.Load(), s.failures.Load(), s.timeouts.Load()
}
