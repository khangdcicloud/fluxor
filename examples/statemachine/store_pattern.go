package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/statemachine"
)

// This example demonstrates Redux/Vuex-style patterns:
// - Store: centralized state management
// - Actions: async operations
// - Mutations: synchronous state changes (commit)
// - Getters: computed values
// - Plugins: middleware
// - on_error: error handling

func main() {
	log.Println("=== Store Pattern Example ===\n")

	// Example 1: Basic Store with Mutations and Actions
	basicStoreExample()

	// Example 2: Store with Getters
	gettersExample()

	// Example 3: Store with Plugins
	pluginsExample()

	// Example 4: Store with Error Handling
	errorHandlingExample()

	// Example 5: State Machine + Store Pattern
	stateMachineStoreExample()
}

func basicStoreExample() {
	log.Println("--- Basic Store Example ---")

	// Create store
	store := statemachine.NewStoreBuilder().
		WithState(&statemachine.StateContext{
			Data: map[string]interface{}{
				"count":   0,
				"loading": false,
			},
		}).
		// Mutations: synchronous state changes (commit)
		WithMutation("increment", func(state *statemachine.StateContext, payload interface{}) error {
			count := state.Data["count"].(int)
			state.Data["count"] = count + 1
			log.Printf("  [Mutation] increment: %d → %d", count, count+1)
			return nil
		}).
		WithMutation("setLoading", func(state *statemachine.StateContext, payload interface{}) error {
			loading := payload.(bool)
			state.Data["loading"] = loading
			log.Printf("  [Mutation] setLoading: %v", loading)
			return nil
		}).
		// Actions: async operations
		WithAction("incrementAsync", func(ctx *statemachine.ActionContext, payload interface{}) error {
			log.Println("  [Action] incrementAsync: starting async operation...")
			
			// Set loading
			ctx.Commit("setLoading", true)
			
			// Simulate async operation
			time.Sleep(100 * time.Millisecond)
			
			// Commit mutation
			ctx.Commit("increment", nil)
			ctx.Commit("setLoading", false)
			
			log.Println("  [Action] incrementAsync: completed")
			return nil
		}).
		Build()

	// Commit synchronous mutation
	log.Println("\n1. Commit synchronous mutation:")
	store.Commit("increment", nil)

	// Dispatch async action
	log.Println("\n2. Dispatch async action:")
	store.Dispatch(context.Background(), "incrementAsync", nil)

	// Check final state
	state := store.GetState()
	log.Printf("\n✅ Final count: %v\n", state.Data["count"])
	log.Println()
}

func gettersExample() {
	log.Println("--- Getters Example ---")

	// Shopping cart store
	store := statemachine.NewStoreBuilder().
		WithState(&statemachine.StateContext{
			Data: map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": 1, "name": "Product A", "price": 10.0, "quantity": 2},
					{"id": 2, "name": "Product B", "price": 20.0, "quantity": 1},
					{"id": 3, "name": "Product C", "price": 15.0, "quantity": 3},
				},
				"taxRate": 0.08,
			},
		}).
		// Getters: computed values
		WithGetter("subtotal", func(state *statemachine.StateContext) interface{} {
			items := state.Data["items"].([]map[string]interface{})
			subtotal := 0.0
			for _, item := range items {
				price := item["price"].(float64)
				quantity := item["quantity"].(int)
				subtotal += price * float64(quantity)
			}
			return subtotal
		}).
		WithGetter("tax", func(state *statemachine.StateContext) interface{} {
			subtotal := state.Data["items"].([]map[string]interface{})
			taxRate := state.Data["taxRate"].(float64)
			total := 0.0
			for _, item := range subtotal {
				price := item["price"].(float64)
				quantity := item["quantity"].(int)
				total += price * float64(quantity)
			}
			return total * taxRate
		}).
		WithGetter("total", func(state *statemachine.StateContext) interface{} {
			items := state.Data["items"].([]map[string]interface{})
			taxRate := state.Data["taxRate"].(float64)
			subtotal := 0.0
			for _, item := range items {
				price := item["price"].(float64)
				quantity := item["quantity"].(int)
				subtotal += price * float64(quantity)
			}
			tax := subtotal * taxRate
			return subtotal + tax
		}).
		WithGetter("itemCount", func(state *statemachine.StateContext) interface{} {
			items := state.Data["items"].([]map[string]interface{})
			count := 0
			for _, item := range items {
				count += item["quantity"].(int)
			}
			return count
		}).
		Build()

	// Get computed values
	subtotal, _ := store.Get("subtotal")
	tax, _ := store.Get("tax")
	total, _ := store.Get("total")
	itemCount, _ := store.Get("itemCount")

	log.Printf("  Subtotal: $%.2f", subtotal)
	log.Printf("  Tax:      $%.2f", tax)
	log.Printf("  Total:    $%.2f", total)
	log.Printf("  Items:    %d", itemCount)
	log.Println()
}

func pluginsExample() {
	log.Println("--- Plugins Example ---")

	// Create Vertx for EventBus plugin
	vertx := core.NewVertx(context.Background())
	defer vertx.Close()

	// Subscribe to mutations
	vertx.EventBus().Consumer("cart.mutation.*").Handler(
		func(ctx core.FluxorContext, msg core.Message) error {
			log.Printf("  [EventBus] Received mutation event")
			return nil
		})

	// Store with plugins
	store := statemachine.NewStoreBuilder().
		WithState(&statemachine.StateContext{
			Data: map[string]interface{}{
				"value": 0,
			},
		}).
		WithMutation("setValue", func(state *statemachine.StateContext, payload interface{}) error {
			state.Data["value"] = payload
			return nil
		}).
		// Plugin 1: Logger
		WithPlugin(statemachine.LoggerPlugin(core.NewDefaultLogger())).
		// Plugin 2: EventBus integration
		WithPlugin(statemachine.EventBusPlugin(vertx.EventBus(), "cart")).
		// Plugin 3: History tracking
		WithPlugin(statemachine.HistoryPlugin(10)).
		// Plugin 4: Persistence (simulated)
		WithPlugin(statemachine.PersistencePlugin(func(state *statemachine.StateContext) error {
			log.Printf("  [Persistence] Saving state: %v", state.Data)
			return nil
		})).
		Build()

	// Commit mutation (triggers all plugins)
	log.Println("\nCommitting mutation (all plugins will be triggered):")
	store.Commit("setValue", 42)

	time.Sleep(100 * time.Millisecond) // Wait for EventBus
	log.Println()
}

func errorHandlingExample() {
	log.Println("--- Error Handling Example ---")

	store := statemachine.NewStoreBuilder().
		WithState(&statemachine.StateContext{
			Data: map[string]interface{}{
				"retryCount": 0,
			},
		}).
		WithMutation("incrementRetry", func(state *statemachine.StateContext, payload interface{}) error {
			count := state.Data["retryCount"].(int)
			state.Data["retryCount"] = count + 1
			return nil
		}).
		WithAction("fetchData", func(ctx *statemachine.ActionContext, payload interface{}) error {
			log.Println("  [Action] fetchData: attempting to fetch...")
			
			// Simulate failure
			return fmt.Errorf("network error: connection timeout")
		}).
		WithAction("fetchDataWithRetry", func(ctx *statemachine.ActionContext, payload interface{}) error {
			maxRetries := 3
			
			for i := 0; i < maxRetries; i++ {
				log.Printf("  [Action] Attempt %d/%d", i+1, maxRetries)
				
				err := ctx.Dispatch("fetchData", nil)
				if err == nil {
					return nil
				}
				
				// Increment retry count
				ctx.Commit("incrementRetry", nil)
				
				if i < maxRetries-1 {
					time.Sleep(100 * time.Millisecond)
				}
			}
			
			return fmt.Errorf("max retries exceeded")
		}).
		// on_error handler
		WithErrorHandler(func(action string, err error, payload interface{}) {
			log.Printf("  [on_error] Action '%s' failed: %v", action, err)
			
			// Could dispatch recovery action, send alert, etc.
		}).
		Build()

	// Dispatch action that will fail
	log.Println("\n1. Simple failure:")
	store.Dispatch(context.Background(), "fetchData", nil)

	// Dispatch action with retry logic
	log.Println("\n2. With retry:")
	store.Dispatch(context.Background(), "fetchDataWithRetry", nil)

	// Check retry count
	state := store.GetState()
	log.Printf("\n✅ Retry count: %v\n", state.Data["retryCount"])
	log.Println()
}

func stateMachineStoreExample() {
	log.Println("--- State Machine + Store Pattern Example ---")

	// Create a state machine
	builder := statemachine.NewBuilder("order", "Order State Machine")
	builder.WithInitialState("pending")
	builder.AddStates(
		statemachine.SimpleState("pending", "Pending"),
		statemachine.SimpleState("processing", "Processing"),
		statemachine.SimpleState("completed", "Completed"),
	)
	builder.AddTransitions(
		statemachine.SimpleTransition("process", "pending", "processing", "process"),
		statemachine.SimpleTransition("complete", "processing", "completed", "complete"),
	)
	definition, _ := builder.Build()

	// Create engine
	engine, _ := statemachine.NewEngine(definition, statemachine.DefaultConfig(), nil)

	// Create store adapter
	adapter, _ := statemachine.NewStateMachineStoreAdapter(engine, map[string]interface{}{
		"orderId": "ORD-123",
		"amount":  99.99,
	})

	// Subscribe to state changes
	adapter.Store().Subscribe(func(mutation string, state *statemachine.StateContext) {
		log.Printf("  [Subscribe] State changed: %s → %s", 
			state.PreviousState, state.CurrentState)
	})

	// Dispatch events via store pattern
	log.Println("\n1. Dispatch 'process' event:")
	adapter.Dispatch(context.Background(), "process", nil)

	log.Println("\n2. Dispatch 'complete' event:")
	adapter.Dispatch(context.Background(), "complete", nil)

	// Get current state via getter
	currentState := adapter.GetCurrentState()
	log.Printf("\n✅ Final state: %s\n", currentState)
	log.Println()
}
