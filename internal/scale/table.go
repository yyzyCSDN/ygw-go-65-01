package scale

import (
	"sort"

	"fnexec/internal/model"
)

// InstanceView is a serializable snapshot of one instance.
type InstanceView struct {
	ID        string              `json:"id"`
	FuncName  string              `json:"func_name"`
	State     model.InstanceState `json:"state"`
	Running   int32               `json:"running"`
	StartedAt string              `json:"started_at"`
}

// Snapshot returns all instances grouped by function in a stable order.
func (s *Scaler) Snapshot() []InstanceView {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(s.instances))
	for id := range s.instances {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]InstanceView, 0, len(ids))
	for _, id := range ids {
		inst := s.instances[id]
		out = append(out, InstanceView{
			ID:        inst.ID,
			FuncName:  inst.FuncName,
			State:     inst.State,
			Running:   inst.RunningCount(),
			StartedAt: inst.StartedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return out
}

// RouteView lists the route order per function.
func (s *Scaler) RouteView() map[string][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]string, len(s.routes))
	for name, ids := range s.routes {
		out[name] = append([]string(nil), ids...)
	}
	return out
}
