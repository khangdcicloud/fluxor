package fsm_test

import (
	"sync"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core/fsm"
)

const (
	StateIdle    fsm.State = "IDLE"
	StateRunning fsm.State = "RUNNING"
	StateStopped fsm.State = "STOPPED"

	EventStart fsm.Event = "START"
	EventStop  fsm.Event = "STOP"
)

func TestFSM(t *testing.T) {
	machine := fsm.NewFSM(StateIdle)

	machine.AddTransition(StateIdle, EventStart, StateRunning)
	machine.AddTransition(StateRunning, EventStop, StateStopped)

	if machine.CurrentState() != StateIdle {
		t.Errorf("expected state %s, got %s", StateIdle, machine.CurrentState())
	}

	if !machine.CanTrigger(EventStart) {
		t.Error("expected EventStart to be valid from Idle")
	}

	if machine.CanTrigger(EventStop) {
		t.Error("expected EventStop to be invalid from Idle")
	}

	// Trigger Start
	var wg sync.WaitGroup
	wg.Add(1)
	machine.SetStateCallback(StateRunning, func(event fsm.Event) {
		defer wg.Done()
		if event != EventStart {
			t.Errorf("expected event %s, got %s", EventStart, event)
		}
	})

	if err := machine.Trigger(EventStart); err != nil {
		t.Fatalf("failed to trigger START: %v", err)
	}

	wg.Wait()

	if machine.CurrentState() != StateRunning {
		t.Errorf("expected state %s, got %s", StateRunning, machine.CurrentState())
	}

	// Trigger Stop
	if err := machine.Trigger(EventStop); err != nil {
		t.Fatalf("failed to trigger STOP: %v", err)
	}

	if machine.CurrentState() != StateStopped {
		t.Errorf("expected state %s, got %s", StateStopped, machine.CurrentState())
	}

	// Invalid transition
	if err := machine.Trigger(EventStart); err == nil {
		t.Error("expected error triggering START from STOPPED")
	}
}
