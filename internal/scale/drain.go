package scale

import (
	"context"
	"time"

	"fnexec/internal/model"
)

// DrainTimeout is how long a scale-down waits for in-flight calls to finish.
const DrainTimeout = 2 * time.Second

// Reclaim removes an instance after its in-flight calls have drained.
func (s *Scaler) Reclaim(ctx context.Context, instanceID string) error {
	inst := s.Get(instanceID)
	if inst == nil {
		return ErrUnknownInstance
	}
	if inst.RunningCount() > 0 {
		s.markDraining(inst)
		if !s.waitDrained(ctx, inst) {
			s.markRunning(inst)
			return context.DeadlineExceeded
		}
	}
	return s.finishReclaim(instanceID)
}

func (s *Scaler) markDraining(inst *model.Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inst.State = model.InstanceDraining
}

func (s *Scaler) markRunning(inst *model.Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	inst.State = model.InstanceRunning
}

func (s *Scaler) waitDrained(ctx context.Context, inst *model.Instance) bool {
	deadline := time.Now().Add(DrainTimeout)
	for inst.RunningCount() > 0 {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(10 * time.Millisecond):
		}
		if time.Now().After(deadline) {
			return false
		}
	}
	return true
}

func (s *Scaler) finishReclaim(instanceID string) error {
	s.mu.Lock()
	inst, ok := s.instances[instanceID]
	if !ok {
		s.mu.Unlock()
		return ErrUnknownInstance
	}
	inst.State = model.InstanceRemoved
	delete(s.instances, instanceID)
	s.removeRoutesLocked(instanceID)
	onReclaim := s.onReclaim
	s.mu.Unlock()
	close(inst.Stop)
	if onReclaim != nil {
		onReclaim(instanceID)
	}
	return nil
}

func (s *Scaler) removeRoutesLocked(instanceID string) {
	for name, ids := range s.routes {
		filtered := ids[:0]
		for _, id := range ids {
			if id != instanceID {
				filtered = append(filtered, id)
			}
		}
		if len(filtered) == 0 {
			delete(s.routes, name)
		} else {
			s.routes[name] = filtered
		}
	}
}
