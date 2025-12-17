package runtime

import (
	"context"
	"errors"
	"sync"

	"github.com/example/goflux/pkg/bus"
	"github.com/example/goflux/pkg/component"
	"github.com/example/goflux/pkg/reactor"
	"github.com/example/goflux/pkg/worker"
)

// ErrRuntimeNotStarted is returned when an operation is performed on a runtime that has not been started.
var ErrRuntimeNotStarted = errors.New("runtime not started")

// Runtime manages the lifecycle of components, reactors, and the worker pool.
type Runtime struct {
	bus         bus.Bus
	reactors    []*reactor.Reactor
	workerPool  *worker.WorkerPool
	nextReactor int
	started     bool
	mu          sync.RWMutex
}

// NewRuntime creates a new Runtime.
type NewRuntimeOptions struct {
	NumReactors  int
	MailboxSize  int
	NumWorkers   int
	QueueSize    int
}

func NewRuntime(opts NewRuntimeOptions, b bus.Bus) *Runtime {
	reactors := make([]*reactor.Reactor, opts.NumReactors)
	for i := 0; i < opts.NumReactors; i++ {
		reactors[i] = reactor.NewReactor(opts.MailboxSize)
	}

	return &Runtime{
		bus:        b,
		reactors:   reactors,
		workerPool: worker.NewWorkerPool(opts.NumWorkers, opts.QueueSize),
	}
}

// Start starts the runtime, including all reactors and the worker pool.
func (r *Runtime) Start() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return
	}

	for _, reactor := range r.reactors {
		reactor.Start()
	}
	r.workerPool.Start()
	r.started = true
}

// Stop stops the runtime, including all reactors and the worker pool.
func (r *Runtime) Stop(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(r.reactors))

	for _, r := range r.reactors {
		go func(r *reactor.Reactor) {
			defer wg.Done()
			r.Stop(ctx)
		}(r)
	}

	wg.Wait()
	r.workerPool.Stop(ctx)
	r.started = false
}

// Deploy deploys a component to the runtime. A new reactor is assigned to the component.
func (r *Runtime) Deploy(ctx context.Context, c component.Component) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return ErrRuntimeNotStarted
	}

	// Assign a reactor to the component
	reactor := r.reactors[r.nextReactor]
	r.nextReactor = (r.nextReactor + 1) % len(r.reactors)

	// Create a bus proxy that ensures all handlers are executed on the assigned reactor.
	busProxy := &reactorBusProxy{
		bus:     r.bus,
		reactor: reactor,
	}

	// Start the component on its assigned reactor and wait for it to complete.
	done := make(chan error, 1)
	reactor.Submit(func() {
		// This is now running on the reactor goroutine.
		done <- c.Start(ctx, busProxy)
	})

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ExecuteBlocking executes a function on the worker pool.
func (r *Runtime) ExecuteBlocking(f func()) error {
	return r.workerPool.Submit(f)
}

// reactorBusProxy is a proxy for the event bus that ensures all handlers are executed on a specific reactor.
type reactorBusProxy struct {
	bus     bus.Bus
	reactor *reactor.Reactor
}

func (p *reactorBusProxy) Publish(msg bus.Message) {
	p.bus.Publish(msg)
}

func (p *reactorBusProxy) Send(msg bus.Message) error {
	return p.bus.Send(msg)
}

func (p *reactorBusProxy) Request(msg bus.Message, replyHandler bus.Handler) {
	// The reply handler must also be executed on the reactor.
	reactorReplyHandler := func(reply bus.Message) {
		p.reactor.Submit(func() {
			replyHandler(reply)
		})
	}
	p.bus.Request(msg, reactorReplyHandler)
}

func (p *reactorBusProxy) Consumer(topic string, handler bus.Handler) {
	p.bus.Consumer(topic, func(msg bus.Message) {
		// When a message is received, we submit it to the reactor for processing.
		// This ensures that the handler is always executed on the correct reactor.
		p.reactor.Submit(func() {
			handler(msg)
		})
	})
}
