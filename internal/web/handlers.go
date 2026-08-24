package web

import (
	"encoding/json"
	"net/http"

	"fnexec/internal/func"
)

// FunctionLister exposes the registered function list.
type FunctionLister interface {
	ListFunctions() []funcs.Entry
}

// FunctionSearcher optionally supports prefix filtering of functions.
type FunctionSearcher interface {
	SearchFunctions(prefix string) []string
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.stats.Snapshot())
}

func (s *Server) handleFunctions(w http.ResponseWriter, r *http.Request) {
	lister, ok := s.stats.(FunctionLister)
	if !ok {
		http.Error(w, "function listing unavailable", http.StatusNotFound)
		return
	}
	if prefix := r.URL.Query().Get("prefix"); prefix != "" {
		if searcher, ok := lister.(FunctionSearcher); ok {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(searcher.SearchFunctions(prefix))
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(lister.ListFunctions())
}

func (s *Server) handleConsole(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if len(s.console) == 0 {
		http.Error(w, "console unavailable", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(s.console)
}
