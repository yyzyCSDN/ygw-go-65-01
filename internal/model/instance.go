package model

import (
	"sync/atomic"
	"time"
)

// InstanceState is the lifecycle state of one execution instance.
type InstanceState string

const (
	InstanceIdle     InstanceState = "idle"
	InstanceBooting  InstanceState = "booting"
	InstanceRunning  InstanceState = "running"
	InstanceDraining InstanceState = "draining"
	InstanceRemoved  InstanceState = "removed"
)

// Instance is a slot that can run one function invocation at a time.
type Instance struct {
	ID        string
	FuncName  string
	State     InstanceState
	Running   int32
	StartedAt time.Time
	Stop      chan struct{}
}

// NewInstance creates an idle instance for the given function.
func NewInstance(id, funcName string) *Instance {
	return &Instance{
		ID:        id,
		FuncName:  funcName,
		State:     InstanceIdle,
		StartedAt: time.Now(),
		Stop:      make(chan struct{}),
	}
}

// AddRunning atomically increments the running call count.
func (in *Instance) AddRunning() int32 {
	return atomic.AddInt32(&in.Running, 1)
}

// SubRunning atomically decrements the running call count.
func (in *Instance) SubRunning() int32 {
	return atomic.AddInt32(&in.Running, -1)
}

// RunningCount returns the current number of running calls.
func (in *Instance) RunningCount() int32 {
	return atomic.LoadInt32(&in.Running)
}
