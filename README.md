# Fluxor

A reactive framework for Go, inspired by Vert.x and Node.js async patterns.

## Quick Start (Primary Pattern)

```go
package main

import (
    "log"
    "github.com/fluxorio/fluxor/pkg/fluxor"
    "github.com/fluxorio/fluxor/pkg/core"
)

func main() {
    // 1. Create app with config
    app, err := fluxor.NewMainVerticle("config.json")
    if err != nil {
        log.Fatal(err)
    }

    // 2. Deploy verticles (order matters)
    app.DeployVerticle(NewApiVerticle())
    app.DeployVerticle(NewWorkerVerticle())

    // 3. Start and block (handles SIGINT/SIGTERM)
    app.Start()
}
```

> **📖 See [docs/PRIMARY_PATTERN.md](docs/PRIMARY_PATTERN.md) for the complete guide.**

## Why This Pattern?

| Feature | Description |
|---------|-------------|
| **Config-driven** | Load from JSON/YAML, auto-inject to verticles |
| **Lifecycle-aware** | Graceful start/stop, signal handling built-in |
| **Cluster-ready** | Swap to NATS EventBus with one option |
| **Verticle-based** | Isolated components with clear boundaries |
| **Fail-fast** | Errors propagate immediately |

## Overview

Fluxor provides:

- **Event-driven architecture** with EventBus (pub/sub, point-to-point, request-reply)
- **Verticle-based deployment** model for isolated units of work
- **n8n-like Workflow Engine** - JSON-defined, event-driven workflows
- **Reactive workflows** with composable steps
- **Future/Promise** abstractions for async operations
- **High-performance HTTP** server (FastHTTP with CCU-based backpressure)
- **Cluster EventBus** via NATS/JetStream

## Architecture

```
pkg/
  core/            - Vertx, EventBus, Verticle, FluxorContext
  fluxor/          - MainVerticle, Future/Promise, Workflows
  web/             - FastHTTPServer, Router, Backpressure
  fx/              - Dependency injection (alternative pattern)
  lite/            - Minimal implementation (~500 LOC)

examples/
  fluxor-project/  - Microservices example (API Gateway + Payment Service)
  todo-api/        - Complete REST API example
```

## Minimal (acyclic) architecture (optional)

If you want a very small dependency graph (no circular dependencies) closer to the “4 package” layout, see:
- `pkg/lite/core`: `Component`, `Bus`, `WorkerPool`, `FluxorContext`
- `pkg/lite/fx`: HTTP-friendly context helpers (`Ok`, `Error`)
- `pkg/lite/web`: `Router` + `HttpVerticle`
- `pkg/lite/fluxor`: `App` runtime (`Deploy`, `Run`)

Run the demo:

```bash
go run ./cmd/lite
```

## Lite-fast (fasthttp, high-RPS)

If your target is **hundreds of thousands of RPS**, use the fasthttp-based lite variant:

```bash
go run ./cmd/litefast
```

Load test (example with `wrk`):

```bash
wrk -t8 -c512 -d30s http://127.0.0.1:8080/ping
```

Practical notes for high RPS:
- Prefer **`Text`** responses over JSON for peak throughput.
- Keep handlers allocation-free; avoid capturing request context into goroutines.
- Set OS limits (ulimit), pin CPU, and tune `GOMAXPROCS` for your machine.

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

### Workflow Engine (n8n-like)

Build complex automation workflows using JSON definitions and EventBus:

```go
import "github.com/fluxorio/fluxor/pkg/workflow"

// Create workflow verticle
wfVerticle := workflow.NewWorkflowVerticle(&workflow.WorkflowVerticleConfig{
    HTTPAddr: ":8081", // Optional HTTP API
})

// Register custom functions
wfVerticle.RegisterFunction("processOrder", func(data interface{}) (interface{}, error) {
    order := data.(map[string]interface{})
    order["processed"] = true
    return order, nil
})

app.DeployVerticle(wfVerticle)
```

**Workflow JSON Definition:**

```json
{
  "id": "order-processing",
  "name": "Order Processing",
  "nodes": [
    {"id": "start", "type": "manual", "next": ["validate"]},
    {"id": "validate", "type": "condition",
     "config": {"field": "amount", "operator": "gt", "value": 0},
     "trueNext": ["process"], "falseNext": ["reject"]},
    {"id": "process", "type": "function",
     "config": {"function": "processOrder"},
     "next": ["notify"]},
    {"id": "notify", "type": "http",
     "config": {"url": "https://api.example.com/notify", "method": "POST"}}
  ]
}
```

**Built-in Node Types:**

| Category | Types |
|----------|-------|
| **Triggers** | `manual`, `webhook`, `schedule`, `event` |
| **Actions** | `function`, `http`, `eventbus`, `set`, `code` |
| **Flow Control** | `condition`, `switch`, `split`, `merge`, `loop`, `wait` |
| **Utility** | `filter`, `map`, `reduce`, `noop`, `error` |

**Programmatic Builder:**

```go
wf := workflow.NewWorkflowBuilder("my-workflow", "My Workflow").
    AddNode("start", "manual").Next("process").Done().
    AddNode("process", "function").
        Config(map[string]interface{}{"function": "myFunc"}).
        Retry(3).Timeout(30*time.Second).Done().
    Build()

wfVerticle.Engine().RegisterWorkflow(wf)
```

See [pkg/workflow/README.md](pkg/workflow/README.md) for complete documentation.

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
// Configure for 67% utilization (leaves 33% headroom for spikes)
maxCCU := 5000
utilizationPercent := 67
config := web.CCUBasedConfigWithUtilization(":8080", maxCCU, utilizationPercent)
server := web.NewFastHTTPServer(vertx, config)
// Server automatically returns 503 when capacity exceeded
```

## Usage Example

### Primary Pattern: MainVerticle

```go
package main

import (
    "log"
    "time"

    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/fluxor"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    app, err := fluxor.NewMainVerticle("config.json")
    if err != nil {
        log.Fatal(err)
    }

    app.DeployVerticle(NewApiVerticle())
    app.Start()
}

type ApiVerticle struct {
    server *web.FastHTTPServer
}

func NewApiVerticle() *ApiVerticle { return &ApiVerticle{} }

func (v *ApiVerticle) Start(ctx core.FluxorContext) error {
    // Get config (auto-injected from config.json)
    addr := ":8080"
    if val, ok := ctx.Config()["http_addr"].(string); ok {
        addr = val
    }

    // Create HTTP server
    v.server = web.NewFastHTTPServer(ctx.Vertx(), web.DefaultFastHTTPServerConfig(addr))
    router := v.server.FastRouter()

    // Define routes
    router.GETFast("/health", func(c *web.FastRequestContext) error {
        return c.JSON(200, map[string]any{"status": "ok"})
    })

    router.POSTFast("/api/process", func(c *web.FastRequestContext) error {
        // Use EventBus for service communication
        reply, err := c.EventBus.Request("worker.process", c.RequestCtx.PostBody(), 5*time.Second)
        if err != nil {
            return c.JSON(502, map[string]any{"error": "service_unavailable"})
        }
        return c.JSON(200, reply.Body())
    })

    go v.server.Start()
    return nil
}

func (v *ApiVerticle) Stop(ctx core.FluxorContext) error {
    if v.server != nil {
        return v.server.Stop()
    }
    return nil
}
```

### With NATS Cluster EventBus

```go
import "context"

func main() {
    app, err := fluxor.NewMainVerticleWithOptions("config.json", fluxor.MainVerticleOptions{
        EventBusFactory: func(ctx context.Context, vertx core.Vertx, cfg map[string]any) (core.EventBus, error) {
            return core.NewClusterEventBusJetStream(ctx, vertx, core.ClusterJetStreamConfig{
                URL:     cfg["nats_url"].(string),
                Prefix:  "myapp",
                Service: "api-gateway",
            })
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    app.DeployVerticle(NewApiVerticle())
    app.DeployVerticle(NewWorkerVerticle())
    app.Start()
}
```

### Alternative: FX Dependency Injection

For complex applications needing advanced DI, see [ARCHITECTURE.md](ARCHITECTURE.md#application-initialization).

## Features

### Core Features
- ✅ Event-driven messaging (pub/sub, point-to-point, request-reply)
- ✅ Verticle deployment model
- ✅ **n8n-like Workflow Engine** - JSON-defined, event-driven automation
- ✅ Reactive workflows
- ✅ Future/Promise abstractions
- ✅ Stack-based task execution
- ✅ Dependency injection
- ✅ High-performance HTTP server (FastHTTP with CCU-based backpressure)
- ✅ Non-blocking I/O patterns
- ✅ Request ID tracking and propagation
- ✅ Comprehensive metrics collection
- ✅ JSON encoding/decoding (standard library, swappable behind `core.JSONEncode/JSONDecode`)
- ✅ Structured logging infrastructure

### Enterprise Features (Production-Ready)
- ✅ **OpenTelemetry distributed tracing** - Jaeger, Zipkin, and Stdout exporters
- ✅ **Prometheus metrics export** - `/metrics` endpoint with custom metrics
- ✅ **JWT/OAuth2 authentication** - Token-based auth with customizable claims
- ✅ **RBAC authorization** - Role-based access control middleware
- ✅ **Security headers middleware** - HSTS, CSP, X-Frame-Options, etc.
- ✅ **CORS middleware** - Configurable cross-origin resource sharing
- ✅ **Rate limiting** - Token bucket algorithm with IP-based limiting
- ✅ **Configuration management** - YAML/JSON with environment variable overrides
- ✅ **Enhanced health checks** - Database, HTTP, and custom health checks
- ✅ **Express-like middleware** - Composable middleware chain
- ✅ **Database connection pooling** - HikariCP-equivalent pooling for Go
- ✅ **Structured logging** - JSON logging with contextual fields

## Installation

```bash
go get github.com/fluxorio/fluxor
```

## Quick Start

### Simple Example

```bash
# Run the basic example
go run cmd/example/main.go
```

### Load Testing

Performance testing with k6 load testing framework:

```bash
# Install k6
brew install k6  # macOS (see loadtest/README.md for other platforms)

# Run the enterprise example
go run cmd/enterprise/main.go

# Run load test (10k concurrent users)
k6 run loadtest/load-test.js

# Run spike test (sudden burst)
k6 run loadtest/spike-test.js

# Run stress test (find breaking point)
k6 run loadtest/stress-test.js
```

**Performance Benchmarks:**
- **227,000 requests/second** (single endpoint)
- **4.4µs P95 latency** under normal load
- **50,000+ RPS** sustained throughput
- **3,350 concurrent users** normal capacity (single instance)
- **10,000+ concurrent users** with horizontal scaling

See [PERFORMANCE.md](PERFORMANCE.md) for complete performance guide and tuning recommendations.

### Enterprise Example (Production-Ready)

The enterprise example demonstrates ALL production features in one comprehensive application:

```bash
# Run the enterprise example
go run cmd/enterprise/main.go

# Or build and run
go build -o enterprise cmd/enterprise/main.go
./enterprise
```

**Features demonstrated in the enterprise example:**
- OpenTelemetry distributed tracing with Jaeger
- Prometheus metrics export
- JWT authentication with token generation
- RBAC authorization (user/admin roles)
- CORS and security headers
- IP-based rate limiting
- Database connection pooling
- Structured JSON logging
- Enhanced health checks
- Express-like middleware chain
- Graceful shutdown

**Endpoints available:**
- `GET /` - Welcome page with feature list
- `GET /health` - Basic health check
- `GET /ready` - Readiness probe
- `GET /health/detailed` - Detailed health with dependency checks
- `GET /metrics` - Prometheus metrics
- `POST /api/auth/login` - Get JWT token
- `GET /api/users` - List users (requires JWT)
- `GET /api/admin/metrics` - Server metrics (requires admin role)

See [`cmd/enterprise/README.md`](cmd/enterprise/README.md) for complete documentation.

## Documentation

### Core Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - System architecture and design patterns
- **[docs/PRIMARY_PATTERN.md](docs/PRIMARY_PATTERN.md)** - Recommended MainVerticle pattern
- **[pkg/workflow/README.md](pkg/workflow/README.md)** - n8n-like Workflow Engine guide
- **[DOCUMENTATION.md](DOCUMENTATION.md)** - Complete API reference and usage guide
- **[CORE_COMPONENTS.md](CORE_COMPONENTS.md)** - Core components definition and interactions
- **[COMPONENT_FLOW.md](COMPONENT_FLOW.md)** - Component flow reference and data flow diagrams
- **[BUILD_AND_TEST.md](BUILD_AND_TEST.md)** - Build and test guide
- **[MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)** - Migration guide for Java/Node.js developers

### Enterprise Features (Day2)

- **[OBSERVABILITY.md](OBSERVABILITY.md)** - Observability guide (logging, metrics, tracing)
- **[SECURITY.md](SECURITY.md)** - Security guide (authentication, authorization, headers, CORS, rate limiting)

### Operations (Day N)

- **[DAY_N_OPERATIONS.md](DAY_N_OPERATIONS.md)** - Day N operations guide (scaling, troubleshooting, incident response, capacity planning, backup/recovery)

### Additional Guides

- **[DATABASE_POOLING.md](DATABASE_POOLING.md)** - Database connection pooling guide (HikariCP equivalent)
- **[BASE_CLASSES.md](pkg/core/BASE_CLASSES.md)** - Premium Pattern base classes documentation
- **[NODEJS_APPROACH.md](NODEJS_APPROACH.md)** - Node.js developer guide
- **[DEVX_EXPERIENCE.md](DEVX_EXPERIENCE.md)** - Developer experience guide

## License

MIT

