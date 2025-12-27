package main

import (
	"encoding/json"
	"log"

	"github.com/fluxorio/fluxor/pkg/fluxor"
	"github.com/fluxorio/fluxor/pkg/workflow"
)

func main() {
	// Create main verticle
	app, err := fluxor.NewMainVerticle("config.json")
	if err != nil {
		log.Fatal(err)
	}

	// Create workflow verticle
	workflowConfig := &workflow.WorkflowVerticleConfig{
		HTTPAddr: ":8081",
	}
	workflowVerticle := workflow.NewWorkflowVerticle(workflowConfig)

	// Register example workflow
	customerSupportWorkflow := createCustomerSupportWorkflow()
	if err := workflowVerticle.Engine().RegisterWorkflow(customerSupportWorkflow); err != nil {
		log.Fatal(err)
	}

	// Deploy workflow verticle
	app.DeployVerticle(workflowVerticle)

	// Start application
	app.Start()
}

// createCustomerSupportWorkflow creates an example AI-powered customer support workflow
func createCustomerSupportWorkflow() *workflow.WorkflowDefinition {
	workflowJSON := `{
		"id": "customer-support-ai",
		"name": "Customer Support AI",
		"description": "AI-powered customer support workflow using aimodule",
		"nodes": [
			{
				"id": "trigger",
				"type": "webhook",
				"name": "Webhook Trigger"
			},
			{
				"id": "ai-chat",
				"type": "aimodule.chat",
				"name": "AI Chat Response",
				"config": {
					"provider": "openai",
					"model": "gpt-4o",
					"prompt": "Bạn là trợ lý hỗ trợ khách hàng MoMo. Trả lời thân thiện bằng tiếng Việt cho câu hỏi sau: {{ $.input.query }}",
					"temperature": 0.7,
					"maxTokens": 500,
					"responseField": "response"
				},
				"next": ["format-response"]
			},
			{
				"id": "format-response",
				"type": "set",
				"name": "Format Response",
				"config": {
					"values": {
						"message": "{{ $.response }}",
						"timestamp": "{{ $.now }}"
					}
				},
				"next": ["output"]
			},
			{
				"id": "output",
				"type": "respond",
				"name": "Output"
			}
		]
	}`

	var def workflow.WorkflowDefinition
	if err := json.Unmarshal([]byte(workflowJSON), &def); err != nil {
		log.Fatal(err)
	}

	return &def
}
