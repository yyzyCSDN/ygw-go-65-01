package model

import "time"

// Event is an incoming trigger that should become one call.
type Event struct {
	ID       string
	FuncName string
	Payload  []byte
	Time     time.Time
}

// NewEvent builds a trigger event with the given identity.
func NewEvent(id, funcName string, payload []byte) Event {
	return Event{
		ID:       id,
		FuncName: funcName,
		Payload:  append([]byte(nil), payload...),
		Time:     time.Now(),
	}
}
