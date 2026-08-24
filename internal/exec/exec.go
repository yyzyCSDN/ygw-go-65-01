package exec

import (
	"context"
	"errors"
	"time"

	"fnexec/internal/cold"
	"fnexec/internal/func"
	"fnexec/internal/model"
	"fnexec/internal/queue"
	"fnexec/internal/retry"
	"fnexec/internal/scale"
)

// ErrInstanceUnavailable is returned when a call cannot be placed on an instance.
var ErrInstanceUnavailable = errors.New("instance unavailable")

// Config wires an executor to the surrounding engine components.
type Config struct {
	Queue          *queue.Queue
	Cold           *cold.Manager
	Scale          *scale.Scaler
	Retry          *retry.Manager
	Funcs          *funcs.Registry
	BatchSize      int
	Workers        int
	DefaultTimeout time.Duration
	HandleLimit    int
}

// Executor dispatches queued calls onto instances and records outcomes.
type Executor struct {
	queue          *queue.Queue
	cold           *cold.Manager
	scale          *scale.Scaler
	retry          *retry.Manager
	funcs          *funcs.Registry
	results        *ResultStore
	claims         *ClaimMap
	runtime        *RuntimePool
	stats          *Stats
	batchSize      int
	workers        int
	defaultTimeout time.Duration
}

// NewExecutor builds an executor from its configuration.
func NewExecutor(cfg Config) *Executor {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 8
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 3
	}
	if cfg.DefaultTimeout <= 0 {
		cfg.DefaultTimeout = 5 * time.Second
	}
	return &Executor{
		queue:          cfg.Queue,
		cold:           cfg.Cold,
		scale:          cfg.Scale,
		retry:          cfg.Retry,
		funcs:          cfg.Funcs,
		results:        NewResultStore(),
		claims:         NewClaimMap(),
		runtime:        NewRuntimePool(cfg.HandleLimit),
		stats:          &Stats{},
		batchSize:      cfg.BatchSize,
		workers:        cfg.Workers,
		defaultTimeout: cfg.DefaultTimeout,
	}
}

// Dispatch executes one call exactly once, recording the outcome. A call that
// already committed a successful outcome is returned as-is without running
// again: a retry triggered by a network blip must never re-execute a call that
// already finished, otherwise its side effects would be replayed downstream.
func (e *Executor) Dispatch(ctx context.Context, call *model.Call) *model.Result {
	if !e.claims.TryClaim(call.ID) {
		return e.results.Get(call.ID)
	}
	defer e.claims.Release(call.ID)

	if committed := e.results.Get(call.ID); committed != nil && committed.Succeeded() {
		call.Status = model.StatusSucceeded
		return committed
	}

	call.Status = model.StatusExecuting
	inst, err := e.pickInstance(ctx, call)
	if err != nil {
		result := e.failure(call, err)
		call.Status = model.StatusFailed
		e.results.Commit(result)
		return result
	}
	result := e.Execute(ctx, inst, call)
	if !result.Succeeded() && e.retry.ShouldRetry(call, result) {
		_ = e.retry.Schedule(call)
	}
	e.results.Commit(result)
	switch result.Outcome {
	case model.OutcomeSuccess:
		call.Status = model.StatusSucceeded
	case model.OutcomeTimeout:
		call.Status = model.StatusTimedOut
	default:
		call.Status = model.StatusFailed
	}
	return result
}

// Execute runs one call on an instance with its configured deadline.
func (e *Executor) Execute(ctx context.Context, inst *model.Instance, call *model.Call) *model.Result {
	handle := e.runtime.Acquire(call.ID)
	defer handle.Release()

	if !e.scale.Acquire(inst) {
		return e.failure(call, ErrInstanceUnavailable)
	}
	defer e.scale.Release(inst)

	fn, err := e.funcs.Lookup(call.FuncName)
	if err != nil {
		return e.failure(call, err)
	}
	runCtx, cancel := context.WithTimeout(ctx, e.timeoutFor(call))
	defer cancel()

	done := make(chan handlerOutcome, 1)
	go func() {
		out, err := fn.Handler(runCtx, call.Payload)
		done <- handlerOutcome{out: out, err: err}
	}()

	select {
	case o := <-done:
		return e.finishResult(call, o)
	case <-runCtx.Done():
		return e.finalizeTimeout(call, func() (handlerOutcome, bool) {
			select {
			case o := <-done:
				return o, true
			default:
				return handlerOutcome{}, false
			}
		})
	case <-inst.Stop:
		return e.failure(call, ErrInstanceUnavailable)
	}
}

func (e *Executor) finishResult(call *model.Call, o handlerOutcome) *model.Result {
	if o.err != nil {
		result := e.failure(call, o.err)
		e.results.Commit(result)
		return result
	}
	result := &model.Result{
		CallID:     call.ID,
		FuncName:   call.FuncName,
		Outcome:    model.OutcomeSuccess,
		Output:     o.out,
		Attempt:    call.Attempt,
		FinishedAt: time.Now(),
	}
	e.results.Commit(result)
	e.stats.RecordSuccess()
	return result
}

func (e *Executor) failure(call *model.Call, err error) *model.Result {
	result := &model.Result{
		CallID:     call.ID,
		FuncName:   call.FuncName,
		Outcome:    model.OutcomeFailure,
		Error:      err.Error(),
		Attempt:    call.Attempt,
		FinishedAt: time.Now(),
	}
	e.stats.RecordFailure()
	return result
}

// pickInstance selects a routed running instance or boots a cold one.
func (e *Executor) pickInstance(ctx context.Context, call *model.Call) (*model.Instance, error) {
	for _, id := range e.scale.Routes(call.FuncName) {
		inst := e.scale.Get(id)
		if inst != nil && e.scale.Ready(inst) {
			return inst, nil
		}
	}
	inst, err := e.cold.Ensure(ctx, call.FuncName)
	if err != nil {
		return nil, err
	}
	e.scale.Register(inst)
	return inst, nil
}

// Results exposes the result store for callers outside the executor.
func (e *Executor) Results() *ResultStore {
	return e.results
}

// Runtime returns the runtime handle pool.
func (e *Executor) Runtime() *RuntimePool {
	return e.runtime
}

// StatsSnapshot returns executor counters for the stats endpoint.
func (e *Executor) StatsSnapshot() StatsSnapshot {
	executions, successes, failures, timeouts := e.stats.Snapshot()
	return StatsSnapshot{
		Executions:  executions,
		Successes:   successes,
		Failures:    failures,
		Timeouts:    timeouts,
		Claims:      e.claims.Len(),
		Results:     e.results.Count(),
		Handles:     e.runtime.Active(),
		BatchSize:   e.batchSize,
		Workers:     e.workers,
		HandleLimit: e.runtime.Limit(),
	}
}
