package fsm

import (
	"context"
	"testing"
	"time"
)

func TestGenericFSM(t *testing.T) {
	// Define types
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

	sm := New[OrderState, OrderEvent](Created)
	defer sm.Close()

	// Use channels to signal completion of configuration since it's async
	// In real apps, config happens before use. Here we need to ensure it's applied.
	// Since the Actor loop processes sequentially, we can just fire a dummy event or rely on timing.
	// Better: We rely on the fact that Configure pushes to the channel, and Fire pushes to the channel.
	// So if we Configure then Fire, Fire is processed AFTER Configure.

	sm.Configure(Created).
		Permit(Pay, Paid).
		OnExit(func(ctx context.Context, t TransitionContext[OrderState, OrderEvent]) error {
			// Log exit
			return nil
		})

	sm.Configure(Paid).
		Permit(Ship, Shipped)

	ctx := context.Background()

	// Test 1: Created -> Paid
	state, err := sm.Fire(ctx, Pay, nil).Await(ctx)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if state != Paid {
		t.Errorf("Expected Paid, got %v", state)
	}

	// Test 2: Paid -> Shipped
	state, err = sm.Fire(ctx, Ship, nil).Await(ctx)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if state != Shipped {
		t.Errorf("Expected Shipped, got %v", state)
	}
}

func TestInternalTransition(t *testing.T) {
	sm := New[string, string]("A")
	defer sm.Close()

	count := 0
	done := make(chan bool)

	sm.Configure("A").
		InternalTransition("Inc", func(ctx context.Context, _ TransitionContext[string, string]) error {
			count++
			return nil
		})

	// Fire
	ctx := context.Background()
	_, err := sm.Fire(ctx, "Inc", nil).Await(ctx)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
	close(done)
}

func TestGuard(t *testing.T) {
	sm := New[string, string]("Start")
	defer sm.Close()

	sm.Configure("Start").
		PermitIf("Go", "End", func(ctx context.Context, t TransitionContext[string, string]) bool {
			val, ok := t.Data.(bool)
			return ok && val
		})

	ctx := context.Background()

	// Should fail
	_, err := sm.Fire(ctx, "Go", false).Await(ctx)
	if err == nil {
		t.Error("Expected guard failure")
	}

	// Should succeed
	state, err := sm.Fire(ctx, "Go", true).Await(ctx)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if state != "End" {
		t.Errorf("Expected End, got %s", state)
	}
}

// TestReentrancy checks if an action can trigger another event
func TestReentrancy(t *testing.T) {
	sm := New[string, string]("A")
	defer sm.Close()

	sm.Configure("A").
		Permit("Event1", "B")

	sm.Configure("B").
		OnEntry(func(ctx context.Context, tx TransitionContext[string, string]) error {
			// Trigger next event asynchronously
			// Note: We don't await here to avoid blocking the actor loop if we were using synchronous calls
			// But since Fire() is async (pushes to channel), this is safe!
			tx.FSM.Fire(ctx, "Event2", nil)
			return nil
		}).
		Permit("Event2", "C")

	ctx := context.Background()
	state, err := sm.Fire(ctx, "Event1", nil).Await(ctx)
	if err != nil {
		t.Fatalf("First fire failed: %v", err)
	}
	if state != "B" {
		t.Errorf("Expected B, got %s", state)
	}

	// Wait for the second transition to happen
	// We can't await the specific future created inside the action easily.
	// But we can poll CurrentState or fire a dummy event to synchronize.
	
	// Let's poll
	for i := 0; i < 10; i++ {
		if sm.CurrentState() == "C" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("Did not reach state C via reentrant fire")
}
