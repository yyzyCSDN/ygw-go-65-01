package scale

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"sync"

	"fnexec/internal/model"
)

// ErrUnknownInstance is returned when an instance ID is not tracked.
var ErrUnknownInstance = errors.New("unknown instance")

// Scaler owns the instance table and the per-function route lists.
type Scaler struct {
	mu        sync.Mutex
	instances map[string]*model.Instance
	routes    map[string][]string
	onReclaim func(instanceID string)
}

// NewScaler creates an empty scaler. onReclaim may be nil and can be
// installed later with SetOnReclaim.
func NewScaler(onReclaim func(instanceID string)) *Scaler {
	return &Scaler{
		instances: make(map[string]*model.Instance),
		routes:    make(map[string][]string),
		onReclaim: onReclaim,
	}
}

// SetOnReclaim installs the reclaim notification callback.
func (s *Scaler) SetOnReclaim(onReclaim func(instanceID string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.onReclaim = onReclaim
}

// NewInstance creates and registers an instance for a function.
func (s *Scaler) NewInstance(funcName string) *model.Instance {
	s.mu.Lock()
	defer s.mu.Unlock()
	inst := model.NewInstance(model.HashID("inst", funcName, strconv.Itoa(len(s.instances)+1)), funcName)
	inst.State = model.InstanceRunning
	s.instances[inst.ID] = inst
	return inst
}

// Register adds an externally provisioned instance (for example one booted by
// the cold-start manager) to the table and its function routes.
func (s *Scaler) Register(inst *model.Instance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.instances[inst.ID]; exists {
		return
	}
	s.instances[inst.ID] = inst
	s.rebuildRoutesLocked(inst.FuncName)
}

// Get returns a tracked instance by ID.
func (s *Scaler) Get(instanceID string) *model.Instance {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.instances[instanceID]
}

// IsLive reports whether an instance is still tracked and not removed.
func (s *Scaler) IsLive(instanceID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	inst, ok := s.instances[instanceID]
	return ok && inst.State != model.InstanceRemoved
}

// Routes returns the dispatch order for a function.
func (s *Scaler) Routes(funcName string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.routes[funcName]))
	copy(out, s.routes[funcName])
	return out
}

// Acquire reserves an instance for one call when it is still running.
func (s *Scaler) Acquire(inst *model.Instance) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if inst.State != model.InstanceRunning {
		return false
	}
	inst.AddRunning()
	return true
}

// Release frees an instance after one call finished.
func (s *Scaler) Release(inst *model.Instance) {
	inst.SubRunning()
}

// Count returns the number of tracked instances for a function.
func (s *Scaler) Count(funcName string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, inst := range s.instances {
		if inst.FuncName == funcName && inst.State != model.InstanceRemoved {
			count++
		}
	}
	return count
}

// Ready reports whether the instance is tracked and accepting calls.
func (s *Scaler) Ready(inst *model.Instance) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.instances[inst.ID]
	return ok && current.State == model.InstanceRunning
}

// InstanceIDs lists all tracked instance IDs.
func (s *Scaler) InstanceIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.instances))
	for id := range s.instances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// RemoveAll reclaims every instance of a function and returns how many were removed.
func (s *Scaler) RemoveAll(funcName string) int {
	s.mu.Lock()
	ids := make([]string, 0)
	for id, inst := range s.instances {
		if inst.FuncName == funcName {
			ids = append(ids, id)
		}
	}
	s.mu.Unlock()
	for _, id := range ids {
		_ = s.Reclaim(context.Background(), id)
	}
	return len(ids)
}
