# Java-Style Abstract Base Classes (Premium Pattern)

Fluxor provides Java-style abstract base classes with default implementations, following the Template Method pattern. This allows developers to extend base functionality while customizing specific behaviors through hook methods.

## Premium Pattern Overview

The **Premium Pattern** is Fluxor's implementation of Java-style abstract base classes in Go. It provides:

- **Enterprise-grade abstractions**: Production-ready base classes with common functionality
- **Template Method Pattern**: Skeleton algorithms with customizable hook methods
- **Code reuse**: Eliminate boilerplate and ensure consistency
- **Type safety**: Compile-time guarantees with Go's type system
- **Developer experience**: Familiar patterns for Java/enterprise developers

This pattern is called "Premium" because it provides:
1. **Higher-level abstractions** than raw interfaces
2. **Default implementations** that work out of the box
3. **Best practices** built-in (lifecycle management, error handling, logging)
4. **Enterprise patterns** (service layer, component composition, handler chains)

---

## Design Pattern: Template Method

All base classes follow the **Template Method Pattern**:
- Base class defines the skeleton of an algorithm
- Hook methods (`doStart`, `doStop`, `doHandle`, etc.) allow subclasses to customize behavior
- Common functionality is provided by default

## Base Classes

### 1. BaseVerticle

**Purpose**: Abstract base class for verticles with common lifecycle management

**Features**:
- Automatic lifecycle management (start/stop state tracking)
- Consumer registration and cleanup
- Convenience methods for EventBus operations
- Context and reference caching

**Usage**:
```go
type MyVerticle struct {
    *core.BaseVerticle
}

// Override hook method
func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    // Custom initialization
    consumer := v.Consumer("my.address")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        return msg.Reply("processed")
    })
    return nil
}

// Optional: Override cleanup
func (v *MyVerticle) doStop(ctx core.FluxorContext) error {
    // Custom cleanup
    return nil
}

// Create and deploy
verticle := &MyVerticle{
    BaseVerticle: core.NewBaseVerticle("my-verticle"),
}
vertx.DeployVerticle(verticle)
```

**Hook Methods**:
- `doStart(ctx)` - Called during Start(), override for custom initialization
- `doStop(ctx)` - Called during Stop(), override for custom cleanup

**Convenience Methods**:
- `Consumer(address)` - Create and register consumer
- `Publish(address, body)` - Publish message
- `Send(address, body)` - Send message
- `EventBus()` - Get EventBus reference
- `Vertx()` - Get Vertx reference
- `Context()` - Get FluxorContext

---

### 2. BaseService

**Purpose**: Abstract base class for service verticles (request-reply pattern)

**Features**:
- Automatic service registration
- Request handling infrastructure
- Reply/Fail convenience methods

**Usage**:
```go
type UserService struct {
    *core.BaseService
}

// Override hook method
func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    // Process request
    userID := msg.Body().(string)
    userData := map[string]interface{}{
        "id":   userID,
        "name": "John Doe",
    }
    return s.Reply(msg, userData)
}

// Create service
service := &UserService{
    BaseService: core.NewBaseService("user-service", "user.service"),
}
vertx.DeployVerticle(service)
```

**Hook Methods**:
- `doHandleRequest(ctx, msg)` - Handle incoming service requests
- `doStart(ctx)` - Custom initialization (inherited from BaseVerticle)
- `doStop(ctx)` - Custom cleanup (inherited from BaseVerticle)

**Convenience Methods**:
- `Reply(msg, body)` - Reply to request
- `Fail(msg, code, message)` - Fail request
- All BaseVerticle methods

---

### 3. BaseHandler

**Purpose**: Abstract base class for message handlers

**Features**:
- Structured logging with request ID
- JSON encoding/decoding utilities
- Reply/Fail convenience methods

**Usage**:
```go
type UserHandler struct {
    *core.BaseHandler
}

// Override hook method
func (h *UserHandler) doHandle(ctx core.FluxorContext, msg core.Message) error {
    // Decode request
    var request map[string]interface{}
    if err := h.DecodeBody(msg, &request); err != nil {
        return h.Fail(msg, 400, "Invalid request")
    }
    
    // Process
    userID := request["id"].(string)
    userData := map[string]interface{}{
        "id":   userID,
        "name": "John Doe",
    }
    
    return h.Reply(msg, userData)
}

// Create handler
handler := &UserHandler{
    BaseHandler: core.NewBaseHandler("user-handler"),
}

// Use in verticle
consumer.Handler(handler.Handle)
```

**Hook Methods**:
- `doHandle(ctx, msg)` - Implement handler logic

**Convenience Methods**:
- `Reply(msg, body)` - Reply to message
- `Fail(msg, code, message)` - Fail message
- `DecodeBody(msg, v)` - Decode JSON message body
- `EncodeBody(data)` - Encode data to JSON

---

### 4. BaseComponent

**Purpose**: Abstract base class for reusable components

**Features**:
- Component lifecycle management
- Parent verticle reference
- Access to parent's EventBus/Vertx

**Usage**:
```go
type DatabaseComponent struct {
    *core.BaseComponent
    connection string
}

// Override hook methods
func (c *DatabaseComponent) doStart(ctx core.FluxorContext) error {
    c.connection = "connected"
    return nil
}

func (c *DatabaseComponent) doStop(ctx core.FluxorContext) error {
    c.connection = "disconnected"
    return nil
}

// Use in verticle
type MyVerticle struct {
    *core.BaseVerticle
    db *DatabaseComponent
}

func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    v.db.SetParent(v.BaseVerticle)
    return v.db.Start(ctx)
}

func (v *MyVerticle) doStop(ctx core.FluxorContext) error {
    return v.db.Stop(ctx)
}
```

**Hook Methods**:
- `doStart(ctx)` - Component initialization
- `doStop(ctx)` - Component cleanup

**Methods**:
- `SetParent(parent)` - Set parent verticle
- `Parent()` - Get parent verticle
- `EventBus()` - Get EventBus from parent
- `Vertx()` - Get Vertx from parent

---

## Benefits of Base Classes

### 1. **Code Reuse**
- Common functionality implemented once
- Reduces boilerplate code
- Consistent patterns across codebase

### 2. **Template Method Pattern**
- Clear extension points (hook methods)
- Enforced structure and lifecycle
- Easy to understand and maintain

### 3. **Java Developer Familiarity**
- Familiar patterns for Java developers
- Abstract base class concept
- Inheritance-like behavior through embedding

### 4. **Type Safety**
- Go's type system ensures correctness
- Compile-time checks
- No runtime reflection needed

### 5. **Composition over Inheritance**
- Go's struct embedding provides composition
- More flexible than traditional inheritance
- Can embed multiple base classes

---

## Comparison: Java vs Go

### Java Abstract Class
```java
public abstract class BaseVerticle {
    protected EventBus eventBus;
    
    public final void start(Vertx vertx) {
        this.eventBus = vertx.eventBus();
        doStart();
    }
    
    protected abstract void doStart();
}
```

### Go Base Class (Fluxor)
```go
type BaseVerticle struct {
    eventBus EventBus
}

func (bv *BaseVerticle) Start(ctx FluxorContext) error {
    bv.eventBus = ctx.EventBus()
    return bv.doStart(ctx)
}

func (bv *BaseVerticle) doStart(ctx FluxorContext) error {
    return nil // Default: no-op, subclasses override
}
```

**Key Differences**:
- Go uses struct embedding instead of inheritance
- Hook methods are regular methods (not abstract)
- Default implementations provided (can be overridden)
- More flexible composition model

---

## Best Practices

1. **Always call base methods**: When overriding, call parent methods if needed
2. **Use hook methods**: Override `doStart`, `doStop`, etc., not `Start`/`Stop`
3. **Register consumers**: Use `Consumer()` method for automatic cleanup
4. **Set parent for components**: Components need parent reference for EventBus access
5. **Handle errors**: Return errors from hook methods for proper error handling

---

## Example: Complete Service

```go
type UserService struct {
    *core.BaseService
    db *DatabaseComponent
}

func NewUserService() *UserService {
    return &UserService{
        BaseService: core.NewBaseService("user-service", "user.service"),
        db: &DatabaseComponent{
            BaseComponent: core.NewBaseComponent("database"),
        },
    }
}

func (s *UserService) doStart(ctx core.FluxorContext) error {
    // Initialize database component
    s.db.SetParent(s.BaseVerticle)
    if err := s.db.Start(ctx); err != nil {
        return err
    }
    return nil
}

func (s *UserService) doStop(ctx core.FluxorContext) error {
    // Cleanup database component
    return s.db.Stop(ctx)
}

func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    // Process request using database component
    userID := msg.Body().(string)
    userData := s.db.GetUser(userID)
    return s.Reply(msg, userData)
}
```

This demonstrates:
- Service extending BaseService
- Component composition
- Lifecycle management
- Request handling

---

---

## Premium Pattern Benefits

### Why Use Premium Patterns?

1. **Faster Development**
   - Less boilerplate code
   - Common patterns pre-implemented
   - Focus on business logic, not infrastructure

2. **Consistency**
   - Enforced patterns across codebase
   - Standard lifecycle management
   - Uniform error handling

3. **Maintainability**
   - Clear extension points
   - Easy to understand structure
   - Reduced cognitive load

4. **Enterprise Ready**
   - Production-tested patterns
   - Built-in observability (logging, request IDs)
   - Proper resource management

5. **Team Collaboration**
   - Familiar patterns for Java developers
   - Clear contracts and interfaces
   - Easy code reviews

### When to Use Premium Patterns

**Use Base Classes When**:
- ✅ Building services or verticles
- ✅ Need consistent lifecycle management
- ✅ Want to reduce boilerplate
- ✅ Team has Java/enterprise background
- ✅ Building reusable components

**Use Raw Interfaces When**:
- ⚠️ Need maximum flexibility
- ⚠️ Building low-level infrastructure
- ⚠️ Performance-critical paths
- ⚠️ Simple, one-off implementations

---

## Premium Pattern Hierarchy

```
BaseVerticle (Foundation)
    ├── BaseService (Request-Reply Services)
    └── [Custom Verticles]

BaseHandler (Message Handlers)
    └── [Custom Handlers]

BaseComponent (Reusable Components)
    └── [Custom Components]
```

**Composition Example**:
```go
BaseService
    ├── BaseVerticle (lifecycle, EventBus)
    └── BaseComponent (database, cache, etc.)
        └── BaseHandler (request processing)
```

---

## Premium Pattern Examples

### Example 1: Enterprise Service

```go
// Premium Pattern: BaseService + BaseComponent
type OrderService struct {
    *core.BaseService
    db    *DatabaseComponent
    cache *CacheComponent
}

func NewOrderService() *OrderService {
    return &OrderService{
        BaseService: core.NewBaseService("order-service", "order.service"),
        db:          NewDatabaseComponent(),
        cache:       NewCacheComponent(),
    }
}

func (s *OrderService) doStart(ctx core.FluxorContext) error {
    // Initialize components (Premium Pattern handles lifecycle)
    s.db.SetParent(s.BaseVerticle)
    s.cache.SetParent(s.BaseVerticle)
    
    if err := s.db.Start(ctx); err != nil {
        return err
    }
    return s.cache.Start(ctx)
}

func (s *OrderService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    // Business logic with premium infrastructure
    orderID := msg.Body().(string)
    
    // Use components (Premium Pattern provides access)
    order, err := s.db.GetOrder(orderID)
    if err != nil {
        return s.Fail(msg, 500, err.Error())
    }
    
    // Cache result
    s.cache.Set(orderID, order)
    
    return s.Reply(msg, order)
}
```

### Example 2: Premium Handler Chain

```go
// Premium Pattern: BaseHandler composition
type AuthHandler struct {
    *core.BaseHandler
    next *OrderHandler
}

type OrderHandler struct {
    *core.BaseHandler
    service *OrderService
}

// Chain handlers (Premium Pattern provides structure)
func (h *AuthHandler) doHandle(ctx core.FluxorContext, msg core.Message) error {
    // Auth logic
    if !isAuthenticated(msg) {
        return h.Fail(msg, 401, "Unauthorized")
    }
    
    // Pass to next handler
    return h.next.Handle(ctx, msg)
}
```

### Example 3: Premium Component

```go
// Premium Pattern: BaseComponent with lifecycle
type DatabaseComponent struct {
    *core.BaseComponent
    pool *sql.DB
}

func (c *DatabaseComponent) doStart(ctx core.FluxorContext) error {
    // Premium Pattern: Access parent's EventBus for notifications
    eventBus := c.EventBus()
    
    // Initialize connection pool
    db, err := sql.Open("postgres", "...")
    if err != nil {
        return err
    }
    
    c.pool = db
    
    // Notify via EventBus (Premium Pattern integration)
    eventBus.Publish("database.ready", map[string]interface{}{
        "component": c.Name(),
    })
    
    return nil
}

func (c *DatabaseComponent) doStop(ctx core.FluxorContext) error {
    // Premium Pattern: Cleanup with parent access
    if c.pool != nil {
        c.pool.Close()
    }
    return nil
}
```

---

## Premium Pattern Best Practices

### 1. **Lifecycle Management**
```go
// ✅ Good: Use hook methods
func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    // Custom initialization
    return nil
}

// ❌ Bad: Don't override Start directly
func (v *MyVerticle) Start(ctx core.FluxorContext) error {
    // This breaks the template method pattern
    return nil
}
```

### 2. **Component Composition**
```go
// ✅ Good: Set parent and manage lifecycle
func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    v.component.SetParent(v.BaseVerticle)
    return v.component.Start(ctx)
}

// ❌ Bad: Forget to set parent
func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    return v.component.Start(ctx) // component.EventBus() will be nil
}
```

### 3. **Error Handling**
```go
// ✅ Good: Return errors from hook methods
func (s *MyService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    if err := validate(msg); err != nil {
        return s.Fail(msg, 400, err.Error())
    }
    return s.Reply(msg, result)
}

// ❌ Bad: Panic or ignore errors
func (s *MyService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    result := process(msg) // What if process fails?
    return s.Reply(msg, result)
}
```

### 4. **Resource Cleanup**
```go
// ✅ Good: Cleanup in doStop
func (v *MyVerticle) doStop(ctx core.FluxorContext) error {
    if v.connection != nil {
        v.connection.Close()
    }
    return nil
}

// ❌ Bad: Leak resources
func (v *MyVerticle) doStop(ctx core.FluxorContext) error {
    // Forgot to close connection
    return nil
}
```

---

## Summary

Fluxor's Premium Pattern (Base Classes) provide:
- ✅ Java-style abstract base class patterns
- ✅ Template Method pattern implementation
- ✅ Default implementations with hook methods
- ✅ Code reuse and consistency
- ✅ Type-safe composition
- ✅ Familiar patterns for Java developers
- ✅ Enterprise-grade abstractions
- ✅ Production-ready infrastructure
- ✅ Best practices built-in
- ✅ Reduced boilerplate code

**The Premium Pattern is Fluxor's way of bringing enterprise Java patterns to Go, making it easier to build production-ready, maintainable applications.**

