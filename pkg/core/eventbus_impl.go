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
	vertx     Vertx                // Store Vertx reference for creating FluxorContext
	executor  concurrency.Executor // Executor for processing messages (hides goroutines)
	logger    Logger               // Logger for error and debug messages
}

// NewEventBus creates a new event bus
func NewEventBus(ctx context.Context, vertx Vertx) EventBus {
	ctx, cancel := context.WithCancel(ctx)

	// Create logger
	logger := NewDefaultLogger()

	// Create Executor for message processing
	// Default config: 10 workers, 1000 queue size
	executorConfig := concurrency.DefaultExecutorConfig()
	executorConfig.Workers = 10
	executorConfig.QueueSize = 1000
	executor := concurrency.NewExecutor(ctx, executorConfig)

	return &eventBus{
		consumers: make(map[string][]*consumer),
		ctx:       ctx,
		cancel:    cancel,
		vertx:     vertx,
		executor:  executor,
		logger:    logger,
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

	// Extract request ID from context if available
	headers := make(map[string]string)
	if requestID := GetRequestID(eb.ctx); requestID != "" {
		headers["X-Request-ID"] = requestID
	}
	msg := newMessage(jsonBody, headers, "", eb)

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

	// Extract request ID from context if available
	headers := make(map[string]string)
	if requestID := GetRequestID(eb.ctx); requestID != "" {
		headers["X-Request-ID"] = requestID
	}
	msg := newMessage(jsonBody, headers, "", eb)

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
	// Extract request ID from context if available
	if requestID := GetRequestID(eb.ctx); requestID != "" {
		headers["X-Request-ID"] = requestID
	}
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

	// Initialize ctx when creating consumer
	// Create FluxorContext for the consumer using eventBus's Vertx reference
	var fluxorCtx FluxorContext
	if eb.vertx != nil {
		fluxorCtx = newContext(eb.ctx, eb.vertx)
	}

	c := &consumer{
		address:  address,
		mailbox:  concurrency.NewBoundedMailbox(100), // Hidden: channel creation
		eventBus: eb,
		ctx:      fluxorCtx,           // Initialize ctx to prevent nil pointer
		done:     make(chan struct{}), // Channel for Completion() notification (closed when mailbox processing stops)
	}

	eb.consumers[address] = append(eb.consumers[address], c)
	return c
}

func (eb *eventBus) Close() error {
	eb.cancel()

	// Shutdown executor gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := eb.executor.Shutdown(shutdownCtx); err != nil {
		eb.logger.Warnf("EventBus executor shutdown timeout: %v", err)
	}

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
	done     chan struct{} // Channel for Completion() notification (closed when mailbox closes)
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
	task := concurrency.NewNamedTask(
		fmt.Sprintf("eventbus-consumer-%s", c.address),
		func(ctx context.Context) error {
			return c.processMessages(ctx)
		},
	)
	if err := c.eventBus.executor.Submit(task); err != nil {
		c.eventBus.logger.Errorf("Failed to submit consumer task for address %s: %v", c.address, err)
		// Close done channel since processing won't start
		close(c.done)
		// Still return consumer - handler can be set later
	}
	return c
}

func (c *consumer) processMessages(ctx context.Context) error {
	// Panic isolation - recover from panics without re-panicking
	// This allows the message processing loop to continue even if one message handler panics
	defer func() {
		if r := recover(); r != nil {
			// Log panic but don't re-panic - maintain panic isolation
			c.eventBus.logger.Errorf("panic in message processing loop for address %s (isolated): %v", c.address, r)
		}
		// Close done channel to notify Completion() when mailbox processing stops
		close(c.done)
	}()

	// Use Mailbox abstraction (hides select statement and channel operations)
	for {
		// Receive message using Mailbox (hides channel receive and select)
		msg, err := c.mailbox.Receive(ctx)
		if err != nil {
			// Mailbox closed or context cancelled
			return err
		}

		// Type assert to Message
		message, ok := msg.(Message)
		if !ok {
			// Invalid message type - skip
			c.eventBus.logger.Warnf("Invalid message type received for address %s", c.address)
			continue
		}

		if c.handler != nil {
			// Use the consumer's context (now properly initialized)
			fluxorCtx := c.ctx
			if fluxorCtx == nil {
				// Fallback: create context if somehow nil (shouldn't happen after Bug 1 fix)
				if c.eventBus.vertx != nil {
					fluxorCtx = newContext(c.eventBus.ctx, c.eventBus.vertx)
				}
			}

			// Wrap handler call in panic recovery for individual messages (panic isolation)
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Log handler panic but don't crash - maintain panic isolation
						c.eventBus.logger.Errorf("handler panic for address %s (isolated): %v", c.address, r)
					}
				}()

				// Call handler - errors are logged but don't crash
				if err := c.handler(fluxorCtx, message); err != nil {
					// Log handler error but don't panic - maintain system stability
					// Try to extract request ID from message headers for better tracing
					requestID := ""
					if headers := message.Headers(); headers != nil {
						if id, ok := headers["X-Request-ID"]; ok {
							requestID = id
						}
					}
					if requestID != "" {
						c.eventBus.logger.Errorf("handler error for address %s (request_id=%s): %v", c.address, requestID, err)
					} else {
						c.eventBus.logger.Errorf("handler error for address %s: %v", c.address, err)
					}
				}
			}()
		} else {
			// Handler is nil - log but don't panic (shouldn't happen in normal flow)
			c.eventBus.logger.Warnf("handler is nil for address %s", c.address)
		}
	}
}

func (c *consumer) Completion() <-chan struct{} {
	// Return the done channel that will be closed when mailbox processing stops
	// This is efficient - no polling, just channel notification
	return c.done
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
