package queue

import (
	"fmt"
	"testing"
	"time"

	"fnexec/internal/model"
)

// TestQueueBatchBoundaryCorrect verifies batch dequeue never loses or
// duplicates calls when the batch size does not divide the queue length.
func TestQueueBatchBoundaryCorrect(t *testing.T) {
	q := New()
	for i := 1; i <= 7; i++ {
		call := model.NewCall(fmt.Sprintf("c%d", i), "echo", nil, time.Now().Add(time.Second))
		if err := q.Enqueue(call); err != nil {
			t.Fatal(err)
		}
	}
	var dequeued []string
	for {
		batch := q.DequeueBatch(3)
		for _, call := range batch.Calls {
			dequeued = append(dequeued, call.ID)
		}
		if !batch.More {
			break
		}
	}
	if len(dequeued) != 7 {
		t.Fatalf("all seven calls must be dequeued exactly once, got %d: %v", len(dequeued), dequeued)
	}
	seen := make(map[string]bool)
	for _, id := range dequeued {
		if seen[id] {
			t.Fatalf("call %s was dequeued twice", id)
		}
		seen[id] = true
	}
	late := model.NewCall("c8", "echo", nil, time.Now().Add(time.Second))
	if err := q.Enqueue(late); err != nil {
		t.Fatal(err)
	}
	batch := q.DequeueBatch(3)
	if len(batch.Calls) != 1 || batch.Calls[0].ID != "c8" {
		t.Fatalf("post-drain enqueue must be dequeued, got %v", batch.Calls)
	}
}
