package main

import (
	"log"

	"github.com/fluxorio/fluxor/pkg/fluxor"

	apigw "github.com/quadgatefoundation/fluxor/fluxor-project/api-gateway/verticles"
	pay "github.com/quadgatefoundation/fluxor/fluxor-project/payment-service/verticles"
)

func main() {
	app, err := fluxor.NewMainVerticle("config.json")
	if err != nil {
		log.Fatal(err)
	}

	// deploy order decision here
	_, _ = app.DeployVerticle(apigw.NewApiGatewayVerticle())
	_, _ = app.DeployVerticle(pay.NewPaymentVerticle())

	_ = app.Start()
}
