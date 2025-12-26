# Fluxor FSM (Finite State Machine)

A lightweight, reactive, generic Finite State Machine for Go, inspired by [Stateless](https://github.com/dotnet-state-machine/stateless) (.NET) and [Spring State Machine](https://spring.io/projects/spring-statemachine) (Java).

Built on top of the Fluxor runtime using the **Actor Model**, utilizing `FutureT` for asynchronous, non-blocking, and thread-safe state transitions.

## Features

- **Generic Types**: Type-safe states and events `StateMachine[S, E]`.
- **Actor Model**: Events are processed sequentially by a dedicated loop, eliminating race conditions and deadlocks.
- **Fluent API**: Builder pattern for easy configuration.
- **Reactive**: `Fire()` executes asynchronously and returns a `FutureT`.
- **Guards**: Conditional transitions.
- **Actions**: Entry, Exit, and Transition actions.
- **Internal Transitions**: Execute actions without changing state or triggering Entry/Exit.

## Usage

```go
package main

import (
    "context"
    "fmt"
    "github.com/fluxorio/fluxor/pkg/fsm"
)

type OrderState string
type OrderEvent int

const (
    Created OrderState = "Created"
    Paid    OrderState = "Paid"
    Shipped OrderState = "Shipped"
)

const (
    Pay OrderEvent = iota
    Ship
)

func main() {
    // 1. Create FSM
    machine := fsm.New[OrderState, OrderEvent](Created)
    defer machine.Close()

    // 2. Configure States
    machine.Configure(Created).
        Permit(Pay, Paid).
        OnExit(func(ctx context.Context, t fsm.TransitionContext[OrderState, OrderEvent]) error {
            fmt.Println("Processing payment...")
            return nil
        })

    machine.Configure(Paid).
        Permit(Ship, Shipped).
        OnEntry(func(ctx context.Context, t fsm.TransitionContext[OrderState, OrderEvent]) error {
            fmt.Println("Payment received!")
            return nil
        })

    machine.Configure(Shipped).
        Ignore(Pay) // Ignore extra payments

    // 3. Fire Events
    ctx := context.Background()
    
    // Returns a FutureT[OrderState]
    future := machine.Fire(ctx, Pay, nil)
    
    newState, err := future.Await(ctx)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Current state: %s\n", newState)
}
```

## Transition Types

- **External** (`Permit`, `PermitIf`): Transitions from Source to Target. Exits Source, Enters Target.
- **Internal** (`InternalTransition`): Executes action but stays in Source. Does NOT Exit Source or Enter Target.

## Integration with Fluxor

The FSM runs its own goroutine (actor loop) to manage state. It is fully non-blocking and safe to use from multiple goroutines. `Fire` returns a `FutureT` that integrates seamlessly with other Fluxor async patterns.
