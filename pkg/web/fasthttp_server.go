package web

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/valyala/fasthttp"
)

// FastHTTPServer implements Server using fasthttp for high performance
// Uses Executor and Mailbox abstractions to hide Go concurrency primitives
// Extends BaseServer for common lifecycle management
type FastHTTPServer struct {
	*core.BaseServer // Embed base server for lifecycle management
	router           *fastRouter
	server           *fasthttp.Server
	addr             string
	requestMailbox   concurrency.Mailbox  // Abstracted: hides chan *fasthttp.RequestCtx
	executor         concurrency.Executor // Abstracted: hides goroutine pool
	maxQueue         int
	workers          int
	// Metrics for monitoring
	queuedRequests     int64 // Atomic counter for queued requests
	rejectedRequests   int64 // Atomic counter for rejected requests (503)
	totalRequests      int64 // Atomic counter for total requests
	successfulRequests int64 // Atomic counter for successful requests (200-299)
	errorRequests      int64 // Atomic counter for error requests (500-599)
	// Backpressure controller for CCU-based limiting
	backpressure *BackpressureController
}

// FastHTTPServerConfig configures the fasthttp server
type FastHTTPServerConfig struct {
	Addr            string
	MaxQueue        int // Bounded queue for backpressure
	Workers         int // Number of worker goroutines
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxConns        int
	ReadBufferSize  int
	WriteBufferSize int
}

// DefaultFastHTTPServerConfig returns default configuration for 100k RPS
func DefaultFastHTTPServerConfig(addr string) *FastHTTPServerConfig {
	return &FastHTTPServerConfig{
		Addr:            addr,
		MaxQueue:        10000, // Bounded queue for backpressure
		Workers:         100,   // Worker goroutines
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		MaxConns:        100000,
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
	}
}

// CCUBasedConfig returns configuration optimized for CCU (Concurrent Users)
// maxCCU: Maximum concurrent users to serve normally
// overflowCCU: Additional CCU that will receive 503 (fail-fast backpressure)
// Formula: QueueSize = maxCCU - Workers (to handle overflow with 503)
func CCUBasedConfig(addr string, maxCCU int, overflowCCU int) *FastHTTPServerConfig {
	// Calculate workers: typically 10-20% of max CCU for optimal throughput
	workers := maxCCU / 10
	if workers < 50 {
		workers = 50 // Minimum workers
	}
	if workers > 500 {
		workers = 500 // Maximum workers to prevent goroutine explosion
	}

	// Queue size = maxCCU - workers
	// This ensures we can queue up to maxCCU requests
	// When queue is full, additional requests get 503 immediately (fail-fast)
	queueSize := maxCCU - workers
	if queueSize < 100 {
		queueSize = 100 // Minimum queue size
	}

	// MaxConns should allow maxCCU + some buffer, but reject overflow
	maxConns := maxCCU + overflowCCU

	return &FastHTTPServerConfig{
		Addr:            addr,
		MaxQueue:        queueSize, // Bounded queue: when full, return 503
		Workers:         workers,   // Worker goroutines for processing
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		MaxConns:        maxConns, // Allow connections but queue controls backpressure
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
	}
}

// CCUBasedConfigWithUtilization returns configuration with target utilization percentage
// maxCCU: Maximum concurrent users capacity
// utilizationPercent: Target utilization under normal load (e.g., 67 for 67%)
// This leaves headroom for traffic spikes while maintaining stability
// Formula: NormalCapacity = maxCCU * (utilizationPercent / 100)
func CCUBasedConfigWithUtilization(addr string, maxCCU int, utilizationPercent int) *FastHTTPServerConfig {
	if utilizationPercent < 1 || utilizationPercent > 100 {
		utilizationPercent = 67 // Default to 67% if invalid
	}

	// Calculate normal capacity (target utilization)
	normalCapacity := int(float64(maxCCU) * float64(utilizationPercent) / 100.0)

	// Calculate workers: 10-20% of normal capacity
	workers := normalCapacity / 10
	if workers < 50 {
		workers = 50 // Minimum workers
	}
	if workers > 500 {
		workers = 500 // Maximum workers
	}

	// Queue size for normal capacity
	queueSize := normalCapacity - workers
	if queueSize < 100 {
		queueSize = 100 // Minimum queue size
	}

	// MaxConns allows up to maxCCU (100% capacity)
	// But backpressure will kick in at normalCapacity (utilizationPercent)
	maxConns := maxCCU

	return &FastHTTPServerConfig{
		Addr:            addr,
		MaxQueue:        queueSize, // Queue sized for normal capacity
		Workers:         workers,   // Workers sized for normal capacity
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		MaxConns:        maxConns, // Allow up to maxCCU connections
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
	}
}

// NewFastHTTPServer creates a new fasthttp server with reactor-based handling
func NewFastHTTPServer(vertx core.Vertx, config *FastHTTPServerConfig) *FastHTTPServer {
	if config == nil {
		config = DefaultFastHTTPServerConfig(":8080")
	}

	router := newFastRouter()

	// Calculate normal CCU capacity (queue + workers)
	// This is the target utilization capacity (e.g., 67% of max)
	normalCapacity := config.MaxQueue + config.Workers

	// Create Mailbox abstraction (hides channel creation)
	requestMailbox := concurrency.NewBoundedMailbox(config.MaxQueue)

	// Create Executor for worker pool (hides goroutine creation)
	// Use vertx context for executor
	vertxCtx := vertx.Context()
	executorConfig := concurrency.ExecutorConfig{
		Workers:   config.Workers,
		QueueSize: config.MaxQueue,
	}
	executor := concurrency.NewExecutor(vertxCtx, executorConfig)

	s := &FastHTTPServer{
		BaseServer:     core.NewBaseServer("fasthttp-server", vertx),
		router:         router,
		addr:           config.Addr,
		requestMailbox: requestMailbox, // Abstracted: hides chan
		executor:       executor,       // Abstracted: hides goroutines
		maxQueue:       config.MaxQueue,
		workers:        config.Workers,
		// Initialize backpressure controller with normal capacity
		// This ensures 67% utilization under normal load
		// Reset interval: 60 seconds (for metrics)
		backpressure: NewBackpressureController(normalCapacity, 60),
		server: &fasthttp.Server{
			ReadTimeout:                   config.ReadTimeout,
			WriteTimeout:                  config.WriteTimeout,
			MaxConnsPerIP:                 config.MaxConns,
			ReadBufferSize:                config.ReadBufferSize,
			WriteBufferSize:               config.WriteBufferSize,
			DisableHeaderNamesNormalizing: false,
			NoDefaultServerHeader:         true,
			ReduceMemoryUsage:             true,
		},
	}

	// Set handler after server is created
	s.server.Handler = s.handleRequest

	// Start request processing workers using Executor (hides goroutine creation)
	s.startRequestWorkers()

	return s
}

// doStart is called by BaseServer.Start() - implements hook method
func (s *FastHTTPServer) doStart() error {
	// Set handler if not already set
	if s.server.Handler == nil {
		s.server.Handler = s.handleRequest
	}
	// Start request processing workers using Executor (hides goroutine creation)
	s.startRequestWorkers()
	// Start listening (blocking call)
	return s.server.ListenAndServe(s.addr)
}

// doStop is called by BaseServer.Stop() - implements hook method
func (s *FastHTTPServer) doStop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close request mailbox (hides channel close)
	s.requestMailbox.Close()

	// Shutdown executor (hides goroutine cleanup)
	if err := s.executor.Shutdown(ctx); err != nil {
		return err
	}

	// Shutdown server
	return s.server.ShutdownWithContext(ctx)
}

// Router returns the router
func (s *FastHTTPServer) Router() Router {
	return s.router
}

// FastRouter returns the fast router for direct access
func (s *FastHTTPServer) FastRouter() *fastRouter {
	return s.router
}

// Metrics returns current server metrics
func (s *FastHTTPServer) Metrics() ServerMetrics {
	bpMetrics := s.backpressure.GetMetrics()
	normalCapacity := int(bpMetrics.NormalCapacity)
	queued := atomic.LoadInt64(&s.queuedRequests)
	queueUtil := float64(queued) / float64(s.maxQueue) * 100
	if queueUtil > 100.0 {
		queueUtil = 100.0
	}
	return ServerMetrics{
		QueuedRequests:     queued,
		RejectedRequests:   atomic.LoadInt64(&s.rejectedRequests),
		QueueCapacity:      s.maxQueue,
		Workers:            s.workers,
		QueueUtilization:   queueUtil,
		NormalCCU:          normalCapacity, // Normal capacity (target utilization, e.g., 67%)
		CurrentCCU:         int(bpMetrics.CurrentLoad),
		CCUUtilization:     bpMetrics.Utilization, // Utilization relative to normal capacity
		TotalRequests:      atomic.LoadInt64(&s.totalRequests),
		SuccessfulRequests: atomic.LoadInt64(&s.successfulRequests),
		ErrorRequests:      atomic.LoadInt64(&s.errorRequests),
	}
}

// ServerMetrics provides server performance metrics
type ServerMetrics struct {
	QueuedRequests     int64   // Current queued requests
	RejectedRequests   int64   // Total rejected requests (503)
	QueueCapacity      int     // Maximum queue capacity
	Workers            int     // Number of worker goroutines
	QueueUtilization   float64 // Queue utilization percentage
	NormalCCU          int     // Normal CCU capacity (target utilization, e.g., 67%)
	CurrentCCU         int     // Current CCU load
	CCUUtilization     float64 // CCU utilization percentage (relative to normal capacity)
	TotalRequests      int64   // Total requests processed (successful + rejected)
	SuccessfulRequests int64   // Total successful requests (200-299)
	ErrorRequests      int64   // Total error requests (500-599)
}

// handleRequest is the main request handler - non-blocking, queues to workers
// Fail-fast: Returns 503 immediately when normal capacity exceeded (backpressure)
// Normal capacity is set to target utilization (e.g., 67%), leaving headroom for spikes
// This prevents system crash by rejecting overflow requests gracefully
func (s *FastHTTPServer) handleRequest(ctx *fasthttp.RequestCtx) {
	// Step 1: Check backpressure controller (normal capacity limiting)
	// Normal capacity = target utilization (e.g., 67% of max)
	// This ensures system operates at target utilization under normal load
	if !s.backpressure.TryAcquire() {
		// Fail-fast: Normal capacity exceeded, reject immediately
		// This maintains target utilization (e.g., 67%) under normal conditions
		atomic.AddInt64(&s.rejectedRequests, 1)
		ctx.Error("Service Unavailable", fasthttp.StatusServiceUnavailable)
		ctx.SetContentType("application/json")
		ctx.WriteString(`{"error":"capacity_exceeded","message":"Server at normal capacity - backpressure applied","code":"BACKPRESSURE"}`)
		return
	}

	// Step 2: Try to queue request using Mailbox abstraction (hides select statement)
	if err := s.requestMailbox.Send(ctx); err != nil {
		// Queue full - fail-fast backpressure: return 503 immediately
		// Release backpressure capacity since we're not processing
		s.backpressure.Release()
		atomic.AddInt64(&s.rejectedRequests, 1)

		ctx.Error("Service Unavailable", fasthttp.StatusServiceUnavailable)
		ctx.SetContentType("application/json")
		ctx.WriteString(`{"error":"queue_full","message":"Server overloaded - backpressure applied","code":"BACKPRESSURE"}`)
		return
	}

	// Request queued successfully
	atomic.AddInt64(&s.queuedRequests, 1)
	// Note: queuedRequests and backpressure released in worker after processing
}

// SetHandler sets the request handler
func (s *FastHTTPServer) SetHandler(handler func(*fasthttp.RequestCtx)) {
	s.server.Handler = handler
}

// startRequestWorkers starts request processing using Executor (hides goroutine creation)
func (s *FastHTTPServer) startRequestWorkers() {
	// Submit worker tasks to executor (hides go func() calls)
	for i := 0; i < s.workers; i++ {
		task := concurrency.NewNamedTask(
			fmt.Sprintf("http-worker-%d", i),
			func(ctx context.Context) error {
				return s.processRequestFromMailbox(ctx)
			},
		)
		if err := s.executor.Submit(task); err != nil {
			// Log error but continue
			s.Logger().Errorf("failed to start worker %d: %v", i, err)
		}
	}
}

// processRequestFromMailbox processes requests from mailbox (hides channel operations)
func (s *FastHTTPServer) processRequestFromMailbox(ctx context.Context) error {
	// Fail-fast: recover from panics to prevent system crash
	// Panic isolation: one worker panic doesn't crash entire system
		defer func() {
			if r := recover(); r != nil {
				// Log panic but don't re-panic to prevent system crash
				s.Logger().Errorf("panic in worker (isolated): %v", r)
			}
		}()

	// Use Mailbox abstraction (hides channel receive and select statement)
	for {
		msg, err := s.requestMailbox.Receive(ctx)
		if err != nil {
			// Mailbox closed or context cancelled
			return err
		}

		// Type assert to RequestCtx
		reqCtx, ok := msg.(*fasthttp.RequestCtx)
		if !ok {
			// Invalid message type - skip
			continue
		}

		// Decrement queued counter when processing starts
		atomic.AddInt64(&s.queuedRequests, -1)

		// Process request with panic isolation
		func() {
			// Release backpressure capacity when request completes
			defer s.backpressure.Release()

			defer func() {
				if r := recover(); r != nil {
					// Handler panic: return 500 error instead of crashing
					// Panic isolation: one request panic doesn't crash system
					// Note: reqCtx here is *fasthttp.RequestCtx, not *FastRequestContext
					// We need to get request ID from the FastRequestContext created in processRequest
					reqCtx.Error("Internal Server Error", fasthttp.StatusInternalServerError)
					reqCtx.SetContentType("application/json")
					// Extract request ID from header if available
					requestID := string(reqCtx.Request.Header.Peek("X-Request-ID"))
					if requestID == "" {
						requestID = "unknown"
					}
					s.Logger().Errorf("handler panic (request_id=%s): %v", requestID, r)
					reqCtx.WriteString(fmt.Sprintf(`{"error":"handler_panic","message":"Request handler failed","request_id":"%s"}`, requestID))
				}
			}()

			s.processRequest(reqCtx)
		}()
	}
}

// processRequest processes a single request
func (s *FastHTTPServer) processRequest(ctx *fasthttp.RequestCtx) {
	// Fail-fast: validate inputs
	if ctx == nil {
		panic("request context cannot be nil")
	}
	if s.Vertx() == nil {
		panic("vertx cannot be nil")
	}
	if s.router == nil {
		panic("router cannot be nil")
	}

	// Generate or extract request ID from headers
	requestID := string(ctx.Request.Header.Peek("X-Request-ID"))
	if requestID == "" {
		requestID = core.GenerateRequestID()
	}

	// Create request context with Vertx
	reqCtx := &FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         ctx,
		Vertx:              s.Vertx(),
		EventBus:           s.EventBus(),
		Params:             make(map[string]string),
		requestID:          requestID,
	}

	// Set request ID in response header for tracing
	ctx.Response.Header.Set("X-Request-ID", requestID)

	// Track request metrics
	atomic.AddInt64(&s.totalRequests, 1)

	// Route request - errors are propagated immediately (fail-fast)
	s.router.ServeFastHTTP(reqCtx)

	// Track response status
	statusCode := ctx.Response.StatusCode()
	if statusCode >= 200 && statusCode < 300 {
		atomic.AddInt64(&s.successfulRequests, 1)
	} else if statusCode >= 500 {
		atomic.AddInt64(&s.errorRequests, 1)
	}
}

// FastRequestContext wraps fasthttp RequestCtx with Fluxor context
// Extends BaseRequestContext for common data storage functionality
type FastRequestContext struct {
	*core.BaseRequestContext // Embed base context for data storage
	RequestCtx               *fasthttp.RequestCtx
	Vertx                    core.Vertx
	EventBus                 core.EventBus
	Params                   map[string]string
	requestID                string // Request ID for tracing
}

// JSON writes JSON response (default format) - fail-fast
func (c *FastRequestContext) JSON(statusCode int, data interface{}) error {
	// Fail-fast: validate status code
	if statusCode < 100 || statusCode > 599 {
		return fmt.Errorf("invalid status code: %d", statusCode)
	}

	c.RequestCtx.SetStatusCode(statusCode)
	c.RequestCtx.SetContentType("application/json")

	// Fail-fast: JSON encoding errors are propagated immediately
	jsonData, err := core.JSONEncode(data)
	if err != nil {
		return fmt.Errorf("json encode error: %w", err)
	}

	c.RequestCtx.Write(jsonData)
	return nil
}

// BindJSON binds JSON request body to a struct - fail-fast
func (c *FastRequestContext) BindJSON(v interface{}) error {
	// Fail-fast: validate target
	if v == nil {
		return fmt.Errorf("cannot bind to nil value")
	}

	body := c.RequestCtx.PostBody()
	if len(body) == 0 {
		return fmt.Errorf("empty request body")
	}

	// Fail-fast: JSON decoding errors are propagated immediately
	return core.JSONDecode(body, v)
}

// Text writes text response
func (c *FastRequestContext) Text(statusCode int, text string) error {
	c.RequestCtx.SetStatusCode(statusCode)
	c.RequestCtx.SetContentType("text/plain")
	c.RequestCtx.WriteString(text)
	return nil
}

// Query returns query parameter value
func (c *FastRequestContext) Query(key string) string {
	return string(c.RequestCtx.QueryArgs().Peek(key))
}

// Param returns path parameter value
func (c *FastRequestContext) Param(key string) string {
	return c.Params[key]
}

// Method returns HTTP method
func (c *FastRequestContext) Method() []byte {
	return c.RequestCtx.Method()
}

// Path returns request path
func (c *FastRequestContext) Path() []byte {
	return c.RequestCtx.Path()
}

// Error writes error response
func (c *FastRequestContext) Error(msg string, statusCode int) {
	c.RequestCtx.Error(msg, statusCode)
}

// RequestID returns the request ID for this request
func (c *FastRequestContext) RequestID() string {
	return c.requestID
}

// Context returns a context with request ID
func (c *FastRequestContext) Context() context.Context {
	ctx := context.Background()
	if c.requestID != "" {
		ctx = core.WithRequestID(ctx, c.requestID)
	}
	return ctx
}
