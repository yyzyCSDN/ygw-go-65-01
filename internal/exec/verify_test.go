package exec

import (
	"context"
	"fmt"
	"testing"
	"time"

	"fnexec/internal/model"
)

// TestRuntimeHandleClosed verifies every execution returns its runtime
// handle so the pool never leaks slots.
func TestRuntimeHandleClosed(t *testing.T) {
	ex, _, _, _, _, _ := testExecutor(t, 0)
	for i := 0; i < 6; i++ {
		call := model.NewCall(fmt.Sprintf("h%d", i), "demo", nil, time.Now().Add(time.Second))
		if result := ex.Dispatch(context.Background(), call); !result.Succeeded() {
			t.Fatalf("dispatch %d failed: %+v", i, result)
		}
	}
	if active := ex.runtime.Active(); active != 0 {
		t.Fatalf("runtime handles must be returned after execution, active=%d", active)
	}
	if len(ex.runtime.InFlight()) != 0 {
		t.Fatalf("no call may hold a runtime handle after execution, in-flight=%v", ex.runtime.InFlight())
	}
}
