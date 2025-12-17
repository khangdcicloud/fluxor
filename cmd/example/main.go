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

	"github.com/fluxor-io/fluxor/pkg/bus"
	"github.com/fluxor-io/fluxor/pkg/component"
	"github.com/fluxor-io/fluxor/pkg/httpx"
	"github.com/fluxor-io/fluxor/pkg/runtime"
)

// GreeterComponent is a simple component that greets people.
type GreeterComponent struct{
	component.Base
}

func (c *GreeterComponent) Name() string {
	return "greeter"
}

func (c *GreeterComponent) OnStart(ctx context.Context, b bus.Bus) error {
	log.Println("GreeterComponent starting")
	b.Subscribe("greeter.hello", func(msg bus.Message) {
		name := msg.Payload.(string)
		log.Printf("GreeterComponent received request: %s\n", name)
		msg.Reply(fmt.Sprintf("Hello, %s!", name))
	})
	return nil
}

// BlockingComponent is a component that performs a blocking operation.
type BlockingComponent struct{
	component.Base
}

func (c *BlockingComponent) Name() string {
	return "blocking"
}

func (c *BlockingComponent) OnStart(ctx context.Context, b bus.Bus) error {
	log.Println("BlockingComponent starting")
	b.Subscribe("blocking.long_operation", func(msg bus.Message) {
		log.Println("BlockingComponent received request")
		// Simulate a long-running blocking operation.
		time.Sleep(2 * time.Second)
		msg.Reply("Done with long operation!")
	})
	return nil
}

func main() {
	// Create a new event bus.
	b := bus.NewBus(1024)

	// Create a new runtime.
	rt := runtime.NewRuntime(runtime.NewRuntimeOptions{
		NumReactors: 4, // 4 CPU-bound reactors
		MailboxSize: 1024,
		NumWorkers:  128,   // A large pool for blocking I/O
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
	httpServer := httpx.NewServer(8080, nil, b)

	// Register the HTTP handlers.
	httpServer.Get("/greet", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			name = "World"
		}

		reply, err := b.Request(r.Context(), "greeter.hello", bus.Message{Payload: name})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, reply.Payload)
	})

	httpServer.Get("/blocking", func(w http.ResponseWriter, r *http.Request) {
		reply, err := b.Request(r.Context(), "blocking.long_operation", bus.Message{})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, reply.Payload)
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
