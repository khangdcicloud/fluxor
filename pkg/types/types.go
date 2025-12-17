package types

import "context"

// Message represents a message passed through the event bus.
type Message struct {
	ReplyTo string
	Payload interface{}
}

// Mailbox is a channel for receiving messages.
type Mailbox chan Message

// Component is the interface for all components.
type Component interface {
	Name() string
	OnStart(ctx context.Context, bus Bus) error
	OnStop(ctx context.Context) error
}

// Bus is the interface for the event bus.
type Bus interface {
	Publish(topic string, msg Message)
	Subscribe(topic string, handler Mailbox) error
	Unsubscribe(topic string, handler Mailbox) error

	Send(topic string, msg Message) error
	Request(ctx context.Context, topic string, msg Message) (Message, error)
}
