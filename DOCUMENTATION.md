# Fluxor Documentation

## Table of Contents

1. [Introduction](#introduction)
2. [Getting Started](#getting-started)
3. [Core Concepts](#core-concepts)
4. [EventBus](#eventbus)
5. [Verticles](#verticles)
6. [HTTP Server](#http-server)
7. [Concurrency Abstractions](#concurrency-abstractions)
8. [Workflows](#workflows)
9. [Best Practices](#best-practices)
10. [Examples](#examples)

---

## Introduction

Fluxor is a **reactive framework for building** scalable, event-driven applications in Go, inspired by Vert.x. It provides high-level abstractions and tools that make it easier to build production-ready systems without dealing with low-level concurrency primitives.

### What is Fluxor?

Fluxor is a **framework for building applications** - it provides:

- **Building blocks**: EventBus, Verticles, Executors, Mailboxes - components to build your app
- **Abstractions**: Hide Go's concurrency primitives (goroutines, channels, select) - build without complexity
- **Patterns**: Event-driven architecture, reactive workflows, fail-fast error handling - proven patterns
- **Tools**: HTTP server, dependency injection, lifecycle management - tools to build faster

Think of Fluxor as a **framework that helps you build** applications faster, safer, and more maintainably.

### Key Features

- **Event-driven architecture** with local event bus for building decoupled systems
- **Verticle-based deployment** for building isolated, scalable components
- **High-performance HTTP** server (100k+ RPS target) for building web services
- **Concurrency abstractions** for building concurrent applications without complexity
- **Fail-fast error handling** for building predictable, reliable systems
- **JSON-first** data format for building interoperable APIs
- **60% resource utilization** target for building stable, production systems

### Philosophy: Framework for Building

Fluxor is designed as a **framework for building applications**. It provides abstractions and patterns that make building complex systems easier:

**Before Fluxor (Direct Go Primitives):**
ch := make(chan Message, 100)
go func() {
    for msg := range ch {
        process(msg)
    }
}()
select {
case ch <- msg:
default:
    return ErrQueueFull
}**With Fluxor (Framework Abstractions):**
mailbox := concurrency.NewBoundedMailbox(100)
executor.Submit(concurrency.TaskFunc(func(ctx context.Context) error {
    msg, err := mailbox.Receive(ctx)
    return process(msg)
}))
mailbox.Send(msg)  // Simple, no select statementsThe framework handles the complexity, so you can focus on **building your application**.
---

## Getting Started

### Installation

```bash
go get github.com/fluxorio/fluxor
```

### Basic Example

```go
package main

import (
    "context"
    "log"
    "reflect"
    
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/fx"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create application with dependency injection
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
    
    app.Wait()
}

func setupApplication(deps map[reflect.Type]interface{}) error {
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)
    eventBus := deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()].(core.EventBus)
    
    // Create HTTP server
    config := web.CCUBasedConfigWithUtilization(":8080", 5000, 60)
    server := web.NewFastHTTPServer(vertx, config)
    
    // Setup routes
    router := server.FastRouter()
    router.GETFast("/", func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, map[string]interface{}{
            "message": "Hello from Fluxor!",
        })
    })
    
    // Start server
    go server.Start()
    
    return nil
}
```

---

## Core Concepts

### 1. Vertx

**Vertx** is the main entry point and runtime coordinator. It manages:

- Verticle lifecycle (deploy/undeploy)
- EventBus instance
- Application context

```go
ctx := context.Background()
vertx := core.NewVertx(ctx)

// Access EventBus
eventBus := vertx.EventBus()

// Deploy verticles
deploymentID, err := vertx.DeployVerticle(myVerticle)
```

### 2. EventBus

**EventBus** provides message passing infrastructure:

- **Publish**: Send message to all consumers (pub/sub)
- **Send**: Send message to one consumer (point-to-point)
- **Request**: Send message and wait for reply (request-reply)

All messages are automatically encoded/decoded to JSON.

### 3. Verticles

**Verticles** are isolated units of deployment. Each verticle:

- Has its own lifecycle (Start/Stop)
- Can register EventBus consumers
- Is isolated from other verticles

### 4. Concurrency Abstractions

Fluxor provides abstractions that hide Go's concurrency primitives:

- **Task**: Unit of work
- **Executor**: Goroutine pool manager
- **Mailbox**: Message passing (hides channels)
- **WorkerPool**: Worker goroutine manager

---

## EventBus

### Publishing Messages

```go
// Publish to all consumers
err := eventBus.Publish("user.created", map[string]interface{}{
    "userId": 123,
    "name": "John",
})

// Send to one consumer (round-robin)
err := eventBus.Send("user.process", userData)
```

### Consuming Messages

```go
consumer := eventBus.Consumer("user.created")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Get message body (automatically decoded from JSON)
    var data map[string]interface{}
    if err := msg.DecodeBody(&data); err != nil {
        return err
    }
    
    // Process message
    log.Printf("User created: %v", data)
    return nil
})
```

### Request-Reply Pattern

```go
// Send request and wait for reply
reply, err := eventBus.Request("user.get", map[string]interface{}{
    "userId": 123,
}, 5*time.Second)

if err != nil {
    return err
}

var user map[string]interface{}
if err := reply.DecodeBody(&user); err != nil {
    return err
}
```

### Reply to Request

```go
consumer := eventBus.Consumer("user.get")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    // Get request data
    var req map[string]interface{}
    msg.DecodeBody(&req)
    
    // Process and reply
    user := getUser(req["userId"].(int))
    return msg.Reply(user)
})
```

---

## Verticles

### Creating a Verticle

```go
type MyVerticle struct {
    eventBus core.EventBus
}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Access EventBus
    v.eventBus = ctx.EventBus()
    
    // Register consumers
    consumer := v.eventBus.Consumer("my.address")
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

### Deploying Verticles

```go
verticle := &MyVerticle{}
deploymentID, err := vertx.DeployVerticle(verticle)
if err != nil {
    log.Fatal(err)
}

// Later, undeploy
err = vertx.UndeployVerticle(deploymentID)
```

### Async Verticles

```go
type AsyncVerticle struct{}

func (v *AsyncVerticle) Start(ctx core.FluxorContext) error {
    return nil
}

func (v *AsyncVerticle) Stop(ctx core.FluxorContext) error {
    return nil
}

func (v *AsyncVerticle) AsyncStart(ctx core.FluxorContext, resultHandler func(error)) {
    go func() {
        // Async initialization
        time.Sleep(1 * time.Second)
        resultHandler(nil)
    }()
}

func (v *AsyncVerticle) AsyncStop(ctx core.FluxorContext, resultHandler func(error)) {
    go func() {
        // Async cleanup
        resultHandler(nil)
    }()
}
```

---

## HTTP Server

### Basic Setup

```go
// Configure for 60% utilization (5000 max CCU = 3000 normal)
config := web.CCUBasedConfigWithUtilization(":8080", 5000, 60)
server := web.NewFastHTTPServer(vertx, config)

router := server.FastRouter()
```

### Routes

```go
// GET route
router.GETFast("/api/users", func(ctx *web.FastRequestContext) error {
    users := getUsers()
    return ctx.JSON(200, users)
})

// POST route with JSON binding
router.POSTFast("/api/users", func(ctx *web.FastRequestContext) error {
    var user map[string]interface{}
    if err := ctx.BindJSON(&user); err != nil {
        return ctx.JSON(400, map[string]string{"error": "invalid json"})
    }
    
    // Process user
    created := createUser(user)
    return ctx.JSON(201, created)
})

// Path parameters
router.GETFast("/api/users/:id", func(ctx *web.FastRequestContext) error {
    id := ctx.Param("id")
    user := getUser(id)
    return ctx.JSON(200, user)
})

// Query parameters
router.GETFast("/api/search", func(ctx *web.FastRequestContext) error {
    query := ctx.Query("q")
    results := search(query)
    return ctx.JSON(200, results)
})
```

### Using EventBus in Handlers

```go
router.POSTFast("/api/events", func(ctx *web.FastRequestContext) error {
    var event map[string]interface{}
    if err := ctx.BindJSON(&event); err != nil {
        return ctx.JSON(400, map[string]string{"error": "invalid json"})
    }
    
    // Publish to EventBus
    if err := ctx.EventBus().Publish("event.received", event); err != nil {
        return ctx.JSON(500, map[string]string{"error": "failed to publish"})
    }
    
    return ctx.JSON(200, map[string]string{"status": "ok"})
})
```

### Metrics Endpoint

```go
router.GETFast("/api/metrics", func(ctx *web.FastRequestContext) error {
    metrics := server.Metrics()
    return ctx.JSON(200, map[string]interface{}{
        "queued_requests":   metrics.QueuedRequests,
        "rejected_requests": metrics.RejectedRequests,
        "normal_ccu":        metrics.NormalCCU,
        "current_ccu":       metrics.CurrentCCU,
        "ccu_utilization":   fmt.Sprintf("%.2f%%", metrics.CCUUtilization),
    })
})
```

---

## Concurrency Abstractions

### Task Interface

```go
import "github.com/fluxorio/fluxor/pkg/core/concurrency"

// Create a task
task := concurrency.TaskFunc(func(ctx context.Context) error {
    // Do work
    return nil
})

// Or with a name
task := concurrency.NewNamedTask("my-task", func(ctx context.Context) error {
    return nil
})
```

### Executor

```go
// Create executor
config := concurrency.DefaultExecutorConfig()
config.Workers = 10
config.QueueSize = 1000

executor := concurrency.NewExecutor(ctx, config)

// Submit tasks
err := executor.Submit(task)

// Submit with timeout
err := executor.SubmitWithTimeout(task, 5*time.Second)

// Get stats
stats := executor.Stats()
log.Printf("Queued: %d, Completed: %d", stats.QueuedTasks, stats.CompletedTasks)

// Shutdown
err := executor.Shutdown(ctx)
```

### Mailbox

```go
// Create mailbox
mailbox := concurrency.NewBoundedMailbox(100)

// Send message (non-blocking)
err := mailbox.Send("hello")

// Receive message (blocking)
msg, err := mailbox.Receive(ctx)

// Try receive (non-blocking)
msg, ok, err := mailbox.TryReceive()
if ok {
    // Message available
}

// Check status
capacity := mailbox.Capacity()
size := mailbox.Size()
closed := mailbox.IsClosed()

// Close mailbox
mailbox.Close()
```

### WorkerPool

```go
// Create worker pool
config := concurrency.DefaultWorkerPoolConfig()
config.Workers = 10
config.QueueSize = 1000

pool := concurrency.NewWorkerPool(ctx, config)

// Start pool
err := pool.Start()

// Submit tasks
err := pool.Submit(task)

// Stop pool
err := pool.Stop(ctx)
```

---

## Workflows

### Creating a Workflow

```go
import "github.com/fluxorio/fluxor/pkg/fluxor"

// Define steps
step1 := fluxor.NewStep("fetch-data", func(ctx context.Context, data interface{}) (interface{}, error) {
    // Fetch data
    return fetchData(), nil
})

step2 := fluxor.NewStep("process-data", func(ctx context.Context, data interface{}) (interface{}, error) {
    // Process data
    return processData(data), nil
})

step3 := fluxor.NewStep("save-data", func(ctx context.Context, data interface{}) (interface{}, error) {
    // Save data
    return saveData(data), nil
})

// Create workflow
workflow := fluxor.NewWorkflow("my-workflow", step1, step2, step3)

// Execute
result, err := workflow.Execute(ctx)
```

---

## Best Practices

### 1. Use Concurrency Abstractions

**Don't:**
```go
ch := make(chan Message, 100)
go func() {
    for msg := range ch {
        process(msg)
    }
}()
```

**Do:**
```go
mailbox := concurrency.NewBoundedMailbox(100)
executor.Submit(concurrency.TaskFunc(func(ctx context.Context) error {
    msg, err := mailbox.Receive(ctx)
    if err != nil {
        return err
    }
    return process(msg)
}))
```

### 2. Fail-Fast Error Handling

Always validate inputs immediately:

```go
func (eb *eventBus) Publish(address string, body interface{}) error {
    // Fail-fast: validate immediately
    if err := ValidateAddress(address); err != nil {
        return err
    }
    // ... continue
}
```

### 3. Use JSON for Messages

All EventBus messages are automatically JSON-encoded:

```go
// This is automatically JSON-encoded
eventBus.Publish("user.created", map[string]interface{}{
    "userId": 123,
    "name": "John",
})
```

### 4. Resource Utilization

Configure servers for 60% utilization to leave headroom:

```go
// 60% of 5000 = 3000 normal capacity
config := web.CCUBasedConfigWithUtilization(":8080", 5000, 60)
```

### 5. Graceful Shutdown

Always shutdown components gracefully:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := server.Stop(); err != nil {
    log.Printf("Shutdown error: %v", err)
}

if err := executor.Shutdown(ctx); err != nil {
    log.Printf("Executor shutdown error: %v", err)
}
```

### 6. Panic Isolation

Panics in handlers are isolated and don't crash the system:

```go
// Panic in handler returns 500, doesn't crash server
router.GETFast("/api/data", func(ctx *web.FastRequestContext) error {
    // If this panics, server continues running
    data := getData()
    return ctx.JSON(200, data)
})
```

---

## Examples

### Complete Application

```go
package main

import (
    "context"
    "log"
    "reflect"
    "time"
    
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/fx"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create application with dependency injection
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
    
    app.Wait()
}

func setupApplication(deps map[reflect.Type]interface{}) error {
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)
    eventBus := deps[reflect.TypeOf((*core.EventBus)(nil)).Elem()].(core.EventBus)
    
    // Deploy verticle
    verticle := &UserServiceVerticle{eventBus: eventBus}
    if _, err := vertx.DeployVerticle(verticle); err != nil {
        return err
    }
    
    // Create HTTP server
    config := web.CCUBasedConfigWithUtilization(":8080", 5000, 60)
    server := web.NewFastHTTPServer(vertx, config)
    router := server.FastRouter()
    
    // Routes
    router.GETFast("/api/users/:id", func(ctx *web.FastRequestContext) error {
        id := ctx.Param("id")
        
        // Request via EventBus
        reply, err := ctx.EventBus().Request("user.get", map[string]interface{}{
            "id": id,
        }, 5*time.Second)
        
        if err != nil {
            return ctx.JSON(500, map[string]string{"error": err.Error()})
        }
        
        var user map[string]interface{}
        if err := reply.DecodeBody(&user); err != nil {
            return ctx.JSON(500, map[string]string{"error": "decode failed"})
        }
        
        return ctx.JSON(200, user)
    })
    
    // Start server
    go server.Start()
    
    return nil
}

type UserServiceVerticle struct {
    eventBus core.EventBus
}

func (v *UserServiceVerticle) Start(ctx core.FluxorContext) error {
    consumer := ctx.EventBus().Consumer("user.get")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        var req map[string]interface{}
        msg.DecodeBody(&req)
        
        // Get user
        user := getUser(req["id"].(string))
        
        // Reply
        return msg.Reply(user)
    })
    return nil
}

func (v *UserServiceVerticle) Stop(ctx core.FluxorContext) error {
    return nil
}

func getUser(id string) map[string]interface{} {
    return map[string]interface{}{
        "id":   id,
        "name": "John Doe",
    }
}
```

---

## API Reference

### Core Interfaces

#### Vertx
```go
type Vertx interface {
    EventBus() EventBus
    DeployVerticle(verticle Verticle) (string, error)
    UndeployVerticle(deploymentID string) error
    Close() error
    Context() context.Context
}
```

#### EventBus
```go
type EventBus interface {
    Publish(address string, body interface{}) error
    Send(address string, body interface{}) error
    Request(address string, body interface{}, timeout time.Duration) (Message, error)
    Consumer(address string) Consumer
    Close() error
}
```

#### Verticle
```go
type Verticle interface {
    Start(ctx FluxorContext) error
    Stop(ctx FluxorContext) error
}
```

#### Concurrency Abstractions

**Task**
```go
type Task interface {
    Execute(ctx context.Context) error
    Name() string
}
```

**Executor**
```go
type Executor interface {
    Submit(task Task) error
    SubmitWithTimeout(task Task, timeout time.Duration) error
    Shutdown(ctx context.Context) error
    Stats() ExecutorStats
}
```

**Mailbox**
```go
type Mailbox interface {
    Send(msg interface{}) error
    Receive(ctx context.Context) (interface{}, error)
    TryReceive() (interface{}, bool, error)
    Close()
    Capacity() int
    Size() int
    IsClosed() bool
}
```

---

## Summary

Fluxor provides a **reactive runtime** for building high-performance, event-driven applications in Go. Key takeaways:

- **Abstract concurrency**: Use Task, Executor, Mailbox instead of goroutines/channels
- **Event-driven**: Use EventBus for component communication
- **Verticle-based**: Deploy isolated components
- **Fail-fast**: Errors detected and reported immediately
- **JSON-first**: All messages automatically JSON-encoded
- **60% utilization**: Configure for stability with headroom

For more details, see [ARCHITECTURE.md](ARCHITECTURE.md).

