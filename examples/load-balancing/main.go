package main

import (
	"github.com/fluxorio/fluxor/examples/load-balancing/verticles"
	"github.com/fluxorio/fluxor/pkg/fluxor"
)

func main() {
	// Create MainVerticle
	app, err := fluxor.NewMainVerticle("")
	if err != nil {
		panic(err)
	}

	workerIDs := []string{"1", "2"}

	// Deploy Workers
	for _, id := range workerIDs {
		_, err := app.DeployVerticle(verticles.NewWorkerVerticle(id))
		if err != nil {
			panic(err)
		}
	}

	// Deploy Master
	_, err = app.DeployVerticle(verticles.NewMasterVerticle(workerIDs))
	if err != nil {
		panic(err)
	}

	// Start application
	if err := app.Start(); err != nil {
		panic(err)
	}
}
