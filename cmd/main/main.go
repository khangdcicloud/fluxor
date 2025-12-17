package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/goflux/pkg/bus"
	"github.com/example/goflux/pkg/component"
	"github.com/example/goflux/pkg/gateway"
	"github.com/example/goflux/pkg/httpx"
	"github.com/example/goflux/pkg/runtime"
)

type GreeterService struct {
	component.Base
	bus bus.Bus
}

func (g *GreeterService) Start(ctx context.Context) error {
	g.bus.Consumer("/greet", g.handleGreet)
	return nil
}

func (g *GreeterService) handleGreet(msg bus.Message) {
	req := msg.Payload.(*httpx.HttpRequest)
	u, _ := url.Parse(req.URL)
	name := u.Query().Get("name")
	if name == "" {
		name = "world"
	}

	if msg.ReplyTo != "" {
		g.bus.Send(msg.ReplyTo, bus.Message{
			Payload:       fmt.Sprintf("hello %s", name),
			CorrelationID: msg.CorrelationID,
		})
	}
}

func (g *GreeterService) Stop(ctx context.Context) error {
	return nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	rt := runtime.NewRuntime(4, 1024) // 4 reactors
	rt.Start()

	// Deploy gateway component
	gw := gateway.NewGateway(rt.ReactorForKey("gateway"), rt.Bus())
	_, err := rt.Deploy(gw, component.DeployOptions{Name: "gateway", Key: "gateway"})
	if err != nil {
		log.Fatalf("Failed to deploy gateway: %v", err)
	}

	// Deploy the greeter service
	_, err = rt.Deploy(&GreeterService{bus: rt.Bus()}, component.DeployOptions{Name: "greeter"})
	if err != nil {
		log.Fatalf("Failed to deploy greeter service: %v", err)
	}

	fmt.Println("Application started. Try sending requests to http://localhost:8080/greet?name=gopher")

	// Wait for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down...")
	cancel()
	rt.Stop(ctx)
	log.Println("Shut down gracefully")
}
