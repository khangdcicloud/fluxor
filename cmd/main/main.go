package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fluxor-io/fluxor/pkg/bus"
	"github.com/fluxor-io/fluxor/pkg/component"
	"github.com/fluxor-io/fluxor/pkg/httpx"
	"github.com/fluxor-io/fluxor/pkg/runtime"
)

// GreeterService is a simple service that greets the world.
type GreeterService struct {
	component.Base
	bus bus.Bus
}

// OnStart is called when the component is started.
func (g *GreeterService) OnStart(ctx context.Context, b bus.Bus) {
	g.bus = b
	g.bus.Consumer("/greet", g.handleGreet)
	log.Println("GreeterService started")
}

func (g *GreeterService) handleGreet(msg bus.Message) {
	req := msg.Payload.(*httpx.HttpRequest)
	u, _ := url.Parse(req.URL)
	name := u.Query().Get("name")
	if name == "" {
		name = "world"
	}

	if msg.ReplyTo != "" {
		// In a real application, you would not want to block the consumer goroutine.
		// Instead, you would send the reply asynchronously.
		g.bus.Send(msg.ReplyTo, bus.Message{
			Payload:       fmt.Sprintf("hello %s", name),
			CorrelationID: msg.CorrelationID,
		})
	}
}

func main() {
	// Create a new in-process event bus.
	b := bus.NewBus()

	// Create a new runtime.
	rt := runtime.NewRuntime(b)

	// Register components.
	rt.Register("gateway", httpx.NewServer(8080, b))
	rt.Register("greeter", &GreeterService{})

	// Start the runtime.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := rt.Start(ctx); err != nil {
		log.Fatalf("failed to start runtime: %v", err)
	}

	log.Println("Runtime started")

	// Wait for a signal to stop the runtime.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")

	// Stop the runtime gracefully.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := rt.Stop(shutdownCtx); err != nil {
		log.Fatalf("failed to stop runtime: %v", err)
	}

	log.Println("Runtime stopped")
}
