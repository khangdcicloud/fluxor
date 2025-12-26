package fsm

import (
	"context"
	"fmt"
	"sync"

	"github.com/fluxorio/fluxor/pkg/fluxor"
)

// TransitionType defines the type of transition
type TransitionType int

const (
	// TransitionExternal causes a state change (exits source, enters target)
	TransitionExternal TransitionType = iota
	// TransitionInternal does not cause a state change (no exit/entry)
	TransitionInternal
)

// Action is a function executed during transitions
type Action[S comparable, E comparable] func(ctx context.Context, t TransitionContext[S, E]) error

// Guard decides if a transition can occur
type Guard[S comparable, E comparable] func(ctx context.Context, t TransitionContext[S, E]) bool

// TransitionContext holds context about the current transition
type TransitionContext[S comparable, E comparable] struct {
	FSM   *StateMachine[S, E]
	Event E
	From  S
	To    S
	Data  any
}

// StateMachine implements a generic Finite State Machine using the Actor Model.
// It processes events sequentially to prevent race conditions and deadlocks.
type StateMachine[S comparable, E comparable] struct {
	currentState S
	states       map[S]*stateConfig[S, E]
	
	// Actor communication
	cmdChan   chan command[S, E]
	stateChan chan S // For synchronous state reads
	closeChan chan struct{}
	once      sync.Once

	// Global interceptors
	onTransition []func(TransitionContext[S, E])
}

// Internal structures
type stateConfig[S comparable, E comparable] struct {
	state       S
	onEntry     []Action[S, E]
	onExit      []Action[S, E]
	transitions map[E]*transition[S, E]
}

type transition[S comparable, E comparable] struct {
	trigger E
	from    S
	to      S
	guard   Guard[S, E]
	actions []Action[S, E]
	kind    TransitionType
}

// command interface for the actor loop
type command[S comparable, E comparable] interface {
	execute(sm *StateMachine[S, E])
}

type fireCommand[S comparable, E comparable] struct {
	ctx     context.Context
	event   E
	data    any
	promise *fluxor.PromiseT[S]
}

func (c fireCommand[S, E]) execute(sm *StateMachine[S, E]) {
	sm.handleFire(c.ctx, c.event, c.data, c.promise)
}

type configCommand[S comparable, E comparable] struct {
	state  S
	config *stateConfig[S, E]
}

func (c configCommand[S, E]) execute(sm *StateMachine[S, E]) {
	sm.states[c.state] = c.config
}

// New creates a new StateMachine with an initial state
func New[S comparable, E comparable](initialState S) *StateMachine[S, E] {
	sm := &StateMachine[S, E]{
		currentState: initialState,
		states:       make(map[S]*stateConfig[S, E]),
		cmdChan:      make(chan command[S, E], 100), // Buffered for performance
		stateChan:    make(chan S),
		closeChan:    make(chan struct{}),
		onTransition: make([]func(TransitionContext[S, E]), 0),
	}
	go sm.loop()
	return sm
}

// loop is the actor loop that processes all state mutations
func (sm *StateMachine[S, E]) loop() {
	for {
		select {
		case cmd := <-sm.cmdChan:
			cmd.execute(sm)
		case sm.stateChan <- sm.currentState:
			// Serve current state reads
		case <-sm.closeChan:
			return
		}
	}
}

// Close stops the state machine loop
func (sm *StateMachine[S, E]) Close() {
	sm.once.Do(func() {
		close(sm.closeChan)
	})
}

// CurrentState returns the current state synchronously
func (sm *StateMachine[S, E]) CurrentState() S {
	select {
	case s := <-sm.stateChan:
		return s
	case <-sm.closeChan:
		return sm.currentState // Return last known state on closed
	}
}

// Configure returns a Builder to configure a state.
// Note: Configuration should ideally happen before firing events.
// The builder updates the internal config map safely via the actor loop.
func (sm *StateMachine[S, E]) Configure(state S) *StateConfigBuilder[S, E] {
	return &StateConfigBuilder[S, E]{
		sm:    sm,
		state: state,
	}
}

// Fire triggers an event and returns a FutureT[S]
func (sm *StateMachine[S, E]) Fire(ctx context.Context, event E, data any) *fluxor.FutureT[S] {
	promise := fluxor.NewPromiseT[S]()
	
	select {
	case sm.cmdChan <- fireCommand[S, E]{
		ctx:     ctx,
		event:   event,
		data:    data,
		promise: promise,
	}:
		// Command sent
	case <-sm.closeChan:
		promise.Fail(fmt.Errorf("state machine is closed"))
	}
	
	return &promise.FutureT
}

// Internal handleFire logic executed by the loop
func (sm *StateMachine[S, E]) handleFire(ctx context.Context, event E, data any, promise *fluxor.PromiseT[S]) {
	currentState := sm.currentState
	
	// 1. Find Config
	config, ok := sm.states[currentState]
	if !ok {
		promise.Fail(fmt.Errorf("no configuration for state %v", currentState))
		return
	}

	// 2. Find Transition
	trans, ok := config.transitions[event]
	if !ok {
		promise.Fail(fmt.Errorf("no transition defined for event %v in state %v", event, currentState))
		return
	}

	// 3. Create Context
	tCtx := TransitionContext[S, E]{
		FSM:   sm,
		Event: event,
		From:  currentState,
		To:    trans.to,
		Data:  data,
	}

	// 4. Check Guard
	if trans.guard != nil && !trans.guard(ctx, tCtx) {
		promise.Fail(fmt.Errorf("guard failed for transition %v -> %v", currentState, trans.to))
		return
	}

	// 5. Execute Exit Actions (if external)
	if trans.kind == TransitionExternal {
		for _, action := range config.onExit {
			if err := action(ctx, tCtx); err != nil {
				promise.Fail(fmt.Errorf("exit action failed: %w", err))
				return
			}
		}
	}

	// 6. Execute Transition Actions
	for _, action := range trans.actions {
		if err := action(ctx, tCtx); err != nil {
			promise.Fail(fmt.Errorf("transition action failed: %w", err))
			return
		}
	}

	// 7. Update State
	sm.currentState = trans.to

	// 8. Execute Entry Actions (if external)
	if trans.kind == TransitionExternal {
		newConfig, ok := sm.states[trans.to]
		if ok {
			for _, action := range newConfig.onEntry {
				if err := action(ctx, tCtx); err != nil {
					// State updated, but entry failed. This is a critical error.
					promise.Fail(fmt.Errorf("entry action failed: %w", err))
					return
				}
			}
		}
	}

	// 9. Notify Listeners
	for _, l := range sm.onTransition {
		l(tCtx)
	}

	promise.Complete(sm.currentState)
}

// OnTransition registers a global transition listener.
// Note: This is not thread-safe if called concurrently with Fire. Call at setup.
func (sm *StateMachine[S, E]) OnTransition(listener func(TransitionContext[S, E])) {
	sm.onTransition = append(sm.onTransition, listener)
}
