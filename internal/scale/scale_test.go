package scale

import (
	"context"
	"testing"
)

func TestAcquireReleaseTracksRunning(t *testing.T) {
	s := NewScaler(nil)
	inst := s.NewInstance("demo")
	if !s.Acquire(inst) {
		t.Fatal("acquire failed")
	}
	if inst.RunningCount() != 1 {
		t.Fatalf("running count must be 1, got %d", inst.RunningCount())
	}
	s.Release(inst)
	if inst.RunningCount() != 0 {
		t.Fatalf("running count must return to 0, got %d", inst.RunningCount())
	}
}

func TestReclaimRemovesInstance(t *testing.T) {
	s := NewScaler(nil)
	inst := s.NewInstance("demo")
	if err := s.Reclaim(context.Background(), inst.ID); err != nil {
		t.Fatal(err)
	}
	if s.IsLive(inst.ID) {
		t.Fatal("reclaimed instance must not be live")
	}
	if len(s.Routes("demo")) != 0 {
		t.Fatal("routes must drop the reclaimed instance")
	}
}

func TestReclaimUnknownInstance(t *testing.T) {
	s := NewScaler(nil)
	if err := s.Reclaim(context.Background(), "missing"); err != ErrUnknownInstance {
		t.Fatalf("expected ErrUnknownInstance, got %v", err)
	}
}

func TestInstanceIDsSorted(t *testing.T) {
	s := NewScaler(nil)
	s.NewInstance("demo")
	s.NewInstance("demo")
	ids := s.InstanceIDs()
	if len(ids) != 2 || ids[0] >= ids[1] {
		t.Fatalf("instance ids must be sorted, got %v", ids)
	}
}
