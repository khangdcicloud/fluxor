# State Machine for Fluxor

A powerful, event-driven state machine implementation for the Fluxor framework, fully integrated with EventBus for distributed state management.

## Features

- **Event-Driven**: Fully integrated with Fluxor's EventBus for distributed state transitions
- **Type-Safe**: Strongly typed states, events, and transitions
- **Guard Conditions**: Conditional transitions with guard functions
- **Actions**: Execute side effects on state entry, exit, and transitions
- **History Tracking**: Automatic tracking of state transitions
- **Persistence**: Pluggable persistence layer for state machine instances
- **Priority Transitions**: Control transition evaluation order
- **Final States**: Support for terminal states
- **Fluent Builder API**: Intuitive API for defining state machines
- **Testing Support**: Comprehensive test coverage and benchmarks

## Table of Contents

1. [Quick Start](#quick-start)
2. [Core Concepts](#core-concepts)
3. [Building State Machines](#building-state-machines)
4. [Deploying State Machines](#deploying-state-machines)
5. [Working with Events](#working-with-events)
6. [Store Pattern (Redux/Vuex Style)](#store-pattern-reduxvuex-style)
7. [Advanced Features](#advanced-features)
8. [Examples](#examples)
9. [API Reference](#api-reference)

## Quick Start

### 1. Define Your State Machine

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/statemachine"
)

func createLightSwitchSM() *statemachine.StateMachineDefinition {
    builder := statemachine.NewBuilder("light-switch", "Light Switch")
    builder.WithInitialState("off")

    // Define states
    builder.AddStates(
        statemachine.NewState("off", "Light Off").
            OnEnter(func(ctx *statemachine.StateContext) error {
                log.Println("Light is now OFF")
                return nil
            }).
            Build(),
            
        statemachine.NewState("on", "Light On").
            OnEnter(func(ctx *statemachine.StateContext) error {
                log.Println("Light is now ON")
                return nil
            }).
            Build(),
    )

    // Define transitions
    builder.AddTransitions(
        statemachine.SimpleTransition("turn-on", "off", "on", "flip"),
        statemachine.SimpleTransition("turn-off", "on", "off", "flip"),
    )

    definition, _ := builder.Build()
    return definition
}
```

### 2. Deploy and Use

```go
// Create Fluxor app
app, _ := fluxor.NewMainVerticle("config.json")

// Create and deploy state machine verticle
smVerticle := statemachine.NewStateMachineVerticle()
app.Vertx().DeployVerticle(smVerticle)

// Register your state machine
definition := createLightSwitchSM()
smVerticle.RegisterStateMachine(definition, nil)

// Create a state machine instance
client := statemachine.NewStateMachineClient(app.Vertx().EventBus())
instanceID, _ := client.CreateInstance(ctx, "light-switch", nil)

// Send events to trigger transitions
client.SendEvent(ctx, "light-switch", instanceID, "flip", nil)
```

## Core Concepts

### State

A **State** represents a specific condition or phase in your state machine. States have:

- **ID**: Unique identifier (StateType)
- **Name**: Human-readable name
- **OnEnter**: Action executed when entering the state
- **OnExit**: Action executed when leaving the state
- **IsFinal**: Marks terminal states

```go
state := statemachine.NewState("processing", "Processing Order").
    WithDescription("Order is being processed").
    OnEnter(func(ctx *statemachine.StateContext) error {
        // Execute when entering this state
        log.Printf("Processing order %s", ctx.MachineID)
        return nil
    }).
    OnExit(func(ctx *statemachine.StateContext) error {
        // Execute when leaving this state
        log.Printf("Finished processing order %s", ctx.MachineID)
        return nil
    }).
    Build()
```

### Transition

A **Transition** defines how the state machine moves from one state to another in response to an event.

```go
transition := statemachine.NewTransition("approve", "pending", "approved", "approve").
    WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
        // Conditional transition
        amount, _ := ctx.Data["amount"].(float64)
        return amount < 10000, nil  // Only approve if under $10k
    }).
    WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
        // Execute during transition
        ctx.Data["approvedBy"] = event.Data["approver"]
        return nil
    }).
    WithPriority(10).  // Higher priority = evaluated first
    Build()
```

### Event

An **Event** triggers state transitions. Events carry data and metadata.

```go
event := statemachine.NewEvent("approve").
    WithData("approver", "john@example.com").
    WithData("comments", "Looks good!").
    WithSource("approval-service").
    Build()
```

### State Context

The **StateContext** provides runtime information and data storage for state machine instances.

```go
type StateContext struct {
    MachineID          string                  // Unique instance ID
    CurrentState       StateType               // Current state
    PreviousState      *StateType             // Previous state
    Data               map[string]interface{} // Instance data
    Context            context.Context         // Go context
    FluxorContext      core.FluxorContext     // Fluxor context
    History            []*HistoryEntry        // Transition history
    StartTime          time.Time              // Instance creation time
    LastTransitionTime time.Time              // Last transition time
}
```

## Building State Machines

### Basic State Machine

```go
builder := statemachine.NewBuilder("order-fsm", "Order State Machine")
builder.WithDescription("Manages order lifecycle").
    WithVersion("1.0").
    WithInitialState("pending")

// Add states
builder.AddStates(
    statemachine.SimpleState("pending", "Pending"),
    statemachine.SimpleState("confirmed", "Confirmed"),
    statemachine.FinalState("completed", "Completed"),
    statemachine.FinalState("cancelled", "Cancelled"),
)

// Add transitions
builder.AddTransitions(
    statemachine.SimpleTransition("confirm", "pending", "confirmed", "confirm"),
    statemachine.SimpleTransition("complete", "confirmed", "completed", "complete"),
    statemachine.SimpleTransition("cancel", "pending", "cancelled", "cancel"),
)

definition, err := builder.Build()
```

### With Guards and Actions

```go
// Guard: Conditional transition
approveTransition := statemachine.NewTransition(
    "approve-low-value", 
    "pending", 
    "approved", 
    "approve",
).WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
    amount, ok := ctx.Data["amount"].(float64)
    if !ok {
        return false, fmt.Errorf("amount not found")
    }
    return amount < 1000, nil  // Only approve amounts under $1000
}).WithAction(func(ctx *statemachine.StateContext, event *statemachine.Event) error {
    // Record approval
    ctx.Data["approvedAt"] = time.Now()
    ctx.Data["approvedBy"] = event.Data["approver"]
    return nil
}).Build()

builder.AddTransition(approveTransition)
```

### Multiple Transitions for Same Event

When multiple transitions match the same event, they're evaluated by priority (highest first):

```go
// High value - requires extra approval
builder.AddTransition(
    statemachine.NewTransition("approve-high", "pending", "needs_review", "approve").
        WithPriority(10).  // Evaluated first
        WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
            amount := ctx.Data["amount"].(float64)
            return amount >= 1000, nil
        }).
        Build(),
)

// Low value - direct approval
builder.AddTransition(
    statemachine.NewTransition("approve-low", "pending", "approved", "approve").
        WithPriority(5).  // Evaluated second
        WithGuard(func(ctx *statemachine.StateContext, event *statemachine.Event) (bool, error) {
            amount := ctx.Data["amount"].(float64)
            return amount < 1000, nil
        }).
        Build(),
)
```

## Deploying State Machines

### Using StateMachineVerticle

```go
// Create Fluxor app
app, err := fluxor.NewMainVerticle("config.json")
if err != nil {
    log.Fatal(err)
}

// Create state machine verticle
smVerticle := statemachine.NewStateMachineVerticle()

// Deploy verticle
_, err = app.Vertx().DeployVerticle(smVerticle)
if err != nil {
    log.Fatal(err)
}

// Register state machine definitions
definition := createOrderStateMachine()
config := statemachine.DefaultConfig()
config.EnableHistory = true
config.EnableEventBus = true

err = smVerticle.RegisterStateMachine(definition, config)
if err != nil {
    log.Fatal(err)
}
```

### Configuration Options

```go
config := &statemachine.StateMachineConfig{
    EnableHistory:      true,                // Track state transitions
    MaxHistorySize:     100,                 // Limit history entries
    EnablePersistence:  false,               // Enable state persistence
    PersistenceStore:   nil,                 // Persistence backend
    EnableEventBus:     true,                // EventBus integration
    EventBusPrefix:     "statemachine",      // EventBus address prefix
    DefaultTimeout:     30 * time.Second,    // Action timeout
}
```

## Working with Events

### Via Client API

```go
client := statemachine.NewStateMachineClient(eventBus)

// Create instance
instanceID, err := client.CreateInstance(ctx, "order-fsm", map[string]interface{}{
    "orderId":  "ORD-123",
    "amount":   99.99,
    "customer": "john@example.com",
})

// Send event
success, err := client.SendEvent(ctx, "order-fsm", instanceID, "confirm", map[string]interface{}{
    "confirmedBy": "system",
    "timestamp":   time.Now(),
})

// Query state
state, err := client.QueryInstance(ctx, "order-fsm", instanceID)
fmt.Printf("Current state: %v\n", state["currentState"])
```

### Via EventBus

```go
// Send event via EventBus
eventBus.Send("statemachine.order-fsm.event", map[string]interface{}{
    "instanceId": instanceID,
    "event":      "confirm",
    "data": map[string]interface{}{
        "confirmedBy": "user@example.com",
    },
})
```

### Subscribing to State Machine Events

```go
// Subscribe to instance creation events
eventBus.Consumer("statemachine.order-fsm.instance.created").
    Handler(func(ctx core.FluxorContext, msg core.Message) error {
        var event map[string]interface{}
        core.JSONDecode(msg.Body(), &event)
        log.Printf("New instance created: %v", event["machineId"])
        return nil
    })

// Subscribe to transition events
eventBus.Consumer("statemachine.order-fsm.transition.completed").
    Handler(func(ctx core.FluxorContext, msg core.Message) error {
        var event map[string]interface{}
        core.JSONDecode(msg.Body(), &event)
        log.Printf("Transition: %v → %v", event["fromState"], event["toState"])
        return nil
    })
```

## Store Pattern (Redux/Vuex Style)

In addition to the event-driven state machine, Fluxor provides a **Store** implementation with Redux/Vuex-style patterns:

### Quick Example

```go
// Create store with mutations, actions, and getters
store := statemachine.NewStoreBuilder().
    WithState(&statemachine.StateContext{
        Data: map[string]interface{}{"count": 0},
    }).
    // Mutation: synchronous state change (commit)
    WithMutation("increment", func(state *statemachine.StateContext, payload interface{}) error {
        count := state.Data["count"].(int)
        state.Data["count"] = count + 1
        return nil
    }).
    // Action: asynchronous operation (dispatch)
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

### Store Concepts

| Concept | Description | Method |
|---------|-------------|--------|
| **Store** | Centralized state container | `NewStore()` |
| **Mutation** | Synchronous state change | `Commit(name, payload)` |
| **Action** | Asynchronous operation | `Dispatch(ctx, name, payload)` |
| **Getter** | Computed value | `Get(name)` |
| **Plugin** | Lifecycle hook/middleware | `WithPlugin()` |
| **on_error** | Error handler | `WithErrorHandler()` |

### Built-in Plugins

- **LoggerPlugin**: Logs all mutations
- **PersistencePlugin**: Persists state after mutations
- **EventBusPlugin**: Publishes mutations to EventBus
- **HistoryPlugin**: Tracks mutation history

### Integrating with State Machine

```go
// Create state machine
definition := buildStateMachine()
engine, _ := statemachine.NewEngine(definition, config, eventBus)

// Create store adapter
adapter, _ := statemachine.NewStateMachineStoreAdapter(engine, initialData)

// Dispatch events via store pattern
adapter.Dispatch(ctx, "approve", eventData)

// Get current state
currentState := adapter.GetCurrentState()
```

For complete documentation, see [`STORE_PATTERN.md`](STORE_PATTERN.md).

## Advanced Features

### History Tracking

```go
// Enable history in config
config := statemachine.DefaultConfig()
config.EnableHistory = true
config.MaxHistorySize = 100

// Access history
instance, _ := engine.GetInstance(ctx, instanceID)
for _, entry := range instance.Context.History {
    fmt.Printf("%s → %s (event: %s, time: %v)\n",
        entry.FromState,
        entry.ToState,
        entry.Event,
        entry.Timestamp,
    )
}
```

### Persistence

Implement the `PersistenceStore` interface:

```go
type MyPersistenceStore struct {
    db *sql.DB
}

func (s *MyPersistenceStore) Save(ctx context.Context, instance *StateMachineInstance) error {
    // Save instance to database
    return nil
}

func (s *MyPersistenceStore) Load(ctx context.Context, instanceID string) (*StateMachineInstance, error) {
    // Load instance from database
    return nil, nil
}

func (s *MyPersistenceStore) Delete(ctx context.Context, instanceID string) error {
    // Delete instance
    return nil
}

func (s *MyPersistenceStore) List(ctx context.Context, definitionID string) ([]*StateMachineInstance, error) {
    // List all instances for a definition
    return nil, nil
}

// Use custom persistence
config := statemachine.DefaultConfig()
config.EnablePersistence = true
config.PersistenceStore = &MyPersistenceStore{db: db}
```

### State Data Management

```go
// Accessing and modifying state data
statemachine.NewState("processing", "Processing").
    OnEnter(func(ctx *statemachine.StateContext) error {
        // Read data
        orderID := ctx.Data["orderId"].(string)
        
        // Write data
        ctx.Data["processedAt"] = time.Now()
        ctx.Data["status"] = "in_progress"
        
        // Access FluxorContext
        ctx.FluxorContext.EventBus().Publish("order.processing", map[string]interface{}{
            "orderId": orderID,
        })
        
        return nil
    }).
    Build()
```

### Checking Transition Availability

```go
// Check if a transition is possible
event := statemachine.NewEvent("approve").Build()
canTransition, err := engine.CanTransition(ctx, instanceID, event)
if canTransition {
    // Transition is possible
    engine.SendEvent(ctx, instanceID, event)
}
```

## Examples

### 1. Order Processing

See [`examples/statemachine/order_processing.go`](../../examples/statemachine/order_processing.go)

States: `pending → payment_processing → confirmed → shipped → delivered`

Features demonstrated:
- Multi-state workflow
- Payment processing simulation
- Shipping and delivery tracking
- Cancellation at various stages

### 2. Approval Workflow

See [`examples/statemachine/approval_workflow.go`](../../examples/statemachine/approval_workflow.go)

States: `draft → pending_l1 → pending_l2 → pending_l3 → approved`

Features demonstrated:
- Multi-level approval process
- Conditional transitions based on amount
- Rejection and revision request handling
- Priority-based transition selection

## API Reference

### Builder API

#### `NewBuilder(id, name string) *Builder`
Creates a new state machine builder.

#### `WithInitialState(state StateType) *Builder`
Sets the initial state.

#### `AddState(state *State) *Builder`
Adds a state to the state machine.

#### `AddTransition(transition *Transition) *Builder`
Adds a transition to the state machine.

#### `Build() (*StateMachineDefinition, error)`
Builds and validates the state machine definition.

### State API

#### `NewState(id StateType, name string) *StateBuilder`
Creates a new state builder.

#### `OnEnter(action StateAction) *StateBuilder`
Sets the on-enter action.

#### `OnExit(action StateAction) *StateBuilder`
Sets the on-exit action.

#### `AsFinal() *StateBuilder`
Marks the state as final.

### Transition API

#### `NewTransition(id string, from StateType, to StateType, event TransitionEvent) *TransitionBuilder`
Creates a new transition builder.

#### `WithGuard(guard TransitionGuard) *TransitionBuilder`
Sets a guard condition.

#### `WithAction(action TransitionAction) *TransitionBuilder`
Sets a transition action.

#### `WithPriority(priority int) *TransitionBuilder`
Sets the transition priority.

### Engine API

#### `NewEngine(definition *StateMachineDefinition, config *StateMachineConfig, eventBus core.EventBus) (*Engine, error)`
Creates a new state machine engine.

#### `CreateInstance(ctx context.Context, initialData map[string]interface{}) (*StateMachineInstance, error)`
Creates a new state machine instance.

#### `SendEvent(ctx context.Context, instanceID string, event *Event) (*TransitionResult, error)`
Sends an event to trigger a transition.

#### `GetInstance(ctx context.Context, instanceID string) (*StateMachineInstance, error)`
Retrieves a state machine instance.

#### `GetCurrentState(ctx context.Context, instanceID string) (StateType, error)`
Gets the current state of an instance.

#### `CanTransition(ctx context.Context, instanceID string, event *Event) (bool, error)`
Checks if a transition is possible.

## Testing

Run tests:

```bash
cd pkg/statemachine
go test -v
```

Run benchmarks:

```bash
go test -bench=. -benchmem
```

## Best Practices

1. **Keep States Focused**: Each state should represent a clear, distinct phase
2. **Use Guards Wisely**: Guards should be side-effect free and fast
3. **Handle Errors**: Always handle errors in actions and guards
4. **Document Transitions**: Use metadata to document business rules
5. **Test State Machines**: Write tests for all states and transitions
6. **Use EventBus**: Leverage EventBus for distributed coordination
7. **Monitor Instances**: Track state machine execution via events
8. **Version Definitions**: Use version field for schema evolution

## Performance

Benchmarks on a MacBook Pro (M1):

```
BenchmarkEngine_CreateInstance-8    100000    12000 ns/op    5000 B/op    45 allocs/op
BenchmarkEngine_SendEvent-8         200000     8000 ns/op    3500 B/op    30 allocs/op
```

- Instance creation: ~12µs per instance
- Event processing: ~8µs per transition
- Memory efficient with bounded allocations

## License

MIT License - See Fluxor main LICENSE file.
