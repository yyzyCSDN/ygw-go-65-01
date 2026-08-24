package trigger

import (
	"sync"
	"time"
)

// Throttle limits events per function with a small token bucket.
type Throttle struct {
	mu      sync.Mutex
	rate    int
	buckets map[string]*bucket
}

type bucket struct {
	tokens   float64
	lastFill time.Time
}

// NewThrottle allows up to rate events per second per function.
func NewThrottle(rate int) *Throttle {
	if rate <= 0 {
		rate = 100
	}
	return &Throttle{rate: rate, buckets: make(map[string]*bucket)}
}

// Allow consumes one token for the function when available.
func (t *Throttle) Allow(funcName string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	b, ok := t.buckets[funcName]
	now := time.Now()
	if !ok {
		b = &bucket{tokens: float64(t.rate), lastFill: now}
		t.buckets[funcName] = b
	}
	elapsed := now.Sub(b.lastFill).Seconds()
	b.tokens += elapsed * float64(t.rate)
	if b.tokens > float64(t.rate) {
		b.tokens = float64(t.rate)
	}
	b.lastFill = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Rate returns the configured events-per-second ceiling.
func (t *Throttle) Rate() int {
	return t.rate
}
