# Store Pattern (Redux/Vuex Style)

The Fluxor State Machine includes a **Store** implementation that provides Redux/Vuex-style state management patterns alongside the event-driven state machine.

## Concepts

### Store
Centralized state container with controlled mutations.

### Mutations (commit)
**Synchronous** state changes. Only mutations can modify state.

### Actions (dispatch)
**Asynchronous** operations that can commit mutations.

### Getters
Computed values derived from state.

### on_error
Error handlers for failed actions.

### Plugins
Middleware that hooks into the store lifecycle.

## Quick Example

```go
// Create store
store := statemachine.NewStoreBuilder().
    WithState(&statemachine.StateContext{
        Data: map[string]interface{}{"count": 0},
    }).
    // Mutation: sync state change
    WithMutation("increment", func(state *statemachine.StateContext, payload interface{}) error {
        count := state.Data["count"].(int)
        state.Data["count"] = count + 1
        return nil
    }).
    // Action: async operation
    WithAction("incrementAsync", func(ctx *statemachine.ActionContext, payload interface{}) error {
        time.Sleep(100 * time.Millisecond) // Simulate async
        return ctx.Commit("increment", nil)
    }).
    // Getter: computed value
    WithGetter("doubleCount", func(state *statemachine.StateContext) interface{} {
        count := state.Data["count"].(int)
        return count * 2
    }).
    // on_error handler
    WithErrorHandler(func(action string, err error, payload interface{}) {
        log.Printf("Action %s failed: %v", action, err)
    }).
    Build()

// Commit mutation (sync)
store.Commit("increment", nil)

// Dispatch action (async)
store.Dispatch(ctx, "incrementAsync", nil)

// Get computed value
doubleCount, _ := store.Get("doubleCount")
```

## API Reference

### Store Creation

```go
// Using builder
store := statemachine.NewStoreBuilder().
    WithState(initialState).
    WithMutation(name, mutationFn).
    WithAction(name, actionFn).
    WithGetter(name, getterFn).
    WithPlugin(pluginFn).
    WithErrorHandler(errorHandler).
    Build()

// Or with options
store := statemachine.NewStore(&statemachine.StoreOptions{
    State:         initialState,
    Mutations:     mutations,
    Actions:       actions,
    Getters:       getters,
    Plugins:       plugins,
    ErrorHandlers: errorHandlers,
})
```

### Mutations

Synchronous state changes:

```go
store.RegisterMutation("setUser", func(state *statemachine.StateContext, payload interface{}) error {
    user := payload.(User)
    state.Data["user"] = user
    state.Data["userId"] = user.ID
    return nil
})

// Commit mutation
store.Commit("setUser", user)
```

### Actions

Asynchronous operations:

```go
store.RegisterAction("login", func(ctx *statemachine.ActionContext, payload interface{}) error {
    credentials := payload.(Credentials)
    
    // Set loading state
    ctx.Commit("setLoading", true)
    
    // Call API
    user, err := authService.Login(credentials)
    if err != nil {
        ctx.Commit("setError", err.Error())
        ctx.Commit("setLoading", false)
        return err
    }
    
    // Update state
    ctx.Commit("setUser", user)
    ctx.Commit("setLoading", false)
    
    // Dispatch another action
    ctx.Dispatch("fetchUserProfile", user.ID)
    
    return nil
})

// Dispatch action
store.Dispatch(ctx, "login", credentials)
```

### Getters

Computed values:

```go
store.RegisterGetter("isAuthenticated", func(state *statemachine.StateContext) interface{} {
    user, ok := state.Data["user"]
    return ok && user != nil
})

store.RegisterGetter("userName", func(state *statemachine.StateContext) interface{} {
    if user, ok := state.Data["user"].(User); ok {
        return user.Name
    }
    return "Guest"
})

// Get computed value
isAuth, _ := store.Get("isAuthenticated")
userName, _ := store.Get("userName")
```

### Subscriptions

React to mutations:

```go
unsubscribe := store.Subscribe(func(mutation string, state *statemachine.StateContext) {
    log.Printf("State changed by %s", mutation)
    
    // Update UI
    // Trigger side effects
    // Persist state
})

// Unsubscribe when done
defer unsubscribe()
```

### Error Handling

```go
store.OnError(func(action string, err error, payload interface{}) {
    log.Printf("Action %s failed: %v", action, err)
    
    // Send to error tracking service
    // Show user notification
    // Trigger recovery action
    store.Dispatch(ctx, "handleError", err)
})
```

## Plugins

Plugins hook into store lifecycle:

### Built-in Plugins

#### Logger Plugin

```go
store := statemachine.NewStoreBuilder().
    WithPlugin(statemachine.LoggerPlugin(logger)).
    Build()
```

#### Persistence Plugin

```go
store := statemachine.NewStoreBuilder().
    WithPlugin(statemachine.PersistencePlugin(func(state *statemachine.StateContext) error {
        return saveToDatabase(state)
    })).
    Build()
```

#### EventBus Plugin

```go
store := statemachine.NewStoreBuilder().
    WithPlugin(statemachine.EventBusPlugin(eventBus, "myapp")).
    Build()

// Publishes: myapp.mutation.{mutationName}
```

#### History Plugin

```go
store := statemachine.NewStoreBuilder().
    WithPlugin(statemachine.HistoryPlugin(100)). // Keep last 100 mutations
    Build()
```

### Custom Plugins

```go
// Time travel plugin
func TimeTravelPlugin() statemachine.Plugin {
    snapshots := make([]*statemachine.StateContext, 0)
    
    return func(store *statemachine.Store) {
        store.Subscribe(func(mutation string, state *statemachine.StateContext) {
            // Save snapshot
            snapshot := *state // Copy
            snapshots = append(snapshots, &snapshot)
        })
        
        // Add custom mutation for time travel
        store.RegisterMutation("timeTravelTo", func(state *statemachine.StateContext, payload interface{}) error {
            index := payload.(int)
            if index < 0 || index >= len(snapshots) {
                return fmt.Errorf("invalid snapshot index")
            }
            *state = *snapshots[index]
            return nil
        })
    }
}
```

## Patterns

### Shopping Cart Example

```go
store := statemachine.NewStoreBuilder().
    WithState(&statemachine.StateContext{
        Data: map[string]interface{}{
            "items": []CartItem{},
            "loading": false,
        },
    }).
    WithMutation("addItem", func(state *statemachine.StateContext, payload interface{}) error {
        item := payload.(CartItem)
        items := state.Data["items"].([]CartItem)
        state.Data["items"] = append(items, item)
        return nil
    }).
    WithMutation("removeItem", func(state *statemachine.StateContext, payload interface{}) error {
        itemId := payload.(string)
        items := state.Data["items"].([]CartItem)
        filtered := make([]CartItem, 0)
        for _, item := range items {
            if item.ID != itemId {
                filtered = append(filtered, item)
            }
        }
        state.Data["items"] = filtered
        return nil
    }).
    WithMutation("setLoading", func(state *statemachine.StateContext, payload interface{}) error {
        state.Data["loading"] = payload
        return nil
    }).
    WithAction("checkout", func(ctx *statemachine.ActionContext, payload interface{}) error {
        ctx.Commit("setLoading", true)
        
        items := ctx.State.Data["items"].([]CartItem)
        order, err := orderService.CreateOrder(items)
        if err != nil {
            ctx.Commit("setLoading", false)
            return err
        }
        
        ctx.Commit("clearCart", nil)
        ctx.Commit("setLoading", false)
        ctx.Dispatch("showOrderConfirmation", order)
        
        return nil
    }).
    WithGetter("itemCount", func(state *statemachine.StateContext) interface{} {
        items := state.Data["items"].([]CartItem)
        return len(items)
    }).
    WithGetter("total", func(state *statemachine.StateContext) interface{} {
        items := state.Data["items"].([]CartItem)
        total := 0.0
        for _, item := range items {
            total += item.Price * float64(item.Quantity)
        }
        return total
    }).
    Build()
```

### Async Data Fetching

```go
store := statemachine.NewStoreBuilder().
    WithState(&statemachine.StateContext{
        Data: map[string]interface{}{
            "data":    nil,
            "loading": false,
            "error":   nil,
        },
    }).
    WithMutation("setLoading", func(state *statemachine.StateContext, payload interface{}) error {
        state.Data["loading"] = payload
        return nil
    }).
    WithMutation("setData", func(state *statemachine.StateContext, payload interface{}) error {
        state.Data["data"] = payload
        state.Data["error"] = nil
        return nil
    }).
    WithMutation("setError", func(state *statemachine.StateContext, payload interface{}) error {
        state.Data["error"] = payload
        state.Data["data"] = nil
        return nil
    }).
    WithAction("fetchData", func(ctx *statemachine.ActionContext, payload interface{}) error {
        ctx.Commit("setLoading", true)
        
        data, err := apiClient.FetchData()
        if err != nil {
            ctx.Commit("setError", err.Error())
            ctx.Commit("setLoading", false)
            return err
        }
        
        ctx.Commit("setData", data)
        ctx.Commit("setLoading", false)
        
        return nil
    }).
    WithGetter("hasData", func(state *statemachine.StateContext) interface{} {
        return state.Data["data"] != nil
    }).
    WithGetter("hasError", func(state *statemachine.StateContext) interface{} {
        return state.Data["error"] != nil
    }).
    Build()
```

## Integrating with State Machine

Use `StateMachineStoreAdapter` to combine state machine with store pattern:

```go
// Create state machine
definition := buildStateMachine()
engine, _ := statemachine.NewEngine(definition, config, eventBus)

// Create store adapter
adapter, _ := statemachine.NewStateMachineStoreAdapter(engine, initialData)

// Access store
store := adapter.Store()

// Subscribe to state changes
store.Subscribe(func(mutation string, state *statemachine.StateContext) {
    log.Printf("State: %s", state.CurrentState)
})

// Dispatch events via store
adapter.Dispatch(ctx, "approve", eventData)

// Get current state via getter
currentState := adapter.GetCurrentState()
```

## Best Practices

### 1. Keep Mutations Pure

Mutations should be **synchronous and side-effect free**:

```go
// ✅ Good
WithMutation("setUser", func(state *statemachine.StateContext, payload interface{}) error {
    state.Data["user"] = payload
    return nil
})

// ❌ Bad - has side effects
WithMutation("setUser", func(state *statemachine.StateContext, payload interface{}) error {
    state.Data["user"] = payload
    sendAnalytics(payload) // Side effect!
    return nil
})
```

### 2. Use Actions for Side Effects

Actions are for **async operations and side effects**:

```go
// ✅ Good
WithAction("saveUser", func(ctx *statemachine.ActionContext, payload interface{}) error {
    user := payload.(User)
    
    // Side effect: API call
    if err := api.SaveUser(user); err != nil {
        return err
    }
    
    // Mutation: update state
    ctx.Commit("setUser", user)
    
    // Side effect: analytics
    analytics.Track("user_saved", user.ID)
    
    return nil
})
```

### 3. Use Getters for Computed Values

Don't compute values in actions/mutations:

```go
// ✅ Good
WithGetter("cartTotal", func(state *statemachine.StateContext) interface{} {
    items := state.Data["items"].([]Item)
    total := 0.0
    for _, item := range items {
        total += item.Price
    }
    return total
})

// ❌ Bad - computing in mutation
WithMutation("addItem", func(state *statemachine.StateContext, payload interface{}) error {
    items := state.Data["items"].([]Item)
    items = append(items, payload.(Item))
    
    // Computing total here
    total := 0.0
    for _, item := range items {
        total += item.Price
    }
    state.Data["total"] = total // Redundant!
    
    return nil
})
```

### 4. Handle Errors Properly

```go
store := statemachine.NewStoreBuilder().
    WithAction("riskyOperation", func(ctx *statemachine.ActionContext, payload interface{}) error {
        // Set loading
        ctx.Commit("setLoading", true)
        
        result, err := doRiskyThing()
        if err != nil {
            // Clear loading
            ctx.Commit("setLoading", false)
            // Set error
            ctx.Commit("setError", err.Error())
            // Propagate error (will trigger on_error)
            return err
        }
        
        // Success
        ctx.Commit("setData", result)
        ctx.Commit("setLoading", false)
        ctx.Commit("setError", nil)
        
        return nil
    }).
    WithErrorHandler(func(action string, err error, payload interface{}) {
        log.Printf("[Store Error] %s: %v", action, err)
        // Could dispatch recovery action
    }).
    Build()
```

### 5. Use Plugins for Cross-Cutting Concerns

```go
store := statemachine.NewStoreBuilder().
    // ... mutations and actions ...
    WithPlugin(statemachine.LoggerPlugin(logger)).
    WithPlugin(statemachine.PersistencePlugin(saveToDB)).
    WithPlugin(statemachine.EventBusPlugin(eventBus, "myapp")).
    WithPlugin(statemachine.HistoryPlugin(50)).
    Build()
```

## Comparison with Redux/Vuex

| Concept | Redux | Vuex | Fluxor Store |
|---------|-------|------|--------------|
| State Container | Store | Store | Store |
| Sync Changes | Reducer | Mutation | Mutation (commit) |
| Async Operations | Action (with middleware) | Action | Action (dispatch) |
| Computed Values | Selectors | Getters | Getters |
| Middleware | Middleware | Plugins | Plugins |
| Error Handling | Middleware | Custom | on_error |
| Event Publishing | - | - | EventBus Plugin |

## Performance

Benchmarks:

```
BenchmarkStore_Commit-8     5000000    300 ns/op    100 B/op    2 allocs/op
BenchmarkStore_Dispatch-8   2000000    800 ns/op    250 B/op    5 allocs/op
```

- Mutations: ~300ns (very fast, synchronous)
- Actions: ~800ns (slightly slower due to indirection)
- Thread-safe with RWMutex
- Minimal allocations

## Running the Example

```bash
go run examples/statemachine/store_pattern.go
```

Example output:

```
--- Basic Store Example ---
[Mutation] increment: 0 → 1
[Action] incrementAsync: starting async operation...
[Mutation] setLoading: true
[Mutation] increment: 1 → 2
[Mutation] setLoading: false
[Action] incrementAsync: completed
✅ Final count: 2

--- Getters Example ---
Subtotal: $75.00
Tax:      $6.00
Total:    $81.00
Items:    6
```

## Summary

The Store pattern provides:

✅ **Predictable state changes** via mutations  
✅ **Async operation handling** via actions  
✅ **Computed values** via getters  
✅ **Error handling** via on_error  
✅ **Extensibility** via plugins  
✅ **Type safety** with Go type system  
✅ **Thread safety** with RWMutex  
✅ **EventBus integration** for distributed systems  

Use the Store pattern when you need Redux/Vuex-style state management combined with Fluxor's event-driven architecture!
