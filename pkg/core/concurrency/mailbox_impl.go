package concurrency

import (
	"context"
	"sync"
	"sync/atomic"
)

// boundedMailbox implements Mailbox using channels internally
// Hides chan type and select statements from public API
type boundedMailbox struct {
	ch       chan interface{} // Hidden: internal channel
	mu       sync.RWMutex
	closed   int32 // Atomic flag
	capacity int
}

// NewBoundedMailbox creates a new bounded mailbox
// Hides channel creation from callers
func NewBoundedMailbox(capacity int) Mailbox {
	if capacity < 1 {
		capacity = 100 // Default capacity
	}

	return &boundedMailbox{
		ch:       make(chan interface{}, capacity), // Hidden: channel creation
		capacity: capacity,
	}
}

// Send implements Mailbox interface
// Hides channel send and select statements
func (mb *boundedMailbox) Send(msg interface{}) error {
	if atomic.LoadInt32(&mb.closed) == 1 {
		return ErrMailboxClosed
	}

	// Try to send (non-blocking for backpressure)
	select {
	case mb.ch <- msg: // Hidden: channel send
		return nil
	default:
		// Mailbox full - backpressure
		return ErrMailboxFull
	}
}

// Receive implements Mailbox interface
// Hides channel receive and select statements
func (mb *boundedMailbox) Receive(ctx context.Context) (interface{}, error) {
	if atomic.LoadInt32(&mb.closed) == 1 {
		return nil, ErrMailboxClosed
	}

	// Receive with context cancellation
	select {
	case msg, ok := <-mb.ch: // Hidden: channel receive
		if !ok {
			return nil, ErrMailboxClosed
		}
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TryReceive implements Mailbox interface
// Hides channel receive and select statements
func (mb *boundedMailbox) TryReceive() (interface{}, bool, error) {
	if atomic.LoadInt32(&mb.closed) == 1 {
		return nil, false, ErrMailboxClosed
	}

	// Try to receive (non-blocking)
	select {
	case msg, ok := <-mb.ch: // Hidden: channel receive
		if !ok {
			return nil, false, ErrMailboxClosed
		}
		return msg, true, nil
	default:
		// Mailbox empty
		return nil, false, nil
	}
}

// Close implements Mailbox interface
// Hides channel close operation
func (mb *boundedMailbox) Close() {
	if atomic.CompareAndSwapInt32(&mb.closed, 0, 1) {
		close(mb.ch) // Hidden: channel close
	}
}

// Capacity implements Mailbox interface
func (mb *boundedMailbox) Capacity() int {
	return mb.capacity
}

// Size implements Mailbox interface
func (mb *boundedMailbox) Size() int {
	return len(mb.ch) // Hidden: channel length
}

// IsClosed implements Mailbox interface
func (mb *boundedMailbox) IsClosed() bool {
	return atomic.LoadInt32(&mb.closed) == 1
}
