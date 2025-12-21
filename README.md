# Fluxor

A reactive framework and runtime abstraction for Go, inspired by Vert.x reactive patterns.

## Overview

Fluxor is a reactive programming framework that provides:

- **Event-driven architecture** with an event bus for pub/sub and point-to-point messaging
- **Verticle-based deployment** model for isolated units of work
- **Reactive workflows** with composable steps
- **Future/Promise** abstractions for asynchronous operations
- **Stack-based task execution** (abstraction over gostacks)
- **Dependency injection** and lifecycle management
- **Web abstractions** (not a web framework, but provides HTTP server abstractions)

## Architecture

```
cmd/
  main.go          - Application entry point

pkg/
  core/            - Core abstractions (EventBus, Verticle, Context, Vertx)
  fx/              - Dependency injection and lifecycle management
  web/             - HTTP/WebSocket abstractions
  fluxor/          - Main framework with runtime abstraction over gostacks
```

## Core Concepts

### Verticles

Verticles are isolated units of deployment, similar to Vert.x verticles:

```go
type MyVerticle struct{}

func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // Initialize verticle
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

### Event Bus

The event bus provides publish-subscribe and point-to-point messaging:

```go
import (
    "time"
    "github.com/fluxorio/fluxor/pkg/core"
)

// Publish (broadcast)
eventBus.Publish("news.feed", "Breaking news!")

// Send (point-to-point)
eventBus.Send("processor.queue", data)

// Request-Reply
msg, err := eventBus.Request("service.address", request, 5*time.Second)
```

### Reactive Workflows

Create composable reactive workflows:

```go
step1 := fluxor.NewStep("step1", func(ctx context.Context, data interface{}) (interface{}, error) {
    // Process data
    return result, nil
})

step2 := fluxor.NewStep("step2", func(ctx context.Context, data interface{}) (interface{}, error) {
    // Process with previous result
    return result, nil
})

workflow := fluxor.NewWorkflow("my-workflow", step1, step2)
workflow.Execute(ctx)
```

### Futures and Promises

Handle asynchronous operations:

```go
future := fluxor.NewFuture()

future.OnSuccess(func(result interface{}) {
    // Handle success
})

future.OnFailure(func(err error) {
    // Handle error
})

// Complete the future
future.Complete("result")

// Or use a promise
promise := fluxor.NewPromise()
promise.Complete("result")
```

### Runtime

The runtime manages task execution and verticle deployment:

```go
runtime := fluxor.NewRuntime(ctx)

// Deploy verticle
runtime.Deploy(verticle)

// Execute task
task := &MyTask{}
runtime.Execute(task)
```

### Request ID Tracking

Fluxor automatically tracks request IDs for observability:

```go
// Request ID is automatically generated/extracted and propagated
// No middleware needed - handled automatically by FastHTTPServer

// Access request ID in handlers
router.GETFast("/api/data", func(ctx *web.FastRequestContext) error {
    requestID := ctx.RequestID()
    // Request ID is automatically included in EventBus messages
    // and set in X-Request-ID response header
    return ctx.JSON(200, map[string]interface{}{
        "data": data,
        "request_id": requestID,
    })
})
```

### Health & Metrics

Built-in health and metrics endpoints:

```go
// Health check endpoint
router.GETFast("/health", func(ctx *web.FastRequestContext) error {
    return ctx.JSON(200, map[string]interface{}{
        "status": "UP",
    })
})

// Readiness check with metrics
router.GETFast("/ready", func(ctx *web.FastRequestContext) error {
    metrics := server.Metrics()
    ready := metrics.QueueUtilization < 90.0 && metrics.CCUUtilization < 90.0
    statusCode := 200
    if !ready {
        statusCode = 503
    }
    return ctx.JSON(statusCode, map[string]interface{}{
        "ready": ready,
        "metrics": metrics,
    })
})

// Detailed metrics endpoint
router.GETFast("/api/metrics", func(ctx *web.FastRequestContext) error {
    metrics := server.Metrics()
    return ctx.JSON(200, metrics)
})
```

### CCU-based Backpressure

Configure server capacity with utilization targets:

```go
// Configure for 60% utilization (leaves 40% headroom for spikes)
maxCCU := 5000
utilizationPercent := 60
config := web.CCUBasedConfigWithUtilization(":8080", maxCCU, utilizationPercent)
server := web.NewFastHTTPServer(vertx, config)
// Server automatically returns 503 when capacity exceeded
```

## Usage Example

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "reflect"
    "syscall"
    "time"

    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/fx"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create Fluxor application
    app, err := fx.New(ctx,
        fx.Provide(fx.NewValueProvider("example-config")),
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

func setupApplication(deps map[reflect.Type]interface{}) error {
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)

    // Deploy verticle
    verticle := &MyVerticle{}
    if _, err := vertx.DeployVerticle(verticle); err != nil {
        return err
    }

    // Create FastHTTP server with CCU-based backpressure
    maxCCU := 5000
    utilizationPercent := 60
    config := web.CCUBasedConfigWithUtilization(":8080", maxCCU, utilizationPercent)
    server := web.NewFastHTTPServer(vertx, config)

    // Setup routes (request ID is automatically handled)
    router := server.FastRouter()

    // Simple JSON endpoint
    router.GETFast("/", func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, map[string]interface{}{
            "message": "Hello from Fluxor!",
            "request_id": ctx.RequestID(),
        })
    })

    // Health check endpoint
    router.GETFast("/health", func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, map[string]interface{}{
            "status": "UP",
        })
    })

    // Readiness check endpoint
    router.GETFast("/ready", func(ctx *web.FastRequestContext) error {
        metrics := server.Metrics()
        ready := metrics.QueueUtilization < 90.0 && metrics.CCUUtilization < 90.0
        statusCode := 200
        if !ready {
            statusCode = 503
        }
        return ctx.JSON(statusCode, map[string]interface{}{
            "ready": ready,
            "metrics": metrics,
        })
    })

    // Start server
    go func() {
        log.Printf("Starting server on %s", config.Addr)
        if err := server.Start(); err != nil {
            log.Printf("Server error: %v", err)
        }
    }()

    return nil
}
```

## Features

- ✅ Event-driven messaging (pub/sub, point-to-point, request-reply)
- ✅ Verticle deployment model
- ✅ Reactive workflows
- ✅ Future/Promise abstractions
- ✅ Stack-based task execution
- ✅ Dependency injection
- ✅ High-performance HTTP server (FastHTTP with CCU-based backpressure)
- ✅ Non-blocking I/O patterns
- ✅ Request ID tracking and propagation
- ✅ Health and readiness endpoints
- ✅ Comprehensive metrics collection
- ✅ High-performance JSON encoding/decoding (Sonic)
- ✅ Structured logging infrastructure
- ✅ **Prometheus metrics export** (Day2)
- ✅ **OpenTelemetry distributed tracing** (Day2)
- ✅ **JWT/OAuth2 authentication** (Day2)
- ✅ **RBAC authorization** (Day2)
- ✅ **Security headers middleware** (Day2)
- ✅ **CORS middleware** (Day2)
- ✅ **Rate limiting** (Day2)
- ✅ **Configuration management** (Day2)
- ✅ **Enhanced health checks** (Day2)
- ✅ **Express-like middleware** (Day2)

## Installation

```bash
go get github.com/fluxorio/fluxor
```

## Documentation

### Core Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System architecture and design patterns
- **[DOCUMENTATION.md](DOCUMENTATION.md)** - Complete API reference and usage guide
- **[CORE_COMPONENTS.md](CORE_COMPONENTS.md)** - Core components definition and interactions
- **[COMPONENT_FLOW.md](COMPONENT_FLOW.md)** - Component flow reference and data flow diagrams
- **[BUILD_AND_TEST.md](BUILD_AND_TEST.md)** - Build and test guide
- **[MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)** - Migration guide for Java/Node.js developers

### Enterprise Features (Day2)

- **[OBSERVABILITY.md](OBSERVABILITY.md)** - Observability guide (logging, metrics, tracing)
- **[SECURITY.md](SECURITY.md)** - Security guide (authentication, authorization, headers, CORS, rate limiting)

### Additional Guides

- **[DATABASE_POOLING.md](DATABASE_POOLING.md)** - Database connection pooling guide (HikariCP equivalent)
- **[BASE_CLASSES.md](pkg/core/BASE_CLASSES.md)** - Premium Pattern base classes documentation
- **[NODEJS_APPROACH.md](NODEJS_APPROACH.md)** - Node.js developer guide
- **[DEVX_EXPERIENCE.md](DEVX_EXPERIENCE.md)** - Developer experience guide

## License

MIT

