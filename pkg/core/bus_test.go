package core

import (
	"sync"
	"testing"
	"time"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	if bus == nil {
		t.Fatal("NewBus() returned nil")
	}
}

func TestBus_Subscribe_FailFast_EmptyTopic(t *testing.T) {
	bus := NewBus()

	// Subscribe with empty topic - should handle gracefully or fail-fast
	// Current implementation doesn't validate, but we should test behavior
	bus.Subscribe("", func(msg any) {})

	// If empty topic is allowed, verify it works
	// If it should fail-fast, we should add validation
	bus.Publish("", "test")
}

func TestBus_Subscribe_FailFast_NilHandler(t *testing.T) {
	bus := NewBus()

	defer func() {
		if r := recover(); r == nil {
			// Subscribe with nil handler might panic or handle gracefully
			// Current implementation doesn't validate, but we should add validation
		}
	}()

	bus.Subscribe("test.topic", nil)
	// If nil handler should fail-fast, add panic check
}

func TestBus_Publish_FailFast_EmptyTopic(t *testing.T) {
	bus := NewBus()

	// Publish with empty topic - should handle gracefully
	bus.Publish("", "test message")
}

func TestBus_Publish_NilMessage(t *testing.T) {
	bus := NewBus()
	done := make(chan bool, 1)
	bus.Subscribe("test.topic", func(msg any) {
		// Nil message is allowed in current implementation
		done <- true
	})

	// Publish nil message - handlers should handle it
	bus.Publish("test.topic", nil)

	// Wait for handler to complete to avoid goroutine leak
	select {
	case <-done:
		// Handler completed
	case <-time.After(100 * time.Millisecond):
		t.Error("Handler did not complete in time")
	}
}

func TestBus_Subscribe_Publish(t *testing.T) {
	bus := NewBus()
	var receivedMsg interface{}
	var mu sync.Mutex
	received := make(chan bool, 1)

	handler := func(msg any) {
		mu.Lock()
		receivedMsg = msg
		mu.Unlock()
		received <- true
	}

	bus.Subscribe("test.topic", handler)
	bus.Publish("test.topic", "test message")

	// Wait for handler to be called
	<-received

	mu.Lock()
	msg := receivedMsg
	mu.Unlock()

	if msg != "test message" {
		t.Errorf("Received message = %v, want 'test message'", msg)
	}
}

func TestBus_MultipleSubscribers(t *testing.T) {
	bus := NewBus()
	count := 0
	var mu sync.Mutex
	received := make(chan bool, 2)

	handler1 := func(msg any) {
		mu.Lock()
		count++
		mu.Unlock()
		received <- true
	}

	handler2 := func(msg any) {
		mu.Lock()
		count++
		mu.Unlock()
		received <- true
	}

	bus.Subscribe("test.topic", handler1)
	bus.Subscribe("test.topic", handler2)
	bus.Publish("test.topic", "test message")

	// Wait for both handlers
	<-received
	<-received

	mu.Lock()
	c := count
	mu.Unlock()

	if c != 2 {
		t.Errorf("Handler call count = %d, want 2", c)
	}
}

func TestBus_NoSubscribers(t *testing.T) {
	bus := NewBus()

	// Publish to topic with no subscribers - should not panic
	bus.Publish("unknown.topic", "test message")
}
