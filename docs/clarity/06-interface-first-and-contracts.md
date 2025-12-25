# Interface-first & Contract Specs (Code is Truth)

Fluxor follows a strict documentation rule:

- **Interface-first**: public behavior is defined by **interfaces** in `pkg/**` first.
- **Contract spec**: the expected semantics (fail-fast, thread-safety, ordering, backpressure, lifecycle) are written down.
- **Code is truth**: the **implementation + tests** are the source of truth; docs must match them.

This document explains how we work in this repo.

## 1) Interface-first: what it means here

### Definition

Before adding a new runtime feature, we:

1. Define a **minimal public interface** (in `pkg/<module>/...`) that users will depend on.
2. Only then implement it behind the interface (structs can change; interfaces are “API contracts”).

### Why

- Keeps the public API stable while implementations evolve.
- Forces clear boundaries between modules (`core`, `web`, `tcp`, `workflow`, ...).
- Enables fail-fast checks at the correct layer (programmer error vs runtime error).

### Example in this repo

- **TCP module**:
  - Interface: `pkg/tcp/server.go` (`tcp.Server`, `tcp.ConnectionHandler`, `tcp.ConnContext`)
  - Implementation: `pkg/tcp/tcp_server.go` (`tcp.TCPServer`)
  - Spec: `docs/clarity/05-tcp-contract.md`
  - Tests-as-spec: `pkg/tcp/*_test.go`

## 2) Contract specs: what must be documented

Contract specs are **behavioral guarantees** for an interface.

At minimum, a contract spec must cover:

- **Lifecycle**: `Start/Stop` semantics, idempotency, blocking/non-blocking, shutdown guarantees.
- **Thread-safety**: which methods are safe concurrently.
- **Fail-fast rules**:
  - What **panics** (programmer errors)
  - What returns **errors** (runtime/operational errors)
- **Backpressure**: bounded queues, rejection behavior, and metrics.
- **Encoding & payload rules**: e.g. EventBus “JSON-first” (`[]byte` bodies vs structs).
- **Observability**: request-id propagation, metrics fields meaning.

### Panic vs error (policy)

In Fluxor we use:

- **panic** for **programmer errors** (invalid address, nil handler, nil vertx, etc.)
- **error** for **runtime errors** (network failures, timeouts, no handlers, etc.)

Concrete example:

- `core.EventBus.Consumer(address)` is **fail-fast** on invalid address (panic).
- `core.EventBus.Send/Publish/Request` returns **errors** for invalid inputs/runtime failures.

## 3) “Code is truth”: how docs stay honest

Docs are not the source of truth by themselves.

Rules:

- If a doc claims a behavior, there must be either:
  - a test asserting it, or
  - a clear implementation point (with stable behavior).
- If tests disagree with docs, **docs must change**.
- If docs describe desired behavior but code doesn’t do it yet, docs must label it as **Future work**.

### Recommended pattern: tests-as-contract

For any new contract statement, add at least one test:

- Fail-fast input validation
- Backpressure rejection
- Panic isolation (per-request / per-connection)
- Start/Stop shutdown guarantee

## 4) Where contract docs live

All contract specs live under:

- `docs/clarity/*-contract.md` (or `*-spec.md`)

Current examples:

- `docs/clarity/05-tcp-contract.md` — TCP server contract

## 5) When to update contract docs

Update contract docs whenever you change any of:

- exported interface methods/signatures
- fail-fast vs error behavior
- message encoding/decoding rules
- lifecycle semantics (blocking vs non-blocking, shutdown behavior)
- backpressure strategy and metrics meaning

