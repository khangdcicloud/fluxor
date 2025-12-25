# Core Runtime Deep Dive: Vertx, Verticle, Context, EventBus

This document provides a detailed analysis of `pkg/core/` focusing on:
1. How deploy/undeploy verticles work
2. Root context lifecycle
3. EventBus instance creation, closure, thread-safety, and ownership
4. **Confusing spots → Suggested fixes**

---

## 1. Deploy/Undeploy Verticles Flow

### DeployVerticle Sequence

```
DeployVerticle(verticle)
    │
    ├─► ValidateVerticle(verticle)     // Fail-fast: nil check
    │
    ├─► v.mu.Lock()                    // Write lock acquired
    │
    ├─► generateDeploymentID()         // "deployment.<uuid>"
    │
    ├─► newContext(v.ctx, v)           // Creates FluxorContext wrapping root ctx
    │
    ├─► Create deployment struct
    │
    ├─► Is AsyncVerticle?
    │   │
    │   ├─► YES: asyncVerticle.AsyncStart(ctx, callback)
    │   │        ⚠️ Non-blocking! Deployment added to map BEFORE completion
    │   │        └─► On error: removes from map in callback
    │   │
    │   └─► NO: verticle.Start(ctx)
    │            ⚠️ Blocking! Lock held during entire Start()
    │
    ├─► v.deployments[deploymentID] = dep
    │
    └─► v.mu.Unlock()
```

### UndeployVerticle Sequence

```
UndeployVerticle(deploymentID)
    │
    ├─► Validate deploymentID (empty check)
    │
    ├─► v.mu.Lock()
    │   ├─► Check existence
    │   ├─► delete(v.deployments, deploymentID)
    │   └─► v.mu.Unlock()              // Lock released BEFORE Stop()
    │
    └─► Is AsyncVerticle?
        │
        ├─► YES: asyncVerticle.AsyncStop(ctx, callback)
        │        └─► Errors logged in callback
        │
        └─► NO: verticle.Stop(ctx)
                 └─► Errors returned immediately
```

### Key Observations

| Aspect | Sync Verticle | Async Verticle |
|--------|---------------|----------------|
| Start blocking | YES (lock held) | NO (returns immediately) |
| Error handling | Returns error | Callback logs, removes from map |
| Added to map | After Start() succeeds | BEFORE AsyncStart() completes |
| Stop blocking | YES | NO |

---

## 2. Root Context Lifecycle

### Context Hierarchy

```
                    ┌─────────────────────────────┐
                    │  Parent context.Context     │
                    │  (passed to NewVertx)       │
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │  vertx.ctx                  │
                    │  context.WithCancel(parent) │
                    │  + vertx.cancel             │
                    └─────────────┬───────────────┘
                                  │
          ┌───────────────────────┼───────────────────────┐
          │                       │                       │
          ▼                       ▼                       ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────┐
│ eventBus.ctx    │   │ deployment[0]   │   │ deployment[N]   │
│ WithCancel(     │   │ .ctx =          │   │ .ctx =          │
│   vertx.ctx)    │   │ FluxorContext   │   │ FluxorContext   │
│ + eventBus.     │   │ wrapping        │   │ wrapping        │
│   cancel        │   │ vertx.ctx       │   │ vertx.ctx       │
└─────────────────┘   └─────────────────┘   └─────────────────┘
```

### Lifecycle Timeline

```
1. NewVertx(parentCtx)
   └─► ctx, cancel := context.WithCancel(parentCtx)
   └─► Creates EventBus (with its own WithCancel)
   └─► vertx ready

2. DeployVerticle(v)
   └─► newContext(vertx.ctx, vertx) → FluxorContext per deployment
   └─► verticle.Start(fluxorCtx)

3. vertx.Close()
   └─► For each deployment: UndeployVerticle(id)
       └─► verticle.Stop(fluxorCtx)
   └─► vertx.cancel()              // Cancels vertx.ctx
   └─► eventBus.Close()
       └─► eventBus.cancel()       // Cancels eventBus.ctx (redundant?)
       └─► executor.Shutdown()
       └─► Close all mailboxes
```

### ⚠️ Context Cancel Order Issue

In `vertx.Close()`:
```go
v.cancel()                 // Line 188: cancels vertx.ctx
return v.eventBus.Close()  // Line 189: eventBus.Close() also cancels
```

**Problem**: `eventBus.ctx` is derived from `vertx.ctx`, so when `v.cancel()` is called, `eventBus.ctx` is already cancelled. The `eventBus.cancel()` call inside `eventBus.Close()` is redundant.

---

## 3. EventBus Instance: Creation, Closure, Thread-Safety, Ownership

### Creation

```go
// In NewVertxWithOptions (vertx.go:60-81)
func NewVertxWithOptions(ctx context.Context, opts VertxOptions) (Vertx, error) {
    ctx, cancel := context.WithCancel(ctx)
    v := &vertx{...}

    if opts.EventBusFactory != nil {
        // Custom EventBus (e.g., NATS cluster)
        bus, err := opts.EventBusFactory(ctx, v)  // ← vertx passed to factory
        v.eventBus = bus
    } else {
        // Default in-memory
        v.eventBus = NewEventBus(ctx, v)          // ← vertx passed here too
    }
}
```

### Closure

```go
// In vertx.Close() (vertx.go:172-190)
func (v *vertx) Close() error {
    // 1. Undeploy all verticles
    for _, dep := range deployments {
        v.UndeployVerticle(dep.id)
    }
    // 2. Cancel root context
    v.cancel()
    // 3. Close EventBus
    return v.eventBus.Close()
}

// In eventBus.Close() (eventbus_impl.go:251-272)
func (eb *eventBus) Close() error {
    eb.cancel()                        // Cancel eventBus context
    eb.executor.Shutdown(ctx)          // Shutdown worker pool
    // Close all consumer mailboxes
    for _, consumers := range eb.consumers {
        for _, c := range consumers {
            c.mailbox.Close()
        }
    }
}
```

### Thread-Safety

| Component | Mutex | Protection Scope |
|-----------|-------|------------------|
| `vertx` | `sync.RWMutex` | `deployments` map |
| `eventBus` | `sync.RWMutex` | `consumers` map |
| `consumer` | `sync.RWMutex` | `handler` field |
| `message` | `sync.RWMutex` | `body`, `headers`, `replyAddress` |

**Concurrent safety:**
- Multiple goroutines can call `Publish/Send/Request` safely (RLock on consumers)
- `DeployVerticle` serializes deployments (Lock)
- Consumer handlers run in executor goroutines with panic isolation

### Ownership Diagram

```
┌─────────────────────────────────────────────────────────┐
│                        vertx                            │
│  OWNS:                                                  │
│  ├── eventBus (interface)                               │
│  ├── deployments map[string]*deployment                 │
│  ├── ctx (context.Context)                              │
│  ├── cancel (context.CancelFunc)                        │
│  └── logger                                             │
└───────────────────────┬─────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────┐
│                      eventBus                           │
│  OWNS:                                                  │
│  ├── consumers map[string][]*consumer                   │
│  ├── executor (concurrency.Executor)                    │
│  ├── ctx (context.Context) ← derived from vertx.ctx    │
│  ├── cancel (context.CancelFunc)                        │
│  └── logger                                             │
│                                                         │
│  REFERENCES (not owns):                                 │
│  └── vertx Vertx  ⚠️ CIRCULAR REFERENCE                │
└─────────────────────────────────────────────────────────┘
```

**⚠️ Circular Reference**: `vertx → eventBus → vertx`

This is intentional (EventBus needs Vertx to create FluxorContext for consumers), but can cause confusion about ownership and lifecycle.

---

## 4. Confusing Spots → Suggested Fixes

### Issue #1: `ctx` Field Name Collision

**Location**: `vertx.go:35`, `eventbus_impl.go:17`, `context.go:33`

**Problem**:
```go
// vertx struct
ctx context.Context      // This is context.Context

// deployment struct  
ctx FluxorContext        // This is FluxorContext!

// vertxContext struct
ctx context.Context      // Back to context.Context
```

Same field name `ctx` with different types across structs.

**Suggested Fix**:
```go
// vertx struct
rootCtx context.Context        // Rename to clarify it's the root

// deployment struct
fluxorCtx FluxorContext        // Explicit type in name

// Or use consistent naming:
// goCtx for context.Context
// fCtx for FluxorContext
```

---

### Issue #2: `newContext` Function Name Misleading

**Location**: `context.go:38`

**Problem**:
```go
func newContext(ctx context.Context, vertx Vertx) FluxorContext
```

Looks like it creates `context.Context` but actually creates `FluxorContext`.

**Suggested Fix**:
```go
func newFluxorContext(ctx context.Context, vertx Vertx) FluxorContext
// Or
func NewVerticleContext(ctx context.Context, vertx Vertx) FluxorContext
```

---

### Issue #3: AsyncVerticle Race Condition Pattern

**Location**: `vertx.go:107-127`

**Problem**:
```go
// Deployment added to map BEFORE AsyncStart completes
if asyncVerticle, ok := verticle.(AsyncVerticle); ok {
    asyncVerticle.AsyncStart(ctx, func(err error) {
        if err != nil {
            // Remove on failure - but other code may have seen it!
            delete(v.deployments, deploymentID)
        }
    })
}
v.deployments[deploymentID] = dep  // Added immediately
return deploymentID, nil           // Returned before AsyncStart completes
```

Callers get a deploymentID that may become invalid if AsyncStart fails.

**Suggested Fix**:
```go
// Option A: Don't add to map until callback confirms success
// Option B: Add "pending" state to deployment struct
type deployment struct {
    id       string
    verticle Verticle
    ctx      FluxorContext
    state    deploymentState  // PENDING, STARTED, STOPPED, FAILED
}

// Option C: Document clearly that deploymentID may become invalid
// and provide DeploymentExists(id) method
```

---

### Issue #4: Lock Held During Sync Verticle Start

**Location**: `vertx.go:94-125`

**Problem**:
```go
v.mu.Lock()
defer v.mu.Unlock()
// ... 
if err := verticle.Start(ctx); err != nil {  // BLOCKS while locked!
    return "", fmt.Errorf("verticle start failed: %w", err)
}
```

If `verticle.Start()` is slow, all other `DeployVerticle` calls block.

**Suggested Fix**:
```go
// Option A: Release lock before Start(), re-acquire after
v.mu.Lock()
deploymentID := generateDeploymentID()
dep := &deployment{...}
v.deployments[deploymentID] = dep
dep.state = STARTING
v.mu.Unlock()

if err := verticle.Start(ctx); err != nil {
    v.mu.Lock()
    delete(v.deployments, deploymentID)
    v.mu.Unlock()
    return "", err
}

v.mu.Lock()
dep.state = STARTED
v.mu.Unlock()

// Option B: Document that Start() should be fast and non-blocking
```

---

### Issue #5: Double Context Cancellation

**Location**: `vertx.go:188-189`, `eventbus_impl.go:252`

**Problem**:
```go
// vertx.Close()
v.cancel()                // Cancels vertx.ctx
return v.eventBus.Close() // eventBus.Close() also calls eb.cancel()

// eventBus.Close()
eb.cancel()               // Redundant - eb.ctx is child of vertx.ctx
```

**Suggested Fix**:
```go
// Option A: Remove eb.cancel() from eventBus.Close() - parent already cancelled
func (eb *eventBus) Close() error {
    // eb.cancel()  // Not needed - vertx.cancel() already cancelled this
    
    // Just cleanup resources
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    eb.executor.Shutdown(shutdownCtx)
    // ...
}

// Option B: Document the intentional redundancy as defense-in-depth
// Add comment explaining why both cancel calls exist
```

---

### Issue #6: Consumer() Panics, Other Methods Return Errors

**Location**: `eventbus_impl.go:223-225`

**Problem**:
```go
func (eb *eventBus) Consumer(address string) Consumer {
    if err := ValidateAddress(address); err != nil {
        FailFast(err)  // PANICS!
    }
    // ...
}

// But Publish, Send, Request all return errors:
func (eb *eventBus) Publish(address string, body interface{}) error {
    if err := ValidateAddress(address); err != nil {
        return err  // Returns error, doesn't panic
    }
}
```

Inconsistent error handling.

**Suggested Fix**:
```go
// Option A: Return (Consumer, error) for consistency
func (eb *eventBus) Consumer(address string) (Consumer, error) {
    if err := ValidateAddress(address); err != nil {
        return nil, err
    }
    // ...
}

// Option B: Document why Consumer panics (fail-fast for programmer errors)
// Consumer creates a builder pattern object, invalid address is a bug
// Add clear godoc comment explaining the panic behavior
```

---

### Issue #7: `vertx` Field in EventBus Creates Circular Dependency

**Location**: `eventbus_impl.go:19`

**Problem**:
```go
type eventBus struct {
    // ...
    vertx Vertx  // EventBus references back to Vertx
}
```

Creates circular dependency: `Vertx → EventBus → Vertx`

**Suggested Fix**:
```go
// Option A: Pass only what's needed (context factory)
type eventBus struct {
    contextFactory func() FluxorContext
}

// Option B: Document the circular reference explicitly
// Add comment explaining why it exists (to create FluxorContext for consumers)
// and that it doesn't cause memory leaks (both cleaned up together)
```

---

### Issue #8: Unclear Ownership of `deployment.ctx`

**Location**: `vertx.go:196-200`

**Problem**:
```go
type deployment struct {
    id       string
    verticle Verticle
    ctx      FluxorContext  // Who owns this? Is it safe to use after undeploy?
}
```

When verticle is undeployed, is `ctx` still valid? Can verticle cache it?

**Suggested Fix**:
```go
// Add documentation:
type deployment struct {
    id       string
    verticle Verticle
    // ctx is the FluxorContext passed to this verticle's Start/Stop methods.
    // It remains valid for the lifetime of the deployment.
    // After UndeployVerticle, this context should not be used.
    // The underlying context.Context will be cancelled when Vertx.Close() is called.
    ctx      FluxorContext
}
```

---

## Summary: Quick Reference

| Issue | Location | Severity | Fix Type | Status |
|-------|----------|----------|----------|--------|
| #1 `ctx` name collision | vertx.go, context.go | Medium | Rename | ✅ FIXED |
| #2 `newContext` misleading | context.go:38 | Low | Rename | ✅ FIXED |
| #3 AsyncVerticle race | vertx.go:107-127 | High | Add state | ✅ FIXED |
| #4 Lock during Start | vertx.go:94-125 | Medium | Release lock | ✅ FIXED |
| #5 Double cancel | vertx.go:188-189 | Low | Document | ✅ FIXED |
| #6 Consumer panics | eventbus.go, eventbus_impl.go | Medium | Document | ✅ FIXED |
| #7 Circular reference | eventbus_impl.go:19 | Low | Document | ✅ FIXED |
| #8 Unclear ownership | vertx.go:196-200 | Low | Document | ✅ FIXED |

---

## Implementation Summary

All 8 issues have been addressed:

### Code Changes

1. **Issue #1**: Renamed `ctx` → `rootCtx` in vertx, `ctx` → `goCtx` in vertxContext, `ctx` → `fluxorCtx` in deployment
2. **Issue #2**: Renamed `newContext()` → `newFluxorContext()` across all files
3. **Issue #3**: Added `DeploymentState` enum with `PENDING`, `STARTED`, `FAILED`, `STOPPING`, `STOPPED` states
4. **Issue #4**: Lock is now released before calling `verticle.Start()`, re-acquired to update state
5. **Issue #5**: Added documentation in `vertx.Close()` explaining the intentional redundancy
6. **Issue #6**: Added comprehensive documentation in `EventBus` interface explaining panic vs error behavior
7. **Issue #7**: Added documentation in `eventBus` struct explaining circular reference and why it's safe
8. **Issue #8**: Added documentation in `deployment` struct explaining ownership and lifecycle

### Files Modified

- `pkg/core/vertx.go`
- `pkg/core/context.go`
- `pkg/core/eventbus.go`
- `pkg/core/eventbus_impl.go`
- `pkg/core/eventbus_cluster_jetstream.go`
- `pkg/core/eventbus_cluster_nats.go`
- `pkg/core/context_test.go`
- `pkg/fluxor/fluxor.go`
