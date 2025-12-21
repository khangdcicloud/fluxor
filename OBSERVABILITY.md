# Observability in Fluxor

Fluxor provides comprehensive observability features including structured logging, Prometheus metrics, and OpenTelemetry distributed tracing.

## Table of Contents

1. [Structured Logging](#structured-logging)
2. [Prometheus Metrics](#prometheus-metrics)
3. [OpenTelemetry Tracing](#opentelemetry-tracing)
4. [Request ID Tracking](#request-id-tracking)
5. [Health Checks](#health-checks)

---

## Structured Logging

Fluxor provides structured logging with JSON support and context-aware logging.

### Basic Usage

```go
import "github.com/fluxorio/fluxor/pkg/core"

// Create logger
logger := core.NewDefaultLogger()

// Plain text logging
logger.Info("Application started")
logger.Errorf("Error: %v", err)

// JSON logging
jsonLogger := core.NewJSONLogger()
jsonLogger.Info("Application started")
// Output: {"timestamp":"2024-01-01T00:00:00Z","level":"INFO","message":"Application started"}
```

### Context-Aware Logging

```go
// Logger automatically extracts request ID from context
ctx := core.WithRequestID(context.Background(), "req-123")
logger := core.NewDefaultLogger().WithContext(ctx)
logger.Info("Processing request")
// Output includes request_id automatically
```

### Structured Fields

```go
logger := core.NewDefaultLogger().WithFields(map[string]interface{}{
    "user_id": "123",
    "action":  "create_user",
})
logger.Info("User created")
// Output includes user_id and action fields
```

### Configuration

```go
logger := core.NewLogger(core.LoggerConfig{
    JSONOutput: true,
    Level:      "INFO", // DEBUG, INFO, WARN, ERROR
})
```

---

## Prometheus Metrics

Fluxor provides Prometheus metrics export for monitoring and alerting.

### Setup

```go
import (
    "github.com/fluxorio/fluxor/pkg/observability/prometheus"
    "github.com/fluxorio/fluxor/pkg/web"
)

// Register metrics endpoint
router := server.FastRouter()
prometheus.RegisterMetricsEndpoint(router, "/metrics")

// Add metrics middleware
router.UseFast(prometheus.FastHTTPMetricsMiddleware())
```

### Available Metrics

**HTTP Metrics:**
- `fluxor_http_requests_total` - Total HTTP requests (counter)
- `fluxor_http_request_duration_seconds` - Request duration (histogram)
- `fluxor_http_request_size_bytes` - Request size (histogram)
- `fluxor_http_response_size_bytes` - Response size (histogram)

**EventBus Metrics:**
- `fluxor_eventbus_messages_total` - Total EventBus messages (counter)
- `fluxor_eventbus_message_duration_seconds` - Message processing duration (histogram)

**Database Metrics:**
- `fluxor_database_connections_open` - Open connections (gauge)
- `fluxor_database_connections_idle` - Idle connections (gauge)
- `fluxor_database_connections_in_use` - Connections in use (gauge)
- `fluxor_database_query_duration_seconds` - Query duration (histogram)

**Server Metrics:**
- `fluxor_server_queued_requests` - Queued requests (gauge)
- `fluxor_server_rejected_requests_total` - Rejected requests (counter)
- `fluxor_server_current_ccu` - Current CCU (gauge)
- `fluxor_server_ccu_utilization` - CCU utilization (gauge)

### Custom Metrics

```go
import "github.com/fluxorio/fluxor/pkg/observability/prometheus"

// Create custom counter
counter := prometheus.Counter("custom_events_total", "Total custom events", "type")
counter.WithLabelValues("user_created").Inc()

// Create custom gauge
gauge := prometheus.Gauge("active_users", "Active users", "region")
gauge.WithLabelValues("us-east").Set(100.0)

// Create custom histogram
histogram := prometheus.Histogram("operation_duration_seconds", "Operation duration", nil, "operation")
histogram.WithLabelValues("process").Observe(0.5)
```

### Integration with FastHTTPServer

```go
// Metrics are automatically collected when using FastHTTPMetricsMiddleware
router.UseFast(prometheus.FastHTTPMetricsMiddleware())

// Update server metrics periodically
go func() {
    ticker := time.NewTicker(5 * time.Second)
    for range ticker.C {
        prometheus.UpdateServerMetrics(server)
    }
}()
```

---

## OpenTelemetry Tracing

Fluxor provides OpenTelemetry integration for distributed tracing.

### Setup

```go
import (
    "context"
    "github.com/fluxorio/fluxor/pkg/observability/otel"
)

// Initialize OpenTelemetry
ctx := context.Background()
err := otel.Initialize(ctx, otel.Config{
    ServiceName:    "my-service",
    ServiceVersion: "1.0.0",
    Exporter:       "jaeger",
    Endpoint:       "http://localhost:14268/api/traces",
    Environment:    "production",
    SampleRate:     1.0, // 100% sampling
})
if err != nil {
    log.Fatal(err)
}
defer otel.Shutdown(ctx)
```

### HTTP Middleware

```go
// Add OpenTelemetry middleware to router
router.UseFast(otel.HTTPMiddleware())

// Automatic span creation for all HTTP requests
router.GETFast("/api/users", func(ctx *web.FastRequestContext) error {
    // Span is automatically created and includes:
    // - HTTP method, path, status code
    // - Request ID correlation
    // - Duration
    return ctx.JSON(200, users)
})
```

### Manual Span Creation

```go
import "go.opentelemetry.io/otel/attribute"

// Create manual span
spanCtx, span := otel.StartSpan(ctx, "database.query",
    trace.WithAttributes(
        attribute.String("query", "SELECT * FROM users"),
        attribute.String("table", "users"),
    ),
)
defer span.End()

// Execute database query
result, err := db.Query(spanCtx, "SELECT * FROM users")
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return err
}

span.SetStatus(codes.Ok, "OK")
```

### EventBus Span Propagation

```go
import "github.com/fluxorio/fluxor/pkg/observability/otel"

// Publish with span propagation
ctx := context.Background()
err := otel.PublishWithSpan(ctx, eventBus, "user.created", userData)

// Send with span propagation
err := otel.SendWithSpan(ctx, eventBus, "user.process", userData)

// Request with span propagation
msg, err := otel.RequestWithSpan(ctx, eventBus, "user.get", requestData, 5*time.Second)

// Wrap consumer handler with span creation
consumer := eventBus.Consumer("user.created")
consumer.Handler(otel.WrapConsumerHandler("user.created", func(ctx core.FluxorContext, msg core.Message) error {
    // Span is automatically created for message processing
    return nil
}))
```

### Supported Exporters

- **Jaeger**: `Exporter: "jaeger"`
- **Zipkin**: `Exporter: "zipkin"`
- **Stdout** (debugging): `Exporter: "stdout"`
- **None** (disabled): `Exporter: "none"`

---

## Request ID Tracking

Request IDs are automatically generated and propagated across all components.

### Automatic Propagation

```go
// Request ID is automatically:
// 1. Generated/extracted from HTTP headers
// 2. Stored in context
// 3. Propagated through EventBus messages
// 4. Included in response headers (X-Request-ID)
// 5. Available in logs via WithContext()

router.GETFast("/api/data", func(ctx *web.FastRequestContext) error {
    requestID := ctx.RequestID() // Get request ID
    
    // Request ID is automatically included in EventBus messages
    eventBus.Publish("data.processed", data)
    
    return ctx.JSON(200, data)
})
```

### Manual Request ID

```go
import "github.com/fluxorio/fluxor/pkg/core"

// Generate new request ID
requestID := core.GenerateRequestID()
ctx := core.WithRequestID(context.Background(), requestID)

// Retrieve from context
id := core.GetRequestID(ctx)
```

---

## Health Checks

Fluxor provides enhanced health checks for dependencies.

### Basic Health Checks

```go
import "github.com/fluxorio/fluxor/pkg/web/health"

// Register health checks
health.Register("database", health.DatabaseCheck(pool))
health.Register("redis", health.HTTPCheck("http://redis:6379/health", 5*time.Second))

// Use health check handlers
router.GETFast("/health", health.Handler())
router.GETFast("/ready", health.ReadyHandler())
```

### Database Health Check

```go
import (
    "github.com/fluxorio/fluxor/pkg/db"
    "github.com/fluxorio/fluxor/pkg/web/health"
)

// Create database pool
pool, _ := db.NewPool(db.DefaultPoolConfig("postgres://...", "postgres"))

// Register health check
health.Register("database", health.DatabaseCheck(pool))

// Or use DatabaseComponent
component := db.NewDatabaseComponent(db.DefaultPoolConfig("postgres://...", "postgres"))
health.Register("database", health.DatabaseComponentCheck(component))
```

### External Service Health Check

```go
// HTTP health check
health.Register("external-api", health.HTTPCheck("http://api.example.com/health", 5*time.Second))

// HTTP health check with custom headers
health.Register("authenticated-api", health.HTTPCheckWithHeaders(
    "http://api.example.com/health",
    5*time.Second,
    map[string]string{
        "Authorization": "Bearer token",
    },
))
```

### Custom Health Checks

```go
// Register custom health check
health.Register("custom", func(ctx context.Context) error {
    // Check custom dependency
    if !isDependencyHealthy() {
        return fmt.Errorf("dependency is down")
    }
    return nil
})

// Register with timeout
health.RegisterWithTimeout("slow-check", func(ctx context.Context) error {
    // Health check with custom timeout
    return checkSlowDependency(ctx)
}, 10*time.Second)
```

### Health Check Response

```json
{
  "status": "UP",
  "timestamp": "2024-01-01T00:00:00Z",
  "checks": {
    "database": {
      "status": "UP",
      "message": "OK",
      "duration": "2ms"
    },
    "redis": {
      "status": "DOWN",
      "message": "connection refused",
      "duration": "5s"
    }
  },
  "request_id": "req-123"
}
```

---

## Best Practices

### 1. Use Structured Logging

```go
// ✅ Good: Structured logging with context
logger := core.NewJSONLogger().WithContext(ctx).WithFields(map[string]interface{}{
    "user_id": userID,
    "action":  "create_order",
})
logger.Info("Order created")

// ❌ Bad: Plain string logging
log.Printf("Order created for user %s", userID)
```

### 2. Enable Metrics Middleware

```go
// ✅ Good: Enable metrics for all requests
router.UseFast(prometheus.FastHTTPMetricsMiddleware())

// ❌ Bad: No metrics collection
```

### 3. Initialize OpenTelemetry Early

```go
// ✅ Good: Initialize at application startup
func main() {
    ctx := context.Background()
    otel.Initialize(ctx, otel.Config{...})
    defer otel.Shutdown(ctx)
    // ... rest of application
}
```

### 4. Use Request ID in Logs

```go
// ✅ Good: Request ID automatically included
logger := core.NewDefaultLogger().WithContext(ctx)
logger.Info("Processing request")

// ❌ Bad: Manual request ID logging
logger.Infof("Request %s: Processing", requestID)
```

### 5. Register Health Checks

```go
// ✅ Good: Register all dependencies
health.Register("database", health.DatabaseCheck(pool))
health.Register("cache", health.HTTPCheck("http://cache:6379/health", 5*time.Second))

// ❌ Bad: No health checks
```

---

## Integration Examples

### Complete Observability Setup

```go
func main() {
    ctx := context.Background()
    
    // Initialize OpenTelemetry
    otel.Initialize(ctx, otel.Config{
        ServiceName: "my-service",
        Exporter:    "jaeger",
        Endpoint:    "http://localhost:14268/api/traces",
    })
    defer otel.Shutdown(ctx)
    
    // Create application
    app, _ := fx.New(ctx, fx.Invoke(fx.NewInvoker(setupApplication)))
    app.Start()
    defer app.Stop()
}

func setupApplication(deps map[reflect.Type]interface{}) error {
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)
    
    // Create server
    config := web.CCUBasedConfigWithUtilization(":8080", 5000, 60)
    server := web.NewFastHTTPServer(vertx, config)
    router := server.FastRouter()
    
    // Add observability middleware
    router.UseFast(
        otel.HTTPMiddleware(),              // OpenTelemetry tracing
        prometheus.FastHTTPMetricsMiddleware(), // Prometheus metrics
        middleware.Logging(middleware.DefaultLoggingConfig()), // Structured logging
    )
    
    // Register metrics endpoint
    prometheus.RegisterMetricsEndpoint(router, "/metrics")
    
    // Register health checks
    health.Register("eventbus", func(ctx context.Context) error {
        // Check EventBus health
        return nil
    })
    router.GETFast("/health", health.Handler())
    router.GETFast("/ready", health.ReadyHandler())
    
    // Routes
    router.GETFast("/api/users", getUserHandler)
    
    go server.Start()
    return nil
}
```

---

## Summary

Fluxor provides comprehensive observability:

- ✅ **Structured Logging**: JSON logging with context and fields
- ✅ **Prometheus Metrics**: HTTP, EventBus, database, and server metrics
- ✅ **OpenTelemetry Tracing**: Distributed tracing with span propagation
- ✅ **Request ID Tracking**: Automatic request correlation
- ✅ **Health Checks**: Database and external service health monitoring

All features work together to provide full observability for production systems.

