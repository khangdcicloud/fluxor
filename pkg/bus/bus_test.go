package bus_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/example/goflux/pkg/bus"
	"github.com/example/goflux/pkg/component"
	"github.com/example/goflux/pkg/runtime"
)

func TestLocalBus_Send(t *testing.T) {
	b := bus.NewLocalBus()
	var wg sync.WaitGroup
	wg.Add(1)

	var receivedPayload interface{}
	b.Consumer("test.topic", func(msg bus.Message) {
		receivedPayload = msg.Payload
		wg.Done()
	})

	b.Send(bus.Message{Topic: "test.topic", Payload: "hello"})

	wg.Wait()

	if receivedPayload != "hello" {
		t.Errorf("Expected payload 'hello', but got '%v'", receivedPayload)
	}
}

func TestLocalBus_Publish(t *testing.T) {
	b := bus.NewLocalBus()
	var counter int32
	var wg sync.WaitGroup
	wg.Add(2)

	b.Consumer("test.topic", func(msg bus.Message) {
		atomic.AddInt32(&counter, 1)
		wg.Done()
	})
	b.Consumer("test.topic", func(msg bus.Message) {
		atomic.AddInt32(&counter, 1)
		wg.Done()
	})

	b.Publish(bus.Message{Topic: "test.topic", Payload: "hello"})

	wg.Wait()

	if atomic.LoadInt32(&counter) != 2 {
		t.Errorf("Expected counter to be 2, but got %d", atomic.LoadInt32(&counter))
	}
}

func TestLocalBus_RequestReply(t *testing.T) {
	b := bus.NewLocalBus()
	var wg sync.WaitGroup
	wg.Add(1)

	b.Consumer("test.request", func(msg bus.Message) {
		msg.Reply("pong")
	})

	var replyPayload interface{}
	b.Request(bus.Message{Topic: "test.request", Payload: "ping"}, func(reply bus.Message) {
		replyPayload = reply.Payload
		wg.Done()
	})

	wg.Wait()

	if replyPayload != "pong" {
		t.Errorf("Expected reply 'pong', but got '%v'", replyPayload)
	}
}

func TestLocalBus_HandlerOnReactor(t *testing.T) {
	b := bus.NewLocalBus()
	rt := runtime.NewRuntime(runtime.NewRuntimeOptions{NumReactors: 1, MailboxSize: 10}, b)
	rt.Start()
	defer rt.Stop(context.Background())

	// A dummy component to get a reactor-backed bus proxy
	type testComponent struct {
		component.Base
		bus      bus.Bus
		received chan struct{}
	}

	// Start is called by the runtime and sets up the component.
	func (c *testComponent) Start(ctx context.Context, b bus.Bus) error {
		c.bus = b
		c.bus.Consumer("test.topic", func(msg bus.Message) {
			// We can't easily assert we're on the right goroutine, but we can be reasonably sure
			// the reactor is working if the message is received.
			close(c.received)
		})
		return nil
	}

	c := &testComponent{received: make(chan struct{})}

	if err := rt.Deploy(context.Background(), c); err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	b.Send(bus.Message{Topic: "test.topic"})

	<-c.received
}
