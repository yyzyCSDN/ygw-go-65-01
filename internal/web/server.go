package web

import (
	"context"
	"net/http"

	"fnexec/internal/model"
)

// Invoker creates a call from an incoming event.
type Invoker interface {
	Invoke(ctx context.Context, ev model.Event) (*model.Call, error)
}

// StatProvider exposes the engine statistics payload.
type StatProvider interface {
	Snapshot() Stats
}

// ResultProvider looks up stored call outcomes.
type ResultProvider interface {
	GetResult(callID string) *model.Result
}

// Server exposes the HTTP API and console page.
type Server struct {
	invoker Invoker
	stats   StatProvider
	results ResultProvider
	console []byte
	mux     *http.ServeMux
}

// NewServer wires the HTTP handlers to the engine.
func NewServer(invoker Invoker, stats StatProvider, results ResultProvider, consoleHTML []byte) *Server {
	return &Server{
		invoker: invoker,
		stats:   stats,
		results: results,
		console: consoleHTML,
	}
}

// Handler returns the fully registered HTTP router.
func (s *Server) Handler() http.Handler {
	if s.mux != nil {
		return s.mux
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("GET /api/stats", s.handleStats)
	mux.HandleFunc("GET /api/functions", s.handleFunctions)
	mux.HandleFunc("POST /api/invoke", s.handleInvoke)
	mux.HandleFunc("GET /api/calls/{id}", s.handleCall)
	mux.HandleFunc("GET /", s.handleConsole)
	s.mux = mux
	return accessLog(mux)
}
