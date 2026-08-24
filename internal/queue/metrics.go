package queue

// Snapshot is a serializable view of the queue state.
type Snapshot struct {
	Pending int `json:"pending"`
	Head    int `json:"head"`
	Total   int `json:"total"`
}

// Snapshot returns queue metrics for the console and stats endpoint.
func (q *Queue) Snapshot() Snapshot {
	q.mu.Lock()
	defer q.mu.Unlock()
	return Snapshot{
		Pending: q.lenLocked(),
		Head:    q.head,
		Total:   len(q.items),
	}
}

// PendingByFunc counts waiting calls grouped by function name.
func (q *Queue) PendingByFunc() map[string]int {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make(map[string]int)
	for _, call := range q.items[q.head:] {
		out[call.FuncName]++
	}
	return out
}
