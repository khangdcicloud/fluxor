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

---


## 2. Update ARCHITECTURE.md - Overview Section

Replace lines 18-36 with:

## Overview

Fluxor is a **reactive framework for building** applications in Go, inspired by Vert.x, that provides:

- **Event-driven architecture** with a local event bus for building decoupled systems
- **Verticle-based deployment** model for building isolated components
- **Reactive workflows** with composable steps for building complex processes
- **High-performance HTTP** server using fasthttp for building web services
- **Fail-fast error handling** for building predictable systems
- **JSON-first** data format for building interoperable APIs

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
- **JSON-first**: All messages automatically encoded/decoded to JSON
- **Fail-fast**: Address and body validation before processing
- **Mailbox abstraction**: Uses `concurrency.Mailbox` to hide channel operations
- **Bounded mailboxes**: Prevents unbounded memory growth (hides bounded channels)
- **Non-blocking**: Message delivery is non-blocking where possible (hides `select` with `default`)

**Message Flow**:
```
Publisher → EventBus → Consumer Mailbox → Handler
                ↓
         (JSON Encoding)
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
HTTP Request → FastHTTP Server → Request Queue (bounded)
                                      ↓
                              Worker Pool (100 workers)
                                      ↓
                              Router → Handler
                                      ↓
                              JSON Response
```

**Key Features**:
- **Bounded queue**: 10,000 request capacity (configurable)
- **Worker pool**: 100 worker goroutines (configurable)
- **Backpressure**: Returns 503 when queue is full
- **JSON-first**: Default response format is JSON
- **Non-blocking**: Request queuing is non-blocking

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
3. EventBus encodes body to JSON (if needed)
   ↓
4. EventBus creates Message
   ↓
5. EventBus routes to all consumers for address
   ↓
6. Message delivered to consumer mailboxes (non-blocking)
   ↓
7. Consumer handlers process messages
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
2. Server attempts to queue request (non-blocking)
   ↓
3a. Queue full → Return 503 (backpressure)
   ↓
3b. Queued → Worker picks up request
   ↓
4. Worker creates FastRequestContext
   ↓
5. Router matches route
   ↓
6. Handler executes (can use EventBus, Vertx)
   ↓
7. Handler returns JSON response
   ↓
8. Response sent to client
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
JSON bytes ([]byte)
    ↓
Message body
    ↓
Consumer receives Message
    ↓
Handler processes (can decode if needed)
```

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

3. **Mailbox Interface**: Abstracts channel operations
   - Hides `chan` type and `select` statements
   - Provides `Send()`, `Receive()`, `TryReceive()` methods
   - Bounded mailboxes with backpressure

4. **WorkerPool Interface**: Abstracts worker goroutine management
   - Hides `go func()` calls
   - Provides lifecycle management (`Start()`, `Stop()`)

#### Implementation Details

- **DefaultExecutor**: Uses channels and goroutines internally (hidden from public API)
- **BoundedMailbox**: Uses channels internally (hidden from public API)
- **DefaultWorkerPool**: Manages worker goroutines internally (hidden from public API)

#### Benefits

- **Maintainability**: Changes to concurrency model don't affect application code
- **Testability**: Mock Executor/Mailbox for unit tests
- **Portability**: Could swap implementation (e.g., use different runtime)
- **Clarity**: Application code focuses on business logic, not concurrency primitives

### Goroutine Usage (Hidden Behind Abstractions)

1. **EventBus Consumers**: Use Mailbox abstraction (hides channels)
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
3. **Panic Recovery**: Panics caught and re-panicked with context

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

- **HTTP**: Designed for 100k+ RPS
- **EventBus**: High-throughput message passing
- **JSON**: Efficient encoding/decoding

### Latency

- **Non-blocking**: All I/O operations non-blocking
- **Bounded queues**: Prevent latency spikes
- **Worker pools**: Predictable processing time

### Memory

- **Bounded**: All queues and channels are bounded
- **Object pooling**: JSON encoders/decoders (future)
- **Garbage collection**: Minimized allocations

### Scalability

- **Horizontal**: Multiple instances can run independently
- **Vertical**: Efficient resource usage per instance
- **Backpressure**: Prevents resource exhaustion

---

## Extension Points

### Custom Verticles

Applications can create custom verticles:

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

