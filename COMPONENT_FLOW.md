# Component Flow Reference

Quick reference guide for component interactions and data flow in Fluxor.

## Core Component Hierarchy

```
Application
│
├── Vertx (Runtime)
│   ├── EventBus
│   │   ├── Publishers
│   │   ├── Consumers
│   │   └── Message Handlers
│   │
│   └── Components
│       ├── DatabaseComponent
│       └── Custom Components
│
└── FastHTTPServer
    ├── Router
    │   ├── Routes
    │   ├── Middleware Chain
    │   └── Handlers
    │
    └── Health Checks
```

---

## Request Processing Flow

### Step-by-Step Flow

```
1. HTTP Request
   │
   └─→ FastHTTPServer.processRequest()
       │
       ├─→ Extract Request ID (X-Request-ID header or generate)
       │
       ├─→ Create FastRequestContext
       │   ├─→ RequestCtx: *fasthttp.RequestCtx
       │   ├─→ Vertx: core.Vertx
       │   ├─→ EventBus: core.EventBus
       │   ├─→ Params: map[string]string
       │   ├─→ data: map[string]interface{} (for middleware)
       │   └─→ requestID: string
       │
       └─→ router.ServeFastHTTP(ctx)

2. Router Processing
   │
   ├─→ Match route (method + path)
   │
   ├─→ Extract path parameters → ctx.Params
   │
   └─→ Execute middleware chain (reverse order)
       │
       └─→ Execute handler

3. Response
   │
   ├─→ Set response headers (X-Request-ID)
   ├─→ Write response body
   └─→ Track metrics
```

---

## Middleware Execution Order

### Standard Middleware Chain

```go
router.UseFast(
    // 1. Recovery (outermost - catches panics)
    middleware.Recovery(...),
    
    // 2. Observability (tracing, metrics, logging)
    otel.HTTPMiddleware(),
    prometheus.FastHTTPMetricsMiddleware(),
    middleware.Logging(...),
    
    // 3. Security (headers, CORS, rate limiting)
    security.Headers(...),
    security.CORS(...),
    security.RateLimit(...),
    
    // 4. Authentication
    auth.JWT(...),
    
    // 5. Authorization (if needed per route)
    // auth.RequireRole("admin"), // Applied per route
    
    // 6. Compression
    middleware.Compression(...),
    
    // 7. Timeout
    middleware.Timeout(...),
)
```

### Execution Flow

```
Request
  │
  ├─→ Recovery (wraps everything)
  │   │
  │   ├─→ OpenTelemetry
  │   │   │
  │   │   ├─→ Prometheus
  │   │   │   │
  │   │   │   ├─→ Logging
  │   │   │   │   │
  │   │   │   │   ├─→ Security Headers
  │   │   │   │   │   │
  │   │   │   │   │   ├─→ CORS
  │   │   │   │   │   │   │
  │   │   │   │   │   │   ├─→ Rate Limiting
  │   │   │   │   │   │   │   │
  │   │   │   │   │   │   │   ├─→ JWT Auth
  │   │   │   │   │   │   │   │   │
  │   │   │   │   │   │   │   │   ├─→ Compression
  │   │   │   │   │   │   │   │   │   │
  │   │   │   │   │   │   │   │   │   ├─→ Timeout
  │   │   │   │   │   │   │   │   │   │   │
  │   │   │   │   │   │   │   │   │   │   └─→ Handler
  │   │   │   │   │   │   │   │   │   │
  │   │   │   │   │   │   │   │   │   └─→ Response flows back
  │   │   │   │   │   │   │   │   │
  │   │   │   │   │   │   │   │   └─→ Update metrics
  │   │   │   │   │   │   │   │
  │   │   │   │   │   │   │   └─→ Record span
  │   │   │   │   │   │   │
  │   │   │   │   │   │   └─→ Log response
  │   │   │   │   │   │
  │   │   │   │   │   └─→ Inject trace context
  │   │   │   │   │
  │   │   │   │   └─→ HTTP Response
```

---

## EventBus Message Flow

### Publish Flow

```
Publisher
  │
  ├─→ eventBus.Publish(address, body)
  │   │
  │   ├─→ Validate address and body
  │   │
  │   ├─→ Encode body to JSON
  │   │
  │   ├─→ Extract Request ID from context
  │   │
  │   └─→ Deliver to all consumers
  │       │
  │       └─→ Consumer Mailbox
  │           │
  │           └─→ Handler Execution
```

### Send Flow (Point-to-Point)

```
Sender
  │
  ├─→ eventBus.Send(address, body)
  │   │
  │   ├─→ Validate address and body
  │   │
  │   ├─→ Encode body to JSON
  │   │
  │   └─→ Deliver to first available consumer
  │       │
  │       └─→ Consumer Mailbox
  │           │
  │           └─→ Handler Execution
```

### Request-Reply Flow

```
Requester
  │
  ├─→ eventBus.Request(address, body, timeout)
  │   │
  │   ├─→ Validate address and body
  │   │
  │   ├─→ Encode body to JSON
  │   │
  │   ├─→ Create reply address (unique)
  │   │
  │   ├─→ Register reply consumer
  │   │
  │   ├─→ Send message
  │   │   │
  │   │   └─→ Consumer processes
  │   │       │
  │   │       └─→ Reply to reply address
  │   │
  │   └─→ Wait for reply (with timeout)
  │       │
  │       └─→ Return message or timeout error
```

---

## Context Data Flow

### Request Context Data

```
FastRequestContext
  │
  ├─→ Request Data
  │   ├─→ Method: []byte
  │   ├─→ Path: []byte
  │   ├─→ Query: string (via Query())
  │   ├─→ Params: map[string]string
  │   └─→ Body: []byte (via PostBody())
  │
  ├─→ Fluxor Services
  │   ├─→ Vertx: core.Vertx
  │   ├─→ EventBus: core.EventBus
  │   └─→ Context(): context.Context (with Request ID)
  │
  └─→ Middleware Data (via Set/Get)
      ├─→ "user": JWT claims / User object
      ├─→ "span_context": OpenTelemetry span context
      └─→ Custom data (any key-value pairs)
```

### Data Access Pattern

```go
// In middleware
ctx.Set("user", userClaims)
ctx.Set("span_context", spanCtx)

// In handler
user := ctx.Get("user")
spanCtx := ctx.Get("span_context")
```

---

## Health Check Flow

### Health Check Execution

```
GET /health or /ready
  │
  └─→ Health Handler
      │
      ├─→ Run all registered checks (parallel)
      │   │
      │   ├─→ Check 1: Database
      │   │   ├─→ Ping database
      │   │   └─→ Return status
      │   │
      │   ├─→ Check 2: Redis
      │   │   ├─→ HTTP check
      │   │   └─→ Return status
      │   │
      │   └─→ Check N: Custom
      │       └─→ Return status
      │
      ├─→ Aggregate results
      │   ├─→ All UP → 200 OK
      │   └─→ Any DOWN → 503 Service Unavailable
      │
      └─→ Return JSON response
          {
            "status": "UP" | "DOWN",
            "checks": {
              "database": { "status": "UP", ... },
              "redis": { "status": "DOWN", ... }
            }
          }
```

---

## Metrics Collection Flow

### Prometheus Metrics

```
HTTP Request
  │
  └─→ FastHTTPMetricsMiddleware
      │
      ├─→ Record start time
      │
      ├─→ Execute handler
      │
      ├─→ Calculate duration
      │
      ├─→ Get status code, sizes
      │
      └─→ Update metrics
          ├─→ HTTP request counter
          ├─→ HTTP request duration histogram
          ├─→ HTTP request size histogram
          └─→ HTTP response size histogram

GET /metrics
  │
  └─→ Prometheus Handler
      │
      └─→ Export all metrics
          └─→ Prometheus scraper
```

---

## Tracing Flow

### OpenTelemetry Tracing

```
HTTP Request
  │
  └─→ HTTPMiddleware
      │
      ├─→ Extract trace context from headers
      │   ├─→ TraceID
      │   └─→ SpanID
      │
      ├─→ Start new span
      │   ├─→ Set span attributes
      │   └─→ Store span context
      │
      ├─→ Execute handler
      │   │
      │   └─→ Handler can create child spans
      │
      ├─→ Record span attributes
      │   ├─→ Status code
      │   ├─→ Duration
      │   └─→ Response size
      │
      └─→ Inject trace context into response headers
          └─→ TraceID, SpanID for downstream services
```

### EventBus Tracing

```
EventBus Message
  │
  └─→ otel.PublishWithSpan/SendWithSpan/RequestWithSpan
      │
      ├─→ Create span
      │   ├─→ Span kind: Producer/Consumer
      │   └─→ Attributes: address, operation
      │
      ├─→ Propagate trace context
      │   └─→ Include in message headers
      │
      └─→ Record span
          ├─→ Duration
          └─→ Status
```

---

## Authentication/Authorization Flow

### JWT Authentication

```
HTTP Request
  │
  └─→ JWT Middleware
      │
      ├─→ Extract token (header/query/cookie)
      │
      ├─→ Validate token
      │   ├─→ Parse JWT
      │   ├─→ Verify signature
      │   └─→ Check expiration
      │
      ├─→ Extract claims
      │
      └─→ Store in context
          └─→ ctx.Set("user", claims)

Handler
  │
  └─→ Access user
      └─→ user := ctx.Get("user")
```

### RBAC Authorization

```
HTTP Request
  │
  └─→ RequireRole("admin") Middleware
      │
      ├─→ Get user from context
      │   └─→ ctx.Get("user")
      │
      ├─→ Extract roles
      │   └─→ From JWT claims or User object
      │
      ├─→ Check role
      │   ├─→ Has role → Continue
      │   └─→ No role → 403 Forbidden
      │
      └─→ Handler execution
```

---

## Component Lifecycle Flow

### Application Startup

```
main()
  │
  ├─→ Load Configuration
  │   └─→ config.LoadWithEnv(...)
  │
  ├─→ Create Vertx
  │   └─→ vertx := core.NewVertx(ctx)
  │
  ├─→ Initialize OpenTelemetry
  │   └─→ otel.Initialize(...)
  │
  ├─→ Register Components
  │   ├─→ DatabaseComponent
  │   └─→ Custom Components
  │
  ├─→ Create FastHTTPServer
  │   └─→ server := web.NewFastHTTPServer(vertx, config)
  │
  ├─→ Setup Router
  │   ├─→ Register middleware
  │   ├─→ Register routes
  │   └─→ Register health checks
  │
  └─→ Start Server
      └─→ server.Start()
```

### Application Shutdown

```
Shutdown Signal
  │
  ├─→ Stop Server
  │   └─→ server.Stop()
  │
  ├─→ Stop Components
  │   └─→ vertx.Stop()
  │
  └─→ Shutdown OpenTelemetry
      └─→ otel.Shutdown(ctx)
```

---

## Error Handling Flow

### Error Propagation

```
Handler Error
  │
  ├─→ Return error
  │
  ├─→ Middleware catches
  │   ├─→ Log error (with Request ID)
  │   ├─→ Record in metrics
  │   └─→ Record in trace span
  │
  └─→ Response
      ├─→ 500 Internal Server Error
      └─→ Error message (if enabled)
```

### Panic Recovery

```
Panic in Handler
  │
  └─→ Recovery Middleware
      │
      ├─→ Recover from panic
      │
      ├─→ Log panic (with stack trace)
      │
      └─→ Response
          ├─→ 500 Internal Server Error
          └─→ Error message
```

---

## Summary

This document provides:

1. **Component Hierarchy**: Visual structure of components
2. **Request Flow**: Step-by-step request processing
3. **Middleware Order**: Standard middleware chain
4. **EventBus Flow**: Message passing patterns
5. **Context Data**: How data flows through context
6. **Health Checks**: Health check execution
7. **Metrics**: Metrics collection
8. **Tracing**: Distributed tracing flow
9. **Auth/Authz**: Authentication and authorization flow
10. **Lifecycle**: Application startup/shutdown
11. **Error Handling**: Error and panic handling

All flows are deterministic and follow clear patterns, preventing uncertainty about system behavior.

