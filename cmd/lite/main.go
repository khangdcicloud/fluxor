package main

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/lite/fluxor"
	"github.com/fluxorio/fluxor/pkg/lite/fx"
	"github.com/fluxorio/fluxor/pkg/lite/web"
)

func main() {
	// 1) Runtime
	app := fluxor.New()

	// Subscribe once (global)
	app.Bus().Subscribe("log", func(msg any) {
		// In a real service: structured logger, redaction, correlation IDs, etc.
		println("log:", msg)
	})

	// 2) Routes / logic
	r := web.NewRouter()

	r.GET("/api/ping", func(c *fx.Context) error {
		return c.Ok(fx.JSON{"msg": "pong", "framework": "Fluxor (lite)"})
	})

	r.GET("/api/work", func(c *fx.Context) error {
		// Demo: WorkerPool + Bus (avoid capturing request context into goroutine)
		wp := c.Worker()
		bus := c.Bus()
		wp.Submit(func(ctx context.Context) {
			bus.Publish("log", "Heavy Task Done")
		})

		return c.Ok(fx.JSON{"status": "processing"})
	})

	// 3) Component
	v := web.NewHttpVerticle("8080", r)

	// 4) Deploy & Run
	app.Deploy(v)
	app.Run()
}
