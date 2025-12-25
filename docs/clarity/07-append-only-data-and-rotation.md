# Append-only Data Rule (Logs/Events) + In-memory-first + Disk Rotation

This repo’s default data philosophy for **event-like data** (logs, events, traces, workflow history, audit streams) is:

- **Append-only**: never mutate existing records in place.
- **In-memory-first**: accept/write fast to RAM.
- **Rotate to disk**: periodically flush/roll segments to durable storage for persistence.

This doc is a **contract-style architecture note**: it defines the rules and the guarantees we expect if/when we implement persistent, log-like storage in Fluxor.

## 1) The append-only rule

### What it means

- Writes are **only** `append(record)`.
- No `update(record)` and no “in-place delete”.
- Correction happens by **appending** a new record (tombstone / compensating event).

### Why

- Simpler concurrency and replication.
- Natural fit for EventBus-style systems.
- Enables cheap recovery (replay).
- Makes audit trails and debugging reliable.

## 2) In-memory-first write path

### Primary goal

Fast ingestion under load, with bounded memory and explicit backpressure (same spirit as `pkg/web` + `pkg/tcp`).

### Suggested write pipeline

1. **Validate & timestamp** (fail-fast for programmer errors).
2. **Append to in-memory buffer** (bounded).
3. **Ack to caller** (depending on durability mode).
4. Background worker **flushes** to disk in append-only segments.

## 3) Rotate-to-disk persistence model

### Segment files (log-structured)

Disk persistence is represented as immutable **segment files**:

- Segment name: `<stream>/<epoch>-<seq>.log` (example only)
- Content: records appended sequentially
- Optional index: `<segment>.idx` mapping logical offsets → file positions

### Rotation

Rotation triggers when any threshold is hit:

- Max segment size (e.g., 64MB / 256MB)
- Max segment age (e.g., 1m / 10m)
- Memory pressure (forced flush)

After rotation:
- Current segment is sealed (immutable).
- New active segment is created.

### Retention

Retention is applied by deleting old **sealed segments**:

- Time-based (keep last N hours/days)
- Size-based (keep last N GB)

## 4) Durability levels (explicit contract)

We should expose explicit durability modes (interface-first):

- **RAM-only**: ack after in-memory append (fastest, volatile).
- **WAL-backed**: ack after write-ahead log append (durable on crash).
- **FSYNC**: ack after fsync on segment (strongest, slowest).

Docs must state which mode is default per module.

## 5) Recovery (code is truth)

On restart:

1. Discover segments on disk.
2. Rebuild indexes (or load `.idx`).
3. Replay records into in-memory structures as needed.
4. Resume appends at the last committed offset.

**Invariant**: records in sealed segments are never rewritten.

## 6) Consistency & ordering guarantees

Define per-stream guarantees:

- **Ordering**: record order is the append order within a stream.
- **Idempotency**: consumers must tolerate replays.
- **At-least-once**: persistence replay implies duplicates are possible unless dedup is implemented.

## 7) Backpressure (must be first-class)

If in-memory buffers are full, we must fail-fast:

- return an error / reject the write
- increment rejection metrics
- never block indefinitely

This mirrors the philosophy already used in `pkg/web` and `pkg/tcp`.

## 8) How this relates to Fluxor modules

- **EventBus**: EventBus itself is messaging, not persistence — but append-only persistence can be a backing store for:
  - workflow history (`pkg/workflow`)
  - audit/event streams emitted by services
  - “outbox” patterns for reliable publication
- **Observability**: logs/traces are naturally append-only and rotation-friendly.

## 9) What to do next (implementation checklist)

If we implement this, we should do it interface-first:

1. Define `pkg/<module>/Store` interface:
   - `Append(stream, record) (offset, error)`
   - `Read(stream, fromOffset, limit) ([]record, error)`
   - `Rotate()` / `Close()` / `Stats()`
2. Add a contract spec in `docs/clarity/*-contract.md`.
3. Add tests for:
   - append-only invariant
   - rotation behavior
   - crash recovery (replay)
   - backpressure rejection

