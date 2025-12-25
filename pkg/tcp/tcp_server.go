package tcp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/core/concurrency"
)

// TCPServer implements a fail-fast, backpressured TCP server.
// It mirrors pkg/web.FastHTTPServer's structure: BaseServer + Mailbox + Executor + Backpressure.
type TCPServer struct {
	*core.BaseServer

	addr   string
	config *TCPServerConfig

	mu       sync.RWMutex
	listener net.Listener
	stopping int32

	connMailbox concurrency.Mailbox
	executor    concurrency.Executor
	workers     int
	maxQueue    int

	startWorkersOnce sync.Once

	handler      ConnectionHandler
	middlewares  []Middleware
	effective    ConnectionHandler
	backpressure *BackpressureController
	maxConns     int
	activeConns  int64 // atomic: in-flight (queued + processing)

	// Metrics (atomic for thread-safety)
	queuedConnections   int64
	rejectedConnections int64
	totalAccepted       int64
	handledConnections  int64
	errorConnections    int64
}

// TCPServerConfig configures the TCP server.
type TCPServerConfig struct {
	Addr string

	// Backpressure: bounded queue + worker pool.
	MaxQueue int
	Workers  int
	// MaxConns bounds concurrent in-flight connections (queued + handling).
	// 0 means unlimited.
	MaxConns int

	// TLSConfig enables TLS when non-nil.
	TLSConfig *tls.Config

	// Connection settings.
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DefaultTCPServerConfig returns a sensible default configuration.
func DefaultTCPServerConfig(addr string) *TCPServerConfig {
	if addr == "" {
		addr = ":9000"
	}
	return &TCPServerConfig{
		Addr:         addr,
		MaxQueue:     1000,
		Workers:      50,
		MaxConns:     0,
		TLSConfig:    nil,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}
}

// NewTCPServer creates a new TCP server.
func NewTCPServer(vertx core.Vertx, config *TCPServerConfig) *TCPServer {
	if config == nil {
		config = DefaultTCPServerConfig(":9000")
	}
	if config.Addr == "" {
		config.Addr = ":9000"
	}
	if config.MaxQueue < 1 {
		config.MaxQueue = 100
	}
	if config.Workers < 1 {
		config.Workers = 1
	}
	if config.MaxConns < 0 {
		config.MaxConns = 0
	}
	if config.ReadTimeout <= 0 {
		config.ReadTimeout = 5 * time.Second
	}
	if config.WriteTimeout <= 0 {
		config.WriteTimeout = 5 * time.Second
	}

	normalCapacity := config.MaxQueue + config.Workers
	connMailbox := concurrency.NewBoundedMailbox(config.MaxQueue)
	executor := concurrency.NewExecutor(vertx.Context(), concurrency.ExecutorConfig{
		Workers:   config.Workers,
		QueueSize: config.MaxQueue,
	})

	s := &TCPServer{
		BaseServer:   core.NewBaseServer("tcp-server", vertx),
		addr:         config.Addr,
		config:       config,
		connMailbox:  connMailbox,
		executor:     executor,
		workers:      config.Workers,
		maxQueue:     config.MaxQueue,
		maxConns:     config.MaxConns,
		backpressure: NewBackpressureController(normalCapacity, 60),
		handler:      defaultConnectionHandler,
	}
	s.effective = s.handler

	// Wire BaseServer hooks (template method pattern).
	s.BaseServer.SetHooks(s.doStart, s.doStop)

	// Start workers once (constructor and Start() both might be called by different patterns).
	s.startConnWorkers()

	return s
}

func defaultConnectionHandler(ctx *ConnContext) error {
	// Default: do nothing. Connection will be closed by server.
	return nil
}

// SetHandler sets the connection handler (fail-fast on nil).
func (s *TCPServer) SetHandler(handler ConnectionHandler) {
	if handler == nil {
		panic("tcp handler cannot be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
	s.rebuildHandlerLocked()
}

// Use adds middleware to the TCP server. Best practice: call before Start().
// Fail-fast: panics if any middleware is nil.
func (s *TCPServer) Use(mw ...Middleware) {
	if len(mw) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range mw {
		if m == nil {
			panic("tcp middleware cannot be nil")
		}
		s.middlewares = append(s.middlewares, m)
	}
	s.rebuildHandlerLocked()
}

func (s *TCPServer) rebuildHandlerLocked() {
	h := s.handler
	// Wrap like web middleware: last added runs outermost.
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		h = s.middlewares[i](h)
	}
	s.effective = h
}

// ListeningAddr returns the actual listening address (useful when Addr is ":0").
// Returns empty string if not currently listening.
func (s *TCPServer) ListeningAddr() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// doStart is called by BaseServer.Start() - implements hook method.
// Note: Like FastHTTPServer, Start() is a blocking call.
func (s *TCPServer) doStart() error {
	// Ensure workers are running.
	s.startConnWorkers()

	var (
		ln  net.Listener
		err error
	)
	if s.config.TLSConfig != nil {
		ln, err = tls.Listen("tcp", s.addr, s.config.TLSConfig)
	} else {
		ln, err = net.Listen("tcp", s.addr)
	}
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()

	// Accept loop (blocking).
	for {
		conn, err := ln.Accept()
		if err != nil {
			// If we're stopping, treat "closed listener" as clean shutdown.
			if atomic.LoadInt32(&s.stopping) == 1 {
				return nil
			}
			// Some platforms wrap the "closed" error; handle that too.
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return err
		}

		atomic.AddInt64(&s.totalAccepted, 1)
		if !s.tryAcquireConnSlot() {
			atomic.AddInt64(&s.rejectedConnections, 1)
			_ = conn.Close()
			continue
		}
		s.enqueueConn(conn)
	}
}

// doStop is called by BaseServer.Stop() - implements hook method.
func (s *TCPServer) doStop() error {
	atomic.StoreInt32(&s.stopping, 1)

	s.mu.Lock()
	ln := s.listener
	s.listener = nil
	s.mu.Unlock()

	// Close listener to break Accept().
	if ln != nil {
		_ = ln.Close()
	}

	// Close mailbox and shutdown executor.
	s.connMailbox.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.executor.Shutdown(ctx)
}

// Metrics returns current server metrics.
func (s *TCPServer) Metrics() ServerMetrics {
	bp := s.backpressure.GetMetrics()
	normalCapacity := int(bp.NormalCapacity)

	queued := atomic.LoadInt64(&s.queuedConnections)
	queueUtil := float64(queued) / float64(s.maxQueue) * 100
	if s.maxQueue <= 0 {
		queueUtil = 0
	} else if queueUtil > 100.0 {
		queueUtil = 100.0
	}

	return ServerMetrics{
		QueuedConnections:   queued,
		RejectedConnections: atomic.LoadInt64(&s.rejectedConnections),
		QueueCapacity:       s.maxQueue,
		Workers:             s.workers,
		QueueUtilization:    queueUtil,
		NormalCCU:           normalCapacity,
		CurrentCCU:          int(bp.CurrentLoad),
		CCUUtilization:      bp.Utilization,
		TotalAccepted:       atomic.LoadInt64(&s.totalAccepted),
		HandledConnections:  atomic.LoadInt64(&s.handledConnections),
		ErrorConnections:    atomic.LoadInt64(&s.errorConnections),
		ActiveConnections:   atomic.LoadInt64(&s.activeConns),
		MaxConns:            s.maxConns,
	}
}

func (s *TCPServer) tryAcquireConnSlot() bool {
	// Unlimited: track active for metrics only.
	if s.maxConns <= 0 {
		atomic.AddInt64(&s.activeConns, 1)
		return true
	}
	for {
		cur := atomic.LoadInt64(&s.activeConns)
		if int(cur) >= s.maxConns {
			return false
		}
		if atomic.CompareAndSwapInt64(&s.activeConns, cur, cur+1) {
			return true
		}
	}
}

func (s *TCPServer) releaseConnSlot() {
	atomic.AddInt64(&s.activeConns, -1)
}

func (s *TCPServer) enqueueConn(conn net.Conn) {
	// Backpressure baseline: reject immediately when normal capacity exceeded.
	if !s.backpressure.TryAcquire() {
		atomic.AddInt64(&s.rejectedConnections, 1)
		s.releaseConnSlot()
		_ = conn.Close()
		return
	}

	// Bounded queue: fail-fast when queue is full.
	if err := s.connMailbox.Send(conn); err != nil {
		s.backpressure.Release()
		atomic.AddInt64(&s.rejectedConnections, 1)
		s.releaseConnSlot()
		_ = conn.Close()
		return
	}

	atomic.AddInt64(&s.queuedConnections, 1)
}

func (s *TCPServer) startConnWorkers() {
	s.startWorkersOnce.Do(func() {
		for i := 0; i < s.workers; i++ {
			task := concurrency.NewNamedTask(
				fmt.Sprintf("tcp-worker-%d", i),
				func(ctx context.Context) error {
					return s.processConnFromMailbox(ctx)
				},
			)
			if err := s.executor.Submit(task); err != nil {
				s.Logger().Errorf("failed to start tcp worker %d: %v", i, err)
			}
		}
	})
}

func (s *TCPServer) processConnFromMailbox(ctx context.Context) error {
	for {
		msg, err := s.connMailbox.Receive(ctx)
		if err != nil {
			return err
		}

		conn, ok := msg.(net.Conn)
		if !ok || conn == nil {
			// Fail-fast: unexpected mailbox payload.
			s.backpressure.Release()
			s.releaseConnSlot()
			continue
		}

		atomic.AddInt64(&s.queuedConnections, -1)

		s.mu.RLock()
		h := s.effective
		s.mu.RUnlock()

		// Per-connection timeouts (best-effort).
		_ = conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
		_ = conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))

		cctx := &ConnContext{
			BaseRequestContext: core.NewBaseRequestContext(),
			Context:            ctx,
			Conn:               conn,
			Vertx:              s.Vertx(),
			EventBus:           s.EventBus(),
			LocalAddr:          conn.LocalAddr(),
			RemoteAddr:         conn.RemoteAddr(),
		}

		// Panic isolation must be per-connection; otherwise a panic would terminate
		// the worker goroutine and stop future processing.
		atomic.AddInt64(&s.handledConnections, 1)
		func() {
			defer func() {
				if r := recover(); r != nil {
					atomic.AddInt64(&s.errorConnections, 1)
					s.Logger().Errorf("panic in tcp handler (isolated): %v", r)
				}
			}()
			if err := h(cctx); err != nil {
				atomic.AddInt64(&s.errorConnections, 1)
				s.Logger().Errorf("tcp handler error: %v", err)
			}
		}()

		_ = conn.Close()
		s.backpressure.Release()
		s.releaseConnSlot()
	}
}
