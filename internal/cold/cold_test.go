package cold

import (
	"context"
	"testing"
	"time"

	"fnexec/internal/func"
	"fnexec/internal/model"
)

func TestEnsureBootsAndCaches(t *testing.T) {
	reg := funcs.NewRegistry()
	reg.Register(&model.Function{
		Name:    "demo",
		Timeout: time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) { return payload, nil },
	})
	mgr := NewManager(reg, NewLocalBooter(0), nil)
	inst, err := mgr.Ensure(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if inst.State != model.InstanceRunning {
		t.Fatalf("instance must be running, got %s", inst.State)
	}
	again, err := mgr.Ensure(context.Background(), "demo")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != inst.ID {
		t.Fatal("second ensure must reuse the cached instance")
	}
}
