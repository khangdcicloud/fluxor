# Fluxor Primary Pattern

> **This is the recommended way to build Fluxor applications.**

## The Pattern

```go
package main

import (
    "context"
    "log"

    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/fluxor"
)

func main() {
    // 1. Create app with config
    app, err := fluxor.NewMainVerticle("config.json")
    if err != nil {
        log.Fatal(err)
    }

    // 2. Deploy verticles
    app.DeployVerticle(NewMyVerticle())

    // 3. Start and block
    app.Start()
}
```

## Why This Pattern?

| Principle | Implementation |
|-----------|----------------|
| **Config-driven** | Load from JSON/YAML, inject into verticles |
| **Lifecycle-aware** | Graceful start/stop, signal handling |
| **Cluster-ready** | Swap EventBus to NATS with one option |
| **Verticle-based** | Isolated components with clear boundaries |
| **Fail-fast** | Errors propagate immediately |

---

## Full Example

### 1. main.go

```go
package main

import (
    "context"
    "log"

    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/fluxor"
)

func main() {
    // Option A: Simple (in-memory EventBus)
    app, err := fluxor.NewMainVerticle("config.json")
    
    // Option B: Clustered (NATS JetStream EventBus)
    app, err := fluxor.NewMainVerticleWithOptions("config.json", fluxor.MainVerticleOptions{
        EventBusFactory: func(ctx context.Context, vertx core.Vertx, cfg map[string]any) (core.EventBus, error) {
            return core.NewClusterEventBusJetStream(ctx, vertx, core.ClusterJetStreamConfig{
                URL:     cfg["nats_url"].(string),
                Prefix:  "myapp",
                Service: "my-service",
            })
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    // Deploy order matters: dependencies first
    app.DeployVerticle(NewDatabaseVerticle())
    app.DeployVerticle(NewCacheVerticle())
    app.DeployVerticle(NewApiVerticle())

    // Blocks until SIGINT/SIGTERM
    app.Start()
}
```

### 2. config.json

```json
{
  "http_addr": ":8080",
  "database": {
    "host": "localhost",
    "port": 5432,
    "name": "mydb"
  },
  "nats_url": "nats://localhost:4222"
}
```

### 3. Verticle Pattern

```go
package verticles

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/web"
)

type ApiVerticle struct {
    server *web.FastHTTPServer
}

func NewApiVerticle() *ApiVerticle {
    return &ApiVerticle{}
}

func (v *ApiVerticle) Start(ctx core.FluxorContext) error {
    // 1. Access runtime from FluxorContext
    vertx := ctx.Vertx()
    bus := ctx.EventBus()
    cfg := ctx.Config()

    // 2. Get config (injected automatically)
    addr := ":8080"
    if val, ok := cfg["http_addr"].(string); ok {
        addr = val
    }

    // 3. Create HTTP server
    v.server = web.NewFastHTTPServer(vertx, web.DefaultFastHTTPServerConfig(addr))
    router := v.server.FastRouter()

    // 4. Define routes
    router.GETFast("/health", func(c *web.FastRequestContext) error {
        return c.JSON(200, map[string]any{"status": "ok"})
    })

    router.POSTFast("/api/action", func(c *web.FastRequestContext) error {
        // Use EventBus for service communication
        reply, err := bus.Request("service.action", c.RequestCtx.PostBody(), 5*time.Second)
        if err != nil {
            return c.JSON(502, map[string]any{"error": "service_unavailable"})
        }
        return c.JSON(200, reply.Body())
    })

    // 5. Start server (non-blocking)
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

### 4. Service Verticle Pattern

```go
type PaymentVerticle struct{}

func NewPaymentVerticle() *PaymentVerticle {
    return &PaymentVerticle{}
}

func (v *PaymentVerticle) Start(ctx core.FluxorContext) error {
    bus := ctx.EventBus()

    // Register EventBus consumer for Request/Reply
    bus.Consumer("payments.authorize").Handler(func(c core.FluxorContext, msg core.Message) error {
        // Decode request
        var req PaymentRequest
        if err := core.JSONDecode(msg.Body().([]byte), &req); err != nil {
            return msg.Reply(PaymentResponse{OK: false, Error: "invalid_request"})
        }

        // Process
        result := processPayment(req)

        // Reply
        return msg.Reply(result)
    })

    return nil
}

func (v *PaymentVerticle) Stop(ctx core.FluxorContext) error {
    return nil
}
```

---

## Key Principles

### 1. One App, Many Verticles

```
┌─────────────────────────────────────────────────────────────┐
│  fluxor.NewMainVerticle("config.json")                      │
│  ├── Vertx (runtime)                                        │
│  ├── EventBus (messaging)                                   │
│  └── Config (injected to all verticles)                     │
└─────────────────────────────────────────────────────────────┘
              │
              ├── Deploy: DatabaseVerticle
              ├── Deploy: CacheVerticle
              ├── Deploy: ApiVerticle
              └── Deploy: WorkerVerticle
```

### Interface-first & Contract Specs (Code is Truth)

When you add a new module (e.g. `pkg/tcp`), follow:
- **Interface-first** (define `pkg/<module>/server.go` / interfaces first)
- **Contract spec** in `docs/clarity/*-contract.md`
- **Tests-as-contract** to keep docs aligned with real behavior

See:
- `docs/clarity/06-interface-first-and-contracts.md`
- `docs/clarity/05-tcp-contract.md`

### 2. Config Injection

```go
// In main.go
app, _ := fluxor.NewMainVerticle("config.json")

// In any verticle
func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    cfg := ctx.Config()
    dbHost := cfg["database"].(map[string]any)["host"].(string)
    // Config is automatically available
}
```

### 3. EventBus Communication

```
┌─────────────┐     Request      ┌─────────────┐
│ ApiVerticle │ ───────────────► │ PaymentSvc  │
│             │ ◄─────────────── │             │
└─────────────┘     Reply        └─────────────┘
       │
       │ Publish
       ▼
┌─────────────┐
│ LogVerticle │  (fire-and-forget)
└─────────────┘
```

### 4. Graceful Shutdown

```go
app.Start()  // Blocks here

// On SIGINT/SIGTERM:
// 1. All verticles Stop() called
// 2. EventBus closed
// 3. Context cancelled
// 4. App exits
```

---

## Comparison: This Pattern vs Others

### ❌ Don't: Manual Vertx Creation

```go
// DON'T do this
func main() {
    ctx := context.Background()
    vertx := core.NewVertx(ctx)
    bus := vertx.EventBus()
    // Manual config loading...
    // Manual signal handling...
    // Easy to miss cleanup...
}
```

### ✅ Do: MainVerticle Pattern

```go
// DO this
func main() {
    app, _ := fluxor.NewMainVerticle("config.json")
    app.DeployVerticle(NewMyVerticle())
    app.Start()  // Everything handled
}
```

### ❌ Don't: FX Dependency Injection

```go
// DON'T do this for simple apps
func main() {
    app, _ := fx.New(ctx,
        fx.Provide(...),
        fx.Invoke(...),
    )
    // Complex, hard to reason about
}
```

### ✅ Do: Explicit Deploy Order

```go
// DO this
func main() {
    app, _ := fluxor.NewMainVerticle("config.json")
    
    // Dependencies first
    app.DeployVerticle(NewDatabaseVerticle())
    app.DeployVerticle(NewCacheVerticle())
    
    // Then dependents
    app.DeployVerticle(NewApiVerticle())
    
    app.Start()
}
```

---

## File Structure

```
myapp/
├── cmd/
│   └── main.go           # MainVerticle bootstrap
├── config.json           # Configuration
├── verticles/
│   ├── api.go           # HTTP endpoints
│   ├── database.go      # Database connections
│   └── worker.go        # Background processing
├── contracts/
│   └── events.go        # EventBus message types
└── go.mod
```

---

## Quick Start

1. **Create main.go**:
   ```go
   app, _ := fluxor.NewMainVerticle("config.json")
   app.DeployVerticle(NewApiVerticle())
   app.Start()
   ```

2. **Create config.json**:
   ```json
   {"http_addr": ":8080"}
   ```

3. **Create verticle**:
   ```go
   type ApiVerticle struct{}
   func (v *ApiVerticle) Start(ctx core.FluxorContext) error { ... }
   func (v *ApiVerticle) Stop(ctx core.FluxorContext) error { return nil }
   ```

4. **Run**:
   ```bash
   go run .
   ```

---

## See Also

- [`examples/fluxor-project/`](../examples/fluxor-project/) - Full microservices example
- [`examples/todo-api/`](../examples/todo-api/) - Complete REST API example
- [`ARCHITECTURE.md`](../ARCHITECTURE.md) - Deep dive into internals
