package bus

import (
	"errors"
	"sync"

	"github.com/google/uuid"
)

var (
	// ErrNoHandler is returned when a message is sent to a topic with no registered handlers.
	ErrNoHandler = errors.New("no handler for topic")
	// ErrRequestTimeout is returned when a request-reply operation times out.
	ErrRequestTimeout = errors.New("request timed out")
)

// Message represents a message on the event bus.
type Message struct {
	Topic         string
	Payload       interface{}
	CorrelationID string
	ReplyTo       string
	Headers       map[string]string

	// isReply is an internal flag to indicate if the message is a reply.
	isReply bool
}

// Reply creates a reply message for the given payload.
func (m *Message) Reply(payload interface{}) Message {
	return Message{
		Topic:         m.ReplyTo,
		Payload:       payload,
		CorrelationID: m.CorrelationID,
		Headers:       m.Headers,
		isReply:       true,
	}
}

// Handler is a function that processes messages.
type Handler func(msg Message)

// Bus is the interface for the event bus.
type Bus interface {
	// Publish sends a message to all subscribers of a topic.
	Publish(msg Message)

	// Send sends a message to a single subscriber of a topic (point-to-point).
	Send(msg Message) error

	// Request sends a request and invokes the replyHandler when a reply is received.
	Request(msg Message, replyHandler Handler)

	// Consumer registers a handler for a topic.
	Consumer(topic string, handler Handler)
}

// localBus is an in-process event bus.
type localBus struct {
	mu           sync.RWMutex
	handlers     map[string][]Handler
	replyHandlers map[string]Handler // a map of correlation IDs to reply handlers
}

// NewBus creates a new in-process event bus.
func NewBus(buffer int) Bus {
	return &localBus{
		handlers:     make(map[string][]Handler),
		replyHandlers: make(map[string]Handler),
	}
}

// handle routes a message to the appropriate handler(s).
func (b *localBus) handle(msg Message) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// If it's a reply, find the specific reply handler.
	if msg.isReply {
		if handler, ok := b.replyHandlers[msg.CorrelationID]; ok {
			// Execute the handler and remove it.
			// This is happening in the sender's reactor, which is what we want.
			go func() {
				handler(msg)
				b.mu.Lock()
				delete(b.replyHandlers, msg.CorrelationID)
				b.mu.Unlock()
			}()
			return nil
		}
		return nil // No-op if no reply handler is found (e.g., timeout already occurred)
	}

	// If it's a regular message, find the topic handlers.
	if handlers, ok := b.handlers[msg.Topic]; ok && len(handlers) > 0 {
		// For Send, deliver to one. For Publish, deliver to all.
		// (This simple implementation delivers to the first for Send)
		for _, handler := range handlers {
			handler(msg)
		}
		return nil
	}

	return ErrNoHandler
}

func (b *localBus) Publish(msg Message) {
	b.handle(msg)
}

func (b *localBus) Send(msg Message) error {
	return b.handle(msg)
}

func (b *localBus) Request(msg Message, replyHandler Handler) {
	// Generate a unique correlation ID.
	msg.CorrelationID = uuid.NewString()
	// The reply topic is the same correlation ID for simplicity.
	msg.ReplyTo = msg.CorrelationID

	b.mu.Lock()
	b.replyHandlers[msg.CorrelationID] = replyHandler
	b.mu.Unlock()

	// The message is sent like any other message.
	b.handle(msg)

	// TODO: Handle timeouts. A real implementation would need a mechanism
	// to clean up the reply handler if a reply is not received within a certain time.
}

func (b *localBus) Consumer(topic string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
}
