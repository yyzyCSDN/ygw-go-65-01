package model

import "time"

// Outcome is the final classification of a finished call.
type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeTimeout Outcome = "timeout"
)

// Result records how one call finished and what it produced.
type Result struct {
	CallID     string
	FuncName   string
	Outcome    Outcome
	Output     []byte
	Error      string
	Attempt    int
	FinishedAt time.Time
}

// Succeeded reports whether the call produced a successful output.
func (r *Result) Succeeded() bool {
	return r != nil && r.Outcome == OutcomeSuccess
}
