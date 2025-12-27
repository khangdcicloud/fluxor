package core

import (
	"context"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
)

// BaseVerticle provides a Java-style abstract base class for verticles
// It implements common lifecycle management and provides hook methods for customization
// Similar to Java's AbstractVerticle pattern
// Each verticle has its own event loop for sequential event processing
type BaseVerticle struct {
	// Name of the verticle (can be set by subclasses)
	name string

	// Context reference (set during Start)
	ctx FluxorContext

	// EventBus reference (cached for convenience)
	eventBus EventBus

	// GoCMD reference (cached for convenience) - GoCMD instance
	gocmd GoCMD

	// Event loop for this verticle (sequential processing)
	// Each verticle has its own event loop - events are processed sequentially
	eventLoop concurrency.Executor

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

// Start implements Verticle.Start
// This is the single entry point - subclasses override Start() directly
// For async operations, use the event loop or implement AsyncVerticle
func (bv *BaseVerticle) Start(ctx FluxorContext) error {
	bv.mu.Lock()
	defer bv.mu.Unlock()

	if bv.started {
		return &EventBusError{Code: "ALREADY_STARTED", Message: "verticle already started"}
	}

	// Set context and references
	bv.ctx = ctx
	bv.eventBus = ctx.EventBus()
	bv.gocmd = ctx.GoCMD()

	// Create event loop for this verticle (1 worker = sequential processing)
	// Each verticle has its own event loop - events are processed sequentially
	gocmdCtx := ctx.GoCMD().Context()
	eventLoopConfig := concurrency.ExecutorConfig{
		Workers:   1,    // Single worker = sequential processing (event loop)
		QueueSize: 1000, // Queue size for events
	}
	bv.eventLoop = concurrency.NewExecutor(gocmdCtx, eventLoopConfig)

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

	// Shutdown event loop gracefully
	if bv.eventLoop != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = bv.eventLoop.Shutdown(shutdownCtx)
		cancel()
		bv.eventLoop = nil
	}

	bv.stopped = true
	return nil
}

// doStart is deprecated - subclasses should override Start() directly
// Kept for backward compatibility
func (bv *BaseVerticle) doStart(ctx FluxorContext) error {
	return nil
}

// doStop is deprecated - subclasses should override Stop() directly
// Kept for backward compatibility
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

// GoCMD returns the GoCMD reference (kept as GoCMD for backward compatibility)
func (bv *BaseVerticle) GoCMD() GoCMD {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.gocmd
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
	// Fail-fast: consumer cannot be nil
	if consumer == nil {
		FailFast(&EventBusError{Code: "INVALID_CONSUMER", Message: "consumer cannot be nil"})
	}
	bv.mu.Lock()
	defer bv.mu.Unlock()
	bv.consumers = append(bv.consumers, consumer)
}

// Consumer creates and registers a consumer for the given address
// Returns the consumer for further configuration
func (bv *BaseVerticle) Consumer(address string) Consumer {
	// Fail-fast: verticle must be started
	if bv.eventBus == nil {
		FailFast(&EventBusError{Code: "NOT_STARTED", Message: "verticle not started - cannot create consumer"})
	}
	consumer := bv.eventBus.Consumer(address)
	bv.RegisterConsumer(consumer)
	return consumer
}

// Publish is a convenience method to publish messages
func (bv *BaseVerticle) Publish(address string, body interface{}) error {
	if bv.eventBus == nil {
		return &EventBusError{Code: "NOT_STARTED", Message: "verticle not started"}
	}
	return bv.eventBus.Publish(address, body)
}

// Send is a convenience method to send messages
func (bv *BaseVerticle) Send(address string, body interface{}) error {
	if bv.eventBus == nil {
		return &EventBusError{Code: "NOT_STARTED", Message: "verticle not started"}
	}
	return bv.eventBus.Send(address, body)
}

// EventLoop returns the event loop executor for this verticle
// Each verticle has its own event loop for sequential event processing
func (bv *BaseVerticle) EventLoop() concurrency.Executor {
	bv.mu.RLock()
	defer bv.mu.RUnlock()
	return bv.eventLoop
}

// RunOnEventLoop executes a task on this verticle's event loop
// Tasks are processed sequentially (single worker)
func (bv *BaseVerticle) RunOnEventLoop(task concurrency.Task) error {
	// Fail-fast: task cannot be nil
	if task == nil {
		FailFast(&EventBusError{Code: "INVALID_TASK", Message: "task cannot be nil"})
	}
	bv.mu.RLock()
	eventLoop := bv.eventLoop
	bv.mu.RUnlock()

	if eventLoop == nil {
		return &EventBusError{Code: "NOT_STARTED", Message: "verticle not started - event loop not available"}
	}

	return eventLoop.Submit(task)
}
