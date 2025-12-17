package bus_test

import (
	"context"
	"testing"
	"time"

	"github.com/fluxor-io/fluxor/pkg/bus"
	"github.com/fluxor-io/fluxor/pkg/types"
)

func TestBus(t *testing.T) {
	b := bus.NewBus(10)

	topic := "test-topic"
	msg := types.NewMessage(topic, "hello")

	handler1 := make(types.Mailbox, 1)
	handler2 := make(types.Mailbox, 1)

	if err := b.Subscribe(topic, handler1); err != nil {
		t.Fatalf("failed to subscribe handler1: %v", err)
	}

	if err := b.Subscribe(topic, handler2); err != nil {
		t.Fatalf("failed to subscribe handler2: %v", err)
	}

	b.Publish(topic, msg)

	// Check if both handlers received the message
	for i := 0; i < 2; i++ {
		select {
		case receivedMsg := <-handler1:
			if receivedMsg.ID != msg.ID {
				t.Errorf("handler1 received wrong message ID: got %s, want %s", receivedMsg.ID, msg.ID)
			}
		case receivedMsg := <-handler2:
			if receivedMsg.ID != msg.ID {
				t.Errorf("handler2 received wrong message ID: got %s, want %s", receivedMsg.ID, msg.ID)
			}
		default:
			t.Fatal("expected a message but got none")
		}
	}

	if err := b.Unsubscribe(topic, handler1); err != nil {
		t.Fatalf("failed to unsubscribe handler1: %v", err)
	}

	b.Publish(topic, msg)

	// Check if only handler2 received the message
	select {
	case <-handler1:
		t.Fatal("handler1 received message after unsubscribe")
	case receivedMsg := <-handler2:
		if receivedMsg.ID != msg.ID {
			t.Errorf("handler2 received wrong message ID: got %s, want %s", receivedMsg.ID, msg.ID)
		}
	default:
		t.Fatal("expected a message but got none")
	}
}

func TestRequest(t *testing.T) {
	b := bus.NewBus(10)

	topic := "test-request-topic"
	requestMsg := types.NewMessage(topic, "request")

	// Create a handler that will reply to the request
	handler := make(types.Mailbox, 1)
	if err := b.Subscribe(topic, handler); err != nil {
		t.Fatalf("failed to subscribe handler: %v", err)
	}

	go func() {
		for msg := range handler {
			replyMsg := types.NewMessage(msg.ReplyTo, "response")
			replyMsg.CorrelationID = msg.ID
			b.Publish(msg.ReplyTo, replyMsg)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	reply, err := b.Request(ctx, topic, requestMsg)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}

	if reply.CorrelationID != requestMsg.ID {
		t.Errorf("unexpected correlation ID. got %q, want %q", reply.CorrelationID, requestMsg.ID)
	}

	if reply.Payload != "response" {
		t.Errorf("unexpected payload. got %q, want %q", reply.Payload, "response")
	}
}
