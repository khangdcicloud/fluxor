package core

import (
	"context"
	"testing"
	"time"
)

func TestEventBus_Publish(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	eb := vertx.EventBus()
	defer eb.Close()

	// Test fail-fast: empty address
	err := eb.Publish("", "test")
	if err == nil {
		t.Error("Publish() with empty address should fail")
	}

	// Test fail-fast: nil body
	err = eb.Publish("test.address", nil)
	if err == nil {
		t.Error("Publish() with nil body should fail")
	}

	// Test valid publish
	err = eb.Publish("test.address", "test message")
	if err != nil {
		t.Errorf("Publish() error = %v", err)
	}
}

func TestEventBus_Send(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	eb := vertx.EventBus()
	defer eb.Close()

	// Test fail-fast: empty address
	err := eb.Send("", "test")
	if err == nil {
		t.Error("Send() with empty address should fail")
	}

	// Test fail-fast: nil body
	err = eb.Send("test.address", nil)
	if err == nil {
		t.Error("Send() with nil body should fail")
	}

	// Test fail-fast: no handlers
	err = eb.Send("test.address", "test")
	if err == nil {
		t.Error("Send() with no handlers should fail")
	}
	if ce, ok := err.(*Error); ok {
		if ce.Code != "NO_HANDLERS" {
			t.Fatalf("Send() error code = %q, want %q", ce.Code, "NO_HANDLERS")
		}
	}

	// Register handler
	consumer := eb.Consumer("test.address")
	received := make(chan bool, 1)
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		received <- true
		return nil
	})

	// Test valid send
	err = eb.Send("test.address", "test message")
	if err != nil {
		t.Errorf("Send() error = %v", err)
	}

	// Wait for message
	select {
	case <-received:
	case <-time.After(1 * time.Second):
		t.Error("Message not received")
	}
}

func TestEventBus_Request(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	eb := vertx.EventBus()
	defer eb.Close()

	// Test fail-fast: invalid timeout
	_, err := eb.Request("test.address", "test", 0)
	if err == nil {
		t.Error("Request() with zero timeout should fail")
	}

	// Test fail-fast: empty address
	_, err = eb.Request("", "test", 1*time.Second)
	if err == nil {
		t.Error("Request() with empty address should fail")
	}

	// Test fail-fast: nil body
	_, err = eb.Request("test.address", nil, 1*time.Second)
	if err == nil {
		t.Error("Request() with nil body should fail")
	}

	// Test fail-fast: no handlers
	_, err = eb.Request("no.handlers", "test", 1*time.Second)
	if err == nil {
		t.Error("Request() with no handlers should fail")
	}
	if ce, ok := err.(*Error); ok {
		if ce.Code != "NO_HANDLERS" {
			t.Fatalf("Request() error code = %q, want %q", ce.Code, "NO_HANDLERS")
		}
	}

	// Register handler
	consumer := eb.Consumer("test.address")
	consumer.Handler(func(ctx FluxorContext, msg Message) error {
		return msg.Reply("reply")
	})

	// Test valid request
	msg, err := eb.Request("test.address", "test", 1*time.Second)
	if err != nil {
		t.Errorf("Request() error = %v", err)
	}
	if msg == nil {
		t.Error("Request() returned nil message")
	}
}

func TestEventBus_Consumer(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	eb := vertx.EventBus()
	defer eb.Close()

	// Test fail-fast: empty address should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Consumer() with empty address should panic")
		}
	}()

	eb.Consumer("")
}

func TestConsumer_Handler_FailFast_NilHandlerPanics(t *testing.T) {
	ctx := context.Background()
	vertx := NewVertx(ctx)
	eb := vertx.EventBus()
	defer eb.Close()

	c := eb.Consumer("test.address")
	defer func() {
		if r := recover(); r == nil {
			t.Error("Handler(nil) should panic (fail-fast)")
		}
	}()
	c.Handler(nil)
}
