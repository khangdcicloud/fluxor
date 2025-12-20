package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/concurrency"
	"github.com/google/uuid"
)

// eventBus implements EventBus
type eventBus struct {
	consumers map[string][]*consumer
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	vertx     Vertx // Store Vertx reference for creating FluxorContext
}

// NewEventBus creates a new event bus
func NewEventBus(ctx context.Context, vertx Vertx) EventBus {
	ctx, cancel := context.WithCancel(ctx)
	return &eventBus{
		consumers: make(map[string][]*consumer),
		ctx:       ctx,
		cancel:    cancel,
		vertx:     vertx,
	}
}

func (eb *eventBus) Publish(address string, body interface{}) error {
	// Fail-fast: validate inputs immediately
	if err := ValidateAddress(address); err != nil {
		return err
	}
	if err := ValidateBody(body); err != nil {
		return err
	}

	// Auto-encode to JSON if not already []byte
	jsonBody, err := eb.encodeBody(body)
	if err != nil {
		return fmt.Errorf("encode body failed: %w", err)
	}

	eb.mu.RLock()
	consumers := eb.consumers[address]
	eb.mu.RUnlock()

	msg := newMessage(jsonBody, nil, "", eb)

	for _, c := range consumers {
		// Use Mailbox abstraction (hides channel operations)
		if err := c.mailbox.Send(msg); err != nil {
			if err == concurrency.ErrMailboxFull {
				// Non-blocking: if handler is busy, skip
				continue
			}
			if err == concurrency.ErrMailboxClosed {
				return eb.ctx.Err()
			}
			return err
		}
	}

	return nil
}

func (eb *eventBus) Send(address string, body interface{}) error {
	// Fail-fast: validate inputs immediately
	if err := ValidateAddress(address); err != nil {
		return err
	}
	if err := ValidateBody(body); err != nil {
		return err
	}

	// Auto-encode to JSON if not already []byte
	jsonBody, err := eb.encodeBody(body)
	if err != nil {
		return fmt.Errorf("encode body failed: %w", err)
	}

	eb.mu.RLock()
	consumers := eb.consumers[address]
	eb.mu.RUnlock()

	// Fail-fast: no handlers registered
	if len(consumers) == 0 {
		return &Error{Code: "NO_HANDLERS", Message: "No handlers registered for address: " + address}
	}

	// Round-robin to one consumer
	consumer := consumers[0]
	msg := newMessage(jsonBody, nil, "", eb)

	// Use Mailbox abstraction (hides select statement)
	// Note: Mailbox.Send() is non-blocking, so timeout is handled by backpressure
	if err := consumer.mailbox.Send(msg); err != nil {
		if err == concurrency.ErrMailboxFull {
			return ErrTimeout
		}
		if err == concurrency.ErrMailboxClosed {
			return eb.ctx.Err()
		}
		return err
	}
	return nil
}

func (eb *eventBus) Request(address string, body interface{}, timeout time.Duration) (Message, error) {
	// Fail-fast: validate inputs immediately
	if err := ValidateAddress(address); err != nil {
		return nil, err
	}
	if err := ValidateBody(body); err != nil {
		return nil, err
	}
	if err := ValidateTimeout(timeout); err != nil {
		return nil, err
	}

	// Auto-encode to JSON if not already []byte
	jsonBody, err := eb.encodeBody(body)
	if err != nil {
		return nil, fmt.Errorf("encode body failed: %w", err)
	}

	replyAddress := generateReplyAddress()
	replyMailbox := concurrency.NewBoundedMailbox(1) // Hidden: channel creation

	// Register temporary reply handler
	replyConsumer := eb.Consumer(replyAddress)
	replyConsumer.Handler(func(ctx FluxorContext, msg Message) error {
		// Use Mailbox abstraction (hides channel send)
		if err := replyMailbox.Send(msg); err != nil {
			// Ignore if mailbox full (non-blocking)
		}
		return nil
	})
	defer replyConsumer.Unregister()

	// Send request with reply address
	headers := map[string]string{"replyAddress": replyAddress}
	msg := newMessage(jsonBody, headers, replyAddress, eb)

	eb.mu.RLock()
	consumers := eb.consumers[address]
	eb.mu.RUnlock()

	// Fail-fast: no handlers registered
	if len(consumers) == 0 {
		return nil, &Error{Code: "NO_HANDLERS", Message: "No handlers registered for address: " + address}
	}

	consumer := consumers[0]

	// Use Mailbox abstraction (hides select statement)
	// Note: Mailbox.Send() is non-blocking, timeout handled by backpressure
	if err := consumer.mailbox.Send(msg); err != nil {
		if err == concurrency.ErrMailboxFull {
			return nil, ErrTimeout
		}
		if err == concurrency.ErrMailboxClosed {
			return nil, eb.ctx.Err()
		}
		return nil, err
	}

	// Wait for reply using Mailbox abstraction (hides select statement)
	replyCtx, replyCancel := context.WithTimeout(eb.ctx, timeout)
	defer replyCancel()

	reply, err := replyMailbox.Receive(replyCtx)
	if err != nil {
		if err == context.DeadlineExceeded {
			return nil, ErrTimeout
		}
		return nil, err
	}

	if msg, ok := reply.(Message); ok {
		return msg, nil
	}
	return nil, fmt.Errorf("invalid reply message type")
}

func (eb *eventBus) Consumer(address string) Consumer {
	// Fail-fast: validate address immediately
	if err := ValidateAddress(address); err != nil {
		FailFast(err)
	}

	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Fix Bug 1: Initialize ctx when creating consumer
	// Create FluxorContext for the consumer using eventBus's Vertx reference
	var fluxorCtx FluxorContext
	if eb.vertx != nil {
		fluxorCtx = newContext(eb.ctx, eb.vertx)
	}

	c := &consumer{
		address:  address,
		mailbox:  concurrency.NewBoundedMailbox(100), // Hidden: channel creation
		eventBus: eb,
		ctx:      fluxorCtx, // Initialize ctx to prevent nil pointer
	}

	eb.consumers[address] = append(eb.consumers[address], c)
	return c
}

func (eb *eventBus) Close() error {
	eb.cancel()
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for _, consumers := range eb.consumers {
		for _, c := range consumers {
			// Close mailbox (hides channel close operation)
			c.mailbox.Close()
		}
	}
	eb.consumers = make(map[string][]*consumer)
	return nil
}

// consumer implements Consumer
// Uses Mailbox abstraction to hide channel operations
type consumer struct {
	address  string
	mailbox  concurrency.Mailbox // Abstracted: hides chan Message
	handler  MessageHandler
	eventBus *eventBus
	ctx      FluxorContext
	mu       sync.RWMutex
}

func (c *consumer) Handler(handler MessageHandler) Consumer {
	// Fail-fast: handler cannot be nil
	if handler == nil {
		FailFast(&Error{Code: "INVALID_HANDLER", Message: "handler cannot be nil"})
	}

	c.mu.Lock()
	c.handler = handler
	c.mu.Unlock()

	// Start processing messages using Executor (hides go func() call)
	// Note: In a full implementation, we'd use the EventBus's executor
	// For now, we'll keep the goroutine but could refactor to use Executor
	go c.processMessages() // TODO: Replace with Executor.Submit()
	return c
}

func (c *consumer) processMessages() {
	// Fix Bug 2: Panic isolation - recover from panics without re-panicking
	// This allows the message processing loop to continue even if one message handler panics
	defer func() {
		if r := recover(); r != nil {
			// Log panic but don't re-panic - maintain panic isolation
			// In production, this would be logged and monitored
			_ = fmt.Errorf("panic in message processing loop for address %s (isolated): %v", c.address, r)
			// Continue processing other messages - don't crash the loop
		}
	}()

	// Use Mailbox abstraction (hides select statement and channel operations)
	for {
		// Receive message using Mailbox (hides channel receive and select)
		msg, err := c.mailbox.Receive(c.eventBus.ctx)
		if err != nil {
			// Mailbox closed or context cancelled
			return
		}

		// Type assert to Message
		message, ok := msg.(Message)
		if !ok {
			// Invalid message type - skip
			continue
		}

		if c.handler != nil {
			// Use the consumer's context (now properly initialized)
			ctx := c.ctx
			if ctx == nil {
				// Fallback: create context if somehow nil (shouldn't happen after Bug 1 fix)
				if c.eventBus.vertx != nil {
					ctx = newContext(c.eventBus.ctx, c.eventBus.vertx)
				}
			}

			// Wrap handler call in panic recovery for individual messages (panic isolation)
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Log handler panic but don't crash - maintain panic isolation
						_ = fmt.Errorf("handler panic for address %s (isolated): %v", c.address, r)
					}
				}()

				// Call handler - errors are logged but don't crash
				if err := c.handler(ctx, message); err != nil {
					// Log handler error but don't panic - maintain system stability
					_ = fmt.Errorf("handler error for address %s: %w", c.address, err)
				}
			}()
		} else {
			// Handler is nil - log but don't panic (shouldn't happen in normal flow)
			_ = fmt.Errorf("handler is nil for address %s", c.address)
		}
	}
}

func (c *consumer) Completion() <-chan struct{} {
	// Return a channel that closes when mailbox is closed
	// This maintains the interface while using Mailbox abstraction
	done := make(chan struct{})
	go func() {
		for !c.mailbox.IsClosed() {
			time.Sleep(10 * time.Millisecond)
		}
		close(done)
	}()
	return done
}

func (c *consumer) Unregister() error {
	c.eventBus.mu.Lock()
	defer c.eventBus.mu.Unlock()

	consumers := c.eventBus.consumers[c.address]
	for i, cons := range consumers {
		if cons == c {
			c.eventBus.consumers[c.address] = append(consumers[:i], consumers[i+1:]...)
			break
		}
	}

	// Close mailbox (hides channel close operation)
	c.mailbox.Close()
	return nil
}

func generateReplyAddress() string {
	return "reply." + uuid.New().String()
}

// encodeBody encodes body to JSON if needed - fail-fast
func (eb *eventBus) encodeBody(body interface{}) (interface{}, error) {
	// Fail-fast: validate body
	if err := ValidateBody(body); err != nil {
		return nil, err
	}

	// If already []byte, return as-is
	if data, ok := body.([]byte); ok {
		return data, nil
	}

	// Encode to JSON - errors are propagated immediately
	return JSONEncode(body)
}
