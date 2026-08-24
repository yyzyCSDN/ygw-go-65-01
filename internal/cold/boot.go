package cold

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"fnexec/internal/model"
)

// Booter creates a fresh instance for a function.
type Booter interface {
	Boot(ctx context.Context, fn *model.Function) (*model.Instance, error)
}

// LocalBooter provisions in-process instances with a simulated startup cost.
type LocalBooter struct {
	startDelay time.Duration
	seq        atomic.Uint64
}

// NewLocalBooter returns a booter with the given simulated startup delay.
func NewLocalBooter(startDelay time.Duration) *LocalBooter {
	return &LocalBooter{startDelay: startDelay}
}

// Boot marks an instance booting, waits the startup delay, then marks it running.
func (b *LocalBooter) Boot(ctx context.Context, fn *model.Function) (*model.Instance, error) {
	id := fmt.Sprintf("inst-%s-%06d", fn.Name, b.seq.Add(1))
	inst := model.NewInstance(id, fn.Name)
	inst.State = model.InstanceBooting
	if b.startDelay > 0 {
		timer := time.NewTimer(b.startDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	inst.State = model.InstanceRunning
	return inst, nil
}
