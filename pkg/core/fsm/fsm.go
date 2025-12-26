package fsm

import (
	"fmt"
	"sync"
)

// State represents a state in the machine
type State string

// Event represents an event that triggers a transition
type Event string

// Transition represents a valid state transition
type Transition struct {
	From  State
	Event Event
	To    State
}

// FSM is a thread-safe finite state machine
type FSM struct {
	currentState State
	transitions  map[State]map[Event]State
	callbacks    map[State]func(event Event)
	mu           sync.RWMutex
}

// NewFSM creates a new FSM with the initial state
func NewFSM(initialState State) *FSM {
	return &FSM{
		currentState: initialState,
		transitions:  make(map[State]map[Event]State),
		callbacks:    make(map[State]func(event Event)),
	}
}

// AddTransition adds a valid transition
func (f *FSM) AddTransition(from State, event Event, to State) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, ok := f.transitions[from]; !ok {
		f.transitions[from] = make(map[Event]State)
	}
	f.transitions[from][event] = to
}

// SetStateCallback sets a callback to be executed when entering a state
func (f *FSM) SetStateCallback(state State, callback func(event Event)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callbacks[state] = callback
}

// CurrentState returns the current state
func (f *FSM) CurrentState() State {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.currentState
}

// Trigger triggers an event
func (f *FSM) Trigger(event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check for valid transition
	toState, ok := f.transitions[f.currentState][event]
	if !ok {
		return fmt.Errorf("invalid transition from state '%s' with event '%s'", f.currentState, event)
	}

	// Update state
	f.currentState = toState

	// Execute callback if any
	if callback, exists := f.callbacks[toState]; exists {
		// execute callback synchronously to ensure state consistency
		// for async callbacks, the user should handle goroutines inside the callback
		callback(event)
	}

	return nil
}

// CanTrigger checks if an event can be triggered from the current state
func (f *FSM) CanTrigger(event Event) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, ok := f.transitions[f.currentState][event]
	return ok
}
