# Fluxor Workflow Engine

An n8n-like workflow engine for Fluxor using EventBus for node communication.

## Features

- **JSON-defined workflows** - Define workflows in JSON, similar to n8n
- **Event-driven execution** - Nodes communicate via EventBus
- **Built-in node types** - HTTP, conditions, loops, transforms, and more
- **Custom functions** - Register your own Go functions as nodes
- **Parallel execution** - Split/merge patterns for concurrent processing
- **Error handling** - Retry, fallback, and error nodes
- **HTTP API** - RESTful API for workflow management

## Quick Start

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/fluxor"
    "github.com/fluxorio/fluxor/pkg/workflow"
)

func main() {
    app, _ := fluxor.NewMainVerticle("")

    // Create workflow verticle with HTTP API
    wfVerticle := workflow.NewWorkflowVerticle(&workflow.WorkflowVerticleConfig{
        HTTPAddr: ":8081",
    })

    // Register custom functions
    wfVerticle.RegisterFunction("myFunction", func(data interface{}) (interface{}, error) {
        // Process data
        return data, nil
    })

    app.DeployVerticle(wfVerticle)
    app.Start()
}
```

## Workflow Definition

```json
{
  "id": "order-processing",
  "name": "Order Processing",
  "nodes": [
    {
      "id": "start",
      "type": "manual",
      "next": ["validate"]
    },
    {
      "id": "validate",
      "type": "condition",
      "config": {
        "field": "amount",
        "operator": "gt",
        "value": 0
      },
      "trueNext": ["process"],
      "falseNext": ["reject"]
    },
    {
      "id": "process",
      "type": "function",
      "config": {"function": "processOrder"},
      "next": ["notify"]
    },
    {
      "id": "notify",
      "type": "http",
      "config": {
        "url": "https://api.example.com/notify",
        "method": "POST"
      }
    },
    {
      "id": "reject",
      "type": "error",
      "config": {"message": "Invalid order amount"}
    }
  ]
}
```

## Node Types

### Trigger Nodes

| Type | Description |
|------|-------------|
| `manual` | Manual trigger (API call) |
| `webhook` | HTTP webhook trigger |
| `schedule` | Cron/interval trigger |
| `event` | EventBus event trigger |

### Action Nodes

| Type | Description | Config |
|------|-------------|--------|
| `function` | Execute registered function | `function`: function name |
| `http` | HTTP request | `url`, `method`, `headers`, `body`, `timeout` |
| `eventbus` | Send to EventBus | `address`, `action` (publish/send/request) |
| `set` | Set variables | `values`: map of key-value pairs |
| `code` | Transform data | `transform`: transformation rules |

### Flow Control Nodes

| Type | Description | Config |
|------|-------------|--------|
| `condition` | If/else branch | `field`, `operator`, `value` |
| `switch` | Multi-way branch | `field`, `cases`, `default` |
| `split` | Parallel execution | (uses all `next` nodes) |
| `merge` | Wait for inputs | `mode`: waitAll/waitAny |
| `loop` | Iterate array | `items`: field name |
| `wait` | Delay | `duration`: e.g., "5s" |

### Utility Nodes

| Type | Description |
|------|-------------|
| `noop` | Pass-through |
| `error` | Throw error |
| `filter` | Filter array |
| `map` | Transform array items |
| `reduce` | Reduce array |

## Condition Operators

- `eq`, `==`, `equals` - Equal
- `ne`, `!=`, `notEquals` - Not equal
- `gt`, `>` - Greater than
- `lt`, `<` - Less than
- `gte`, `>=` - Greater than or equal
- `lte`, `<=` - Less than or equal
- `contains` - Contains
- `exists` - Not null
- `empty` - Is empty
- `notEmpty` - Is not empty

## Template Variables

Use `{{field}}` syntax in strings to reference data:

```json
{
  "type": "http",
  "config": {
    "url": "https://api.example.com/users/{{userId}}",
    "headers": {
      "Authorization": "Bearer {{token}}"
    }
  }
}
```

## Programmatic Workflow Building

```go
wf := workflow.NewWorkflowBuilder("my-workflow", "My Workflow").
    AddNode("start", "manual").
    Next("process").
    Done().
    AddNode("process", "function").
    Config(map[string]interface{}{"function": "myFunc"}).
    Next("end").
    Retry(3).
    Timeout(30 * time.Second).
    Done().
    AddNode("end", "noop").
    Done().
    Build()

engine.RegisterWorkflow(wf)
```

## HTTP API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/workflows` | GET | List all workflows |
| `/workflows` | POST | Register workflow |
| `/workflows/:id/execute` | POST | Execute workflow |
| `/executions/:id` | GET | Get execution status |
| `/executions/:id/cancel` | POST | Cancel execution |
| `/health` | GET | Health check |

## Event-Driven Execution

Workflows use EventBus internally:

```
workflow.{workflowId}.execute     → Start workflow
workflow.{workflowId}.node.{id}   → Execute specific node
```

You can also trigger workflows via EventBus:

```go
// Register event trigger
workflow.RegisterEventTrigger(eventBus, engine, workflow.EventTriggerConfig{
    Address:    "orders.new",
    WorkflowID: "order-processing",
})

// Trigger from anywhere
eventBus.Publish("orders.new", orderData)
```

## Example: Order Processing Pipeline

```
┌──────────┐
│  Start   │
└────┬─────┘
     │
     ▼
┌──────────────┐     ┌──────────┐
│   Validate   │────►│  Reject  │
│  amount > 0  │ no  └──────────┘
└──────┬───────┘
       │ yes
       ▼
┌──────────────┐
│   Process    │
│  (function)  │
└──────┬───────┘
       │
       ▼
┌──────────────┐
│  Calculate   │
│   Discount   │
└──────┬───────┘
       │
       ▼
┌──────────────────┐     ┌──────────┐
│  Check Amount    │────►│ Standard │
│  > 100 (premium) │ no  └────┬─────┘
└──────┬───────────┘          │
       │ yes                  │
       ▼                      │
┌──────────┐                  │
│ Premium  │                  │
└────┬─────┘                  │
     │                        │
     └────────────┬───────────┘
                  │
                  ▼
            ┌──────────┐
            │  Format  │
            │ Response │
            └──────────┘
```

## See Also

- [examples/workflow-demo](../../examples/workflow-demo/) - Full working example
- [PRIMARY_PATTERN.md](../../docs/PRIMARY_PATTERN.md) - Fluxor patterns
