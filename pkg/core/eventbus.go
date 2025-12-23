package core

import (
	"fmt"
	"sync"
	"time"
)

// Message represents a message on the event bus
type Message interface {
	// Body returns the message body
	Body() interface{}

	// Headers returns the message headers
	Headers() map[string]string

	// ReplyAddress returns the reply address if this is a request message
	ReplyAddress() string

	// Reply sends a reply to this message
	Reply(body interface{}) error

	// DecodeBody decodes the message body into v
	DecodeBody(v interface{}) error

	// Fail indicates that processing failed
	Fail(failureCode int, message string) error
}

// message implements Message
type message struct {
	body         interface{}
	headers      map[string]string
	replyAddress string
	eventBus     EventBus
	mu           sync.RWMutex
}

func newMessage(body interface{}, headers map[string]string, replyAddress string, eventBus EventBus) Message {
	if headers == nil {
		headers = make(map[string]string)
	}
	return &message{
		body:         body,
		headers:      headers,
		replyAddress: replyAddress,
		eventBus:     eventBus,
	}
}

func (m *message) Body() interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.body
}

func (m *message) Headers() map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make(map[string]string)
	for k, v := range m.headers {
		result[k] = v
	}
	return result
}

func (m *message) ReplyAddress() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.replyAddress
}

func (m *message) Reply(body interface{}) error {
	if m.replyAddress == "" {
		return ErrNoReplyAddress
	}
	return m.eventBus.Send(m.replyAddress, body)
}

func (m *message) DecodeBody(v interface{}) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if data, ok := m.body.([]byte); ok {
		return JSONDecode(data, v)
	}
	return fmt.Errorf("body is not []byte, got %T", m.body)
}

func (m *message) Fail(failureCode int, message string) error {
	// In a real implementation, this would send a failure response
	return m.Reply(map[string]interface{}{
		"failureCode": failureCode,
		"message":     message,
	})
}

// EventBus provides publish-subscribe and point-to-point messaging
// Default data format is JSON
type EventBus interface {
	// Publish publishes a message to all handlers registered for the address
	// Body is automatically JSON encoded if not already []byte
	Publish(address string, body interface{}) error

	// Send sends a point-to-point message
	// Body is automatically JSON encoded if not already []byte
	Send(address string, body interface{}) error

	// Request sends a message and expects a reply
	// Body is automatically JSON encoded if not already []byte
	Request(address string, body interface{}, timeout time.Duration) (Message, error)

	// Consumer creates a consumer for the given address
	Consumer(address string) Consumer

	// Close closes the event bus
	Close() error
}

// Consumer represents a message consumer
type Consumer interface {
	// Handler sets the message handler
	Handler(handler MessageHandler) Consumer

	// Completion returns a channel that will be closed when the consumer is closed
	Completion() <-chan struct{}

	// Unregister unregisters the consumer
	Unregister() error
}

// MessageHandler handles incoming messages
type MessageHandler func(ctx FluxorContext, msg Message) error

// Errors
var (
	ErrNoReplyAddress = &Error{Code: "NO_REPLY_ADDRESS", Message: "No reply address available"}
	ErrTimeout        = &Error{Code: "TIMEOUT", Message: "Request timeout"}
)

// Error represents an event bus error
type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}
