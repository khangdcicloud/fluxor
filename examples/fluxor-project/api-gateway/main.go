package main

import (
	"context"
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"

	"github.com/quadgatefoundation/fluxor/examples/fluxor-project/api-gateway/verticles"
)

func main() {
	app, err := fluxor.NewMainVerticleWithOptions("config.json", fluxor.MainVerticleOptions{
		EventBusFactory: func(ctx context.Context, gocmd core.GoCMD, cfg map[string]any) (core.EventBus, error) {
			natsCfg, _ := cfg["nats"].(map[string]any)
			url, _ := natsCfg["url"].(string)
			prefix, _ := natsCfg["prefix"].(string)
			return core.NewClusterEventBusJetStream(ctx, gocmd, core.ClusterJetStreamConfig{
				URL:     url,
				Prefix:  prefix,
				Service: "api-gateway",
			})
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	_, _ = app.DeployVerticle(verticles.NewApiGatewayVerticle())
	_ = app.Start()
}
