package engine

import (
	"context"
	"time"

	"fnexec/internal/scale"
)

func (e *Engine) scaleLoop(ctx context.Context) {
	ticker := time.NewTicker(e.options.ScaleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.reconcile(ctx)
		}
	}
}

func (e *Engine) reconcile(ctx context.Context) {
	demand := e.Queue.PendingByFunc()
	for _, entry := range e.Funcs.Snapshot() {
		fn, err := e.Funcs.Lookup(entry.Name)
		if err != nil {
			continue
		}
		desired := scale.Recommend(demand[entry.Name], fn.MinInstances, fn.MaxInstances)
		if e.Scale.Count(entry.Name) != desired {
			_ = e.Scale.Resize(entry.Name, desired)
		}
	}
}
