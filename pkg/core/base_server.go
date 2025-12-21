package core

import (
	"sync"
)

// BaseServer provides a Java-style abstract base class for HTTP servers
// It implements common lifecycle management and provides hook methods for customization
// Similar to Java's abstract base class pattern
type BaseServer struct {
	// Name of the server (can be set by subclasses)
	name string

	// Vertx reference
	vertx Vertx

	// State management
	mu      sync.RWMutex
	started bool
	stopped bool

	// Logger for server operations
	logger Logger
}

// NewBaseServer creates a new BaseServer
func NewBaseServer(name string, vertx Vertx) *BaseServer {
	if vertx == nil {
		// Fail-fast: vertx cannot be nil
		panic("vertx cannot be nil")
	}
	return &BaseServer{
		name:   name,
		vertx:  vertx,
		logger: NewDefaultLogger(),
	}
}

// Start implements Server.Start with template method pattern
// Subclasses should override doStart() for custom initialization
func (bs *BaseServer) Start() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.started {
		return &Error{Code: "ALREADY_STARTED", Message: "server already started"}
	}

	// Call hook method for subclass customization
	if err := bs.doStart(); err != nil {
		return err
	}

	bs.started = true
	return nil
}

// Stop implements Server.Stop with template method pattern
// Subclasses should override doStop() for custom cleanup
func (bs *BaseServer) Stop() error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bs.stopped {
		return nil // Already stopped
	}

	// Call hook method for subclass customization
	if err := bs.doStop(); err != nil {
		return err
	}

	bs.stopped = true
	return nil
}

// doStart is a hook method for subclasses to override
// Default implementation does nothing
func (bs *BaseServer) doStart() error {
	return nil
}

// doStop is a hook method for subclasses to override
// Default implementation does nothing
func (bs *BaseServer) doStop() error {
	return nil
}

// Name returns the server name
func (bs *BaseServer) Name() string {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.name
}

// Vertx returns the Vertx reference
func (bs *BaseServer) Vertx() Vertx {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.vertx
}

// EventBus returns the EventBus reference
func (bs *BaseServer) EventBus() EventBus {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	if bs.vertx == nil {
		return nil
	}
	return bs.vertx.EventBus()
}

// Logger returns the logger instance
func (bs *BaseServer) Logger() Logger {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.logger
}

// SetLogger sets a custom logger for this server
func (bs *BaseServer) SetLogger(logger Logger) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.logger = logger
}

// IsStarted returns whether the server has been started
func (bs *BaseServer) IsStarted() bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.started
}

// IsStopped returns whether the server has been stopped
func (bs *BaseServer) IsStopped() bool {
	bs.mu.RLock()
	defer bs.mu.RUnlock()
	return bs.stopped
}

