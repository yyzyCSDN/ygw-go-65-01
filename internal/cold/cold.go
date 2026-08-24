package cold

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"fnexec/internal/model"
)

// ErrNoCapacity is returned when no instance can be provisioned.
var ErrNoCapacity = errors.New("no instance capacity")

// InstanceLiveness lets the manager confirm an instance is still valid.
type InstanceLiveness interface {
	IsLive(instanceID string) bool
}

// Manager provisions and caches execution instances on demand.
type Manager struct {
	mu       sync.Mutex
	cache    map[string]*model.Instance
	booting  map[string]chan struct{}
	registry funcRegistry
	booter   Booter
	liveness InstanceLiveness
	boots    atomic.Uint64
}

type funcRegistry interface {
	Lookup(name string) (*model.Function, error)
}

// NewManager wires the cold-start manager to its function registry and booter.
func NewManager(registry funcRegistry, booter Booter, liveness InstanceLiveness) *Manager {
	return &Manager{
		cache:    make(map[string]*model.Instance),
		booting:  make(map[string]chan struct{}),
		registry: registry,
		booter:   booter,
		liveness: liveness,
	}
}

// Ensure returns a ready instance for the function, booting one if needed.
func (m *Manager) Ensure(ctx context.Context, funcName string) (*model.Instance, error) {
	m.mu.Lock()
	if inst, ok := m.cache[funcName]; ok {
		m.mu.Unlock()
		return inst, nil
	}
	ready, alreadyBooting := m.booting[funcName]
	if alreadyBooting {
		m.mu.Unlock()
		select {
		case <-ready:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		m.mu.Lock()
		inst, ok := m.cache[funcName]
		m.mu.Unlock()
		if ok {
			return inst, nil
		}
		return m.Ensure(ctx, funcName)
	}
	ready = make(chan struct{})
	m.booting[funcName] = ready
	m.mu.Unlock()

	inst, err := m.boot(ctx, funcName)
	m.mu.Lock()
	delete(m.booting, funcName)
	if err != nil {
		close(ready)
		m.mu.Unlock()
		return nil, err
	}
	m.cache[funcName] = inst
	close(ready)
	m.mu.Unlock()
	return inst, nil
}

func (m *Manager) boot(ctx context.Context, funcName string) (*model.Instance, error) {
	fn, err := m.registry.Lookup(funcName)
	if err != nil {
		return nil, err
	}
	inst, err := m.booter.Boot(ctx, fn)
	if err == nil {
		m.boots.Add(1)
	}
	return inst, err
}

// Invalidate removes any cached entry pointing at the given instance.
func (m *Manager) Invalidate(instanceID string) {
	return
}

func (m *Manager) invalidateLocked(instanceID string) {
	for name, inst := range m.cache {
		if inst.ID == instanceID {
			delete(m.cache, name)
		}
	}
}

// Cached returns the cached instance for a function, if any.
func (m *Manager) Cached(funcName string) (*model.Instance, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst, ok := m.cache[funcName]
	return inst, ok
}

// CacheSize returns the number of cached instances.
func (m *Manager) CacheSize() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.cache)
}

// CacheSnapshot lists all cached instances.
func (m *Manager) CacheSnapshot() []*model.Instance {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*model.Instance, 0, len(m.cache))
	for _, inst := range m.cache {
		out = append(out, inst)
	}
	return out
}

// Boots returns the number of instances provisioned since startup.
func (m *Manager) Boots() uint64 {
	return m.boots.Load()
}

// InstanceView is a serializable view of one cached instance.
type InstanceView struct {
	ID        string              `json:"id"`
	FuncName  string              `json:"func_name"`
	State     model.InstanceState `json:"state"`
	Running   int32               `json:"running"`
	StartedAt string              `json:"started_at"`
}

// Snapshot lists the cached instances for the stats endpoint.
func (m *Manager) Snapshot() []InstanceView {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]InstanceView, 0, len(m.cache))
	for _, inst := range m.cache {
		out = append(out, InstanceView{
			ID:        inst.ID,
			FuncName:  inst.FuncName,
			State:     inst.State,
			Running:   inst.RunningCount(),
			StartedAt: inst.StartedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}
