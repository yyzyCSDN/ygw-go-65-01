package trigger

import (
	"context"
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	"fnexec/internal/func"
	"fnexec/internal/model"
	"fnexec/internal/queue"
)

// ErrThrottled is returned when the per-function rate limit is exceeded.
var ErrThrottled = errors.New("trigger throttled")

// Trigger turns incoming events into queued calls.
type Trigger struct {
	queue     *queue.Queue
	registry  *funcs.Registry
	throttle  *Throttle
	accepted  atomic.Int64
	rejected  atomic.Int64
	throttled atomic.Int64
}

// New builds a trigger over the queue and registry.
func New(q *queue.Queue, registry *funcs.Registry, throttle *Throttle) *Trigger {
	return &Trigger{queue: q, registry: registry, throttle: throttle}
}

// Handle validates an event, looks up its function and enqueues a call.
func (t *Trigger) Handle(ctx context.Context, ev model.Event) (*model.Call, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fn, err := t.registry.Lookup(ev.FuncName)
	if err != nil {
		t.rejected.Add(1)
		return nil, err
	}
	if t.throttle != nil && !t.throttle.Allow(ev.FuncName) {
		t.throttled.Add(1)
		return nil, ErrThrottled
	}
	if ev.ID == "" {
		ev.ID = model.HashID("ev", ev.FuncName, strconv.FormatInt(ev.Time.UnixNano(), 10))
	}
	call := model.NewCall(
		model.HashID(ev.ID, ev.FuncName),
		ev.FuncName,
		ev.Payload,
		time.Now().Add(fn.Timeout),
	)
	if err := t.queue.Enqueue(call); err != nil {
		t.rejected.Add(1)
		return nil, err
	}
	t.accepted.Add(1)
	return call, nil
}

// Stats is a serializable view of trigger counters.
type Stats struct {
	Accepted  int64 `json:"accepted"`
	Rejected  int64 `json:"rejected"`
	Throttled int64 `json:"throttled"`
}

// Stats returns the trigger counters.
func (t *Trigger) Stats() Stats {
	return Stats{
		Accepted:  t.accepted.Load(),
		Rejected:  t.rejected.Load(),
		Throttled: t.throttled.Load(),
	}
}
