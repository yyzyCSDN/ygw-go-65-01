package main

import (
	"context"
	"fmt"
	"time"

	"fnexec/internal/cold"
	"fnexec/internal/func"
	"fnexec/internal/model"
	"fnexec/internal/scale"
)

func main() {
	reg := funcs.NewRegistry()
	reg.Register(&model.Function{
		Name:    "demo",
		Timeout: time.Second,
		Handler: func(ctx context.Context, payload []byte) ([]byte, error) { return payload, nil },
	})
	scaler := scale.NewScaler(nil)
	mgr := cold.NewManager(reg, cold.NewLocalBooter(0), scaler)
	scaler.SetOnReclaim(func(id string) { mgr.Invalidate(id) })

	ctx := context.Background()
	inst, err := mgr.Ensure(ctx, "demo")
	if err != nil {
		panic(err)
	}
	fmt.Printf("booted: %s state=%s cached=%d\n", inst.ID, inst.State, mgr.CacheSize())

	// Autoscaler reclaims the instance (scale-down path via resize).
	if err := scaler.Resize("demo", 0); err != nil {
		panic(err)
	}
	fmt.Printf("after resize-down reclaim: scaler live=%v cached=%d\n", scaler.IsLive(inst.ID), mgr.CacheSize())

	// Next call: must NOT return the reclaimed instance.
	inst2, err := mgr.Ensure(ctx, "demo")
	if err != nil {
		panic(err)
	}
	if inst2.ID == inst.ID {
		fmt.Printf("FAIL: Ensure returned the reclaimed instance %s (state=%s)\n", inst2.ID, inst2.State)
		return
	}
	fmt.Printf("PASS: next call got fresh instance %s state=%s (old %s no longer served)\n", inst2.ID, inst2.State, inst.ID)
}
