package core

import (
	"sync"
)

// BaseVerticle provides a Java-style abstract base class for verticles
// It implements common lifecycle management and provides hook methods for customization
// Similar to Java's AbstractVerticle pattern
type BaseVerticle struct {
	// Name of the verticle (can be set by subclasses)
	name string

	// Context reference (set during Start)
	ctx FluxorContext

	// EventBus reference (cached for convenience)
	eventBus EventBus

	// Vertx reference (cached for convenience)
	vertx Vertx

	// State management
	mu      sync.RWMutex
	started bool
	stopped bool

	// Consumers registered by this verticle (for cleanup)
	consumers []Consumer
}

// NewBaseVerticle creates a new BaseVerticle
func NewBaseVerticle(name string) *BaseVerticle {
	return &BaseVerticle{
		name:      name,
		consumers: make([]Consumer, 0),
	}
}

// Start implements Verticle.Start with template method pattern
// Subclasses should override doStart() for custom initialization
func (bv *BaseVerticle) Start(ctx FluxorContext) error {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	if bv.started {
		return &Error{Code: "ALREADY_STARTED", Message: "verticle already started"}
	}

	// Set context and references
	bv.ctx = ctx
	bv.eventBus = ctx.EventBus()
	bv.vertx = ctx.Vertx()

	// Call hook method for subclass customization
	if err := bv.doStart(ctx); err != nil {
		return err
	}

	bv.started = true
	return nil
}

// Stop implements Verticle.Stop with template method pattern
// Subclasses should override doStop() for custom cleanup
func (bv *BaseVerticle) Stop(ctx FluxorContext) error {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	if bv.stopped {
		return nil // Already stopped
	}

	// Call hook method for subclass customization
	if err := bv.doStop(ctx); err != nil {
		return err
	}

	// Cleanup registered consumers
	for _, consumer := range bv.consumers {
		_ = consumer.Unregister()
	}
	bv.consumers = nil

	bv.stopped = true
	return nil
}

// doStart is a hook method for subclasses to override
// Default implementation does nothing
func (bv *BaseVerticle) doStart(ctx FluxorContext) error {
	return nil
}

// doStop is a hook method for subclasses to override
// Default implementation does nothing
func (bv *BaseVerticle) doStop(ctx FluxorContext) error {
	return nil
}

// Name returns the verticle name
func (bv *BaseVerticle) Name() string {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.name
}

// Context returns the FluxorContext (set during Start)
func (bv *BaseVerticle) Context() FluxorContext {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.ctx
}

// EventBus returns the EventBus reference
func (bv *BaseVerticle) EventBus() EventBus {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.eventBus
}

// Vertx returns the Vertx reference
func (bv *BaseVerticle) Vertx() Vertx {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.vertx
}

// IsStarted returns whether the verticle has been started
func (bv *BaseVerticle) IsStarted() bool {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.started
}

// IsStopped returns whether the verticle has been stopped
func (bv *BaseVerticle) IsStopped() bool {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.stopped
}

// RegisterConsumer registers a consumer for automatic cleanup
// This is a convenience method for subclasses
func (bv *BaseVerticle) RegisterConsumer(consumer Consumer) {
	bv.mu.Lock()
	defer bv.mu.Unlock()
	bv.consumers = append(bv.consumers, consumer)
}

// Consumer creates and registers a consumer for the given address
// Returns the consumer for further configuration
func (bv *BaseVerticle) Consumer(address string) Consumer {
	if bv.eventBus == nil {
		panic("verticle not started - cannot create consumer")
	}
	consumer := bv.eventBus.Consumer(address)
	bv.RegisterConsumer(consumer)
	return consumer
}

// Publish is a convenience method to publish messages
func (bv *BaseVerticle) Publish(address string, body interface{}) error {
	if bv.eventBus == nil {
		return &Error{Code: "NOT_STARTED", Message: "verticle not started"}
	}
	return bv.eventBus.Publish(address, body)
}

// Send is a convenience method to send messages
func (bv *BaseVerticle) Send(address string, body interface{}) error {
	if bv.eventBus == nil {
		return &Error{Code: "NOT_STARTED", Message: "verticle not started"}
	}
	return bv.eventBus.Send(address, body)
}
