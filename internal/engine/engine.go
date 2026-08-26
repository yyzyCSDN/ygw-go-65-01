package engine

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync/atomic"
	"time"

	"fnexec/internal/cold"
	"fnexec/internal/exec"
	"fnexec/internal/func"
	"fnexec/internal/model"
	"fnexec/internal/queue"
	"fnexec/internal/retry"
	"fnexec/internal/scale"
	"fnexec/internal/trigger"
	"fnexec/internal/web"
)

// Version is the FnExec engine version reported by the console and probes.
const Version = "1.0.0"

// Options configures a complete engine instance.
type Options struct {
	HTTPAddr       string
	BatchSize      int
	Workers        int
	BootDelay      time.Duration
	DefaultTimeout time.Duration
	HandleLimit    int
	ScaleInterval  time.Duration
	ConsoleHTML    []byte
}

// Engine wires every component and owns the HTTP lifecycle.
type Engine struct {
	Funcs     *funcs.Registry
	Queue     *queue.Queue
	Cold      *cold.Manager
	Scale     *scale.Scaler
	Retry     *retry.Manager
	Exec      *exec.Executor
	Trigger   *trigger.Trigger
	Web       *web.Server
	httpSrv   *http.Server
	httpLn    net.Listener
	cancel    context.CancelFunc
	started   atomic.Bool
	startedAt time.Time
	options   Options
}

// New builds a fully wired engine.
func New(opts Options) (*Engine, error) {
	if opts.HTTPAddr == "" {
		opts.HTTPAddr = "127.0.0.1:8080"
	}
	if opts.ScaleInterval <= 0 {
		opts.ScaleInterval = 2 * time.Second
	}
	registry := funcs.NewRegistry()
	if err := funcs.RegisterBuiltins(registry); err != nil {
		return nil, err
	}
	callQueue := queue.New()
	scaler := scale.NewScaler(nil)
	coldMgr := cold.NewManager(registry, cold.NewLocalBooter(opts.BootDelay), scaler)
	scaler.SetOnReclaim(func(instanceID string) { coldMgr.Invalidate(instanceID) })
	retrier := retry.NewManager(retry.Config{Queue: callQueue, MaxRetries: 2})
	executor := exec.NewExecutor(exec.Config{
		Queue:          callQueue,
		Cold:           coldMgr,
		Scale:          scaler,
		Retry:          retrier,
		Funcs:          registry,
		BatchSize:      opts.BatchSize,
		Workers:        opts.Workers,
		DefaultTimeout: opts.DefaultTimeout,
		HandleLimit:    opts.HandleLimit,
	})
	retrier.SetChecker(executor.Results())
	trig := trigger.New(callQueue, registry, trigger.NewThrottle(200))
	eng := &Engine{
		Funcs:     registry,
		Queue:     callQueue,
		Cold:      coldMgr,
		Scale:     scaler,
		Retry:     retrier,
		Exec:      executor,
		Trigger:   trig,
		startedAt: time.Now(),
		options:   opts,
	}
	eng.Web = web.NewServer(eng, eng, eng, opts.ConsoleHTML)
	return eng, nil
}

// Start launches the executor, autoscaler and HTTP server.
func (e *Engine) Start(parent context.Context) error {
	ctx, cancel := context.WithCancel(parent)
	e.cancel = cancel
	go e.Exec.Run(ctx)
	go e.scaleLoop(ctx)

	ln, err := net.Listen("tcp", e.options.HTTPAddr)
	if err != nil {
		cancel()
		return err
	}
	e.httpLn = ln
	e.httpSrv = &http.Server{Handler: e.Web.Handler()}
	go func() {
		_ = e.httpSrv.Serve(ln)
	}()
	e.started.Store(true)
	return nil
}

// Addr returns the address the HTTP server is listening on.
func (e *Engine) Addr() string {
	if e.httpLn == nil {
		return e.options.HTTPAddr
	}
	return e.httpLn.Addr().String()
}

// Ready reports whether the HTTP server has started.
func (e *Engine) Ready() bool {
	return e.started.Load()
}

// Stop cancels the workers and shuts the HTTP server down.
func (e *Engine) Stop(ctx context.Context) error {
	if e.cancel != nil {
		e.cancel()
	}
	var shutdownErr error
	if e.httpSrv != nil {
		if err := e.httpSrv.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			shutdownErr = err
		}
	}
	for _, entry := range e.Funcs.Snapshot() {
		_ = e.Scale.RemoveAll(entry.Name)
	}
	_ = e.Queue.Close()
	return shutdownErr
}

// Invoke turns an event into a queued call.
func (e *Engine) Invoke(ctx context.Context, ev model.Event) (*model.Call, error) {
	return e.Trigger.Handle(ctx, ev)
}

// GetResult returns the stored outcome of a call.
func (e *Engine) GetResult(callID string) *model.Result {
	return e.Exec.Results().Get(callID)
}

// ListFunctions returns the registered function list.
func (e *Engine) ListFunctions() []funcs.Entry {
	return e.Funcs.Snapshot()
}

// SearchFunctions returns function names matching a prefix.
func (e *Engine) SearchFunctions(prefix string) []string {
	return e.Funcs.Search(prefix)
}

// Snapshot assembles the stats payload for the console.
func (e *Engine) Snapshot() web.Stats {
	return web.Stats{
		Version:       Version,
		UptimeSeconds: int64(time.Since(e.startedAt).Seconds()),
		Functions:     e.Funcs.Snapshot(),
		Queue:         e.Queue.Snapshot(),
		Instances:     e.Scale.Snapshot(),
		Routes:        e.Scale.RouteView(),
		Exec:          e.Exec.StatsSnapshot(),
		Retries:       e.Retry.Retries(),
		ColdCache:     e.Cold.CacheSize(),
		ColdInstances: e.Cold.Snapshot(),
		Trigger:       e.Trigger.Stats(),
	}
}
