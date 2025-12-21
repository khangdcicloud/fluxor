# Fluxor Architecture

## Table of Contents

1. [Overview](#overview)
2. [Architectural Principles](#architectural-principles)
3. [System Architecture](#system-architecture)
4. [Core Components](#core-components)
5. [Component Interactions](#component-interactions)
6. [Data Flow](#data-flow)
7. [Design Patterns](#design-patterns)
8. [Concurrency Model](#concurrency-model)
9. [Error Handling & Fail-Fast](#error-handling--fail-fast)
10. [Performance Characteristics](#performance-characteristics)
11. [Observability](#observability)

---


## Overview

Fluxor is a **reactive framework for building** applications in Go, inspired by Vert.x, that provides:

- **Event-driven architecture** with a local event bus for building decoupled systems
- **Verticle-based deployment** model for building isolated components
- **Reactive workflows** with composable steps for building complex processes
- **High-performance HTTP** server using fasthttp for building web services
- **Fail-fast error handling** for building predictable systems
- **JSON-first** data format for building interoperable APIs

For a complete working example of a Fluxor application, see [`cmd/main.go`](cmd/main.go), which demonstrates the standard initialization pattern, verticle deployment, HTTP server setup, and graceful shutdown.

### Key Characteristics

- **Framework for building**: Provides building blocks, abstractions, and patterns to construct applications
- **Structural concurrency**: Enforced concurrency patterns, not accidental - build safely
- **Standalone**: Build applications without external dependencies (control-plane, mesh, etc.)
- **Fail-fast**: Build predictable systems with immediate error detection
- **Non-blocking**: Build high-performance systems with non-blocking I/O

---

## Architectural Principles

### 1. Framework for Building

Fluxor is a **framework** that provides building blocks and patterns for constructing applications:

- **Components**: Verticles, EventBus, Executors, Mailboxes - building blocks for your app
- **Abstractions**: Hide complexity - build without worrying about goroutines/channels
- **Patterns**: Event-driven, reactive, fail-fast - proven patterns for building systems
- **Tools**: HTTP server, DI, workflows - tools to build faster

Applications are **built using** Fluxor's framework components, not just deployed into a runtime. The framework provides the structure and abstractions, you build your application logic on top.
### 2. Structural Concurrency

Concurrency is **designed and enforced**, not left to chance:

- Bounded queues prevent unbounded goroutine growth
- Worker pools for blocking operations
- Reactor-based request handling
- Explicit backpressure mechanisms

### 3. Fail-Fast

Errors are detected and reported **immediately**:

- Input validation happens before processing
- Errors propagate immediately, never silently ignored
- Panics are caught and re-panicked with context
- Invalid state causes immediate failure

### 4. Message-First Design

Communication happens through **message passing**, not shared state:

- Event bus for pub/sub and point-to-point messaging
- JSON as default serialization format
- Request-reply patterns for synchronous operations
- Decoupled components through messaging

---

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│  (User Code: Verticles, Handlers, Workflows, Tasks)         │
└──────────────────────┬──────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│                    Fluxor Runtime                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │     FX       │  │   Fluxor     │  │     Web      │      │
│  │ (DI/Lifecycle)│  │  (Workflows) │  │  (HTTP/WS)  │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                 │                  │              │
│  ┌──────▼─────────────────▼──────────────────▼──────┐       │
│  │              Core Layer                          │       │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐       │       │
│  │  │  Vertx   │  │ EventBus │  │ Context  │       │       │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘       │       │
│  │       │             │              │             │       │
│  │  ┌────▼─────────────▼──────────────▼─────┐       │       │
│  │  │         Verticle Deployment            │       │       │
│  │  └───────────────────────────────────────┘       │       │
│  └──────────────────────────────────────────────────┘       │
│                                                              │
│  ┌──────────────────────────────────────────────────┐        │
│  │         Stack Manager (gostacks abstraction)   │        │
│  └──────────────────────────────────────────────────┘        │
└──────────────────────────────────────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────────────┐
│              Go Runtime & System Resources                  │
│  (Goroutines, Channels, Network I/O, File System)          │
└─────────────────────────────────────────────────────────────┘
```

---

## Application Initialization

The canonical example of Fluxor application initialization can be found in [`cmd/main.go`](cmd/main.go). This section describes the standard pattern for bootstrapping a Fluxor application.

### Initialization Pattern

Fluxor applications follow a consistent initialization pattern using the FX dependency injection framework:

```go
func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create Fluxor application with dependency injection
    app, err := fx.New(ctx,
        fx.Provide(fx.NewValueProvider("example-config")), // Optional custom providers
        fx.Invoke(fx.NewInvoker(setupApplication)),
    )
    if err != nil {
        log.Fatalf("Failed to create Fluxor app: %v", err)
    }

    // Start the application
    if err := app.Start(); err != nil {
        log.Fatalf("Failed to start Fluxor app: %v", err)
    }

    // Setup graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    // Wait for shutdown signal
    <-sigChan
    log.Println("Shutting down...")

    if err := app.Stop(); err != nil {
        log.Fatalf("Error stopping app: %v", err)
    }
}
```

### Key Points

1. **Context Management**: Always create a cancellable context for lifecycle management
2. **FX Framework**: Use `fx.New(ctx, ...)` which requires a context and returns `(*Fluxor, error)`
3. **Auto-Provision**: Vertx and EventBus are automatically provided by FX - no need to call `fx.Provide(core.NewVertx)`
4. **Invoker Pattern**: Use `fx.NewInvoker()` to wrap setup functions
5. **Error Handling**: Always check errors from `fx.New()` and `app.Start()`
6. **Graceful Shutdown**: Implement signal handling for clean shutdown

### Setup Function Pattern

The setup function receives dependencies via a type-based map:

```go
func setupApplication(deps map[reflect.Type]interface{}) error {
    // Extract dependencies using type reflection
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)
    eventBus := deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()].(core.EventBus)

    // Deploy verticles
    verticle := &ExampleVerticle{eventBus: eventBus}
    if _, err := vertx.DeployVerticle(verticle); err != nil {
        return fmt.Errorf("failed to deploy verticle: %w", err)
    }

    // Create and configure HTTP server
    config := web.CCUBasedConfigWithUtilization(":8080", 5000, 60)
    server := web.NewFastHTTPServer(vertx, config)
    
    // Setup routes
    router := server.FastRouter()
    // ... route definitions ...

    // Start server in goroutine
    go server.Start()

    return nil
}
```

### Dependencies Available

The FX framework automatically provides these dependencies to setup functions:

- **`core.Vertx`**: Main runtime coordinator
- **`core.EventBus`**: Message passing infrastructure

Additional dependencies can be provided using `fx.Provide()` with custom providers.

---

## Core Components

### 1. Vertx (`pkg/core/vertx.go`)

**Purpose**: Main entry point and runtime coordinator

**Responsibilities**:
- Manages verticle lifecycle (deploy/undeploy)
- Provides access to EventBus
- Maintains deployment registry
- Coordinates shutdown

**Key Interfaces**:
```go
type Vertx interface {
    EventBus() EventBus
    DeployVerticle(verticle Verticle) (string, error)
    UndeployVerticle(deploymentID string) error
    Close() error
    Context() context.Context
}
```

**Architecture Notes**:
- Single Vertx instance per application
- Thread-safe deployment operations
- Context propagation for cancellation

---

### 2. EventBus (`pkg/core/eventbus.go`, `pkg/core/eventbus_impl.go`)

**Purpose**: Message passing infrastructure

**Responsibilities**:
- Publish-subscribe messaging
- Point-to-point messaging
- Request-reply patterns
- Message routing and delivery

**Key Interfaces**:
```go
type EventBus interface {
    Publish(address string, body interface{}) error
    Send(address string, body interface{}) error
    Request(address string, body interface{}, timeout time.Duration) (Message, error)
    Consumer(address string) Consumer
    Close() error
}
```

**Architecture Notes**:
- **JSON-first**: All messages automatically encoded/decoded to JSON using Sonic (high-performance)
- **Fail-fast**: Address and body validation before processing
- **Mailbox abstraction**: Uses `concurrency.Mailbox` to hide channel operations
- **Bounded mailboxes**: Prevents unbounded memory growth (hides bounded channels)
- **Non-blocking**: Message delivery is non-blocking where possible (hides `select` with `default`)
- **Executor-based processing**: Uses `concurrency.Executor` internally for message processing (hides goroutines)
- **Logging**: All errors and panics are logged using Logger interface (internal implementation detail)
- **Request ID propagation**: Request IDs from context are automatically included in messages for tracing

**Message Flow**:
```
Publisher → EventBus → Consumer Mailbox → Handler
                ↓
         (Sonic JSON Encoding)
                ↓
         (Request ID Propagation)
```

---

### 3. Verticle (`pkg/core/verticle.go`)

**Purpose**: Isolated unit of deployment

**Responsibilities**:
- Encapsulate business logic
- Lifecycle management (Start/Stop)
- Event bus consumer registration
- Resource management

**Key Interfaces**:
```go
type Verticle interface {
    Start(ctx FluxorContext) error
    Stop(ctx FluxorContext) error
}

type AsyncVerticle interface {
    Verticle
    AsyncStart(ctx FluxorContext, resultHandler func(error))
    AsyncStop(ctx FluxorContext, resultHandler func(error))
}
```

**Architecture Notes**:
- **Isolation**: Each verticle is isolated from others
- **Lifecycle**: Explicit start/stop phases
- **Context**: Receives FluxorContext for runtime access
- **Async support**: Can handle asynchronous initialization

---

### 4. FluxorContext (`pkg/core/context.go`)

**Purpose**: Execution context for verticles and handlers

**Responsibilities**:
- Provide access to Vertx and EventBus
- Configuration management
- Deployment operations
- Context propagation

**Key Interfaces**:
```go
type FluxorContext interface {
    Context() context.Context
    EventBus() EventBus
    Vertx() Vertx
    Config() map[string]interface{}
    SetConfig(key string, value interface{})
    Deploy(verticle Verticle) (string, error)
    Undeploy(deploymentID string) error
}
```

**Architecture Notes**:
- **Immutable Vertx reference**: Context holds reference to Vertx
- **Mutable config**: Configuration can be modified per context
- **Fail-fast**: Nil checks and validation

---

### 5. Runtime (`pkg/fluxor/runtime.go`)

**Purpose**: Runtime abstraction over gostacks

**Responsibilities**:
- Task execution management
- Stack-based workflow execution
- Integration with Vertx
- Workflow orchestration

**Key Interfaces**:
```go
type Runtime interface {
    Start(ctx context.Context) error
    Stop() error
    Execute(task Task) error
    Deploy(verticle core.Verticle) (string, error)
    Vertx() core.Vertx
}
```

**Architecture Notes**:
- **Stack abstraction**: Uses StackManager for task execution
- **Integration**: Wraps Vertx for unified API
- **Workflow support**: Executes reactive workflows

---

### 6. FastHTTPServer (`pkg/web/fasthttp_server.go`)

**Purpose**: High-performance HTTP server

**Responsibilities**:
- HTTP request handling
- Request routing
- Worker pool management
- Backpressure handling

**Architecture**:
```
HTTP Request → Request ID Middleware (extract/generate ID)
                    ↓
              FastHTTP Server → Backpressure Check (CCU-based)
                                      ↓
                              Request Queue (bounded)
                                      ↓
                              Worker Pool (100 workers)
                                      ↓
                              Router → Handler
                                      ↓
                              JSON Response (Sonic encoding)
                                      ↓
                              Response with X-Request-ID header
                                      ↓
                              Metrics Update
```

**Health & Readiness Endpoints**:
- **`/health`**: Basic health check endpoint
  - Returns 200 with system status
  - Includes EventBus and Executor status
  - Always available for basic health monitoring
  
- **`/ready`**: Readiness check endpoint
  - Returns 200 when system is ready to accept traffic
  - Returns 503 when capacity utilization is high (≥90%)
  - Includes detailed metrics (CCU, queue utilization, request counts)
  - Used by load balancers and orchestration systems

**Metrics Structure**:
```go
type ServerMetrics struct {
    QueuedRequests     int64   // Current queued requests
    RejectedRequests   int64   // Total rejected requests (503)
    QueueCapacity      int     // Maximum queue capacity
    Workers            int     // Number of worker goroutines
    QueueUtilization   float64 // Queue utilization percentage
    NormalCCU          int     // Normal CCU capacity (target utilization)
    CurrentCCU         int     // Current CCU load
    CCUUtilization     float64 // CCU utilization percentage
    TotalRequests      int64   // Total requests processed
    SuccessfulRequests int64   // Total successful requests (200-299)
    ErrorRequests      int64   // Total error requests (500-599)
}
```

**Key Features**:
- **Bounded queue**: 10,000 request capacity (configurable)
- **Worker pool**: 100 worker goroutines (configurable)
- **CCU-based backpressure**: Two-layer backpressure system
  - Normal capacity: Operates at target utilization (e.g., 60% of max CCU)
  - Queue-based: Additional protection when queue is full
  - Returns 503 when capacity exceeded (fail-fast)
- **JSON-first**: Default response format is JSON (using Sonic for encoding)
- **Non-blocking**: Request queuing is non-blocking
- **Request ID tracking**: Automatic request ID generation/propagation via middleware
- **Health endpoints**: Built-in `/health` and `/ready` endpoints
- **Enhanced metrics**: Comprehensive request and performance metrics

**Configuration**:
```go
type FastHTTPServerConfig struct {
    Addr          string
    MaxQueue      int           // Bounded queue size
    Workers       int           // Worker goroutines
    ReadTimeout   time.Duration
    WriteTimeout  time.Duration
    MaxConns      int
    ReadBufferSize int
    WriteBufferSize int
}
```

---

### 7. FX (`pkg/fx/fx.go`)

**Purpose**: Dependency injection and lifecycle management

**Responsibilities**:
- Dependency provision
- Lifecycle management
- Application startup/shutdown
- Component wiring

**Architecture Notes**:
- **Provider pattern**: Functions that provide dependencies
- **Invoker pattern**: Functions that consume dependencies
- **Type-based injection**: Uses reflection for dependency resolution
- **Lifecycle hooks**: Start/Stop/Wait for application lifecycle

---

## Component Interactions

### Deployment Flow

```
1. Application calls Vertx.DeployVerticle(verticle)
   ↓
2. Vertx validates verticle (fail-fast)
   ↓
3. Vertx creates FluxorContext
   ↓
4. Vertx calls verticle.Start(ctx)
   ↓
5. Verticle registers EventBus consumers
   ↓
6. Verticle is added to deployment registry
   ↓
7. Deployment ID returned
```

### Message Flow (Publish)

```
1. Publisher calls EventBus.Publish(address, body)
   ↓
2. EventBus validates address and body (fail-fast)
   ↓
3. EventBus extracts request ID from context (if available)
   ↓
4. EventBus encodes body to JSON using Sonic (if needed)
   ↓
5. EventBus creates Message with request ID
   ↓
6. EventBus routes to all consumers for address
   ↓
7. Message delivered to consumer mailboxes (non-blocking)
   ↓
8. Executor processes message (hides goroutine)
   ↓
9. Consumer handlers process messages
   ↓
10. Errors/panics logged via Logger interface
```

### Message Flow (Request-Reply)

```
1. Requester calls EventBus.Request(address, body, timeout)
   ↓
2. EventBus validates inputs (fail-fast)
   ↓
3. EventBus generates reply address
   ↓
4. EventBus registers temporary reply consumer
   ↓
5. EventBus sends request message
   ↓
6. Handler receives message and calls msg.Reply(response)
   ↓
7. Reply delivered to temporary consumer
   ↓
8. Requester receives reply message
```

### HTTP Request Flow

```
1. HTTP Request arrives at FastHTTP Server
   ↓
2. Request ID middleware extracts/generates request ID
   ↓
3. Server attempts to queue request (non-blocking)
   ↓
4a. Queue full → Return 503 (backpressure)
   ↓
4b. Queued → Worker picks up request
   ↓
5. Worker creates FastRequestContext with request ID
   ↓
6. Router matches route
   ↓
7. Handler executes (can use EventBus, Vertx)
   ↓
8. Handler returns JSON response (Sonic encoding)
   ↓
9. Response sent to client with X-Request-ID header
   ↓
10. Metrics updated (total, successful, error requests)
```

---

## Data Flow

### JSON Serialization Flow

```
Application Data (struct/map/string)
    ↓
EventBus.encodeBody()
    ↓
JSONEncode() [fail-fast validation]
    ↓
Sonic.Marshal() [JIT compilation, SIMD optimizations]
    ↓
JSON bytes ([]byte)
    ↓
Message body
    ↓
Consumer receives Message
    ↓
Handler processes (can decode if needed)
    ↓
Sonic.Unmarshal() [high-performance decoding]
```

**JSON Implementation Details**:

- **Sonic Integration**: Uses `github.com/bytedance/sonic` for high-performance JSON encoding/decoding
- **Performance**: 2-3x faster than standard library (`encoding/json`)
  - Encoding: ~1289 ns/op vs ~2523 ns/op (standard library)
  - Decoding: ~1388 ns/op vs ~4228 ns/op (standard library)
- **Optimizations**: 
  - JIT (Just-In-Time) compilation for type-specific encoders/decoders
  - SIMD (Single Instruction Multiple Data) optimizations for vectorized operations
  - Internal buffer pooling and memory management
- **Fail-fast**: Input validation before encoding/decoding
- **Compatibility**: Drop-in replacement for standard JSON operations

### Configuration Flow

```
Application Startup
    ↓
FX.New() with Providers
    ↓
Dependencies resolved
    ↓
Vertx created with context
    ↓
Verticles deployed with FluxorContext
    ↓
Context.Config() available to verticles
```

---

## Design Patterns

### 1. Reactor Pattern

**Implementation**: Worker pool with bounded queue

- **Single responsibility**: Each worker processes one request at a time
- **Non-blocking**: Request queuing doesn't block
- **Backpressure**: Queue full → immediate rejection

### 2. Event-Driven Architecture

**Implementation**: EventBus with pub/sub and point-to-point

- **Decoupling**: Components communicate through messages
- **Scalability**: Multiple consumers per address
- **Flexibility**: Dynamic consumer registration

### 3. Dependency Injection

**Implementation**: FX framework

- **Type-based**: Dependencies resolved by type
- **Lifecycle-aware**: Start/Stop hooks
- **Composable**: Multiple providers and invokers

### 4. Fail-Fast Pattern

**Implementation**: Validation and error propagation

- **Early validation**: Inputs validated before processing
- **Immediate errors**: Errors returned immediately
- **No silent failures**: All errors are visible

### 5. Verticle Pattern (from Vert.x)

**Implementation**: Verticle interface and deployment

- **Isolation**: Each verticle is independent
- **Lifecycle**: Explicit start/stop
- **Deployment**: Dynamic deployment/undeployment

---

## Concurrency Model

### Concurrency Abstraction Layer

Fluxor abstracts Go's concurrency primitives (goroutines, channels, `select` statements) behind high-level APIs to improve maintainability and testability.

#### Core Abstractions (`pkg/core/concurrency/`)

1. **Task Interface**: Represents units of work
   - Hides goroutine creation behind task submission
   - Supports cancellation via context
   - `Task.Execute(ctx)` method for execution

2. **Executor Interface**: Abstracts goroutine pool management
   - Hides channel operations for task queuing
   - Provides bounded execution with backpressure
   - `Submit()`, `Shutdown()`, `Stats()` methods
   - Uses `SimpleLogger` interface internally for error logging
     - Separate from `core.Logger` to avoid import cycles between `core` and `concurrency` packages
     - Provides minimal logging interface (Error/Errorf) for internal use

3. **Mailbox Interface**: Abstracts channel operations
   - Hides `chan` type and `select` statements
   - Provides `Send()`, `Receive()`, `TryReceive()` methods
   - Bounded mailboxes with backpressure

4. **WorkerPool Interface**: Abstracts worker goroutine management
   - Hides `go func()` calls
   - Provides lifecycle management (`Start()`, `Stop()`)

#### Implementation Details

- **DefaultExecutor**: Uses channels and goroutines internally (hidden from public API)
  - Creates its own `SimpleLogger` instance for error logging
  - Logs task execution failures via logger interface
- **BoundedMailbox**: Uses channels internally (hidden from public API)
- **DefaultWorkerPool**: Manages worker goroutines internally (hidden from public API)
  - Uses `SimpleLogger` interface for error logging

#### Benefits

- **Maintainability**: Changes to concurrency model don't affect application code
- **Testability**: Mock Executor/Mailbox for unit tests
- **Portability**: Could swap implementation (e.g., use different runtime)
- **Clarity**: Application code focuses on business logic, not concurrency primitives

### Goroutine Usage (Hidden Behind Abstractions)

1. **EventBus Consumers**: Use Executor abstraction (hides goroutines) - processes messages via Executor
2. **HTTP Workers**: Use Executor abstraction (hides goroutine pool)
3. **Runtime Tasks**: Use Task/Executor pattern (hides goroutine creation)
4. **Verticle Lifecycle**: Use Executor for async operations (hides goroutines)

### Synchronization

- **Mutexes**: Used for shared state (deployments, consumers)
- **Mailbox**: Abstracted message passing (hides channels)
- **Executor**: Abstracted task execution (hides goroutines)
- **Context**: Used for cancellation and timeout

### Backpressure

- **Bounded Mailboxes**: Request mailbox, message mailboxes (hides bounded channels)
- **Immediate rejection**: Mailbox full → 503 or error (hides `select` with `default`)
- **No blocking**: All operations are non-blocking where possible (hides blocking channel operations)

---

## Error Handling & Fail-Fast

### Validation Points

1. **Input Validation**: Address, body, timeout, verticle validation
2. **State Validation**: Nil checks, empty checks
3. **Configuration Validation**: Status codes, buffer sizes

### Error Propagation

1. **Immediate Return**: Errors returned immediately, not deferred
2. **Error Wrapping**: Errors include context for debugging
3. **Panic Recovery**: Panics caught and isolated (logged, not re-panicked) - maintains system stability

### Fail-Fast Mechanisms

```go
// Example: Address validation
func (eb *eventBus) Publish(address string, body interface{}) error {
    if err := ValidateAddress(address); err != nil {
        return err  // Fail immediately
    }
    // ... continue processing
}
```

---

## Performance Characteristics

### Throughput

- **HTTP**: Designed for 100k+ RPS with CCU-based backpressure
- **EventBus**: High-throughput message passing with Executor-based processing
- **JSON**: High-performance encoding/decoding using Sonic
  - Encoding: ~1289 ns/op (2x faster than standard library)
  - Decoding: ~1388 ns/op (3x faster than standard library)
  - Parallel encoding: ~173.6 ns/op under concurrent load

### Latency

- **Non-blocking**: All I/O operations non-blocking
- **Bounded queues**: Prevent latency spikes
- **Worker pools**: Predictable processing time
- **Request ID overhead**: Minimal (<1ns per request for UUID generation)

### Memory

- **Bounded**: All queues and channels are bounded
- **Sonic optimizations**: Internal buffer pooling and memory management
- **Garbage collection**: Minimized allocations through pooling
- **Request ID**: UUID strings (36 bytes per request ID)

### Scalability

- **Horizontal**: Multiple instances can run independently
- **Vertical**: Efficient resource usage per instance
- **Backpressure**: Two-layer system (CCU + Queue) prevents resource exhaustion
- **Metrics**: Atomic counters for minimal overhead

---

## Observability

Fluxor provides comprehensive observability features for production systems, including request tracking, structured logging, health monitoring, and detailed metrics.

### Request ID Tracking

**Purpose**: Enable distributed tracing and request correlation across components

**Implementation**:
- **Context-based propagation**: Request IDs stored in `context.Context`
- **UUID generation**: Uses `github.com/google/uuid` for unique identifiers
- **Automatic propagation**: Request IDs flow through EventBus messages and HTTP requests
- **HTTP integration**: Request ID middleware extracts/generates IDs and sets `X-Request-ID` header

**Usage**:
```go
// Generate new request ID
requestID := core.GenerateRequestID()
ctx := core.WithRequestID(ctx, requestID)

// Retrieve from context
id := core.GetRequestID(ctx)

// In HTTP handlers
requestID := ctx.RequestID() // Available in FastRequestContext
```

**Benefits**:
- Trace requests across EventBus messages
- Correlate logs and errors with specific requests
- Debug distributed flows within a single process
- Minimal overhead (<1ns per request)

### Logging Infrastructure

**Purpose**: Centralized, structured logging with pluggable implementations

**Implementation**:
- **Logger interface**: Pluggable `Logger` interface (`pkg/core/logger.go`)
- **Default implementation**: Uses standard `log` package with level prefixes
- **Integration points**: EventBus, Executor, WorkerPool, FastHTTPServer
- **Log levels**: Error, Warn, Info, Debug

**Logger Interface**:
```go
type Logger interface {
    Error(args ...interface{})
    Errorf(format string, args ...interface{})
    Warn(args ...interface{})
    Warnf(format string, args ...interface{})
    Info(args ...interface{})
    Infof(format string, args ...interface{})
    Debug(args ...interface{})
    Debugf(format string, args ...interface{})
}
```

**Usage in Components**:
- **EventBus**: Logs errors and panics during message processing
- **Executor**: Logs task execution failures
- **WorkerPool**: Logs worker task failures
- **FastHTTPServer**: Logs handler panics and errors

**Custom Loggers**:
The Logger interface can be swapped with custom implementations (e.g., Zap, Logrus, structured loggers) without changing application code.

**Note on Logger Interfaces**:
- **`core.Logger`**: Full-featured logger interface used by EventBus, FastHTTPServer, and other core components
- **`concurrency.SimpleLogger`**: Minimal logger interface used by Executor and WorkerPool
  - Separate interface to avoid import cycles between `core` and `concurrency` packages
  - Provides only Error/Errorf methods for internal error logging
  - Both interfaces serve the same purpose but are kept separate for architectural reasons

### Health & Readiness Endpoints

**Purpose**: Enable orchestration systems and load balancers to monitor system health

**Endpoints**:

1. **`/health`** - Basic Health Check
   - **Status**: Always returns 200 when server is running
   - **Response**: JSON with system status
   - **Use case**: Basic liveness checks
   ```json
   {
     "status": "UP",
     "checks": [
       {"name": "eventbus", "status": "UP"},
       {"name": "executor", "status": "UP"}
     ],
     "request_id": "..."
   }
   ```

2. **`/ready`** - Readiness Check
   - **Status**: Returns 200 when ready, 503 when capacity exceeded
   - **Response**: JSON with detailed metrics
   - **Use case**: Kubernetes readiness probes, load balancer health checks
   ```json
   {
     "status": "UP",
     "metrics": {
       "queued_requests": 0,
       "rejected_requests": 0,
       "queue_capacity": 10000,
       "workers": 100,
       "queue_utilization": 0.0,
       "normal_ccu": 3000,
       "current_ccu": 0,
       "ccu_utilization": 0.0,
       "total_requests": 0,
       "successful_requests": 0,
       "error_requests": 0
     },
     "request_id": "..."
   }
   ```

**Readiness Logic**:
- Returns `UP` when CCU utilization < 90% and queue utilization < 90%
- Returns `DOWN` when approaching capacity limits
- Prevents new traffic when system is overloaded

### Metrics Collection

**Purpose**: Provide detailed performance and operational metrics

**Metrics Structure** (`ServerMetrics`):
- **Request Metrics**:
  - `TotalRequests`: Total requests received
  - `SuccessfulRequests`: Requests handled successfully (2xx)
  - `ErrorRequests`: Requests resulting in errors (5xx)
  - `RejectedRequests`: Requests rejected due to backpressure (503)
  
- **Capacity Metrics**:
  - `QueuedRequests`: Current requests in queue
  - `QueueCapacity`: Maximum queue size
  - `QueueUtilization`: Queue utilization percentage
  - `Workers`: Number of worker goroutines
  
- **CCU Metrics**:
  - `NormalCCU`: Normal capacity (target utilization, e.g., 60% of max)
  - `CurrentCCU`: Current concurrent request load
  - `CCUUtilization`: Utilization relative to normal capacity

**Access**:
```go
metrics := server.Metrics()
// Access individual metrics
totalRequests := metrics.TotalRequests
ccuUtilization := metrics.CCUUtilization
```

**Benefits**:
- Monitor system health in real-time
- Detect capacity issues before failures
- Track performance trends
- Enable autoscaling decisions

### Observability Integration Points

```
HTTP Request
    ↓
Request ID Middleware (generate/extract)
    ↓
FastHTTPServer (log errors, update metrics)
    ↓
EventBus (propagate request ID, log errors)
    ↓
Consumer Handler (access request ID from context)
    ↓
Logger (structured logging with request ID)
    ↓
Metrics (atomic counters, minimal overhead)
```

---

## Extension Points

### Premium Pattern: Base Classes

Fluxor provides **Premium Pattern** base classes following Java-style abstract base class patterns. These provide default implementations with hook methods for customization.

**Available Base Classes**:
- `BaseVerticle` - Abstract base for verticles with lifecycle management
- `BaseService` - Abstract base for request-reply services
- `BaseHandler` - Abstract base for message handlers
- `BaseComponent` - Abstract base for reusable components

**Benefits**:
- Reduced boilerplate code
- Consistent patterns across codebase
- Built-in lifecycle management
- Enterprise-grade abstractions

See `pkg/core/BASE_CLASSES.md` for detailed documentation.

### Custom Verticles (Premium Pattern)

Applications can create custom verticles using the Premium Pattern:

```go
// Using Premium Pattern (BaseVerticle)
type MyVerticle struct {
    *core.BaseVerticle
}

// Override hook method (Template Method Pattern)
func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    consumer := v.Consumer("my.address") // BaseVerticle convenience method
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        // Handle message
        return msg.Reply("processed")
    })
    return nil
}

// Optional: Override cleanup
func (v *MyVerticle) doStop(ctx core.FluxorContext) error {
    // Custom cleanup
    return nil
}

// Create and deploy
verticle := &MyVerticle{
    BaseVerticle: core.NewBaseVerticle("my-verticle"),
}
vertx.DeployVerticle(verticle)
```

### Custom Verticles (Raw Interface)

Applications can also implement the interface directly:

```go
type MyVerticle struct{}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    consumer := ctx.EventBus().Consumer("my.address")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        // Handle message
        return nil
    })
    return nil
}

func (v *MyVerticle) Stop(ctx core.FluxorContext) error {
    // Cleanup
    return nil
}
```

### Custom Workflows

Applications can create reactive workflows:

```go
step1 := fluxor.NewStep("step1", func(ctx context.Context, data interface{}) (interface{}, error) {
    // Process data
    return result, nil
})

workflow := fluxor.NewWorkflow("my-workflow", step1, step2)
workflow.Execute(ctx)
```

### Custom Handlers

Applications can create HTTP handlers:

```go
router.GETFast("/api/data", func(ctx *web.FastRequestContext) error {
    // Access EventBus, Vertx through context
    return ctx.JSON(200, data)
})
```

---

## Future Enhancements

1. **Scheduler Abstraction**: Task scheduling with delay/periodic execution
2. **Cluster EventBus**: Distributed event bus (NATS integration)
3. **Supervision**: Automatic restart strategies for failed verticles
4. **Observability**: Metrics and tracing integration
5. **Config Hot-Swap**: Runtime configuration updates
6. **Reactor Model**: Single goroutine per reactor with bounded mailbox (alternative implementation)

---

## Summary

Fluxor provides a **reactive runtime** for building high-performance, event-driven applications in Go. The architecture emphasizes:

- **Structural concurrency** through bounded queues and worker pools
- **Fail-fast** error handling for predictable behavior
- **Message-first** design for decoupled components
- **JSON-first** data format for interoperability
- **High performance** through non-blocking I/O and efficient resource usage

The framework is designed to be **standalone**, **predictable**, and **scalable**, enabling developers to build complex backend systems with confidence.

