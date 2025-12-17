package types

import "github.com/google/uuid"

// Message represents a message passed on the bus.
type Message struct {
	ID            string
	CorrelationID string
	Topic         string
	ReplyTo       string
	Payload       interface{}
}

// NewMessage creates a new message with a unique ID.
func NewMessage(topic string, payload interface{}) Message {
	return Message{
		ID:      uuid.New().String(),
		Topic:   topic,
		Payload: payload,
		ReplyTo: "",
	}
}

// Mailbox is a channel for receiving messages.
type Mailbox chan Message
