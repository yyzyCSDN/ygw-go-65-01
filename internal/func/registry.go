package funcs

import (
	"errors"
	"sort"
	"strings"
	"sync"

	"fnexec/internal/model"
)

// ErrDuplicateName is returned when a function is already registered.
var ErrDuplicateName = errors.New("function already registered")

// ErrNotFound is returned when a function name is not registered.
var ErrNotFound = errors.New("function not registered")

// Registry holds the set of callable functions.
type Registry struct {
	mu    sync.RWMutex
	funcs map[string]*model.Function
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{funcs: make(map[string]*model.Function)}
}

// Register adds a function unless the name is taken.
func (r *Registry) Register(fn *model.Function) error {
	if err := Validate(fn); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.funcs[fn.Name]; exists {
		return ErrDuplicateName
	}
	r.funcs[fn.Name] = fn
	return nil
}

// Lookup returns the function registered under name.
func (r *Registry) Lookup(name string) (*model.Function, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	fn, ok := r.funcs[name]
	if !ok {
		return nil, ErrNotFound
	}
	return fn, nil
}

// Has reports whether a function with the given name exists.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.funcs[name]
	return ok
}

// Names returns all registered names in sorted order.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.funcs))
	for name := range r.funcs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of registered functions.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.funcs)
}

// Search returns function names beginning with the given prefix.
func (r *Registry) Search(prefix string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0)
	for name := range r.funcs {
		if len(prefix) == 0 || strings.HasPrefix(name, prefix) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
