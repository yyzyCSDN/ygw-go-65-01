package queue

import (
	"testing"
	"time"

	"fnexec/internal/model"
)

// TestDequeuePartialBatch covers the non-multiple boundary case: a queue
// length that is not divisible by the batch size. The final batch must take
// only the remaining calls, report More=false, and leave the queue drained
// with every call dequeued exactly once.
func TestDequeuePartialBatch(t *testing.T) {
	const total = 5
	const batch = 3
	q := New()
	want := map[string]bool{}
	for i := 0; i < total; i++ {
		id := "c" + string(rune('0'+i))
		want[id] = true
		call := model.NewCall(id, "echo", []byte("x"), time.Now().Add(time.Second))
		if err := q.Enqueue(call); err != nil {
			t.Fatal(err)
		}
	}

	seen := map[string]bool{}
	var batches [][]*model.Call

	// 5 calls, batch 3 -> first batch 3 (More=true), second batch 2 (More=false).
	first := q.DequeueBatch(batch)
	if len(first.Calls) != 3 || !first.More {
		t.Fatalf("first batch must have 3 calls and more=true, got %d/%v", len(first.Calls), first.More)
	}
	batches = append(batches, first.Calls)

	second := q.DequeueBatch(batch)
	if len(second.Calls) != 2 || second.More {
		t.Fatalf("second batch must have the 2 remaining calls and more=false, got %d/%v", len(second.Calls), second.More)
	}
	batches = append(batches, second.Calls)

	if q.Len() != 0 {
		t.Fatalf("queue must be drained, got %d", q.Len())
	}

	for bi, b := range batches {
		for _, c := range b {
			if seen[c.ID] {
				t.Fatalf("call %s dequeued more than once (batch %d)", c.ID, bi)
			}
			seen[c.ID] = true
		}
	}
	for id := range want {
		if !seen[id] {
			t.Fatalf("call %s was never dequeued", id)
		}
	}

	// After draining, a fresh enqueue must be delivered and the dispatcher
	// must not be told the queue is empty.
	fresh := model.NewCall("cfresh", "echo", nil, time.Now().Add(time.Second))
	if err := q.Enqueue(fresh); err != nil {
		t.Fatal(err)
	}
	again := q.DequeueBatch(batch)
	if len(again.Calls) != 1 || again.More {
		t.Fatalf("post-drain dequeue must return the fresh call once, got %d/%v", len(again.Calls), again.More)
	}
	if again.Calls[0].ID != "cfresh" {
		t.Fatalf("post-drain dequeue returned wrong call %q", again.Calls[0].ID)
	}
}
