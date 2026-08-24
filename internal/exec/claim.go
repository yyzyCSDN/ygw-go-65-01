package exec

import "sync"

// ClaimMap tracks call IDs that are currently being dispatched so that one
// call is never executed by two workers at the same time.
type ClaimMap struct {
	mu     sync.Mutex
	claims map[string]struct{}
}

// NewClaimMap creates an empty claim set.
func NewClaimMap() *ClaimMap {
	return &ClaimMap{claims: make(map[string]struct{})}
}

// TryClaim atomically claims a call ID unless it is already claimed.
func (c *ClaimMap) TryClaim(callID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.claims[callID]; exists {
		return false
	}
	c.claims[callID] = struct{}{}
	return true
}

// Release removes a claim after the call finished.
func (c *ClaimMap) Release(callID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.claims, callID)
}

// Len returns the number of in-flight claims.
func (c *ClaimMap) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.claims)
}
