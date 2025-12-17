package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/goflux/pkg/bus"
	"github.com/example/goflux/pkg/inspector"
	"github.com/example/goflux/pkg/runtime"
)

func main() {
	// Create a new runtime.
	opts := runtime.NewRuntimeOptions{
		NumReactors: 4,
		MailboxSize: 1024,
		NumWorkers:  8,
		QueueSize:   1024,
	}
	b := bus.NewBus(1024)
	rt := runtime.NewRuntime(opts, b)

	// Start the runtime.
	rt.Start()

	// Create and deploy the inspector.
	inspector := inspector.NewInspector(":8080", rt)
	if err := rt.Deploy(context.Background(), inspector); err != nil {
		panic(err)
	}

	// Wait for a signal to shut down.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	// Shut down the runtime gracefully.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rt.Stop(ctx)
}
