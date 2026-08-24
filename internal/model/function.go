package model

import (
	"context"
	"time"
)

// Handler executes a function body for one invocation.
type Handler func(ctx context.Context, payload []byte) ([]byte, error)

// Function is a registered callable together with its execution policy.
type Function struct {
	Name         string
	Handler      Handler
	Timeout      time.Duration
	MaxRetries   int
	MinInstances int
	MaxInstances int
}
