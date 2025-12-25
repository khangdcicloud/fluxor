package tcp

import (
	"context"
	"net"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Server represents a TCP server abstraction, aligned with pkg/web.Server.
type Server interface {
	// Start starts the server (blocking, like HTTP servers).
	Start() error

	// Stop stops the server gracefully.
	Stop() error

	// SetHandler sets the connection handler (fail-fast on nil).
	SetHandler(handler ConnectionHandler)

	// Metrics returns current server metrics.
	Metrics() ServerMetrics
}

// ConnectionHandler handles a single TCP connection.
// Implementations should be fail-fast and must not block forever.
// The server closes the connection after handler returns.
type ConnectionHandler func(ctx *ConnContext) error

// ConnContext provides per-connection context and convenient access to Vertx/EventBus.
// Mirrors web.RequestContext's shape (BaseRequestContext + Context + Vertx + EventBus).
type ConnContext struct {
	*core.BaseRequestContext

	Context  context.Context
	Conn     net.Conn
	Vertx    core.Vertx
	EventBus core.EventBus

	LocalAddr  net.Addr
	RemoteAddr net.Addr
}

// ServerMetrics provides TCP server performance metrics.
type ServerMetrics struct {
	QueuedConnections   int64   // Current queued connections
	RejectedConnections int64   // Total rejected connections (backpressure)
	QueueCapacity       int     // Maximum queue capacity
	Workers             int     // Number of worker goroutines
	QueueUtilization    float64 // Queue utilization percentage
	NormalCCU           int     // Normal capacity (target utilization baseline)
	CurrentCCU          int     // Current load (relative to normal capacity)
	CCUUtilization      float64 // Utilization percentage (relative to normal capacity)
	TotalAccepted       int64   // Total connections accepted
	HandledConnections  int64   // Total connections handled
	ErrorConnections    int64   // Total connections that returned error
}
