package fluxor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/fluxor-io/fluxor/pkg/core"
)

type Runtime struct {
	bus         *core.EventBus
	deployments map[string]core.Reactor // Registry
	mu          sync.RWMutex
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

func New() *Runtime {
	ctx, cancel := context.WithCancel(context.Background())
	return &Runtime{
		bus:         core.NewEventBus(),
		deployments: make(map[string]core.Reactor),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Implement core.RuntimeRef
func (r *Runtime) EventBus() *core.EventBus { return r.bus }

func (r *Runtime) Deploy(reactor core.Reactor, config map[string]any) string {
	id := uuid.New().String()
	
	// Inject Dependencies vào Context
	fctx := core.NewFluxorContext(r.ctx, r, id, config)

	r.mu.Lock()
	r.deployments[id] = reactor
	r.mu.Unlock()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		// Khởi động Reactor
		if err := reactor.OnStart(fctx); err != nil {
			slog.Error("Failed to deploy reactor", "id", id, "error", err)
			r.Undeploy(id)
			return
		}
		// Reactor chạy ngầm bên trong OnStart hoặc các goroutine con của nó
	}()

	return id
}

func (r *Runtime) Undeploy(id string) {
	r.mu.Lock()
	reactor, exists := r.deployments[id]
	if !exists {
		r.mu.Unlock()
		return
	}
	delete(r.deployments, id)
	r.mu.Unlock()

	if err := reactor.OnStop(); err != nil {
		slog.Error("Error stopping reactor", "id", id, "error", err)
	}
}

func (r *Runtime) Shutdown() {
	slog.Info("System shutting down...")
	r.cancel() // Signal context cancellation
	
	r.mu.Lock()
	for id, reactor := range r.deployments {
		reactor.OnStop()
		delete(r.deployments, id)
	}
	r.mu.Unlock()
	
	// Wait với Timeout
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("Shutdown graceful complete")
	case <-time.After(5 * time.Second):
		slog.Warn("Shutdown timed out")
	}
}
