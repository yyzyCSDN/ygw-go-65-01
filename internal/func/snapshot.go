package funcs

import "sort"

// Entry is a serializable view of one registered function.
type Entry struct {
	Name         string `json:"name"`
	TimeoutMS    int64  `json:"timeout_ms"`
	MaxRetries   int    `json:"max_retries"`
	MinInstances int    `json:"min_instances"`
	MaxInstances int    `json:"max_instances"`
}

// Snapshot returns a sorted, serializable view of the registry.
func (r *Registry) Snapshot() []Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Entry, 0, len(r.funcs))
	for _, fn := range r.funcs {
		out = append(out, Entry{
			Name:         fn.Name,
			TimeoutMS:    fn.Timeout.Milliseconds(),
			MaxRetries:   fn.MaxRetries,
			MinInstances: fn.MinInstances,
			MaxInstances: fn.MaxInstances,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
