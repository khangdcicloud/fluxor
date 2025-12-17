package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/goflux/pkg/bus"
	"github.com/example/goflux/pkg/component"
	"github.com/example/goflux/pkg/httpx"
	"github.com/example/goflux/pkg/runtime"
)

// GreeterComponent is a simple component that greets people.
type GreeterComponent struct{}

func (c *GreeterComponent) Start(ctx context.Context, b bus.Bus) error {
	log.Println("GreeterComponent starting")
	b.Consumer("greeter.hello", func(msg bus.Message) {
		name := msg.Payload.(string)
		log.Printf("GreeterComponent received request: %s\n", name)
		msg.Reply(fmt.Sprintf("Hello, %s!", name))
	})
	return nil
}

func (c *GreeterComponent) Stop(ctx context.Context) error {
	log.Println("GreeterComponent stopping")
	return nil
}

// BlockingComponent is a component that performs a blocking operation.
type BlockingComponent struct{}

func (c *BlockingComponent) Start(ctx context.Context, b bus.Bus) error {
	log.Println("BlockingComponent starting")
	b.Consumer("blocking.long_operation", func(msg bus.Message) {
		log.Println("BlockingComponent received request")
		// Simulate a long-running blocking operation.
		time.Sleep(2 * time.Second)
		msg.Reply("Done with long operation!")
	})
	return nil
}

func (c *BlockingComponent) Stop(ctx context.Context) error {
	log.Println("BlockingComponent stopping")
	return nil
}

func main() {
	// Create a new event bus.
	b := bus.NewLocalBus()

	// Create a new runtime.
	rt := runtime.NewRuntime(runtime.NewRuntimeOptions{
		NumReactors: 4,       // 4 CPU-bound reactors
		MailboxSize: 1024,
		NumWorkers:  128,     // A large pool for blocking I/O
		QueueSize:   65536,
	}, b)

	// Start the runtime.
	rt.Start()
	defer rt.Stop(context.Background())

	// Deploy the components.
	if err := rt.Deploy(context.Background(), &GreeterComponent{}); err != nil {
		log.Fatalf("Failed to deploy GreeterComponent: %v", err)
	}
	if err := rt.Deploy(context.Background(), &BlockingComponent{}); err != nil {
		log.Fatalf("Failed to deploy BlockingComponent: %v", err)
	}

	// Create a new HTTP server.
	httpServer := httpx.NewServer(":8080", b)

	// Register the HTTP handlers.
	httpServer.Handle("/greet", func(b bus.Bus, w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "World"
		}

		b.Request(bus.Message{Topic: "greeter.hello", Payload: name}, func(reply bus.Message) {
			fmt.Fprintln(w, reply.Payload)
		})
	})

	httpServer.Handle("/blocking", func(b bus.Bus, w http.ResponseWriter, r *http.Request) {
		rt.ExecuteBlocking(func() {
			b.Request(bus.Message{Topic: "blocking.long_operation"}, func(reply bus.Message) {
				fmt.Fprintln(w, reply.Payload)
			})
		})
	})

	// Start the HTTP server.
	go func() {
		if err := httpServer.Start(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Wait for a shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown the HTTP server.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Stop(ctx); err != nil {
		log.Fatalf("HTTP server shutdown failed: %v", err)
	}

	log.Println("Server gracefully shut down")
}
