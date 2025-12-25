## Fluxor architecture overview (big picture)

This document is an onboarding-friendly overview of how the repo fits together: **entrypoints → runtime/core → EventBus → modules → examples**.

### Repo layout (what lives where)

- **`cmd/`**: runnable entrypoints (examples/binaries)
  - `cmd/main.go`: “kitchen-sink” example using DI + FastHTTPServer + metrics
  - `cmd/enterprise/main.go`: production-style example (security/otel/prometheus/db)
  - `cmd/lite/`, `cmd/litefast/`: minimal/fast variants built on `pkg/lite/*`
- **`pkg/`**: the framework modules
  - `pkg/core`: **Vertx**, **Verticle**, **FluxorContext**, **EventBus**, validation, request-id, concurrency wrappers
  - `pkg/web`: **FastHTTPServer** + **FastRouter** + middleware primitives
  - `pkg/fx`: DI/lifecycle bootstrap (constructs Vertx + injects dependencies)
  - `pkg/fluxor`: workflow/runtime utilities + `MainVerticle` bootstrap
  - `pkg/db`: database pool/component
  - `pkg/observability/*`: OpenTelemetry + Prometheus integrations
  - `pkg/lite/*`: smaller acyclic module graph (core/fx/web/fluxor)
- **`examples/`**: example apps built on top of the framework
  - `examples/fluxor-project`: multi-process example using clustered EventBus (NATS/JetStream)
  - `examples/todo-api`: a fuller web API example

### Terminology (standardized)

For consistency, this repo standardizes names in `TERMINOLOGY.md`. Short version:

- **Vertx**: the runtime instance (`core.Vertx`)
- **Verticle**: a deployable unit (`core.Verticle`)
- **FluxorContext**: runtime context passed to Verticles and EventBus handlers (`core.FluxorContext`)
- **EventBus**: messaging abstraction (`core.EventBus`)
- **FastHTTPServer/FastRouter/FastRequestContext**: HTTP layer (`pkg/web`)
- **Request ID**: “Request ID”, header `X-Request-ID`, code `requestID`

Note: there is **no Redux-like Dispatcher/Store** concept in `pkg/` today. The closest analog to “dispatch” is **publishing/sending** on the **EventBus**.

---

## Big picture: request/message flow

### One diagram (Mermaid)

```mermaid
flowchart TD
  %% Entry points
  subgraph EP[Entrypoints (cmd/*, examples/*)]
    EP1[cmd/main.go<br/>fx.New + FastHTTPServer]
    EP2[cmd/enterprise/main.go<br/>fx.New + middleware + db + otel/prometheus]
    EP3[examples/fluxor-project/*<br/>fluxor.MainVerticle + clustered EventBus]
  end

  %% Bootstrap
  EP1 --> FX[fx.New(ctx)<br/>DI + lifecycle]
  EP2 --> FX
  EP3 --> MV[fluxor.NewMainVerticleWithOptions(configPath, opts)]

  %% Core runtime
  FX --> VX[core.NewVertxWithOptions(ctx, VertxOptions)]
  MV --> VX

  VX --> EB[Vertx.EventBus()<br/>core.EventBus]
  VX --> DEP[Vertx.DeployVerticle(v)]
  DEP --> VSTART[Verticle.Start(FluxorContext)]
  VSTART -->|register consumers| EB

  %% HTTP
  VX --> HTTP[web.NewFastHTTPServer(vertx, config)]
  HTTP --> ROUTER[FastRouter + middleware]
  ROUTER --> HANDLER[FastRequestHandler]
  HANDLER -->|publish/send/request| EB

  %% EventBus options
  EB -->|default| MEM[In-memory EventBus<br/>(pkg/core/eventbus_impl.go)]
  EB -->|optional via EventBusFactory| CLUSTER[Clustered EventBus<br/>(NATS / JetStream)]
  CLUSTER --> NATS[(NATS / JetStream)]
```

### The same flow in words

- **Entrypoint** (a `main()` under `cmd/*` or an example under `examples/*`)
  - chooses a bootstrap style:
    - **DI bootstrap** via `pkg/fx` (see `cmd/main.go`, `cmd/enterprise/main.go`)
    - **MainVerticle bootstrap** via `pkg/fluxor` (see `examples/fluxor-project/*`)
- **Runtime creation**
  - all roads lead to **`core.NewVertxWithOptions(...)`**, which constructs a **Vertx** and its **EventBus**
  - `VertxOptions.EventBusFactory` can swap the EventBus implementation (in-memory vs clustered)
- **Deployment**
  - `Vertx.DeployVerticle(v)` creates a **FluxorContext** and calls `v.Start(ctx)`
  - Verticles typically register EventBus consumers and/or start servers
- **I/O (HTTP)**
  - `web.NewFastHTTPServer(vertx, ...)` creates FastHTTPServer and a FastRouter
  - request handling creates a **FastRequestContext**, extracts/generates **Request ID**, and routes through middleware/handlers
  - handlers can call EventBus (`Publish`, `Send`, `Request`) and return JSON responses

---

## Entrypoints: what to run and what they demonstrate

### `cmd/main.go` (simple + demonstrative)

- Uses `fx.New(ctx, ...)` to build the app and inject **Vertx** and **EventBus**.
- Deploys an example Verticle.
- Starts a **FastHTTPServer** with CCU/backpressure config and exposes:
  - `/`, `/api/status`, `/api/echo`
  - `/health` (alias), `/live`, `/ready`
  - `/metrics` (Prometheus handler)

### `cmd/enterprise/main.go` (production-style wiring)

- Adds:
  - structured logging (`core.NewJSONLogger()`)
  - config load + env overrides (`pkg/config`)
  - OpenTelemetry init (`pkg/observability/otel`)
  - Prometheus metrics endpoint (`pkg/observability/prometheus`)
  - middleware chain: recovery, logging, CORS, headers, rate-limit, compression, auth/RBAC
  - database component/pool (`pkg/db`)
- Still uses the same core primitives: **Vertx**, **Verticle**, **EventBus**, **FastHTTPServer**.

### `examples/fluxor-project` (multi-service messaging over NATS/JetStream)

- Uses `fluxor.NewMainVerticleWithOptions(...)` to:
  - load config from file
  - create **Vertx** with an `EventBusFactory` returning **JetStream-backed clustered EventBus**
  - deploy service Verticles and block on signals (`Start()`)
- Demonstrates cross-process request/reply:
  - api-gateway receives HTTP → `EventBus.Request("payments.authorize", ...)`
  - payment-service consumes `"payments.authorize"` and `Reply(...)`

---

## Runtime/core: the stable “center”

### Vertx (`pkg/core/vertx.go`)

- Owns the application root `context.Context`
- Manages Verticle lifecycle:
  - `DeployVerticle(v)` / `UndeployVerticle(id)` / `Close()`
- Exposes the configured EventBus via `EventBus()`
- Supports swapping EventBus via `VertxOptions.EventBusFactory`

### FluxorContext (`pkg/core/context.go`)

- Passed into Verticles and EventBus handlers
- Provides:
  - `Context()` (Go context)
  - `Vertx()` and `EventBus()`
  - `Config()` key/value map (used heavily by `fluxor.MainVerticle` config injection)

### Verticle (`pkg/core/verticle.go`)

- A deployable unit: `Start(ctx)` and `Stop(ctx)`
- Pattern: a Verticle starts consumers, servers, workflows, etc.

---

## EventBus: how messages move

### Interface (common)

`core.EventBus` supports:
- `Publish(address, body)` (fanout to all consumers)
- `Send(address, body)` (point-to-point / queue semantics)
- `Request(address, body, timeout)` (request/reply)
- `Consumer(address).Handler(fn)` to subscribe

### Default implementation: in-memory (`pkg/core/eventbus_impl.go`)

- JSON-first encoding (`core.JSONEncode/JSONDecode`)
- request/reply implemented with a generated reply address + temporary reply consumer
- bounded execution via `pkg/core/concurrency` abstractions (Executor/Mailbox)

### Cluster implementations (`pkg/core/eventbus_cluster_nats.go`, `pkg/core/eventbus_cluster_jetstream.go`)

- **NATS**: maps addresses to NATS subjects `<prefix>.{pub|send|req}.<address>`
- **JetStream**:
  - `Publish`/`Send` are durable via JetStream streams
  - `Request` uses core NATS request/reply for lower latency
  - `Publish` semantics are “fanout per service” (each service group gets a copy)

---

## Modules (how to think about them)

- **`pkg/core`**: the lowest-level stable API surface (runtime + messaging + context)
- **`pkg/web`**: HTTP server + router + middleware pipeline; uses Vertx/EventBus
- **`pkg/fx`**: convenience bootstrap to wire dependencies and lifecycle
- **`pkg/fluxor`**: workflow/runtime utilities + `MainVerticle` bootstrap for “main-like” apps
- **`pkg/observability/*`**: middleware + helpers for metrics/tracing
- **`pkg/db`**: database pooling + health integration
- **`pkg/lite/*`**: alternate minimal dependency graph (use when you want a smaller surface)

