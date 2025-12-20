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

func (v *MyVerticle) Start(ctx core.Context) error {
    // Initialize verticle
    consumer := ctx.EventBus().Consumer("my.address")
    consumer.Handler(func(ctx core.Context, msg core.Message) error {
        // Handle message
        return nil
    })
    return nil
}

func (v *MyVerticle) Stop(ctx core.Context) error {
    // Cleanup
    return nil
}
```

### Event Bus

The event bus provides publish-subscribe and point-to-point messaging:

```go
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

## Usage Example

```go
package main

import (
    "context"
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/fx"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    ctx := context.Background()
    
    app, _ := fx.New(ctx,
        fx.Provide(fx.NewValueProvider("config")),
        fx.Invoke(fx.NewInvoker(setupApp)),
    )
    
    app.Start()
    defer app.Stop()
}

func setupApp(deps map[reflect.Type]interface{}) error {
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)
    
    // Deploy verticle
    vertx.DeployVerticle(&MyVerticle{})
    
    // Setup HTTP server
    server := web.NewServer(vertx, ":8080")
    server.Router().GET("/", handler)
    server.Start()
    
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
- ✅ HTTP server abstractions
- ✅ Non-blocking I/O patterns

## Installation

```bash
go get github.com/fluxorio/fluxor
```

## License

MIT

