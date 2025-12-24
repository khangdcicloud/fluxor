package main

import (
	"context"
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"

	apigw "github.com/quadgatefoundation/fluxor/fluxor-project/api-gateway/verticles"
	pay "github.com/quadgatefoundation/fluxor/fluxor-project/payment-service/verticles"
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
				Service: "all-in-one",
			})
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// deploy order decision here
	_, _ = app.DeployVerticle(apigw.NewApiGatewayVerticle())
	_, _ = app.DeployVerticle(pay.NewPaymentVerticle())

	_ = app.Start()
}
