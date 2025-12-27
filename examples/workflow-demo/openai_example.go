package main

import (
	"fmt"
	"log"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"
	"github.com/fluxorio/fluxor/pkg/workflow"
	"github.com/fluxorio/fluxor/pkg/web"
)

// Example: Using OpenAI node in workflow
func ExampleOpenAIWorkflow() {
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
	app.DeployVerticle(NewOpenAIGateway(wfVerticle))

	fmt.Println("🚀 OpenAI Workflow Demo Running")
	fmt.Println("   API Gateway: http://localhost:8080")
	fmt.Println("")
	fmt.Println("📋 Try this command:")
	fmt.Println(`   curl -X POST http://localhost:8080/api/chat -H "Content-Type: application/json" -d '{"message":"Hello, how are you?"}'`)

	app.Start()
}

// OpenAIGateway provides HTTP API for OpenAI workflows
type OpenAIGateway struct {
	wfVerticle *workflow.WorkflowVerticle
	server     *web.FastHTTPServer
}

func NewOpenAIGateway(wfVerticle *workflow.WorkflowVerticle) *OpenAIGateway {
	return &OpenAIGateway{wfVerticle: wfVerticle}
}

func (v *OpenAIGateway) Start(ctx core.FluxorContext) error {
	// Create workflow with OpenAI node
	wf := workflow.NewWorkflowBuilder("openai-chat", "OpenAI Chat Workflow").
		AddNode("start", "manual").
		Name("Start").
		Next("openai").
		Done().
		AddNode("openai", "openai").
		Name("OpenAI Chat").
		Config(map[string]interface{}{
			"model":  "gpt-3.5-turbo",
			"prompt": "{{ $.input.message }}",
			"temperature": 0.7,
			"maxTokens": 500,
		}).
		Next("format").
		Done().
		AddNode("format", "set").
		Name("Format Response").
		Config(map[string]interface{}{
			"values": map[string]interface{}{
				"success": true,
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

	router.POSTFast("/api/chat", func(c *web.FastRequestContext) error {
		var input interface{}
		if err := c.BindJSON(&input); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid JSON"})
		}

		// Execute workflow
		execID, err := v.wfVerticle.Engine().ExecuteWorkflow(c.Context(), "openai-chat", input)
		if err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		// Get result (simplified - in production, use proper async handling)
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

func (v *OpenAIGateway) Stop(ctx core.FluxorContext) error {
	if v.server != nil {
		return v.server.Stop()
	}
	return nil
}

