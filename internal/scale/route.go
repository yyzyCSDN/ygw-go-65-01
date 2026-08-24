package scale

import (
	"sort"

	"fnexec/internal/model"
)

// RebuildRoutes regenerates the route list for one function from the live
// instance table, keeping a deterministic order.
func (s *Scaler) RebuildRoutes(funcName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rebuildRoutesLocked(funcName)
}

func (s *Scaler) rebuildRoutesLocked(funcName string) {
	ids := make([]string, 0)
	for id, inst := range s.instances {
		if inst.FuncName == funcName && inst.State != model.InstanceRemoved {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	s.routes[funcName] = ids
}
