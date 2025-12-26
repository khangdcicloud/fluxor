package fsm

import "context"

// StateConfigBuilder provides a fluent API for configuring states.
// It queues configuration updates to the StateMachine.
type StateConfigBuilder[S comparable, E comparable] struct {
	sm    *StateMachine[S, E]
	state S
}

// Helper to safely update config in the actor loop
func (b *StateConfigBuilder[S, E]) updateConfig(update func(*stateConfig[S, E])) {
	// We send a command to the actor loop to update the configuration
	// This ensures thread safety even if Configure is called dynamically (though not recommended)
	b.sm.cmdChan <- &updateConfigCommand[S, E]{
		state:  b.state,
		update: update,
	}
}

type updateConfigCommand[S comparable, E comparable] struct {
	state  S
	update func(*stateConfig[S, E])
}

func (c *updateConfigCommand[S, E]) execute(sm *StateMachine[S, E]) {
	config, ok := sm.states[c.state]
	if !ok {
		config = &stateConfig[S, E]{
			state:       c.state,
			onEntry:     make([]Action[S, E], 0),
			onExit:      make([]Action[S, E], 0),
			transitions: make(map[E]*transition[S, E]),
		}
		sm.states[c.state] = config
	}
	c.update(config)
}

// Permit defines an allowed transition
func (b *StateConfigBuilder[S, E]) Permit(event E, nextState S) *StateConfigBuilder[S, E] {
	return b.PermitIf(event, nextState, nil)
}

// PermitIf defines a allowed transition if the guard returns true
func (b *StateConfigBuilder[S, E]) PermitIf(event E, nextState S, guard Guard[S, E]) *StateConfigBuilder[S, E] {
	b.updateConfig(func(c *stateConfig[S, E]) {
		c.transitions[event] = &transition[S, E]{
			trigger: event,
			from:    b.state,
			to:      nextState,
			guard:   guard,
			actions: make([]Action[S, E], 0),
			kind:    TransitionExternal,
		}
	})
	return b
}

// PermitWithAction defines a transition that executes an action
func (b *StateConfigBuilder[S, E]) PermitWithAction(event E, nextState S, action Action[S, E]) *StateConfigBuilder[S, E] {
	b.updateConfig(func(c *stateConfig[S, E]) {
		c.transitions[event] = &transition[S, E]{
			trigger: event,
			from:    b.state,
			to:      nextState,
			actions: []Action[S, E]{action},
			kind:    TransitionExternal,
		}
	})
	return b
}

// Ignore defines an event that should be ignored
func (b *StateConfigBuilder[S, E]) Ignore(event E) *StateConfigBuilder[S, E] {
	return b.InternalTransition(event, func(_ context.Context, _ TransitionContext[S, E]) error {
		return nil
	})
}

// OnEntry adds an action to be executed when entering this state
func (b *StateConfigBuilder[S, E]) OnEntry(action Action[S, E]) *StateConfigBuilder[S, E] {
	b.updateConfig(func(c *stateConfig[S, E]) {
		c.onEntry = append(c.onEntry, action)
	})
	return b
}

// OnExit adds an action to be executed when exiting this state
func (b *StateConfigBuilder[S, E]) OnExit(action Action[S, E]) *StateConfigBuilder[S, E] {
	b.updateConfig(func(c *stateConfig[S, E]) {
		c.onExit = append(c.onExit, action)
	})
	return b
}

// InternalTransition defines a transition that executes an action but stays in the same state
func (b *StateConfigBuilder[S, E]) InternalTransition(event E, action Action[S, E]) *StateConfigBuilder[S, E] {
	b.updateConfig(func(c *stateConfig[S, E]) {
		c.transitions[event] = &transition[S, E]{
			trigger: event,
			from:    b.state,
			to:      b.state,
			actions: []Action[S, E]{action},
			kind:    TransitionInternal,
		}
	})
	return b
}
