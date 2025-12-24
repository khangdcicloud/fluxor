package core

import "sync"

// Bus is an in-process pub/sub bus.
//
// This is intentionally minimal and "fire-and-forget". For production use you may
// want backpressure, bounded queues, and error propagation.
type Bus struct {
	mu   sync.RWMutex
	subs map[string][]func(any)
}

func NewBus() *Bus {
	return &Bus{subs: make(map[string][]func(any))}
}

func (b *Bus) Subscribe(topic string, handler func(any)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[topic] = append(b.subs[topic], handler)
}

func (b *Bus) Publish(topic string, msg any) {
	b.mu.RLock()
	handlers := append([]func(any){}, b.subs[topic]...)
	b.mu.RUnlock()

	for _, h := range handlers {
		// Fire-and-forget.
		go h(msg)
	}
}
