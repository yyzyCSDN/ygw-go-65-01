package exec

import "sync"

// RuntimePool bounds the number of runtime handles held by the executor.
type RuntimePool struct {
	mu       sync.Mutex
	slots    chan struct{}
	inflight map[string]struct{}
}

// Handle is one acquired runtime slot.
type Handle struct {
	pool     *RuntimePool
	callID   string
	released bool
}

// NewRuntimePool creates a pool that allows up to limit concurrent handles.
func NewRuntimePool(limit int) *RuntimePool {
	if limit <= 0 {
		limit = 64
	}
	return &RuntimePool{slots: make(chan struct{}, limit), inflight: make(map[string]struct{})}
}

// Acquire reserves a runtime handle for a call.
func (p *RuntimePool) Acquire(callID string) *Handle {
	p.slots <- struct{}{}
	p.mu.Lock()
	p.inflight[callID] = struct{}{}
	p.mu.Unlock()
	return &Handle{pool: p, callID: callID}
}

// Release returns the handle to the pool.
func (h *Handle) Release() {
	if h == nil || h.released {
		return
	}
	h.pool.mu.Lock()
	h.released = true
	delete(h.pool.inflight, h.callID)
	h.pool.mu.Unlock()
}

// Active returns how many handles are currently reserved.
func (p *RuntimePool) Active() int {
	return len(p.slots)
}

// Limit returns the configured handle ceiling.
func (p *RuntimePool) Limit() int {
	return cap(p.slots)
}

// InFlight lists the call IDs currently holding a handle.
func (p *RuntimePool) InFlight() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, 0, len(p.inflight))
	for id := range p.inflight {
		out = append(out, id)
	}
	return out
}
