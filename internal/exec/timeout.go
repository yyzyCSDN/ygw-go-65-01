package exec

import (
	"time"

	"fnexec/internal/model"
)

// handlerOutcome carries the result of one function body run.
type handlerOutcome struct {
	out []byte
	err error
}

// finalizeTimeout settles a call whose deadline expired. If the handler
// already produced a result, that result wins. finalizeTimeout only records
// the outcome; retry scheduling is owned by Dispatch, which is the single
// authority that re-enqueues a call. Scheduling here as well would double-enqueue
// the same call and let a second worker re-execute it once the first releases
// its claim.
func (e *Executor) finalizeTimeout(call *model.Call, drain func() (handlerOutcome, bool)) *model.Result {
	if o, ok := drain(); ok {
		return e.finishResult(call, o)
	}
	if e.results.CommittedSuccess(call.ID) {
		return e.results.Get(call.ID)
	}
	result := &model.Result{
		CallID:     call.ID,
		FuncName:   call.FuncName,
		Outcome:    model.OutcomeTimeout,
		Attempt:    call.Attempt,
		FinishedAt: time.Now(),
	}
	e.results.Commit(result)
	e.stats.RecordTimeout()
	return result
}

// timeoutFor returns the effective deadline duration for a call.
func (e *Executor) timeoutFor(call *model.Call) time.Duration {
	if call == nil || call.Deadline.IsZero() {
		return e.defaultTimeout
	}
	remaining := time.Until(call.Deadline)
	if remaining <= 0 {
		return time.Nanosecond
	}
	return remaining
}
