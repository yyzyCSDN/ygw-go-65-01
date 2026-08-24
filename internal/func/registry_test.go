package funcs

import (
	"context"
	"testing"
	"time"

	"fnexec/internal/model"
)

func TestRegistryLifecycle(t *testing.T) {
	r := NewRegistry()
	fn := &model.Function{
		Name:    "ping",
		Timeout: time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) { return payload, nil },
	}
	if err := r.Register(fn); err != nil {
		t.Fatal(err)
	}
	if !r.Has("ping") {
		t.Fatal("ping must be present")
	}
	if err := r.Register(fn); err != ErrDuplicateName {
		t.Fatalf("duplicate register must fail with ErrDuplicateName, got %v", err)
	}
	got, err := r.Lookup("ping")
	if err != nil || got.Name != "ping" {
		t.Fatalf("lookup failed: %v", err)
	}
}

func TestRegisterBuiltins(t *testing.T) {
	r := NewRegistry()
	if err := RegisterBuiltins(r); err != nil {
		t.Fatal(err)
	}
	if r.Count() != 5 {
		t.Fatalf("expected 5 builtins, got %d", r.Count())
	}
	if len(r.Snapshot()) != 5 {
		t.Fatal("snapshot must list all builtins")
	}
}

func TestValidatorRejectsBadNames(t *testing.T) {
	r := NewRegistry()
	valid := &model.Function{
		Name:    "x",
		Timeout: time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) { return nil, nil },
	}
	if err := r.Register(valid); err != nil {
		t.Fatalf("single-letter name must be valid: %v", err)
	}
	for _, name := range []string{"", "A", "1abc", "has space"} {
		fn := &model.Function{
			Name:    name,
			Timeout: time.Second,
			Handler: func(ctx context.Context, payload []byte) ([]byte, error) { return nil, nil },
		}
		if err := r.Register(fn); err == nil {
			t.Fatalf("name %q must be rejected", name)
		}
	}
}
