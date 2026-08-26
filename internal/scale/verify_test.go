package scale

import "testing"

// TestScaleTableUpdatedAfterResize verifies the route table follows the
// instance count after scaling up and down.
func TestScaleTableUpdatedAfterResize(t *testing.T) {
	s := NewScaler(nil)
	if err := s.Resize("demo", 3); err != nil {
		t.Fatal(err)
	}
	routes := s.Routes("demo")
	if len(routes) != 3 {
		t.Fatalf("routes must contain all new instances after scale-up, got %v", routes)
	}
	if s.Count("demo") != 3 {
		t.Fatalf("instance count must be 3, got %d", s.Count("demo"))
	}
	if err := s.Resize("demo", 1); err != nil {
		t.Fatal(err)
	}
	routes = s.Routes("demo")
	if len(routes) != 1 {
		t.Fatalf("routes must shrink after scale-down, got %v", routes)
	}
}
