package fsm

import (
	"context"
	"errors"
	"testing"
)

const (
	StateIdle    State = "IDLE"
	StateRunning State = "RUNNING"
	StatePaused  State = "PAUSED"
	StateStopped State = "STOPPED"

	EventStart  Event = "START"
	EventPause  Event = "PAUSE"
	EventResume Event = "RESUME"
	EventStop   Event = "STOP"
)

func TestFSM_Transitions(t *testing.T) {
	fsm := NewFSM(StateIdle, nil)

	// Define transitions
	fsm.AddTransition(StateIdle, EventStart, StateRunning, nil)
	fsm.AddTransition(StateRunning, EventPause, StatePaused, nil)
	fsm.AddTransition(StatePaused, EventResume, StateRunning, nil)
	fsm.AddTransition(StateRunning, EventStop, StateStopped, nil)
	fsm.AddTransition(StatePaused, EventStop, StateStopped, nil)

	ctx := context.Background()

	// Initial state
	if fsm.Current() != StateIdle {
		t.Errorf("expected state %s, got %s", StateIdle, fsm.Current())
	}

	// Idle -> Running
	if err := fsm.Fire(ctx, EventStart, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if fsm.Current() != StateRunning {
		t.Errorf("expected state %s, got %s", StateRunning, fsm.Current())
	}

	// Running -> Paused
	if err := fsm.Fire(ctx, EventPause, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if fsm.Current() != StatePaused {
		t.Errorf("expected state %s, got %s", StatePaused, fsm.Current())
	}

	// Paused -> Running
	if err := fsm.Fire(ctx, EventResume, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if fsm.Current() != StateRunning {
		t.Errorf("expected state %s, got %s", StateRunning, fsm.Current())
	}

	// Running -> Stopped
	if err := fsm.Fire(ctx, EventStop, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if fsm.Current() != StateStopped {
		t.Errorf("expected state %s, got %s", StateStopped, fsm.Current())
	}
}

func TestFSM_InvalidTransition(t *testing.T) {
	fsm := NewFSM(StateIdle, nil)
	fsm.AddTransition(StateIdle, EventStart, StateRunning, nil)

	ctx := context.Background()

	// Invalid event for Idle
	if err := fsm.Fire(ctx, EventStop, nil); err == nil {
		t.Error("expected error for invalid transition")
	}

	if fsm.Current() != StateIdle {
		t.Errorf("state should remain %s", StateIdle)
	}
}

func TestFSM_Actions(t *testing.T) {
	fsm := NewFSM(StateIdle, nil)
	
	entryCalled := false
	exitCalled := false
	transitionCalled := false

	fsm.OnExit(StateIdle, func(ctx context.Context, event Event, data interface{}) error {
		exitCalled = true
		return nil
	})

	fsm.OnEntry(StateRunning, func(ctx context.Context, event Event, data interface{}) error {
		entryCalled = true
		return nil
	})

	fsm.AddTransition(StateIdle, EventStart, StateRunning, func(ctx context.Context, event Event, data interface{}) error {
		transitionCalled = true
		return nil
	})

	ctx := context.Background()
	if err := fsm.Fire(ctx, EventStart, nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !exitCalled {
		t.Error("OnExit action not called")
	}
	if !transitionCalled {
		t.Error("Transition action not called")
	}
	if !entryCalled {
		t.Error("OnEntry action not called")
	}
}

func TestFSM_ActionError(t *testing.T) {
	fsm := NewFSM(StateIdle, nil)
	expectedErr := errors.New("transition error")

	fsm.AddTransition(StateIdle, EventStart, StateRunning, func(ctx context.Context, event Event, data interface{}) error {
		return expectedErr
	})

	ctx := context.Background()
	err := fsm.Fire(ctx, EventStart, nil)
	if err == nil {
		t.Error("expected error")
	}
	
	// State should NOT change if action fails
	// Note: In this implementation, if action fails, state is NOT updated because update happens at the end
	// However, OnExit was already executed. This is a design choice.
	if fsm.Current() != StateIdle {
		t.Errorf("state should remain %s on transition failure", StateIdle)
	}
}
