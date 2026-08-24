package queue

import (
	"testing"
	"time"

	"fnexec/internal/model"
)

func TestEnqueueLenPeek(t *testing.T) {
	q := New()
	call := model.NewCall("c1", "echo", []byte("hi"), time.Now().Add(time.Second))
	if err := q.Enqueue(call); err != nil {
		t.Fatal(err)
	}
	if q.Len() != 1 {
		t.Fatalf("expected length 1, got %d", q.Len())
	}
	if got := q.Peek(); got == nil || got.ID != "c1" {
		t.Fatalf("peek must return the first call")
	}
	if q.Len() != 1 {
		t.Fatalf("peek must not remove the call, got len %d", q.Len())
	}
}

func TestDequeueFullBatches(t *testing.T) {
	q := New()
	for i := 1; i <= 6; i++ {
		call := model.NewCall("c"+string(rune('0'+i)), "echo", []byte("x"), time.Now().Add(time.Second))
		if err := q.Enqueue(call); err != nil {
			t.Fatal(err)
		}
	}
	first := q.DequeueBatch(3)
	if len(first.Calls) != 3 || !first.More {
		t.Fatalf("first batch must have 3 calls and more=true, got %d/%v", len(first.Calls), first.More)
	}
	second := q.DequeueBatch(3)
	if len(second.Calls) != 3 || second.More {
		t.Fatalf("second batch must have 3 calls and more=false, got %d/%v", len(second.Calls), second.More)
	}
	if q.Len() != 0 {
		t.Fatalf("queue must be drained, got %d", q.Len())
	}
}

func TestRetryPutsCallFirst(t *testing.T) {
	q := New()
	call := model.NewCall("c1", "echo", nil, time.Now().Add(time.Second))
	if err := q.Enqueue(call); err != nil {
		t.Fatal(err)
	}
	retried := model.NewCall("c1", "echo", nil, time.Now().Add(time.Second))
	if err := q.Retry(retried); err != nil {
		t.Fatal(err)
	}
	if got := q.Peek(); got == nil || got.ID != "c1" {
		t.Fatal("retried call must be at the head")
	}
}

func TestCloseRejectsEnqueue(t *testing.T) {
	q := New()
	call := model.NewCall("c1", "echo", nil, time.Now().Add(time.Second))
	if err := q.Close(); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(call); err != ErrClosed {
		t.Fatalf("enqueue after close must fail with ErrClosed, got %v", err)
	}
}
