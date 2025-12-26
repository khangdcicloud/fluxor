package fsm

import (
	"context"
	"fmt"
	"sync"

	"github.com/fluxorio/fluxor/pkg/core"
)

// State represents a state in the finite state machine
type State string

// Event represents an event that triggers a transition
type Event string

// Action is a function executed during transitions
type Action func(ctx context.Context, event Event, data interface{}) error

// Transition represents a state transition
type Transition struct {
	From   State
	To     State
	Event  Event
	Action Action
}

// FSM represents a Finite State Machine
type FSM struct {
	currentState State
	transitions  map[State]map[Event]*Transition
	onEntry      map[State]Action
	onExit       map[State]Action
	mu           sync.RWMutex
	logger       core.Logger
}

// NewFSM creates a new Finite State Machine
func NewFSM(initialState State, logger core.Logger) *FSM {
	if logger == nil {
		logger = core.NewDefaultLogger()
	}
	return &FSM{
		currentState: initialState,
		transitions:  make(map[State]map[Event]*Transition),
		onEntry:      make(map[State]Action),
		onExit:       make(map[State]Action),
		logger:       logger,
	}
}

// AddTransition adds a transition to the FSM
func (f *FSM) AddTransition(from State, event Event, to State, action Action) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.transitions[from]; !ok {
		f.transitions[from] = make(map[Event]*Transition)
	}

	f.transitions[from][event] = &Transition{
		From:   from,
		To:     to,
		Event:  event,
		Action: action,
	}
}

// OnEntry registers an action to be executed when entering a state
func (f *FSM) OnEntry(state State, action Action) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onEntry[state] = action
}

// OnExit registers an action to be executed when exiting a state
func (f *FSM) OnExit(state State, action Action) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.onExit[state] = action
}

// Fire triggers an event and executes the transition if valid
func (f *FSM) Fire(ctx context.Context, event Event, data interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	from := f.currentState
	
	// Find transition
	stateTransitions, ok := f.transitions[from]
	if !ok {
		return fmt.Errorf("no transitions defined for state %s", from)
	}

	transition, ok := stateTransitions[event]
	if !ok {
		return fmt.Errorf("no transition defined for state %s on event %s", from, event)
	}

	to := transition.To

	f.logger.Infof("FSM Transition: %s -> %s on event %s", from, to, event)

	// Execute OnExit for current state
	if exitAction, ok := f.onExit[from]; ok {
		if err := exitAction(ctx, event, data); err != nil {
			return fmt.Errorf("failed to execute OnExit action for state %s: %w", from, err)
		}
	}

	// Execute Transition Action
	if transition.Action != nil {
		if err := transition.Action(ctx, event, data); err != nil {
			return fmt.Errorf("failed to execute transition action for %s -> %s: %w", from, to, err)
		}
	}

	// Execute OnEntry for next state
	if entryAction, ok := f.onEntry[to]; ok {
		if err := entryAction(ctx, event, data); err != nil {
			return fmt.Errorf("failed to execute OnEntry action for state %s: %w", to, err)
		}
	}

	// Update state
	f.currentState = to

	return nil
}

// Current returns the current state
func (f *FSM) Current() State {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.currentState
}

// CanFire checks if an event can trigger a transition from the current state
func (f *FSM) CanFire(event Event) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stateTransitions, ok := f.transitions[f.currentState]
	if !ok {
		return false
	}

	_, ok = stateTransitions[event]
	return ok
}
