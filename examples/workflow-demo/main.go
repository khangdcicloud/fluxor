// Example: n8n-like workflow engine using EventBus
//
// This demonstrates how to create and execute event-driven workflows
// similar to n8n or Zapier.
//
// Run: go run ./examples/workflow-demo
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fluxor"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/workflow"
)

func main() {
	app, err := fluxor.NewMainVerticle("")
	if err != nil {
		log.Fatal(err)
	}

	// Deploy workflow verticle
	wfVerticle := workflow.NewWorkflowVerticle(&workflow.WorkflowVerticleConfig{
		HTTPAddr: ":8081", // Workflow management API
	})

	// Register custom functions
	wfVerticle.RegisterFunction("processOrder", processOrder)
	wfVerticle.RegisterFunction("calculateDiscount", calculateDiscount)
	wfVerticle.RegisterFunction("formatResponse", formatResponse)

	app.DeployVerticle(wfVerticle)

	// Deploy API gateway
	app.DeployVerticle(NewApiGateway(wfVerticle))

	fmt.Println("🚀 Workflow Demo Running")
	fmt.Println("   API Gateway: http://localhost:8080")
	fmt.Println("   Workflow API: http://localhost:8081")
	fmt.Println("")
	fmt.Println("📋 Try these commands:")
	fmt.Println("")
	fmt.Println("   # Create order-processing workflow")
	fmt.Println(`   curl -X POST http://localhost:8081/workflows -H "Content-Type: application/json" -d @workflow.json`)
	fmt.Println("")
	fmt.Println("   # Execute workflow via API")
	fmt.Println(`   curl -X POST http://localhost:8080/api/orders -H "Content-Type: application/json" -d '{"orderId":"123","amount":150}'`)
	fmt.Println("")
	fmt.Println("   # Check execution status")
	fmt.Println("   curl http://localhost:8081/executions/{executionId}")
	fmt.Println("")

	app.Start()
}

// Custom functions for workflow nodes

func processOrder(data interface{}) (interface{}, error) {
	order, _ := data.(map[string]interface{})
	order["processed"] = true
	order["processedAt"] = time.Now().Format(time.RFC3339)
	return order, nil
}

func calculateDiscount(data interface{}) (interface{}, error) {
	order, _ := data.(map[string]interface{})
	amount, _ := order["amount"].(float64)

	var discount float64
	if amount > 100 {
		discount = amount * 0.1 // 10% discount
	} else if amount > 50 {
		discount = amount * 0.05 // 5% discount
	}

	order["discount"] = discount
	order["finalAmount"] = amount - discount
	return order, nil
}

func formatResponse(data interface{}) (interface{}, error) {
	order, _ := data.(map[string]interface{})
	return map[string]interface{}{
		"success": true,
		"order":   order,
		"message": "Order processed successfully",
	}, nil
}

// ApiGateway verticle

type ApiGateway struct {
	wfVerticle *workflow.WorkflowVerticle
	server     *web.FastHTTPServer
}

func NewApiGateway(wfVerticle *workflow.WorkflowVerticle) *ApiGateway {
	return &ApiGateway{wfVerticle: wfVerticle}
}

func (v *ApiGateway) Start(ctx core.FluxorContext) error {
	// Create the order-processing workflow programmatically
	wf := workflow.NewWorkflowBuilder("order-processing", "Order Processing Workflow").
		AddNode("start", "manual").
		Name("Start").
		Next("validate").
		Done().
		AddNode("validate", "condition").
		Name("Validate Order").
		Config(map[string]interface{}{
			"field":    "amount",
			"operator": "gt",
			"value":    0,
		}).
		TrueNext("process").
		FalseNext("invalid").
		Done().
		AddNode("process", "function").
		Name("Process Order").
		Config(map[string]interface{}{
			"function": "processOrder",
		}).
		Next("discount").
		Done().
		AddNode("discount", "function").
		Name("Calculate Discount").
		Config(map[string]interface{}{
			"function": "calculateDiscount",
		}).
		Next("check-amount").
		Done().
		AddNode("check-amount", "condition").
		Name("Check Final Amount").
		Config(map[string]interface{}{
			"field":    "finalAmount",
			"operator": "gt",
			"value":    100,
		}).
		TrueNext("premium").
		FalseNext("standard").
		Done().
		AddNode("premium", "set").
		Name("Mark Premium").
		Config(map[string]interface{}{
			"values": map[string]interface{}{
				"tier": "premium",
			},
		}).
		Next("format").
		Done().
		AddNode("standard", "set").
		Name("Mark Standard").
		Config(map[string]interface{}{
			"values": map[string]interface{}{
				"tier": "standard",
			},
		}).
		Next("format").
		Done().
		AddNode("format", "function").
		Name("Format Response").
		Config(map[string]interface{}{
			"function": "formatResponse",
		}).
		Done().
		AddNode("invalid", "set").
		Name("Invalid Order").
		Config(map[string]interface{}{
			"values": map[string]interface{}{
				"error":   "invalid_amount",
				"message": "Order amount must be greater than 0",
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
	v.server = web.NewFastHTTPServer(ctx.Vertx(), config)
	router := v.server.FastRouter()

	// API endpoints
	router.GETFast("/health", func(c *web.FastRequestContext) error {
		return c.JSON(200, map[string]interface{}{"status": "ok"})
	})

	router.POSTFast("/api/orders", func(c *web.FastRequestContext) error {
		var input interface{}
		if err := c.BindJSON(&input); err != nil {
			return c.JSON(400, map[string]interface{}{"error": "invalid JSON"})
		}

		// Execute workflow
		execID, err := v.wfVerticle.Engine().ExecuteWorkflow(c.Context(), "order-processing", input)
		if err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		// Wait a bit for simple workflows to complete
		time.Sleep(100 * time.Millisecond)

		// Get result
		execCtx, err := v.wfVerticle.Engine().GetExecution(execID)
		if err != nil {
			return c.JSON(500, map[string]interface{}{"error": err.Error()})
		}

		// Return the last node output or format node output
		if result, ok := execCtx.NodeOutputs["format"]; ok {
			return c.JSON(200, result)
		}
		if result, ok := execCtx.NodeOutputs["invalid"]; ok {
			return c.JSON(400, result)
		}

		return c.JSON(200, map[string]interface{}{
			"executionId": execID,
			"outputs":     execCtx.NodeOutputs,
		})
	})

	go v.server.Start()
	return nil
}

func (v *ApiGateway) Stop(ctx core.FluxorContext) error {
	if v.server != nil {
		return v.server.Stop()
	}
	return nil
}
