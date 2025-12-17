package reactor

import (
	"context"
	"errors"
	"time"
)

var (
	ErrBackpressure = errors.New("reactor mailbox is full")
)

// Event represents a task to be executed by the Reactor.
type Event func()

// Reactor is an event loop that processes events sequentially in a single goroutine.
type Reactor struct {
	mailbox chan Event
	stop    chan struct{}
}

// NewReactor creates a new Reactor with a given mailbox size.
func NewReactor(mailboxSize int) *Reactor {
	return &Reactor{
		mailbox: make(chan Event, mailboxSize),
		stop:    make(chan struct{}),
	}
}

// Start begins the reactor's event loop.
func (r *Reactor) Start() {
	go r.run()
}

// Stop gracefully stops the reactor. It waits for all pending events to be processed.
func (r *Reactor) Stop(ctx context.Context) {
	// Send a stop signal which will be processed after all other events.
	r.Submit(func() {
		close(r.stop)
	})

	// Wait for the stop signal to be processed or for the context to be done.
	select {
	case <-r.stop:
		// reactor has stopped
	case <-ctx.Done():
		// timeout waiting for reactor to stop
	}
}

// run is the reactor's event loop.
func (r *Reactor) run() {
	for {
		select {
		case event := <-r.mailbox:
			event()
		case <-r.stop:
			return
		}
	}
}

// Submit sends an event to the reactor's mailbox for processing.
// It returns ErrBackpressure if the mailbox is full.
func (r *Reactor) Submit(event Event) error {
	select {
	case r.mailbox <- event:
		return nil
	default:
		return ErrBackpressure
	}
}

// Schedule submits an event to be executed after a specified duration.
func (r *Reactor) Schedule(event Event, delay time.Duration) *time.Timer {
	return time.AfterFunc(delay, func() {
		// It's possible for this to be called after the reactor is stopped.
		// We will try to submit, but ignore backpressure errors if the reactor is shutting down.
		_ = r.Submit(event)
	})
}
