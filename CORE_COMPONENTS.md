# Core Components and Flow

This document defines the core components of Fluxor and their interactions to prevent uncertainty and ensure clear understanding of the system architecture.

## Table of Contents

1. [Core Components](#core-components)
2. [Request Flow](#request-flow)
3. [Component Interactions](#component-interactions)
4. [Day2 Integration](#day2-integration)
5. [Data Flow Diagrams](#data-flow-diagrams)

---

## Core Components

### 1. Vertx (Runtime)

**Purpose**: The core runtime that manages the application lifecycle and provides access to shared resources.

**Location**: `pkg/core/vertx.go`

**Responsibilities**:
- Application lifecycle management (start/stop)
- EventBus creation and management
- Context management
- Component registration

**Key Methods**:
```go
type Vertx interface {
    Start() error
    Stop() error
    EventBus() EventBus
    Context() context.Context
    RegisterComponent(component Component) error
}
```

**Flow**:
```
Application Start
    ↓
Vertx.Start()
    ↓
Initialize EventBus
    ↓
Start Registered Components
    ↓
Application Ready
```

---

### 2. EventBus

**Purpose**: Message passing infrastructure for publish-subscribe, point-to-point, and request-reply messaging.

**Location**: `pkg/core/eventbus_impl.go`

**Responsibilities**:
- Message routing
- Consumer registration
- Message encoding/decoding (JSON)
- Request ID propagation

**Key Methods**:
```go
type EventBus interface {
    Publish(address string, body interface{}) error
    Send(address string, body interface{}) error
    Request(address string, body interface{}, timeout time.Duration) (Message, error)
    Consumer(address string) Consumer
}
```

**Message Flow**:
```
Publisher → EventBus → Consumer(s)
    ↓
JSON Encoding
    ↓
Request ID Propagation
    ↓
Message Delivery
```

---

### 3. FastHTTPServer

**Purpose**: High-performance HTTP server with CCU-based backpressure.

**Location**: `pkg/web/fasthttp_server.go`

**Responsibilities**:
- HTTP request handling
- Request ID generation/extraction
- CCU-based backpressure
- Request routing
- Metrics collection

**Key Components**:
```go
type FastHTTPServer struct {
    vertx    core.Vertx
    router   *fastRouter
    config   CCUBasedConfig
    // ... metrics, backpressure
}
```

**Request Processing Flow**:
```
HTTP Request
    ↓
Extract/Generate Request ID
    ↓
CCU Check (Backpressure)
    ↓
Router Matching
    ↓
Middleware Chain
    ↓
Handler Execution
    ↓
Response
```

---

### 4. Router

**Purpose**: HTTP routing and middleware management.

**Location**: `pkg/web/fast_router.go`

**Responsibilities**:
- Route registration
- Path matching
- Parameter extraction
- Middleware chain execution

**Key Types**:
```go
type FastRouter struct {
    routes     []*fastRoute
    middleware []FastMiddleware
}

type FastMiddleware func(handler FastRequestHandler) FastRequestHandler
type FastRequestHandler func(ctx *FastRequestContext) error
```

**Middleware Chain Flow**:
```
Request
    ↓
Middleware[0] → Middleware[1] → ... → Middleware[N]
    ↓
Handler
    ↓
Response (reverse order)
```

---

### 5. FastRequestContext

**Purpose**: Request context wrapper providing access to request data and Fluxor services.

**Location**: `pkg/web/fasthttp_server.go`

**Responsibilities**:
- Request data access (method, path, params, body)
- Response writing
- Context storage (Set/Get)
- Request ID access
- EventBus access

**Key Methods**:
```go
type FastRequestContext struct {
    RequestCtx *fasthttp.RequestCtx
    Vertx      core.Vertx
    EventBus   core.EventBus
    Params     map[string]string
    data       map[string]interface{} // For middleware data
    requestID  string
}

func (c *FastRequestContext) Context() context.Context
func (c *FastRequestContext) RequestID() string
func (c *FastRequestContext) Set(key string, value interface{})
func (c *FastRequestContext) Get(key string) interface{}
```

---

## Request Flow

### Complete HTTP Request Flow

```
1. HTTP Request Arrives
   │
   ├─→ FastHTTPServer.processRequest()
   │   │
   │   ├─→ Extract Request ID from header (X-Request-ID)
   │   │   OR Generate new Request ID
   │   │
   │   ├─→ Create FastRequestContext
   │   │   ├─→ Set Request ID
   │   │   ├─→ Attach Vertx
   │   │   ├─→ Attach EventBus
   │   │   └─→ Initialize Params map
   │   │
   │   └─→ router.ServeFastHTTP(ctx)
   │
2. Router Processing
   │
   ├─→ Match route by method + path
   │   │
   │   ├─→ Extract path parameters
   │   │   └─→ Store in ctx.Params
   │   │
   │   └─→ Apply middleware chain (reverse order)
   │       │
   │       ├─→ Middleware[N] → Middleware[N-1] → ... → Middleware[0]
   │       │   │
   │       │   ├─→ Each middleware can:
   │       │   │   ├─→ Read/Modify request
   │       │   │   ├─→ Store data in ctx.Set()
   │       │   │   ├─→ Call next() handler
   │       │   │   └─→ Read/Modify response
   │       │   │
   │       │   └─→ Handler execution
   │       │       │
   │       │       ├─→ Business logic
   │       │       ├─→ Database queries
   │       │       ├─→ EventBus messages
   │       │       └─→ Response writing
   │       │
   │       └─→ Response flows back through middleware
   │
3. Response
   │
   ├─→ Set response headers (including X-Request-ID)
   ├─→ Write response body
   └─→ Track metrics
```

---

## Component Interactions

### 1. Vertx → EventBus

```
Vertx
  │
  ├─→ Creates EventBus on Start()
  │
  └─→ Provides EventBus to components
      │
      └─→ Components can publish/send messages
```

### 2. FastHTTPServer → Router

```
FastHTTPServer
  │
  ├─→ Creates fastRouter
  │
  ├─→ Routes requests to router.ServeFastHTTP()
  │
  └─→ Router matches routes and executes handlers
```

### 3. Router → Middleware → Handler

```
Router
  │
  ├─→ Maintains middleware chain
  │
  ├─→ Applies middleware in reverse order
  │
  └─→ Executes handler after all middleware
```

### 4. Handler → EventBus

```
Handler (via FastRequestContext)
  │
  ├─→ Access EventBus: ctx.EventBus()
  │
  ├─→ Publish: ctx.EventBus().Publish(address, data)
  │
  └─→ Send: ctx.EventBus().Send(address, data)
```

---

## Day2 Integration

### How Day2 Features Integrate

#### 1. Configuration Management

**Integration Point**: Application startup

```
Application Start
    ↓
Load Config (pkg/config)
    ├─→ Load from YAML/JSON
    ├─→ Apply environment overrides
    └─→ Validate configuration
    ↓
Initialize Components with Config
```

**Flow**:
```go
// At application startup
var cfg AppConfig
config.LoadWithEnv("config.yaml", "APP", &cfg)

// Use config for component initialization
server := web.NewFastHTTPServer(vertx, cfg.Server)
```

---

#### 2. Prometheus Metrics

**Integration Point**: FastHTTPServer and Router

```
HTTP Request
    ↓
FastHTTPMetricsMiddleware (pkg/observability/prometheus)
    ├─→ Record request start time
    ├─→ Execute handler
    ├─→ Record duration, status, sizes
    └─→ Update metrics
    ↓
Response
```

**Metrics Collection Flow**:
```
Request → Metrics Middleware → Handler → Metrics Update → Response
```

**Metrics Endpoint**:
```
GET /metrics
    ↓
Prometheus Scraper
    ↓
Metrics Export
```

---

#### 3. OpenTelemetry Tracing

**Integration Point**: Router middleware and EventBus

```
HTTP Request
    ↓
HTTPMiddleware (pkg/observability/otel)
    ├─→ Extract trace context from headers
    ├─→ Start span
    ├─→ Store span context in ctx.Set("span_context")
    ├─→ Execute handler
    ├─→ Record span attributes (status, duration)
    └─→ Inject trace context into response
    ↓
Response
```

**EventBus Integration**:
```
EventBus.Publish/Send/Request
    ↓
otel.PublishWithSpan/SendWithSpan/RequestWithSpan
    ├─→ Create span
    ├─→ Propagate trace context
    └─→ Record message metrics
```

---

#### 4. Authentication/Authorization

**Integration Point**: Router middleware

```
HTTP Request
    ↓
JWT/OAuth2 Middleware (pkg/web/middleware/auth)
    ├─→ Extract token
    ├─→ Validate token
    ├─→ Extract claims
    └─→ Store in ctx.Set("user", claims)
    ↓
RBAC Middleware (if needed)
    ├─→ Extract user from ctx.Get("user")
    ├─→ Check roles/permissions
    └─→ Allow/Deny request
    ↓
Handler (can access user via ctx.Get("user"))
```

---

#### 5. Security Middleware

**Integration Point**: Router middleware (early in chain)

```
HTTP Request
    ↓
Security Headers Middleware
    ├─→ Add HSTS, CSP, X-Frame-Options, etc.
    └─→ Continue to next middleware
    ↓
CORS Middleware
    ├─→ Check origin
    ├─→ Handle preflight (OPTIONS)
    └─→ Add CORS headers
    ↓
Rate Limiting Middleware
    ├─→ Check rate limit (by IP/user)
    ├─→ Allow/Reject (429)
    └─→ Continue if allowed
    ↓
Next Middleware
```

---

#### 6. Enhanced Health Checks

**Integration Point**: Router routes

```
GET /health or /ready
    ↓
Health Handler (pkg/web/health)
    ├─→ Run all registered checks
    │   ├─→ Database check
    │   ├─→ External service check
    │   └─→ Custom checks
    ├─→ Aggregate results
    └─→ Return status (200/503)
```

**Health Check Registration Flow**:
```
Component Initialization
    ↓
Register Health Check
    ├─→ health.Register("database", health.DatabaseCheck(pool))
    └─→ health.Register("redis", health.HTTPCheck(...))
    ↓
Health Endpoint Available
```

---

#### 7. Express-like Middleware

**Integration Point**: Router middleware chain

```
HTTP Request
    ↓
Recovery Middleware (first)
    ├─→ Panic recovery
    └─→ Error response on panic
    ↓
Logging Middleware
    ├─→ Log request start
    ├─→ Execute handler
    └─→ Log response
    ↓
Compression Middleware
    ├─→ Check Accept-Encoding
    └─→ Compress response if needed
    ↓
Timeout Middleware
    ├─→ Set request timeout
    └─→ Return 504 on timeout
    ↓
Handler
```

---

## Data Flow Diagrams

### Complete Request Flow with Day2 Features

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Request Arrives                     │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              FastHTTPServer.processRequest()                 │
│  • Extract/Generate Request ID                              │
│  • Create FastRequestContext                                │
│  • Attach Vertx, EventBus                                   │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                    Router Matching                          │
│  • Match route by method + path                            │
│  • Extract path parameters                                  │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              Middleware Chain (Reverse Order)               │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Recovery Middleware                                │    │
│  │  • Panic recovery                                  │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ OpenTelemetry Middleware                           │    │
│  │  • Extract trace context                           │    │
│  │  • Start span                                      │    │
│  │  • Store span context                              │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Prometheus Metrics Middleware                       │    │
│  │  • Record request start                             │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Logging Middleware                                  │    │
│  │  • Log request                                      │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Security Headers Middleware                        │    │
│  │  • Add security headers                            │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ CORS Middleware                                    │    │
│  │  • Handle CORS                                     │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Rate Limiting Middleware                           │    │
│  │  • Check rate limit                                │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Authentication Middleware (JWT/OAuth2)            │    │
│  │  • Validate token                                   │    │
│  │  • Store user in ctx.Set("user")                   │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Authorization Middleware (RBAC)                    │    │
│  │  • Check roles/permissions                          │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Compression Middleware                              │    │
│  │  • Compress response if needed                      │    │
│  └────────────────────────────────────────────────────┘    │
│                        │                                     │
│                        ▼                                     │
│  ┌────────────────────────────────────────────────────┐    │
│  │ Timeout Middleware                                 │    │
│  │  • Enforce timeout                                 │    │
│  └────────────────────────────────────────────────────┘    │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                      Handler Execution                       │
│  • Business logic                                            │
│  • Database queries (with health checks)                    │
│  • EventBus messages (with tracing)                          │
│  • Response writing                                          │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              Response Flow (Reverse Order)                  │
│  • Update metrics                                            │
│  • Record span attributes                                    │
│  • Log response                                              │
│  • Inject trace context                                      │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│                      HTTP Response                           │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Dependencies

### Dependency Graph

```
Application
    │
    ├─→ Vertx (Core Runtime)
    │   │
    │   ├─→ EventBus
    │   │   └─→ Message Handlers
    │   │
    │   └─→ Components
    │       ├─→ DatabaseComponent
    │       └─→ Custom Components
    │
    └─→ FastHTTPServer
        │
        ├─→ Router
        │   │
        │   ├─→ Middleware Chain
        │   │   ├─→ Recovery
        │   │   ├─→ OpenTelemetry
        │   │   ├─→ Prometheus
        │   │   ├─→ Logging
        │   │   ├─→ Security Headers
        │   │   ├─→ CORS
        │   │   ├─→ Rate Limiting
        │   │   ├─→ Authentication
        │   │   ├─→ Authorization
        │   │   ├─→ Compression
        │   │   └─→ Timeout
        │   │
        │   └─→ Handlers
        │
        └─→ Health Checks
            ├─→ Database Health
            ├─→ External Service Health
            └─→ Custom Health Checks
```

---

## Key Principles

### 1. Fail-Fast
- All components validate inputs immediately
- Errors are propagated immediately, not silently ignored
- Invalid state is detected early

### 2. Request ID Propagation
- Request ID is generated/extracted at entry point
- Propagated through all components
- Included in logs, traces, and EventBus messages

### 3. Middleware Order
- Recovery (outermost) - catches panics
- Observability (OpenTelemetry, Prometheus, Logging)
- Security (Headers, CORS, Rate Limiting)
- Authentication/Authorization
- Business logic (Handler)

### 4. Context Storage
- Use `ctx.Set(key, value)` to store data
- Use `ctx.Get(key)` to retrieve data
- Common keys: "user", "span_context", "request_id"

### 5. Component Lifecycle
- Components implement `Component` interface
- Started via `Vertx.RegisterComponent()`
- Stopped via `Vertx.Stop()`

---

## Summary

This document defines:

1. **Core Components**: Vertx, EventBus, FastHTTPServer, Router, FastRequestContext
2. **Request Flow**: Complete flow from HTTP request to response
3. **Component Interactions**: How components interact with each other
4. **Day2 Integration**: How Day2 features integrate into the core flow
5. **Data Flow Diagrams**: Visual representation of request processing

All components follow clear contracts and interactions, preventing uncertainty about how the system works.

