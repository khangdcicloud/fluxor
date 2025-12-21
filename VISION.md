# Vision, Differentiators, and MVP Scope

## Go Vert.x‑style Reactive Runtime Framework

---

## 1. Vision

### One‑sentence vision

> **Create a lightweight, standalone reactive runtime for Go that enforces structural concurrency (reactors, worker isolation, event bus), enabling a new generation of high‑performance, predictable, and “fun” backend systems beyond traditional HTTP servers and heavy gateways.**

### Longer vision

Modern backends are increasingly **event‑driven, high‑concurrency, and middleware‑heavy**, yet current Go solutions either:

* rely on unstructured goroutine usage, or
* depend on heavy infrastructure (gateways, meshes, control‑planes).

This project envisions a **runtime‑first framework**, inspired by Vert.x (Netty‑based) and OTP ideas, but implemented **natively on Go’s runtime**, to let developers build **independent services** that feel like:

* a **mini runtime / platform**, not just a web app,
* fast like a Go binary,
* disciplined like Vert.x,
* operable as a standalone service (OOB of any platform).

The framework is meant to **unlock new backend architectures**, not just optimize old ones.

---

## 2. Core Philosophy

### Runtime over framework

* This is **not** an HTTP framework.
* This is **not** an API Gateway product.
* This is a **runtime** that applications are built *on top of*.

### Structural concurrency, not accidental concurrency

* Concurrency must be **designed and enforced**, not left to “spawn goroutines and hope”.
* Predictability > raw throughput.
* Backpressure is a feature, not a failure.

### Standalone by default

* Services built on this runtime can run:

  * without a control‑plane,
  * without a service mesh,
  * without a database dependency at runtime.
* Cloud and platform integrations are optional, not required.

---

## 3. Key Differentiators

### 3.1 Vert.x‑style runtime semantics in Go (unique gap)

* Vert.x’s real value is **not Netty**, but:

  * event loops
  * worker isolation
  * deployment model
  * event bus
* Go already provides Netty‑level non‑blocking I/O natively.
* This framework applies **Vert.x runtime discipline directly on Go’s runtime**.

> **No existing Go framework does this end‑to‑end.**

---

### 3.2 Enforced reactor model (not “best practice”, but rule)

* One reactor = one goroutine
* Sequential execution per reactor
* Bounded mailbox
* No blocking allowed inside reactor

This removes:

* goroutine explosions
* lock contention chaos
* hidden latency spikes

---

### 3.3 Explicit worker isolation (executeBlocking semantics)

* Blocking and heavy work must go through a **WorkerPool**
* Fixed size, bounded queue
* Backpressure instead of silent degradation

This mirrors Vert.x’s strongest guarantee.

---

### 3.4 First‑class Event Bus (message‑first design)

* Message passing over shared state
* Publish / send / request‑reply
* Correlation IDs and timeouts
* Reactor‑bound handlers

This enables:

* microservice‑style composition *inside* one process
* clean decoupling without network overhead

---

### 3.5 “Out‑of‑Band” (OOB) runtime nature

Like Vert.x in Java:

* Not tied to servlet containers
* Not tied to DI frameworks
* Not tied to enterprise stacks

Services built on this runtime:

* start fast
* behave like independent systems
* feel closer to Node.js / OTP than traditional Go apps

---

### 3.6 Fun as a first‑class goal

“Fun” here means:

* Deploying components dynamically
* Watching runtime state (reactors, mailboxes, components)
* Using timers, bus messaging, and supervision naturally
* Feeling like you’re programming a **live system**, not wiring handlers

This increases:

* developer productivity
* architectural clarity
* long‑term maintainability

---

## 4. What This Enables (New Backend Patterns)

With this framework, teams can build:

* Event‑driven microservices without Kafka/mesh overhead
* Middleware backends that embed business logic safely
* BFFs that aggregate logic without thread/goroutine chaos
* Edge or offline‑capable services using config snapshots
* APIM / Gateway cores **without** heavy platforms

This is not incremental improvement — it’s **a new class of backend runtime**.

---

## 5. MVP Scope (Strict and Focused)

### 5.1 MVP Goals

The MVP must prove:

* Structural concurrency works in Go
* Runtime discipline is enforceable
* Services feel “Vert.x‑like” to develop
* High concurrency does not cause chaos

---

### 5.2 Included in MVP

#### Core runtime

* Reactor (event loop)

  * single goroutine
  * bounded mailbox
  * timers (one‑shot + periodic)
* WorkerPool

  * fixed size
  * bounded queue
  * cancellation support
* Runtime

  * deploy / undeploy components
  * bind components to reactors
  * panic isolation (no process crash)
* Component (Verticle‑like abstraction)

  * Start / Stop lifecycle

#### Messaging

* Local Event Bus

  * publish
  * send
  * request‑reply with timeout
  * correlation IDs
  * reactor‑bound handlers

#### Dev experience

* Small runtime inspector (code‑level or debug endpoint):

  * list reactors
  * mailbox sizes
  * deployed components
* Example microservice using:

  * multiple components
  * event bus request‑reply
  * worker pool for blocking work
  * optional HTTP adapter as trigger

---

### 5.3 Explicitly OUT of MVP

* No API Gateway features
* No load balancing
* No APIM policies
* No cluster bus
* No persistence abstractions
* No Kubernetes / Istio / service mesh
* No UI or admin console
* No plugins/WASM (future only)

---

## 6. Success Criteria for MVP

The MVP is successful if:

* Reactor code never blocks
* Backpressure is observable and testable
* No unbounded goroutine growth under load
* Event bus handlers always execute on reactors
* Example service remains stable under concurrency
* Developers report the runtime feels:

  * different from net/http
  * closer to Vert.x / Node.js / OTP

---

## 7. Long‑Term Direction (Post‑MVP)

After MVP validation:

* Config snapshot hot‑swap
* Cluster event bus (NATS)
* WASM‑based plugins/policies
* Supervision strategies (restart, backoff)
* Runtime‑level observability

---

## 8. Final Statement

This project does not aim to replace existing Go frameworks.
It aims to **fill a missing layer**:

> **A disciplined, reactive runtime for Go that makes building complex, high‑concurrency backend systems both predictable and enjoyable.**

**End of document**
