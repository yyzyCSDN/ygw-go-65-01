package model

import "time"

// Status describes the lifecycle of one function invocation.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusExecuting Status = "executing"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusTimedOut  Status = "timed-out"
	StatusRetrying  Status = "retrying"
)

// Call is one unit of work: a single invocation of a registered function.
type Call struct {
	ID        string
	FuncName  string
	Payload   []byte
	Status    Status
	Attempt   int
	Deadline  time.Time
	CreatedAt time.Time
}

// NewCall builds a queued call with the given deadline.
func NewCall(id, funcName string, payload []byte, deadline time.Time) *Call {
	return &Call{
		ID:        id,
		FuncName:  funcName,
		Payload:   append([]byte(nil), payload...),
		Status:    StatusQueued,
		Attempt:   1,
		Deadline:  deadline,
		CreatedAt: time.Now(),
	}
}
