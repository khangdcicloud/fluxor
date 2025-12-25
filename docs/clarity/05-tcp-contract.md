# TCP Server Contract (`pkg/tcp`)

This document specifies the **contract** (expected behavior, guarantees, and failure modes) for the TCP server module implemented in `pkg/tcp`.

## Scope

Applies to:
- `pkg/tcp.TCPServer` (constructor + lifecycle + backpressure)
- `pkg/tcp.Server` / `pkg/tcp.ConnectionHandler`
- `pkg/tcp.ConnContext`
- `pkg/tcp.ServerMetrics`

Non-goals:
- Protocol framing (length-prefix, line protocol, etc.)
- TLS (`crypto/tls`) integration (future work)

## Public API Shape

- **Create**: `tcp.NewTCPServer(vertx, config)`
- **Lifecycle**: `Start() error`, `Stop() error`
- **Handler**: `SetHandler(ConnectionHandler)`
- **Metrics**: `Metrics() ServerMetrics`
- **Listening address**: `(*TCPServer).ListeningAddr() string` (helper for `Addr=":0"`)

## Lifecycle Semantics

### Start()

- **Blocking call**: `Start()` blocks while the server is accepting connections, until the server is stopped or an unrecoverable accept error occurs.
- **Idempotency**: calling `Start()` more than once returns an error with code `ALREADY_STARTED`.
- **State**: `BaseServer.IsStarted()` becomes `true` as soon as `Start()` begins running the start hook.
- **Error handling**:
  - If bind/listen fails, `Start()` returns that error.
  - If accept fails with a non-shutdown error, `Start()` returns that error.
  - If the listener is closed due to shutdown, `Start()` returns `nil`.

### Stop()

- **Graceful shutdown**: `Stop()` closes the listener (unblocks `Accept()`), closes the mailbox, then shuts down the executor with a bounded timeout.
- **Idempotency**: multiple `Stop()` calls are safe; subsequent calls should return `nil`.
- **Guarantee**: after `Stop()` returns successfully, `Start()` should return shortly afterward (no long hang).

## Concurrency Model

The server is structured like `pkg/web.FastHTTPServer`:

- **Accept loop**: accepts connections and **enqueues** them.
- **Mailbox (bounded queue)**: `concurrency.Mailbox` holds `net.Conn` items.
- **Executor (worker pool)**: `concurrency.Executor` runs worker tasks which pull connections from the mailbox.
- **Handler execution**: each connection is processed by exactly one worker, calling your `ConnectionHandler`.

## Backpressure (Fail-Fast)

Backpressure is applied in two layers, both **fail-fast**:

1. **Normal capacity gate** (baseline utilization):
   - A simple controller (`BackpressureController`) limits concurrent in-flight load to a configured baseline.
   - When normal capacity is exceeded, new connections are rejected immediately.

2. **Bounded queue**:
   - If the mailbox is full, the connection is rejected immediately.

### Rejection behavior

When rejecting a connection due to backpressure:
- The server **closes the connection immediately** (no protocol-level error response is guaranteed).
- `ServerMetrics.RejectedConnections` increments.

## Handler Contract (`ConnectionHandler`)

Signature:

```go
type ConnectionHandler func(ctx *tcp.ConnContext) error
```

Rules:
- **Fail-fast on nil**: calling `SetHandler(nil)` panics.
- **Panic isolation**: panics are recovered **per-connection**; one handler panic must not crash the process or kill the worker.
- **One-shot handling**:
  - The server closes `ctx.Conn` after the handler returns (even on error).
  - Handlers should treat the connection as owned only for the duration of the call.
- **No infinite blocking**: handlers should avoid blocking forever; use deadlines/timeouts and/or the provided context.

## ConnContext Contract

`tcp.ConnContext` mirrors the useful parts of `web.RequestContext`:
- `BaseRequestContext`: thread-safe key/value storage for per-connection data.
- `Context`: worker context (cancellable when the executor shuts down).
- `Vertx` and `EventBus`: convenience references for integrating TCP flows with the runtime.
- `LocalAddr` and `RemoteAddr`: snapshot addresses for logging/observability.

## Timeouts / Deadlines

The server applies best-effort per-connection deadlines:
- `SetReadDeadline(now + ReadTimeout)`
- `SetWriteDeadline(now + WriteTimeout)`

Notes:
- This is **best-effort** and depends on `net.Conn` implementation behavior.
- Protocol-level framing and read loops inside your handler must still be written defensively.

## Metrics Contract

`Metrics()` provides:
- **Queue**: queued count, capacity, utilization
- **Workers**: worker pool size
- **Backpressure**: normal capacity baseline + current load + utilization
- **Connection counters**: accepted, handled, errored, rejected

Counters are **monotonic** (only increase), except instantaneous fields like queued/utilization.

## Thread-Safety

- `Metrics()` is safe to call concurrently.
- `SetHandler()` is protected by a mutex and is safe to call, but best practice is to set it **before** `Start()`.

## Known Limitations / Future Work

- No TLS support in `pkg/tcp` yet.
- No protocol framing helpers (length-prefix, line-based, etc.).
- No first-class integration with `pkg/web` middleware concepts (auth/rate-limit) yet; expected to be added as reusable handler wrappers.

