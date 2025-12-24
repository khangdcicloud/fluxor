package main

import (
	"log"

	"github.com/fluxorio/fluxor/pkg/fluxor"

	"github.com/quadgatefoundation/fluxor/fluxor-project/payment-service/verticles"
)

func main() {
	app, err := fluxor.NewMainVerticle("config.json")
	if err != nil {
		log.Fatal(err)
	}

	_, _ = app.DeployVerticle(verticles.NewPaymentVerticle())
	_ = app.Start()
}
