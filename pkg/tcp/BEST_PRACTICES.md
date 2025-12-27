# TCP Server Best Practices

This document summarizes best practices found across 6 branches (`cursor/tcp-server-setup-*`) and the current main branch implementation.

## Architecture Overview

The TCP server follows a fail-fast, backpressured design that mirrors `pkg/web.FastHTTPServer`:
- **BaseServer**: Core server lifecycle management
- **Mailbox**: Bounded queue for connection handling
- **Executor**: Worker pool for concurrent processing
- **Backpressure**: Protection against overload

## Key Best Practices

### 1. Constructor Pattern

**Best Practice**: Use `core.GoCMD` instead of `core.Vertx` for consistency with the rest of the codebase.

```go
// ✅ Current main branch (recommended)
func NewTCPServer(gocmd core.GoCMD, config *TCPServerConfig) *TCPServer {
    executor := concurrency.NewExecutor(gocmd.Context(), ...)
    BaseServer: core.NewBaseServer("tcp-server", gocmd),
}

// ❌ Older branches (deprecated)
func NewTCPServer(vertx core.Vertx, config *TCPServerConfig) *TCPServer {
    executor := concurrency.NewExecutor(vertx.Context(), ...)
    BaseServer: core.NewBaseServer("tcp-server", vertx),
}
```

**Rationale**: 
- `GoCMD` is the primary runtime interface in Fluxor
- Provides better integration with the event bus and deployment system
- Consistent with HTTP server patterns

### 2. ConnContext Structure

**Best Practice**: Use `GoCMD` field in `ConnContext` for consistency.

```go
// ✅ Current main branch (recommended)
type ConnContext struct {
    *core.BaseRequestContext
    Context  context.Context
    Conn     net.Conn
    GoCMD    core.GoCMD      // ✅ Use GoCMD
    EventBus core.EventBus
    LocalAddr  net.Addr
    RemoteAddr net.Addr
}

// ❌ Older branches (deprecated)
type ConnContext struct {
    *core.BaseRequestContext
    Context  context.Context
    Conn     net.Conn
    Vertx    core.Vertx      // ❌ Deprecated
    EventBus core.EventBus
    LocalAddr  net.Addr
    RemoteAddr net.Addr
}
```

### 3. Fail-Fast Design Principles

**Best Practice**: Always validate inputs and fail immediately on invalid state.

```go
// ✅ Fail-fast on nil handler
func (s *TCPServer) SetHandler(handler ConnectionHandler) {
    if handler == nil {
        panic("tcp handler cannot be nil")
    }
    // ...
}

// ✅ Fail-fast on nil middleware
func (s *TCPServer) Use(mw ...Middleware) {
    for _, m := range mw {
        if m == nil {
            panic("tcp middleware cannot be nil")
        }
    }
    // ...
}
```

### 4. Configuration Defaults

**Best Practice**: Provide sensible defaults with validation.

```go
func DefaultTCPServerConfig(addr string) *TCPServerConfig {
    if addr == "" {
        addr = ":9000"
    }
    return &TCPServerConfig{
        Addr:         addr,
        MaxQueue:     1000,        // Bounded queue
        Workers:      50,           // Worker pool size
        MaxConns:     0,            // 0 = unlimited
        TLSConfig:    nil,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 5 * time.Second,
    }
}

// Validate and normalize in constructor
func NewTCPServer(gocmd core.GoCMD, config *TCPServerConfig) *TCPServer {
    if config == nil {
        config = DefaultTCPServerConfig(":9000")
    }
    if config.MaxQueue < 1 {
        config.MaxQueue = 100
    }
    if config.Workers < 1 {
        config.Workers = 1
    }
    // ...
}
```

### 5. Backpressure Management

**Best Practice**: Implement multi-layer backpressure protection.

```go
// Layer 1: MaxConns limit (if configured)
if !s.tryAcquireConnSlot() {
    atomic.AddInt64(&s.rejectedConnections, 1)
    _ = conn.Close()
    continue
}

// Layer 2: Backpressure controller (normal capacity)
if !s.backpressure.TryAcquire() {
    atomic.AddInt64(&s.rejectedConnections, 1)
    s.releaseConnSlot()
    _ = conn.Close()
    return
}

// Layer 3: Bounded mailbox queue
if err := s.connMailbox.Send(conn); err != nil {
    s.backpressure.Release()
    atomic.AddInt64(&s.rejectedConnections, 1)
    s.releaseConnSlot()
    _ = conn.Close()
    return
}
```

**Key Points**:
- Normal capacity = `MaxQueue + Workers` (target utilization baseline)
- Reject immediately when normal capacity exceeded (fail-fast)
- Reset backpressure metrics periodically (default: 60 seconds)

### 6. Connection Handling

**Best Practice**: Isolate panics per-connection to prevent worker termination.

```go
func (s *TCPServer) processConnFromMailbox(ctx context.Context) error {
    // ...
    atomic.AddInt64(&s.handledConnections, 1)
    func() {
        defer func() {
            if r := recover(); r != nil {
                atomic.AddInt64(&s.errorConnections, 1)
                s.Logger().Error(fmt.Sprintf("panic in tcp handler (isolated): %v", r))
            }
        }()
        if err := h(cctx); err != nil {
            atomic.AddInt64(&s.errorConnections, 1)
            s.Logger().Error(fmt.Sprintf("tcp handler error: %v", err))
        }
    }()
    
    _ = conn.Close()  // Always close connection
    s.backpressure.Release()
    s.releaseConnSlot()
}
```

**Key Points**:
- Panic isolation prevents worker goroutine termination
- Always close connection after handling
- Release backpressure and connection slot in all cases

### 7. Timeout Management

**Best Practice**: Set per-connection timeouts to prevent resource leaks.

```go
// Per-connection timeouts (best-effort)
_ = conn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
_ = conn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
```

**Rationale**:
- Prevents connections from hanging indefinitely
- Best-effort (ignore errors) to avoid breaking valid connections
- Configurable per server instance

### 8. Graceful Shutdown

**Best Practice**: Handle shutdown cleanly with atomic flags.

```go
func (s *TCPServer) doStop() error {
    atomic.StoreInt32(&s.stopping, 1)
    
    s.mu.Lock()
    ln := s.listener
    s.listener = nil
    s.mu.Unlock()
    
    if ln != nil {
        _ = ln.Close()  // Close listener to stop Accept()
    }
    
    // Wait for workers to finish (executor handles this)
    return nil
}

// In accept loop
conn, err := ln.Accept()
if err != nil {
    if atomic.LoadInt32(&s.stopping) == 1 {
        return nil  // Clean shutdown
    }
    if errors.Is(err, net.ErrClosed) {
        return nil  // Listener closed
    }
    return err
}
```

### 9. Metrics Collection

**Best Practice**: Use atomic operations for thread-safe metrics.

```go
// Metrics fields (atomic for thread-safety)
queuedConnections   int64
rejectedConnections int64
totalAccepted       int64
handledConnections  int64
errorConnections    int64
activeConns         int64

// Update metrics atomically
atomic.AddInt64(&s.totalAccepted, 1)
atomic.AddInt64(&s.queuedConnections, 1)
atomic.AddInt64(&s.handledConnections, 1)
```

### 10. Middleware Pattern

**Best Practice**: Wrap handlers with middleware, last added runs outermost.

```go
func (s *TCPServer) rebuildHandlerLocked() {
    h := s.handler
    // Wrap like web middleware: last added runs outermost
    for i := len(s.middlewares) - 1; i >= 0; i-- {
        h = s.middlewares[i](h)
    }
    s.effective = h
}
```

**Rationale**: Consistent with HTTP middleware pattern, allows chaining multiple middleware layers.

### 11. Worker Initialization

**Best Practice**: Use `sync.Once` to ensure workers start exactly once.

```go
startWorkersOnce sync.Once

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
                // Handle error
            }
        }
    })
}
```

**Rationale**: Prevents duplicate worker initialization if constructor and Start() are both called.

## Summary of Differences Across Branches

| Aspect | Main Branch (Current) | Older Branches |
|--------|----------------------|----------------|
| Constructor | `NewTCPServer(gocmd core.GoCMD, ...)` | `NewTCPServer(vertx core.Vertx, ...)` |
| ConnContext | `GoCMD core.GoCMD` | `Vertx core.Vertx` |
| Executor Context | `gocmd.Context()` | `vertx.Context()` |
| BaseServer | `core.NewBaseServer(..., gocmd)` | `core.NewBaseServer(..., vertx)` |

## Recommended Implementation

The **main branch** implementation is recommended because:
1. ✅ Uses `GoCMD` for consistency with the rest of the codebase
2. ✅ Better integration with event bus and deployment system
3. ✅ Aligns with HTTP server patterns
4. ✅ More maintainable and future-proof

## Testing Best Practices

1. **Fail-fast tests**: Test nil handler/middleware panics
2. **Backpressure tests**: Verify connection rejection when capacity exceeded
3. **Graceful shutdown**: Test clean shutdown without errors
4. **Metrics tests**: Verify atomic metric updates
5. **Timeout tests**: Verify connection timeouts work correctly

## Performance Considerations

1. **Worker pool size**: Default 50 workers balances throughput and resource usage
2. **Queue size**: Default 1000 provides buffer for connection bursts
3. **Backpressure**: Normal capacity = queue + workers (1050 default)
4. **Atomic operations**: All metrics use atomic for thread-safety without locks
5. **Connection limits**: `MaxConns: 0` means unlimited (can be set for resource protection)

