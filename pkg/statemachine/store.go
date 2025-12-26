package statemachine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
)

// Store represents a centralized state store (Redux/Vuex pattern).
type Store struct {
	state      *StateContext
	mutations  map[string]Mutation
	actions    map[string]Action
	getters    map[string]Getter
	plugins    []Plugin
	mu         sync.RWMutex
	subscribers []Subscriber
	errorHandlers []ErrorHandler
	logger     core.Logger
}

// Mutation is a synchronous state change (like Vuex mutations).
type Mutation func(state *StateContext, payload interface{}) error

// Action is an async operation that can commit mutations (like Vuex actions).
type Action func(ctx *ActionContext, payload interface{}) error

// Getter computes derived state (like Vuex getters).
type Getter func(state *StateContext) interface{}

// Subscriber is called after each mutation.
type Subscriber func(mutation string, state *StateContext)

// ErrorHandler handles errors from actions.
type ErrorHandler func(action string, err error, payload interface{})

// Plugin can hook into store lifecycle.
type Plugin func(store *Store)

// ActionContext provides context for actions.
type ActionContext struct {
	Store   *Store
	Context context.Context
	Commit  func(mutation string, payload interface{}) error
	Dispatch func(action string, payload interface{}) error
	State   *StateContext
}

// StoreOptions configures a store.
type StoreOptions struct {
	State          *StateContext
	Mutations      map[string]Mutation
	Actions        map[string]Action
	Getters        map[string]Getter
	Plugins        []Plugin
	ErrorHandlers  []ErrorHandler
}

// NewStore creates a new store.
func NewStore(options *StoreOptions) *Store {
	if options.State == nil {
		options.State = &StateContext{
			Data:    make(map[string]interface{}),
			History: make([]*HistoryEntry, 0),
		}
	}

	store := &Store{
		state:         options.State,
		mutations:     options.Mutations,
		actions:       options.Actions,
		getters:       options.Getters,
		plugins:       options.Plugins,
		subscribers:   make([]Subscriber, 0),
		errorHandlers: options.ErrorHandlers,
		logger:        core.NewDefaultLogger(),
	}

	if store.mutations == nil {
		store.mutations = make(map[string]Mutation)
	}
	if store.actions == nil {
		store.actions = make(map[string]Action)
	}
	if store.getters == nil {
		store.getters = make(map[string]Getter)
	}
	if store.errorHandlers == nil {
		store.errorHandlers = make([]ErrorHandler, 0)
	}

	// Initialize plugins
	for _, plugin := range store.plugins {
		plugin(store)
	}

	return store
}

// Commit applies a mutation synchronously.
func (s *Store) Commit(mutation string, payload interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	mutationFn, ok := s.mutations[mutation]
	if !ok {
		return fmt.Errorf("mutation not found: %s", mutation)
	}

	// Apply mutation
	if err := mutationFn(s.state, payload); err != nil {
		return fmt.Errorf("mutation %s failed: %w", mutation, err)
	}

	// Notify subscribers
	for _, subscriber := range s.subscribers {
		subscriber(mutation, s.state)
	}

	s.logger.Debugf("Committed mutation: %s", mutation)

	return nil
}

// Dispatch executes an action asynchronously.
func (s *Store) Dispatch(ctx context.Context, action string, payload interface{}) error {
	actionFn, ok := s.actions[action]
	if !ok {
		return fmt.Errorf("action not found: %s", action)
	}

	// Create action context
	actionCtx := &ActionContext{
		Store:   s,
		Context: ctx,
		State:   s.GetState(),
		Commit: func(mutation string, payload interface{}) error {
			return s.Commit(mutation, payload)
		},
		Dispatch: func(nestedAction string, nestedPayload interface{}) error {
			return s.Dispatch(ctx, nestedAction, nestedPayload)
		},
	}

	// Execute action (async)
	err := actionFn(actionCtx, payload)
	if err != nil {
		// Call error handlers
		s.handleError(action, err, payload)
		return fmt.Errorf("action %s failed: %w", action, err)
	}

	s.logger.Debugf("Dispatched action: %s", action)

	return nil
}

// DispatchSync executes an action synchronously (blocking).
func (s *Store) DispatchSync(ctx context.Context, action string, payload interface{}) error {
	return s.Dispatch(ctx, action, payload)
}

// Get retrieves a getter value.
func (s *Store) Get(getter string) (interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	getterFn, ok := s.getters[getter]
	if !ok {
		return nil, fmt.Errorf("getter not found: %s", getter)
	}

	return getterFn(s.state), nil
}

// GetState returns a copy of the current state.
func (s *Store) GetState() *StateContext {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a shallow copy to prevent external modification
	stateCopy := *s.state
	return &stateCopy
}

// Subscribe adds a subscriber that's called after each mutation.
func (s *Store) Subscribe(subscriber Subscriber) func() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.subscribers = append(s.subscribers, subscriber)

	// Return unsubscribe function
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, sub := range s.subscribers {
			if &sub == &subscriber {
				s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
				break
			}
		}
	}
}

// RegisterMutation registers a new mutation.
func (s *Store) RegisterMutation(name string, mutation Mutation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mutations[name] = mutation
}

// RegisterAction registers a new action.
func (s *Store) RegisterAction(name string, action Action) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actions[name] = action
}

// RegisterGetter registers a new getter.
func (s *Store) RegisterGetter(name string, getter Getter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getters[name] = getter
}

// OnError registers an error handler.
func (s *Store) OnError(handler ErrorHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errorHandlers = append(s.errorHandlers, handler)
}

// handleError calls all registered error handlers.
func (s *Store) handleError(action string, err error, payload interface{}) {
	s.mu.RLock()
	handlers := make([]ErrorHandler, len(s.errorHandlers))
	copy(handlers, s.errorHandlers)
	s.mu.RUnlock()

	for _, handler := range handlers {
		handler(action, err, payload)
	}
}

// StoreBuilder provides fluent API for building stores.
type StoreBuilder struct {
	options *StoreOptions
}

// NewStoreBuilder creates a new store builder.
func NewStoreBuilder() *StoreBuilder {
	return &StoreBuilder{
		options: &StoreOptions{
			Mutations:     make(map[string]Mutation),
			Actions:       make(map[string]Action),
			Getters:       make(map[string]Getter),
			Plugins:       make([]Plugin, 0),
			ErrorHandlers: make([]ErrorHandler, 0),
		},
	}
}

// WithState sets initial state.
func (b *StoreBuilder) WithState(state *StateContext) *StoreBuilder {
	b.options.State = state
	return b
}

// WithMutation adds a mutation.
func (b *StoreBuilder) WithMutation(name string, mutation Mutation) *StoreBuilder {
	b.options.Mutations[name] = mutation
	return b
}

// WithAction adds an action.
func (b *StoreBuilder) WithAction(name string, action Action) *StoreBuilder {
	b.options.Actions[name] = action
	return b
}

// WithGetter adds a getter.
func (b *StoreBuilder) WithGetter(name string, getter Getter) *StoreBuilder {
	b.options.Getters[name] = getter
	return b
}

// WithPlugin adds a plugin.
func (b *StoreBuilder) WithPlugin(plugin Plugin) *StoreBuilder {
	b.options.Plugins = append(b.options.Plugins, plugin)
	return b
}

// WithErrorHandler adds an error handler.
func (b *StoreBuilder) WithErrorHandler(handler ErrorHandler) *StoreBuilder {
	b.options.ErrorHandlers = append(b.options.ErrorHandlers, handler)
	return b
}

// Build creates the store.
func (b *StoreBuilder) Build() *Store {
	return NewStore(b.options)
}

// Common Plugins

// LoggerPlugin logs all mutations and actions.
func LoggerPlugin(logger core.Logger) Plugin {
	return func(store *Store) {
		store.Subscribe(func(mutation string, state *StateContext) {
			logger.Infof("[Mutation] %s", mutation)
		})
	}
}

// PersistencePlugin persists state after mutations.
func PersistencePlugin(persistFn func(*StateContext) error) Plugin {
	return func(store *Store) {
		store.Subscribe(func(mutation string, state *StateContext) {
			if err := persistFn(state); err != nil {
				store.logger.Errorf("Failed to persist state after %s: %v", mutation, err)
			}
		})
	}
}

// EventBusPlugin publishes mutations to EventBus.
func EventBusPlugin(eventBus core.EventBus, prefix string) Plugin {
	return func(store *Store) {
		store.Subscribe(func(mutation string, state *StateContext) {
			address := fmt.Sprintf("%s.mutation.%s", prefix, mutation)
			eventBus.Publish(address, map[string]interface{}{
				"mutation": mutation,
				"state":    state.Data,
				"timestamp": time.Now(),
			})
		})
	}
}

// HistoryPlugin tracks mutation history.
func HistoryPlugin(maxSize int) Plugin {
	return func(store *Store) {
		history := make([]*HistoryEntry, 0)
		
		store.Subscribe(func(mutation string, state *StateContext) {
			entry := &HistoryEntry{
				FromState:    state.CurrentState,
				ToState:      state.CurrentState,
				Event:        TransitionEvent(mutation),
				Timestamp:    time.Now(),
				TransitionID: mutation,
				Data:         copyMap(state.Data),
			}
			
			history = append(history, entry)
			
			// Trim history if needed
			if maxSize > 0 && len(history) > maxSize {
				history = history[len(history)-maxSize:]
			}
			
			state.History = history
		})
	}
}

// StateMachineStoreAdapter adapts a state machine to work with Store pattern.
type StateMachineStoreAdapter struct {
	engine *Engine
	store  *Store
}

// NewStateMachineStoreAdapter creates an adapter.
func NewStateMachineStoreAdapter(engine *Engine, initialData map[string]interface{}) (*StateMachineStoreAdapter, error) {
	// Create initial state context
	instance, err := engine.CreateInstance(context.Background(), initialData)
	if err != nil {
		return nil, err
	}

	// Create store
	store := NewStoreBuilder().
		WithState(instance.Context).
		WithMutation("transition", func(state *StateContext, payload interface{}) error {
			// Mutation is handled by action
			return nil
		}).
		WithAction("sendEvent", func(ctx *ActionContext, payload interface{}) error {
			eventData, ok := payload.(map[string]interface{})
			if !ok {
				return fmt.Errorf("invalid event payload")
			}

			eventName, ok := eventData["event"].(string)
			if !ok {
				return fmt.Errorf("event name required")
			}

			event := NewEvent(TransitionEvent(eventName)).
				WithDataMap(eventData).
				Build()

			result, err := engine.SendEvent(ctx.Context, ctx.State.MachineID, event)
			if err != nil {
				return err
			}

			if !result.Success {
				return result.Error
			}

			// Commit the state change
			return ctx.Commit("transition", result)
		}).
		WithGetter("currentState", func(state *StateContext) interface{} {
			return state.CurrentState
		}).
		WithGetter("history", func(state *StateContext) interface{} {
			return state.History
		}).
		WithGetter("data", func(state *StateContext) interface{} {
			return state.Data
		}).
		Build()

	return &StateMachineStoreAdapter{
		engine: engine,
		store:  store,
	}, nil
}

// Store returns the underlying store.
func (a *StateMachineStoreAdapter) Store() *Store {
	return a.store
}

// Dispatch sends an event to the state machine via store action.
func (a *StateMachineStoreAdapter) Dispatch(ctx context.Context, event string, data map[string]interface{}) error {
	payload := map[string]interface{}{
		"event": event,
	}
	for k, v := range data {
		payload[k] = v
	}

	return a.store.Dispatch(ctx, "sendEvent", payload)
}

// GetCurrentState returns the current state.
func (a *StateMachineStoreAdapter) GetCurrentState() StateType {
	state, _ := a.store.Get("currentState")
	return state.(StateType)
}
