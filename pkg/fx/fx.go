package fx

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	
	"github.com/fluxorio/fluxor/pkg/core"
)

// Fluxor is the dependency injection and lifecycle management framework
type Fluxor struct {
	vertx      core.Vertx
	providers  []Provider
	invokers   []Invoker
	lifecycle  *lifecycle
	mu         sync.RWMutex
}

// Provider provides a value to the dependency injection container
type Provider interface {
	// Provide returns the value and optionally an error
	Provide() (interface{}, error)
}

// Invoker is a function that will be invoked after all providers are initialized
type Invoker interface {
	// Invoke is called with the provided dependencies
	Invoke(deps map[reflect.Type]interface{}) error
}

// New creates a new Fluxor instance
func New(ctx context.Context, options ...Option) (*Fluxor, error) {
	vertx := core.NewVertx(ctx)
	
	fx := &Fluxor{
		vertx:     vertx,
		providers: make([]Provider, 0),
		invokers:  make([]Invoker, 0),
		lifecycle: newLifecycle(),
	}
	
	for _, opt := range options {
		if err := opt(fx); err != nil {
			return nil, err
		}
	}
	
	return fx, nil
}

// Option configures a Fluxor instance
type Option func(*Fluxor) error

// Provide registers a provider
func Provide(provider Provider) Option {
	return func(fx *Fluxor) error {
		fx.mu.Lock()
		defer fx.mu.Unlock()
		fx.providers = append(fx.providers, provider)
		return nil
	}
}

// Invoke registers an invoker
func Invoke(invoker Invoker) Option {
	return func(fx *Fluxor) error {
		fx.mu.Lock()
		defer fx.mu.Unlock()
		fx.invokers = append(fx.invokers, invoker)
		return nil
	}
}

// Start starts the Fluxor application
func (fx *Fluxor) Start() error {
	fx.mu.Lock()
	defer fx.mu.Unlock()
	
	// Build dependency map
	deps := make(map[reflect.Type]interface{})
	
	// Provide all dependencies
	for _, provider := range fx.providers {
		value, err := provider.Provide()
		if err != nil {
			return fmt.Errorf("provider error: %w", err)
		}
		
		valueType := reflect.TypeOf(value)
		if valueType != nil {
			deps[valueType] = value
		}
	}
	
	// Add Vertx to dependencies
	deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()] = fx.vertx
	deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()] = fx.vertx.EventBus()
	
	// Invoke all invokers
	for _, invoker := range fx.invokers {
		if err := invoker.Invoke(deps); err != nil {
			return fmt.Errorf("invoker error: %w", err)
		}
	}
	
	fx.lifecycle.start()
	return nil
}

// Stop stops the Fluxor application
func (fx *Fluxor) Stop() error {
	fx.mu.Lock()
	defer fx.mu.Unlock()
	
	fx.lifecycle.stop()
	return fx.vertx.Close()
}

// Vertx returns the Vertx instance
func (fx *Fluxor) Vertx() core.Vertx {
	return fx.vertx
}

// Wait waits for the application to stop
func (fx *Fluxor) Wait() error {
	return fx.lifecycle.wait()
}

// lifecycle manages application lifecycle
type lifecycle struct {
	started chan struct{}
	stopped chan struct{}
	mu      sync.Mutex
}

func newLifecycle() *lifecycle {
	return &lifecycle{
		started: make(chan struct{}),
		stopped: make(chan struct{}),
	}
}

func (l *lifecycle) start() {
	l.mu.Lock()
	defer l.mu.Unlock()
	close(l.started)
}

func (l *lifecycle) stop() {
	l.mu.Lock()
	defer l.mu.Unlock()
	select {
	case <-l.stopped:
	default:
		close(l.stopped)
	}
}

func (l *lifecycle) wait() error {
	<-l.stopped
	return nil
}

