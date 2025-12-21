package core

import (
	"sync"
)

// BaseComponent provides a Java-style abstract base class for components
// Components are reusable units that can be embedded in verticles
// Similar to Java's Component pattern
type BaseComponent struct {
	// Component name
	name string

	// Parent verticle (if embedded in a verticle)
	parent *BaseVerticle

	// State
	mu      sync.RWMutex
	started bool
}

// NewBaseComponent creates a new BaseComponent
func NewBaseComponent(name string) *BaseComponent {
	return &BaseComponent{
		name: name,
	}
}

// SetParent sets the parent verticle
func (bc *BaseComponent) SetParent(parent *BaseVerticle) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.parent = parent
}

// Start initializes the component
func (bc *BaseComponent) Start(ctx FluxorContext) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if bc.started {
		return &Error{Code: "ALREADY_STARTED", Message: "component already started"}
	}

	// Call hook method
	if err := bc.doStart(ctx); err != nil {
		return err
	}

	bc.started = true
	return nil
}

// Stop stops the component
func (bc *BaseComponent) Stop(ctx FluxorContext) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	if !bc.started {
		return nil
	}

	// Call hook method
	if err := bc.doStop(ctx); err != nil {
		return err
	}

	bc.started = false
	return nil
}

// doStart is a hook method for subclasses
func (bc *BaseComponent) doStart(ctx FluxorContext) error {
	return nil
}

// doStop is a hook method for subclasses
func (bc *BaseComponent) doStop(ctx FluxorContext) error {
	return nil
}

// Name returns the component name
func (bc *BaseComponent) Name() string {
	return bc.name
}

// Parent returns the parent verticle (if any)
func (bc *BaseComponent) Parent() *BaseVerticle {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.parent
}

// IsStarted returns whether the component is started
func (bc *BaseComponent) IsStarted() bool {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	return bc.started
}

// EventBus returns the EventBus from parent verticle
func (bc *BaseComponent) EventBus() EventBus {
	if bc.parent != nil {
		return bc.parent.EventBus()
	}
	return nil
}

// Vertx returns the Vertx from parent verticle
func (bc *BaseComponent) Vertx() Vertx {
	if bc.parent != nil {
		return bc.parent.Vertx()
	}
	return nil
}
