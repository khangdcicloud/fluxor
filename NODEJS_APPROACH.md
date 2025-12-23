# Node.js Developer's Guide to Fluxor

If you're a Node.js developer, this guide shows you how to approach Fluxor using familiar patterns and mental models.

## Mental Model: What is Fluxor?

**Think of Fluxor as:**
- **Express.js** + **EventEmitter** + **Worker Threads** = Fluxor
- A reactive framework that gives you Express-like routing with event-driven architecture
- Built for high performance (100k+ RPS) with built-in backpressure handling

## Quick Start: Your First Fluxor App

### Node.js Approach
```javascript
// app.js
const express = require('express');
const app = express();

app.use(express.json());

app.get('/', (req, res) => {
    res.json({ message: 'Hello from Node.js!' });
});

app.listen(8080, () => {
    console.log('Server running on port 8080');
});
```

### Fluxor Approach
```go
// main.go
package main

import (
    "context"
    "log"
    "reflect"
    
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/fx"
    "github.com/fluxorio/fluxor/pkg/web"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Create application (like Express app)
    app, err := fx.New(ctx,
        fx.Invoke(fx.NewInvoker(setupApplication)),
    )
    if err != nil {
        log.Fatalf("Failed to create app: %v", err)
    }
    
    if err := app.Start(); err != nil {
        log.Fatalf("Failed to start app: %v", err)
    }
    
    app.Wait() // Keep server running
}

func setupApplication(deps map[reflect.Type]interface{}) error {
    vertx := deps[reflect.TypeOf((*core.Vertx)(nil)).Elem()].(core.Vertx)
    
    // Create HTTP server (like Express)
    config := web.CCUBasedConfigWithUtilization(":8080", 5000, 67)
    server := web.NewFastHTTPServer(vertx, config)
    router := server.FastRouter()
    
    // Routes (like Express routes)
    router.GETFast("/", func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, map[string]interface{}{
            "message": "Hello from Fluxor!",
        })
    })
    
    // Start server (like app.listen())
    go server.Start()
    
    return nil
}
```

## Pattern Mapping: Node.js → Fluxor

### 1. Express Routes → Fluxor Routes

**Node.js:**
```javascript
app.get('/api/users/:id', (req, res) => {
    const id = req.params.id;
    const user = getUser(id);
    res.json(user);
});

app.post('/api/users', async (req, res) => {
    const user = await createUser(req.body);
    res.status(201).json(user);
});
```

**Fluxor:**
```go
router.GETFast("/api/users/:id", func(ctx *web.FastRequestContext) error {
    id := ctx.Param("id")  // Like req.params.id
    user := getUser(id)
    return ctx.JSON(200, user)  // Like res.json()
})

router.POSTFast("/api/users", func(ctx *web.FastRequestContext) error {
    var user map[string]interface{}
    if err := ctx.BindJSON(&user); err != nil {  // Like req.body
        return ctx.JSON(400, map[string]interface{}{
            "error": "invalid json",
        })
    }
    
    created := createUser(user)
    return ctx.JSON(201, created)  // Like res.status(201).json()
})
```

### 2. EventEmitter → EventBus

**Node.js:**
```javascript
const EventEmitter = require('events');
const emitter = new EventEmitter();

// Emit event
emitter.emit('user.created', { userId: 123, name: 'John' });

// Listen to event
emitter.on('user.created', (data) => {
    console.log('User created:', data);
});
```

**Fluxor:**
```go
eventBus := vertx.EventBus()

// Publish event (like emit)
eventBus.Publish("user.created", map[string]interface{}{
    "userId": 123,
    "name":   "John",
})

// Consume event (like on)
consumer := eventBus.Consumer("user.created")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var data map[string]interface{}
    msg.DecodeBody(&data)  // Automatically JSON-decoded
    log.Printf("User created: %v", data)
    return nil
})
```

### 3. Promises/Async-Await → Futures

**Node.js:**
```javascript
async function fetchUser(id) {
    const user = await db.getUser(id);
    return user;
}

// Usage
const user = await fetchUser(123);
console.log(user);
```

**Fluxor:**
```go
// Create a future (like Promise)
future := fluxor.NewFuture()

// Complete it asynchronously (like resolve)
go func() {
    user := db.GetUser(id)
    future.Complete(user)
}()

// Await result (like await)
result, err := future.Await(ctx)
if err != nil {
    return err
}
log.Printf("User: %v", result)
```

### 4. Middleware → Handlers

**Node.js:**
```javascript
app.use((req, res, next) => {
    req.requestId = generateId();
    next();
});

app.use(authMiddleware);
app.use(cors());
```

**Fluxor:**
```go
// Request ID is automatic! No middleware needed.
// Access via ctx.RequestID()

// For custom middleware, wrap handlers:
func withAuth(handler func(*web.FastRequestContext) error) func(*web.FastRequestContext) error {
    return func(ctx *web.FastRequestContext) error {
        // Auth logic here
        if !isAuthenticated(ctx) {
            return ctx.JSON(401, map[string]string{"error": "unauthorized"})
        }
        return handler(ctx)
    }
}

router.GETFast("/api/protected", withAuth(func(ctx *web.FastRequestContext) error {
    return ctx.JSON(200, map[string]string{"data": "protected"})
}))
```

### 5. Error Handling

**Node.js:**
```javascript
app.get('/api/data', async (req, res, next) => {
    try {
        const data = await fetchData();
        res.json(data);
    } catch (err) {
        next(err);  // Pass to error handler
    }
});

app.use((err, req, res, next) => {
    res.status(500).json({ error: err.message });
});
```

**Fluxor:**
```go
router.GETFast("/api/data", func(ctx *web.FastRequestContext) error {
    data, err := fetchData()
    if err != nil {
        // Return error directly (automatically becomes 500)
        return ctx.JSON(500, map[string]interface{}{
            "error": err.Error(),
        })
    }
    return ctx.JSON(200, data)
})

// Panics are automatically caught and return 500
// No need for try-catch - Go's error handling is explicit
```

### 6. JSON Request/Response

**Node.js:**
```javascript
app.post('/api/echo', express.json(), (req, res) => {
    res.json({
        echo: req.body,
        message: "Echo successful"
    });
});
```

**Fluxor:**
```go
router.POSTFast("/api/echo", func(ctx *web.FastRequestContext) error {
    var data map[string]interface{}
    if err := ctx.BindJSON(&data); err != nil {  // Like express.json()
        return ctx.JSON(400, map[string]interface{}{
            "error": "invalid json",
        })
    }
    
    return ctx.JSON(200, map[string]interface{}{
        "echo":    data,  // Like req.body
        "message": "Echo successful",
    })
})
```

## Key Differences to Remember

### 1. Error Handling
- **Node.js**: Try-catch or callback errors
- **Fluxor**: Explicit error returns `(result, error)`
- **Always check**: `if err != nil { return err }`

### 2. Async Operations
- **Node.js**: `async/await` or `.then()`
- **Fluxor**: `future.Await(ctx)` or callbacks with `OnSuccess/OnFailure`
- **Context required**: Always pass `ctx` for cancellation

### 3. JSON Handling
- **Node.js**: `express.json()` middleware
- **Fluxor**: `ctx.BindJSON(&data)` - explicit binding
- **Automatic**: EventBus messages are automatically JSON-encoded/decoded

### 4. Request Context
- **Node.js**: `req` and `res` objects
- **Fluxor**: `*web.FastRequestContext` (single object)
- **Methods**: `ctx.JSON()`, `ctx.Param()`, `ctx.Query()`, `ctx.BindJSON()`

### 5. Module System
- **Node.js**: `require()` and `module.exports`
- **Fluxor**: Packages (directories) with uppercase = public
- **Import**: `import "github.com/yourproject/user"`

## Common Patterns

### Pattern 1: REST API with EventBus

**Node.js:**
```javascript
app.post('/api/users', async (req, res) => {
    const user = await createUser(req.body);
    emitter.emit('user.created', user);
    res.json(user);
});
```

**Fluxor:**
```go
router.POSTFast("/api/users", func(ctx *web.FastRequestContext) error {
    var user map[string]interface{}
    if err := ctx.BindJSON(&user); err != nil {
        return ctx.JSON(400, map[string]interface{}{"error": "invalid json"})
    }
    
    created := createUser(user)
    
    // Publish event
    ctx.EventBus().Publish("user.created", created)
    
    return ctx.JSON(201, created)
})
```

### Pattern 2: Request-Reply Pattern

**Node.js:**
```javascript
// Service A
app.get('/api/data', async (req, res) => {
    const result = await requestService('data.get', { id: req.query.id });
    res.json(result);
});

// Service B
emitter.on('data.get', async (data) => {
    const result = await fetchData(data.id);
    emitter.emit('data.get.reply', result);
});
```

**Fluxor:**
```go
// Service A
router.GETFast("/api/data", func(ctx *web.FastRequestContext) error {
    id := ctx.Query("id")
    
    // Request-reply pattern
    reply, err := ctx.EventBus().Request("data.get", map[string]interface{}{
        "id": id,
    }, 5*time.Second)
    
    if err != nil {
        return ctx.JSON(500, map[string]interface{}{"error": err.Error()})
    }
    
    var result map[string]interface{}
    reply.DecodeBody(&result)
    return ctx.JSON(200, result)
})

// Service B
consumer := eventBus.Consumer("data.get")
consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
    var req map[string]interface{}
    msg.DecodeBody(&req)
    
    result := fetchData(req["id"].(string))
    return msg.Reply(result)  // Automatic reply
})
```

### Pattern 3: Background Jobs

**Node.js:**
```javascript
app.post('/api/process', (req, res) => {
    // Queue job
    jobQueue.add('process', req.body);
    res.json({ status: 'processing' });
});

// Worker
jobQueue.process('process', async (job) => {
    await processData(job.data);
});
```

**Fluxor:**
```go
router.POSTFast("/api/process", func(ctx *web.FastRequestContext) error {
    var data map[string]interface{}
    ctx.BindJSON(&data)
    
    // Send to worker via EventBus
    ctx.EventBus().Send("process.job", data)
    
    return ctx.JSON(200, map[string]interface{}{
        "status": "processing",
    })
})

// Worker verticle
type ProcessWorker struct {
    eventBus core.EventBus
}

func (w *ProcessWorker) Start(ctx core.FluxorContext) error {
    consumer := ctx.EventBus().Consumer("process.job")
    consumer.Handler(func(ctx core.FluxorContext, msg core.Message) error {
        var data map[string]interface{}
        msg.DecodeBody(&data)
        processData(data)  // Background processing
        return nil
    })
    return nil
}
```

## Development Workflow

### Node.js Workflow
```bash
npm install
npm run dev  # Hot reload
npm test
npm start
```

### Fluxor Workflow
```bash
go mod download
go run cmd/main.go  # Run directly
go test ./...       # Run tests
go build -o app cmd/main.go  # Build binary
```

## Performance Comparison

| Metric | Node.js (Express) | Fluxor |
|--------|------------------|--------|
| RPS | ~10k-20k | 100k+ |
| Memory | Higher | Lower |
| Startup | Fast | Very Fast |
| Concurrency | Event Loop | Goroutines + Event Bus |

## Next Steps

1. **Read the examples**: Check out [`cmd/main.go`](cmd/main.go) for a complete working example
2. **Learn Go basics**: Understand structs, interfaces, and error handling
3. **Try the patterns**: Start with Express-like routes, then add EventBus
4. **Read documentation**: See [DOCUMENTATION.md](DOCUMENTATION.md) for detailed API reference

## Questions?

- **"How do I do X in Fluxor?"** → Check [DOCUMENTATION.md](DOCUMENTATION.md)
- **"What's the architecture?"** → Read [ARCHITECTURE.md](ARCHITECTURE.md)
- **"How do I migrate my Node.js app?"** → See [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)

## Summary

Fluxor gives you:
- ✅ Express-like routing (`router.GETFast()`)
- ✅ Event-driven architecture (EventBus like EventEmitter)
- ✅ High performance (100k+ RPS)
- ✅ Built-in backpressure handling
- ✅ Type safety (Go's static typing)
- ✅ JSON-first (automatic encoding/decoding)

**Think of it as**: "Express.js but in Go with better performance and event-driven architecture built-in."

