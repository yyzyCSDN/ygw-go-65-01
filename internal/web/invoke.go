package web

import (
	"encoding/json"
	"net/http"
	"time"

	"fnexec/internal/model"
)

type invokeRequest struct {
	ID       string          `json:"id"`
	FuncName string          `json:"func"`
	Payload  json.RawMessage `json:"payload"`
}

type invokeResponse struct {
	CallID string `json:"call_id"`
	Status string `json:"status"`
}

func (s *Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	var req invokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	ev := model.Event{
		ID:       req.ID,
		FuncName: req.FuncName,
		Payload:  req.Payload,
		Time:     time.Now(),
	}
	call, err := s.invoker.Invoke(r.Context(), ev)
	if err != nil {
		http.Error(w, "invoke failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(invokeResponse{CallID: call.ID, Status: string(model.StatusQueued)})
}

func (s *Server) handleCall(w http.ResponseWriter, r *http.Request) {
	callID := r.PathValue("id")
	result := s.results.GetResult(callID)
	if result == nil {
		http.Error(w, "call not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}
