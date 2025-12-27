package main

import (
	"fmt"
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"
	"github.com/fluxorio/fluxor/pkg/workflow"
	"github.com/fluxorio/fluxor/pkg/web"
)

// Example: Using Cursor AI node in workflow
func ExampleCursorWorkflow() {
	app, err := fluxor.NewMainVerticle("")
	if err != nil {
		log.Fatal(err)
	}

	// Create workflow verticle
	wfVerticle := workflow.NewWorkflowVerticle(&workflow.WorkflowVerticleConfig{
		HTTPAddr: ":8081",
	})

	app.DeployVerticle(wfVerticle)

	// Deploy API gateway
	app.DeployVerticle(NewCursorGateway(wfVerticle))

	fmt.Println("🚀 Cursor AI Workflow Demo Running")
	fmt.Println("   API Gateway: http://localhost:8080")
	fmt.Println("")
	fmt.Println("📋 Try this command:")
	fmt.Println(`   curl -X POST http://localhost:8080/api/cursor -H "Content-Type: application/json" -d '{"prompt":"Write a Go function to calculate factorial"}'`)

	app.Start()
}

// CursorGateway provides HTTP API for Cursor AI workflows
type CursorGateway struct {
	wfVerticle *workflow.WorkflowVerticle
	server     *web.FastHTTPServer
}

func NewCursorGateway(wfVerticle *workflow.WorkflowVerticle) *CursorGateway {
	return &CursorGateway{wfVerticle: wfVerticle}
}

func (v *CursorGateway) Start(ctx core.FluxorContext) error {
	// Create workflow with Cursor AI node
	wf := workflow.NewWorkflowBuilder("cursor-ai", "Cursor AI Workflow").
		AddNode("start", "manual").
		Name("Start").
		Next("cursor").
		Done().
		AddNode("cursor", "ai").
		Name("Cursor AI").
		Config(map[string]interface{}{
			"provider":    "cursor",
			"model":       "gpt-4",
			"prompt":      "{{ $.input.prompt }}",
			"temperature": 0.7,
			"maxTokens":  2000,
		}).
		Next("format").
		Done().
		AddNode("format", "set").
		Name("Format Response").
		Config(map[string]interface{}{
			"values": map[string]interface{}{
				"success": true,
				"provider": "cursor",
			},
		}).
		Done().
		Build()

	// Register workflow
	if err := v.wfVerticle.Engine().RegisterWorkflow(wf); err != nil {
		return err
	}

	// Create HTTP server
	config := web.DefaultFastHTTPServerConfig(":8080")
	v.server = web.NewFastHTTPServer(ctx.GoCMD(), config)
	router := v.server.FastRouter()

	// API endpoints
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{"status": "ok"})
	})

	router.POSTFast("/api/cursor", func(c *web.FastRequestContext) error {
		var input interface{}
		if err := c.BindJSON(&input); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid JSON"})
		}

		// Execute workflow
		execID, err := v.wfVerticle.Engine().ExecuteWorkflow(c.Context(), "cursor-ai", input)
		if err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		// Get result
		execCtx, err := v.wfVerticle.Engine().GetExecution(execID)
		if err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		return c.JSON(200, map[string]interface{}{
			"executionId": execID,
			"outputs":     execCtx.NodeOutputs,
		})
	})

	go v.server.Start()
	return nil
}

func (v *CursorGateway) Stop(ctx core.FluxorContext) error {
	if v.server != nil {
		return v.server.Stop()
	}
	return nil
}

