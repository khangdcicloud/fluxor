package statemachine

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

func TestStore_Commit(t *testing.T) {
	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: map[string]interface{}{
				"count": 0,
			},
		}).
		WithMutation("increment", func(state *StateContext, payload interface{}) error {
			count := state.Data["count"].(int)
			state.Data["count"] = count + 1
			return nil
		}).
		Build()

	// Commit mutation
	err := store.Commit("increment", nil)
	if err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Check state
	state := store.GetState()
	if state.Data["count"] != 1 {
		t.Errorf("Expected count to be 1, got %v", state.Data["count"])
	}
}

func TestStore_Dispatch(t *testing.T) {
	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: map[string]interface{}{
				"value": 0,
			},
		}).
		WithMutation("setValue", func(state *StateContext, payload interface{}) error {
			state.Data["value"] = payload
			return nil
		}).
		WithAction("updateValue", func(ctx *ActionContext, payload interface{}) error {
			// Async operation (simulated)
			time.Sleep(10 * time.Millisecond)
			
			// Commit mutation
			return ctx.Commit("setValue", payload)
		}).
		Build()

	// Dispatch action
	err := store.Dispatch(context.Background(), "updateValue", 42)
	if err != nil {
		t.Fatalf("Failed to dispatch: %v", err)
	}

	// Check state
	state := store.GetState()
	if state.Data["value"] != 42 {
		t.Errorf("Expected value to be 42, got %v", state.Data["value"])
	}
}

func TestStore_Getter(t *testing.T) {
	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: map[string]interface{}{
				"firstName": "John",
				"lastName":  "Doe",
			},
		}).
		WithGetter("fullName", func(state *StateContext) interface{} {
			return state.Data["firstName"].(string) + " " + state.Data["lastName"].(string)
		}).
		Build()

	// Get computed value
	fullName, err := store.Get("fullName")
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}

	if fullName != "John Doe" {
		t.Errorf("Expected 'John Doe', got %v", fullName)
	}
}

func TestStore_Subscribe(t *testing.T) {
	mutationCalled := false
	mutationName := ""

	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: make(map[string]interface{}),
		}).
		WithMutation("testMutation", func(state *StateContext, payload interface{}) error {
			return nil
		}).
		Build()

	// Subscribe
	unsubscribe := store.Subscribe(func(mutation string, state *StateContext) {
		mutationCalled = true
		mutationName = mutation
	})
	defer unsubscribe()

	// Commit mutation
	store.Commit("testMutation", nil)

	if !mutationCalled {
		t.Error("Expected subscriber to be called")
	}

	if mutationName != "testMutation" {
		t.Errorf("Expected mutation name 'testMutation', got %s", mutationName)
	}
}

func TestStore_OnError(t *testing.T) {
	errorCalled := false
	var capturedError error

	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: make(map[string]interface{}),
		}).
		WithAction("failingAction", func(ctx *ActionContext, payload interface{}) error {
			return context.DeadlineExceeded
		}).
		WithErrorHandler(func(action string, err error, payload interface{}) {
			errorCalled = true
			capturedError = err
		}).
		Build()

	// Dispatch failing action
	err := store.Dispatch(context.Background(), "failingAction", nil)
	if err == nil {
		t.Error("Expected error from failing action")
	}

	if !errorCalled {
		t.Error("Expected error handler to be called")
	}

	if capturedError != context.DeadlineExceeded {
		t.Errorf("Expected DeadlineExceeded error, got %v", capturedError)
	}
}

func TestStore_NestedActions(t *testing.T) {
	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: map[string]interface{}{
				"step": 0,
			},
		}).
		WithMutation("setStep", func(state *StateContext, payload interface{}) error {
			state.Data["step"] = payload
			return nil
		}).
		WithAction("step1", func(ctx *ActionContext, payload interface{}) error {
			ctx.Commit("setStep", 1)
			return ctx.Dispatch("step2", nil)
		}).
		WithAction("step2", func(ctx *ActionContext, payload interface{}) error {
			return ctx.Commit("setStep", 2)
		}).
		Build()

	// Dispatch nested actions
	err := store.Dispatch(context.Background(), "step1", nil)
	if err != nil {
		t.Fatalf("Failed to dispatch: %v", err)
	}

	// Check final state
	state := store.GetState()
	if state.Data["step"] != 2 {
		t.Errorf("Expected step to be 2, got %v", state.Data["step"])
	}
}

func TestLoggerPlugin(t *testing.T) {
	logger := core.NewDefaultLogger()
	
	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: make(map[string]interface{}),
		}).
		WithMutation("test", func(state *StateContext, payload interface{}) error {
			return nil
		}).
		WithPlugin(LoggerPlugin(logger)).
		Build()

	// Commit should trigger logger
	store.Commit("test", nil)
	// Just verify no panic
}

func TestEventBusPlugin(t *testing.T) {
	vertx := core.NewVertx(context.Background())
	defer vertx.Close()

	eventReceived := false
	vertx.EventBus().Consumer("store.mutation.test").Handler(
		func(ctx core.FluxorContext, msg core.Message) error {
			eventReceived = true
			return nil
		})

	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: make(map[string]interface{}),
		}).
		WithMutation("test", func(state *StateContext, payload interface{}) error {
			return nil
		}).
		WithPlugin(EventBusPlugin(vertx.EventBus(), "store")).
		Build()

	// Commit should publish to EventBus
	store.Commit("test", nil)

	// Give EventBus time to process
	time.Sleep(100 * time.Millisecond)

	if !eventReceived {
		t.Error("Expected event to be received via EventBus")
	}
}

func TestHistoryPlugin(t *testing.T) {
	store := NewStoreBuilder().
		WithState(&StateContext{
			Data:    make(map[string]interface{}),
			History: make([]*HistoryEntry, 0),
		}).
		WithMutation("change", func(state *StateContext, payload interface{}) error {
			state.Data["value"] = payload
			return nil
		}).
		WithPlugin(HistoryPlugin(5)).
		Build()

	// Make multiple mutations
	for i := 0; i < 3; i++ {
		store.Commit("change", i)
	}

	// Check history
	state := store.GetState()
	if len(state.History) != 3 {
		t.Errorf("Expected 3 history entries, got %d", len(state.History))
	}
}

func TestStateMachineStoreAdapter(t *testing.T) {
	// Create a simple state machine
	def := createTestStateMachine()
	engine, err := NewEngine(def, DefaultConfig(), nil)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	// Create adapter
	adapter, err := NewStateMachineStoreAdapter(engine, map[string]interface{}{
		"test": "data",
	})
	if err != nil {
		t.Fatalf("Failed to create adapter: %v", err)
	}

	// Check initial state
	currentState := adapter.GetCurrentState()
	if currentState != "initial" {
		t.Errorf("Expected initial state, got %s", currentState)
	}

	// Dispatch event via store
	err = adapter.Dispatch(context.Background(), "start", nil)
	if err != nil {
		t.Fatalf("Failed to dispatch: %v", err)
	}

	// Check new state
	currentState = adapter.GetCurrentState()
	if currentState != "running" {
		t.Errorf("Expected running state, got %s", currentState)
	}
}

func TestStore_ConcurrentCommits(t *testing.T) {
	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: map[string]interface{}{
				"count": 0,
			},
		}).
		WithMutation("increment", func(state *StateContext, payload interface{}) error {
			count := state.Data["count"].(int)
			state.Data["count"] = count + 1
			return nil
		}).
		Build()

	// Concurrent commits
	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			store.Commit("increment", nil)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Check final count
	state := store.GetState()
	if state.Data["count"] != 100 {
		t.Errorf("Expected count to be 100, got %v", state.Data["count"])
	}
}

func BenchmarkStore_Commit(b *testing.B) {
	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: make(map[string]interface{}),
		}).
		WithMutation("test", func(state *StateContext, payload interface{}) error {
			state.Data["value"] = payload
			return nil
		}).
		Build()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Commit("test", i)
	}
}

func BenchmarkStore_Dispatch(b *testing.B) {
	store := NewStoreBuilder().
		WithState(&StateContext{
			Data: make(map[string]interface{}),
		}).
		WithMutation("test", func(state *StateContext, payload interface{}) error {
			state.Data["value"] = payload
			return nil
		}).
		WithAction("update", func(ctx *ActionContext, payload interface{}) error {
			return ctx.Commit("test", payload)
		}).
		Build()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Dispatch(ctx, "update", i)
	}
}
