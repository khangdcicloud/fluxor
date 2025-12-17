package bus

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/fluxor-io/fluxor/pkg/types"
	"github.com/google/uuid"
)

type Bus interface {
	Publish(topic string, msg types.Message)
	Subscribe(topic string, handler types.Mailbox) error
	Unsubscribe(topic string, handler types.Mailbox) error

	Send(topic string, msg types.Message) error
	Request(ctx context.Context, topic string, msg types.Message) (types.Message, error)
}

type localBus struct {
	subscribers map[string][]types.Mailbox
	mu          sync.RWMutex
	capacity    int
}

func NewBus(capacity int) Bus {
	return &localBus{
		subscribers: make(map[string][]types.Mailbox),
		capacity:    capacity,
	}
}

func (b *localBus) Publish(topic string, msg types.Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subscribers, ok := b.subscribers[topic]; ok {
		for _, sub := range subscribers {
			// Non-blocking send
			select {
			case sub <- msg:
			default:
				log.Printf("Failed to publish message to topic %s: subscriber channel is full", topic)
			}
		}
	}
}

func (b *localBus) Subscribe(topic string, handler types.Mailbox) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[topic] = append(b.subscribers[topic], handler)
	return nil
}

func (b *localBus) Unsubscribe(topic string, handler types.Mailbox) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if subscribers, ok := b.subscribers[topic]; ok {
		for i, sub := range subscribers {
			if sub == handler {
				b.subscribers[topic] = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (b *localBus) Send(topic string, msg types.Message) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if subscribers, ok := b.subscribers[topic]; ok {
		for _, sub := range subscribers {
			sub <- msg
		}
		return nil
	}
	return errors.New("no subscribers for topic: " + topic)
}

func (b *localBus) Request(ctx context.Context, topic string, msg types.Message) (types.Message, error) {
	replyTopic := newReplyTopic()
	msg.ReplyTo = replyTopic

	replyMailbox := make(types.Mailbox, 1)
	if err := b.Subscribe(replyTopic, replyMailbox); err != nil {
		return types.Message{}, err
	}
	defer b.Unsubscribe(replyTopic, replyMailbox)

	if err := b.Send(topic, msg); err != nil {
		return types.Message{}, err
	}

	select {
	case reply := <-replyMailbox:
		return reply, nil
	case <-ctx.Done():
		return types.Message{}, ctx.Err()
	}
}

func newReplyTopic() string {
	return "reply." + uuid.New().String()
}
