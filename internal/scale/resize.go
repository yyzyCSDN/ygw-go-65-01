package scale

import (
	"errors"

	"fnexec/internal/model"
)

// ErrInvalidSize is returned when a resize target is out of bounds.
var ErrInvalidSize = errors.New("invalid instance count")

// Resize adjusts the number of instances for a function and refreshes routes.
func (s *Scaler) Resize(funcName string, desired int) error {
	if desired < 0 || desired > 16 {
		return ErrInvalidSize
	}
	current := s.Count(funcName)
	for i := current; i < desired; i++ {
		s.NewInstance(funcName)
	}
	for i := current; i > desired; i-- {
		s.reclaimOldest(funcName)
	}
	return nil
}

func (s *Scaler) reclaimOldest(funcName string) {
	s.mu.Lock()
	var oldest *model.Instance
	for _, inst := range s.instances {
		if inst.FuncName != funcName || inst.State == model.InstanceRemoved {
			continue
		}
		if oldest == nil || inst.StartedAt.Before(oldest.StartedAt) {
			oldest = inst
		}
	}
	if oldest == nil {
		s.mu.Unlock()
		return
	}
	id := oldest.ID
	delete(s.instances, id)
	onReclaim := s.onReclaim
	oldest.State = model.InstanceRemoved
	s.mu.Unlock()
	close(oldest.Stop)
	if onReclaim != nil {
		onReclaim(id)
	}
}
