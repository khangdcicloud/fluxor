package core

import "sync"

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
	defer b.mu.RUnlock()
	if handlers, ok := b.subs[topic]; ok {
		for _, h := range handlers {
			// Fire and forget (trong thực tế có thể wrap goroutine)
			go h(msg)
		}
	}
}
