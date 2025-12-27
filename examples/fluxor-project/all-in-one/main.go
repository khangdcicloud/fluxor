package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"

	apigw "github.com/quadgatefoundation/fluxor/examples/fluxor-project/api-gateway/verticles"
	pay "github.com/quadgatefoundation/fluxor/examples/fluxor-project/payment-service/verticles"
)

func main() {
	logger := core.NewDefaultLogger()
	logger.Info("Starting all-in-one server...")

	app, err := fluxor.NewMainVerticleWithOptions("config.json", fluxor.MainVerticleOptions{
		EventBusFactory: func(ctx context.Context, gocmd core.GoCMD, cfg map[string]any) (core.EventBus, error) {
			natsCfg, _ := cfg["nats"].(map[string]any)
			url, _ := natsCfg["url"].(string)
			prefix, _ := natsCfg["prefix"].(string)
			eventBus, err := core.NewClusterEventBusJetStream(ctx, gocmd, core.ClusterJetStreamConfig{
				URL:     url,
				Prefix:  prefix,
				Service: "all-in-one",
			})
			if err == nil {
				core.Info(fmt.Sprintf("Connecting to NATS: %s (prefix: %s)", url, prefix))
			}
			return eventBus, err
		},
	})
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to create main verticle: %v", err))
		os.Exit(1)
	}

	// deploy order decision here
	logger.Info("Deploying API Gateway verticle...")
	_, err = app.DeployVerticle(apigw.NewApiGatewayVerticle())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deploy API Gateway verticle: %v", err))
	} else {
		logger.Info("API Gateway verticle deployed successfully")
	}

	logger.Info("Deploying Payment Service verticle...")
	_, err = app.DeployVerticle(pay.NewPaymentVerticle())
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to deploy Payment Service verticle: %v", err))
	} else {
		logger.Info("Payment Service verticle deployed successfully")
	}

	logger.Info("Starting application...")
	if err := app.Start(); err != nil {
		logger.Error(fmt.Sprintf("Failed to start application: %v", err))
		os.Exit(1)
	}

	logger.Info("All-in-one server started successfully!")
}
