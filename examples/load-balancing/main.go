package main

import (
	"github.com/fluxorio/fluxor/examples/load-balancing/verticles"
	"github.com/fluxorio/fluxor/pkg/fluxor"
)

func main() {
	// Create MainVerticle
	app, err := fluxor.NewMainVerticle("examples/load-balancing/config.json")
	if err != nil {
		panic(err)
	}

	workerIDs := []string{"1", "2"}
	if raw, ok := app.Config()["workers"]; ok {
		if arr, ok := raw.([]interface{}); ok && len(arr) > 0 {
			tmp := make([]string, 0, len(arr))
			for _, v := range arr {
				if s, ok := v.(string); ok && s != "" {
					tmp = append(tmp, s)
				}
			}
			if len(tmp) > 0 {
				workerIDs = tmp
			}
		}
	}

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
