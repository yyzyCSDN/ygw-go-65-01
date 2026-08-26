package queue

import (
	"errors"
	"sync"

	"fnexec/internal/model"
)

// ErrClosed is returned when a call is enqueued after the queue was closed.
var ErrClosed = errors.New("queue is closed")

// Queue holds pending calls in arrival order.
type Queue struct {
	mu     sync.Mutex
	items  []*model.Call
	head   int
	notify chan struct{}
	closed bool
}

// New creates an empty call queue.
func New() *Queue {
	return &Queue{notify: make(chan struct{}, 1)}
}

// Enqueue appends a call and wakes a waiting dispatcher.
func (q *Queue) Enqueue(call *model.Call) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	q.items = append(q.items, call)
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return nil
}

// Len returns the number of calls still waiting to be dequeued.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.lenLocked()
}

func (q *Queue) lenLocked() int {
	remaining := len(q.items) - q.head
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Notify returns the channel signaled when calls arrive.
func (q *Queue) Notify() <-chan struct{} {
	return q.notify
}

// Close permanently rejects new calls.
func (q *Queue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.closed = true
	select {
	case q.notify <- struct{}{}:
	default:
	}
	return nil
}

// Peek returns the oldest waiting call without removing it.
func (q *Queue) Peek() *model.Call {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.head >= len(q.items) {
		return nil
	}
	return q.items[q.head]
}

func (q *Queue) compactLocked() {
	if q.head >= len(q.items) {
		q.items = q.items[:0]
		q.head = 0
	}
}
