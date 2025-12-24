package main

import (
	"context"
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"

	"github.com/quadgatefoundation/fluxor/examples/fluxor-project/payment-service/verticles"
)

func main() {
	app, err := fluxor.NewMainVerticleWithOptions("config.json", fluxor.MainVerticleOptions{
		EventBusFactory: func(ctx context.Context, vertx core.Vertx, cfg map[string]any) (core.EventBus, error) {
			natsCfg, _ := cfg["nats"].(map[string]any)
			url, _ := natsCfg["url"].(string)
			prefix, _ := natsCfg["prefix"].(string)
			return core.NewClusterEventBusJetStream(ctx, vertx, core.ClusterJetStreamConfig{
				URL:     url,
				Prefix:  prefix,
				Service: "payment-service",
			})
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	_, _ = app.DeployVerticle(verticles.NewPaymentVerticle())
	_ = app.Start()
}
