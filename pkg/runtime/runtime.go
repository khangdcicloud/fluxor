package runtime

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/example/goflux/pkg/bus"
	"github.com/example/goflux/pkg/component"
	"github.com/example/goflux/pkg/reactor"
	"github.com/example/goflux/pkg/worker"
)

// ErrRuntimeNotStarted is returned when an operation is performed on a runtime that has not been started.
var ErrRuntimeNotStarted = errors.New("runtime not started")

// Status represents the current status of the Runtime.
type Status struct {
	Reactors    []ReactorStatus
	WorkerPool  worker.Status
	Deployments []DeploymentStatus
}

// ReactorStatus represents the status of a single Reactor.
type ReactorStatus struct {
	ReactorID int
	Status    reactor.Status
}

// DeploymentStatus represents the status of a single deployed component.
type DeploymentStatus struct {
	ComponentName string
	ReactorID     int
}

// deployment tracks a deployed component and its assigned reactor.
type deployment struct {
	component component.Component
	reactorID int
}

// Runtime manages the lifecycle of components, reactors, and the worker pool.
type Runtime struct {
	bus         bus.Bus
	reactors    []*reactor.Reactor
	workerPool  *worker.WorkerPool
	deployments map[string]*deployment
	nextReactor int
	started     bool
	mu          sync.RWMutex
}

// NewRuntime creates a new Runtime.
type NewRuntimeOptions struct {
	NumReactors int
	MailboxSize int
	NumWorkers  int
	QueueSize   int
}

func NewRuntime(opts NewRuntimeOptions, b bus.Bus) *Runtime {
	reactors := make([]*reactor.Reactor, opts.NumReactors)
	for i := 0; i < opts.NumReactors; i++ {
		reactors[i] = reactor.NewReactor(opts.MailboxSize)
	}

	return &Runtime{
		bus:         b,
		reactors:    reactors,
		workerPool:  worker.NewWorkerPool(opts.NumWorkers, opts.QueueSize),
		deployments: make(map[string]*deployment),
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
	reactorID := r.nextReactor
	reactor := r.reactors[reactorID]
	r.nextReactor = (r.nextReactor + 1) % len(r.reactors)

	// Create a bus proxy that ensures all handlers are executed on the assigned reactor.
	busProxy := &reactorBusProxy{
		bus:     r.bus,
		reactor: reactor,
	}

	// Store the deployment information.
	componentName := getComponentName(c)
	r.deployments[componentName] = &deployment{
		component: c,
		reactorID: reactorID,
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

// Status returns the current status of the runtime.
func (r *Runtime) Status() Status {
	r.mu.RLock()
	defer r.mu.RUnlock()

	reactorStatuses := make([]ReactorStatus, len(r.reactors))
	for i, reactor := range r.reactors {
		reactorStatuses[i] = ReactorStatus{
			ReactorID: i,
			Status:    reactor.Status(),
		}
	}

	deploymentStatuses := make([]DeploymentStatus, 0, len(r.deployments))
	for name, dep := range r.deployments {
		deploymentStatuses = append(deploymentStatuses, DeploymentStatus{
			ComponentName: name,
			ReactorID:     dep.reactorID,
		})
	}

	return Status{
		Reactors:    reactorStatuses,
		WorkerPool:  r.workerPool.Status(),
		Deployments: deploymentStatuses,
	}
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

// getComponentName returns a name for the given component.
func getComponentName(c component.Component) string {
	// This is a simple implementation that uses the type name of the component.
	// A more robust implementation might use a custom name provider.
	val := reflect.ValueOf(c)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return fmt.Sprintf("%s.%s", val.Type().PkgPath(), val.Type().Name())
}
