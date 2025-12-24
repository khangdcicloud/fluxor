package main

import (
	"github.com/fluxorio/fluxor/pkg/lite/fluxor"
	"github.com/fluxorio/fluxor/pkg/lite/fx"
	"github.com/fluxorio/fluxor/pkg/lite/webfast"
)

func main() {
	app := fluxor.New()

	// Use the component-scoped core context for logs, bus, worker (bound in verticle start)
	router := webfast.NewRouter()

	// For 500k RPS testing, keep handler extremely small and avoid JSON.
	router.GET("/ping",
		func(c *fx.FastContext) error {
			return c.Text(200, "pong")
		},
		webfast.Cache(webfast.CacheConfig{
			CacheControl: "public, max-age=60, immutable",
			ETag:         `"ping-v1"`,
		}),
	)

	router.GET("/users/:id", func(c *fx.FastContext) error {
		return c.Text(200, c.Param("id"))
	})

	v := webfast.NewFastHTTPVerticle(":8080", router)
	app.Deploy(v)
	app.Run()
}
