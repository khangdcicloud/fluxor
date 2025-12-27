# Cursor AI Integration in Fluxor

## Overview

Cursor AI is integrated into Fluxor as an AI provider option in the Workflow Engine. This allows you to use Cursor's AI capabilities in your event-driven workflows.

## Integration Points

### 1. Workflow Engine (`pkg/workflow`)

Cursor is supported as a provider in the generic AI node handler:

```go
// In pkg/workflow/nodes_ai.go
func AINodeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
    provider := "openai" // default
    if p, ok := input.Config["provider"].(string); ok && p != "" {
        provider = strings.ToLower(p) // Supports "cursor"
    }
    // ... handles Cursor API calls
}
```

### 2. Provider Configuration

Cursor uses OpenAI-compatible API:
- **Base URL**: `https://api.openai.com/v1` (default, can be overridden)
- **Endpoint**: `/chat/completions`
- **Default Model**: `gpt-4`
- **Env Var**: `CURSOR_API_KEY` (or `OPENAI_API_KEY` if compatible)

### 3. Using in Workflows

#### JSON Workflow Definition

```json
{
  "id": "cursor-workflow",
  "name": "Cursor AI Workflow",
  "nodes": [
    {
      "id": "start",
      "type": "webhook",
      "name": "Webhook Trigger"
    },
    {
      "id": "cursor",
      "type": "ai",
      "name": "Cursor AI",
      "config": {
        "provider": "cursor",
        "apiKey": "sk-...",  // Optional if CURSOR_API_KEY env var is set
        "model": "gpt-4",
        "prompt": "{{ $.input.prompt }}",
        "temperature": 0.7,
        "maxTokens": 2000,
        "responseField": "response"
      },
      "next": ["format"]
    },
    {
      "id": "format",
      "type": "set",
      "name": "Format Response",
      "config": {
        "values": {
          "success": true,
          "provider": "cursor"
        }
      }
    }
  ]
}
```

#### Go Code Example

```go
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
        "maxTokens":   2000,
    }).
    Next("format").
    Done().
    Build()
```

### 4. Integration with fluxorcli

When you create a new app with `fluxorcli new`, you can add Cursor AI workflows:

```bash
# Create new app
fluxorcli new myapp
cd myapp

# Add workflow verticle
# In main.go, deploy workflow verticle:
app.DeployVerticle(workflow.NewWorkflowVerticle(&workflow.WorkflowVerticleConfig{
    HTTPAddr: ":8081",
}))

# Register Cursor workflow
cursorWorkflow := createCursorWorkflow()
wfVerticle.Engine().RegisterWorkflow(cursorWorkflow)
```

### 5. HTTP API Integration

Example API endpoint that uses Cursor AI:

```go
router.POSTFast("/api/cursor", func(c *web.FastRequestContext) error {
    var input interface{}
    if err := c.BindJSON(&input); err != nil {
        return c.JSON(400, map[string]interface{}{"error": "invalid JSON"})
    }

    // Execute workflow with Cursor AI
    execID, err := wfVerticle.Engine().ExecuteWorkflow(c.Context(), "cursor-ai", input)
    if err != nil {
        return c.JSON(500, map[string]interface{}{"error": err.Error()})
    }

    // Get result
    execCtx, err := wfVerticle.Engine().GetExecution(execID)
    if err != nil {
        return c.JSON(500, map[string]interface{}{"error": err.Error()})
    }

    return c.JSON(200, map[string]interface{}{
        "executionId": execID,
        "outputs":     execCtx.NodeOutputs,
    })
})
```

### 6. Environment Variables

Set API key via environment variable:

```bash
export CURSOR_API_KEY="sk-..."
# Or if using OpenAI-compatible endpoint:
export OPENAI_API_KEY="sk-..."
```

### 7. Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `provider` | string | `"openai"` | Set to `"cursor"` for Cursor AI |
| `apiKey` | string | - | API key (or use env var) |
| `baseURL` | string | `"https://api.openai.com/v1"` | API base URL |
| `model` | string | `"gpt-4"` | Model name |
| `prompt` | string | - | Prompt template (supports `{{ $.input.field }}`) |
| `messages` | array | - | Chat messages array |
| `temperature` | float | `1.0` | Temperature (0-2) |
| `maxTokens` | int | `1000` | Max tokens |
| `timeout` | string | `"60s"` | Request timeout |
| `responseField` | string | `"response"` | Field name for response |

### 8. Example: Complete Integration

See `examples/workflow-demo/cursor_example.go` for a complete example:

```bash
cd examples/workflow-demo
go run cursor_example.go

# Test with:
curl -X POST http://localhost:8080/api/cursor \
  -H "Content-Type: application/json" \
  -d '{"prompt":"Write a Go function to calculate factorial"}'
```

## Implementation Details

### Provider Detection

The workflow engine detects Cursor provider and:
1. Uses OpenAI-compatible endpoint (`/chat/completions`)
2. Sets default model to `gpt-4`
3. Looks for `CURSOR_API_KEY` env var
4. Uses standard OpenAI request/response format

### Template Processing

Prompts support template syntax:
- `{{ $.input.field }}` - Access input data
- `{{ $.response }}` - Access previous node output
- `{{ $.now }}` - Current timestamp

### Error Handling

Cursor API errors are handled the same as other AI providers:
- Network errors → retry with exponential backoff
- Rate limits → return error with retry-after
- Invalid API key → return error immediately

## See Also

- [Workflow Engine README](pkg/workflow/README.md)
- [AI Module Documentation](pkg/aimodule/README.md)
- [Workflow Examples](examples/workflow-demo/)

