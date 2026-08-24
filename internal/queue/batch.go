package queue

import "fnexec/internal/model"

// Batch is a group of calls handed to one dispatcher pass.
type Batch struct {
	Calls []*model.Call
	More  bool
}

// DequeueBatch removes up to n calls from the head of the queue.
// More reports whether the queue still holds calls after this batch,
// including calls enqueued after the batch was cut.
func (q *Queue) DequeueBatch(n int) Batch {
	q.mu.Lock()
	defer q.mu.Unlock()
	if n <= 0 {
		n = 1
	}
	if q.head >= len(q.items) {
		q.compactLocked()
		return Batch{More: false}
	}
	end := q.head + n
	if end > len(q.items) {
		end = len(q.items)
	}
	batch := make([]*model.Call, 0, end-q.head)
	batch = append(batch, q.items[q.head:end]...)
	// Advance head to end, the clamped tail of the slice we actually took.
	// Advancing by n instead would overshoot len(q.items) whenever the batch
	// was shorter than n (the queue length is not a multiple of the batch
	// size), stranding calls after head and flipping More to false early.
	q.head = end
	more := q.head < len(q.items)
	return Batch{Calls: batch, More: more}
}

// Retry puts a call back at the head of the queue for another attempt.
func (q *Queue) Retry(call *model.Call) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	q.items = append([]*model.Call{call}, q.items[q.head:]...)
	q.head = 0
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return nil
}
