package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fluxor-io/fluxor/pkg/core"
	"github.com/fluxor-io/fluxor/pkg/fluxor"
)

// --- Ping Reactor ---
type PingReactor struct{}

func (p *PingReactor) OnStart(ctx *core.FluxorContext) error {
	logger := ctx.Log() // Logger này đã có DeploymentID
	logger.Info("PingReactor Started")

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Ctx().Done():
				return
			case <-ticker.C:
				msg := fmt.Sprintf("PING from %s", ctx.ID())
				ctx.Bus().Publish("ping-topic", msg)
			}
		}
	}()
	return nil
}

func (p *PingReactor) OnStop() error { return nil }

// --- Pong Reactor ---
type PongReactor struct{}

func (p *PongReactor) OnStart(ctx *core.FluxorContext) error {
	logger := ctx.Log()
	ch := core.Subscribe[string](ctx.Bus(), "ping-topic")

	go func() {
		for msg := range ch {
			// In ra log để chứng minh Deployment ID hoạt động
			logger.Info("Received", "payload", msg)
		}
	}()
	return nil
}

func (p *PongReactor) OnStop() error { return nil }

// --- Main ---
func main() {
	rt := fluxor.New()

	// Deploy Ping
	rt.Deploy(&PingReactor{}, nil)

	// Deploy Pong (Scale 2 workers để thấy ID khác nhau)
	rt.Deploy(&PongReactor{}, nil)
	rt.Deploy(&PongReactor{}, nil)

	// Wait for Ctrl+C
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	rt.Shutdown()
}
