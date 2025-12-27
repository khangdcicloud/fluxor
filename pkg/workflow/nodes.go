package workflow

import (
	"context"
	"fmt"
	"time"
)

// noOpHandler passes data through unchanged.
func noOpHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	return &NodeOutput{Data: input.Data}, nil
}

// setHandler sets variables in the execution context.
func setHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config should contain "values" map
	values, ok := input.Config["values"].(map[string]interface{})
	if !ok {
		return &NodeOutput{Data: input.Data}, nil
	}

	// Merge values into output
	output := make(map[string]interface{})
	if data, ok := input.Data.(map[string]interface{}); ok {
		for k, v := range data {
			output[k] = v
		}
	}
	for k, v := range values {
		output[k] = v
	}

	return &NodeOutput{Data: output}, nil
}

// conditionHandler evaluates a condition and determines next nodes.
func conditionHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config should contain:
	// - "field": field to check
	// - "operator": eq, ne, gt, lt, gte, lte, contains, exists
	// - "value": value to compare against

	field, _ := input.Config["field"].(string)
	operator, _ := input.Config["operator"].(string)
	expectedValue := input.Config["value"]

	// Get actual value from input data
	var actualValue interface{}
	if data, ok := input.Data.(map[string]interface{}); ok {
		actualValue = data[field]
	}

	result := evaluateCondition(actualValue, operator, expectedValue)

	output := &NodeOutput{Data: input.Data}
	if result {
		// Return signal to use trueNext nodes
		output.Data = map[string]interface{}{
			"_conditionResult": true,
			"_originalData":    input.Data,
		}
	} else {
		// Return signal to use falseNext nodes
		output.Data = map[string]interface{}{
			"_conditionResult": false,
			"_originalData":    input.Data,
		}
	}

	return output, nil
}

func evaluateCondition(actual interface{}, operator string, expected interface{}) bool {
	switch operator {
	case "eq", "==", "equals":
		return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
	case "ne", "!=", "notEquals":
		return fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected)
	case "gt", ">":
		return toFloat(actual) > toFloat(expected)
	case "lt", "<":
		return toFloat(actual) < toFloat(expected)
	case "gte", ">=":
		return toFloat(actual) >= toFloat(expected)
	case "lte", "<=":
		return toFloat(actual) <= toFloat(expected)
	case "contains":
		return contains(actual, expected)
	case "exists":
		return actual != nil
	case "empty":
		return isEmpty(actual)
	case "notEmpty":
		return !isEmpty(actual)
	default:
		return actual == expected
	}
}

func toFloat(v interface{}) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	default:
		return 0
	}
}

func contains(actual, expected interface{}) bool {
	switch a := actual.(type) {
	case string:
		if e, ok := expected.(string); ok {
			return len(a) > 0 && len(e) > 0 &&
				(a == e || len(a) > len(e) && (a[:len(e)] == e || a[len(a)-len(e):] == e))
		}
	case []interface{}:
		for _, item := range a {
			if fmt.Sprintf("%v", item) == fmt.Sprintf("%v", expected) {
				return true
			}
		}
	}
	return false
}

func isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	switch a := v.(type) {
	case string:
		return a == ""
	case []interface{}:
		return len(a) == 0
	case map[string]interface{}:
		return len(a) == 0
	}
	return false
}

// waitHandler delays execution for a specified duration.
func waitHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config: "duration" in milliseconds or as string like "5s"
	var duration time.Duration

	if ms, ok := input.Config["duration"].(float64); ok {
		duration = time.Duration(ms) * time.Millisecond
	} else if dur, ok := input.Config["duration"].(string); ok {
		var err error
		duration, err = time.ParseDuration(dur)
		if err != nil {
			return nil, fmt.Errorf("invalid duration: %s", dur)
		}
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(duration):
		return &NodeOutput{Data: input.Data}, nil
	}
}

// errorHandler throws an error to stop or branch execution.
func errorHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	message, _ := input.Config["message"].(string)
	if message == "" {
		message = "workflow error"
	}
	return nil, fmt.Errorf("%s", message)
}

// loopHandler iterates over an array and executes next nodes for each item.
func loopHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config:
	// - "items": field name containing array, or use input data directly
	// - "batchSize": number of items to process in parallel (default: 1)

	var items []interface{}

	if itemsField, ok := input.Config["items"].(string); ok {
		if data, ok := input.Data.(map[string]interface{}); ok {
			if arr, ok := data[itemsField].([]interface{}); ok {
				items = arr
			}
		}
	} else if arr, ok := input.Data.([]interface{}); ok {
		items = arr
	}

	if len(items) == 0 {
		return &NodeOutput{Data: input.Data}, nil
	}

	// Return items for processing
	return &NodeOutput{
		Data: map[string]interface{}{
			"_loopItems":    items,
			"_loopIndex":    0,
			"_loopTotal":    len(items),
			"_originalData": input.Data,
		},
	}, nil
}

// splitHandler creates parallel branches.
func splitHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Split creates parallel execution paths
	// All "next" nodes will execute in parallel
	return &NodeOutput{
		Data: map[string]interface{}{
			"_parallel":     true,
			"_originalData": input.Data,
		},
	}, nil
}

// mergeHandler waits for multiple inputs and merges them.
func mergeHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config:
	// - "mode": "waitAll" (default), "waitAny", "append"
	// - "expectedInputs": number of inputs to wait for

	mode, _ := input.Config["mode"].(string)
	if mode == "" {
		mode = "waitAll"
	}

	// Merge data is handled by the engine
	return &NodeOutput{
		Data: map[string]interface{}{
			"_merge":        true,
			"_mergeMode":    mode,
			"_originalData": input.Data,
		},
	}, nil
}

// switchHandler provides multi-way branching based on value.
func switchHandler(ctx context.Context, input *NodeInput) (*NodeOutput, error) {
	// Config:
	// - "field": field to switch on
	// - "cases": map of value -> next node IDs
	// - "default": default next node IDs

	field, _ := input.Config["field"].(string)
	cases, _ := input.Config["cases"].(map[string]interface{})
	defaultNext, _ := input.Config["default"].([]interface{})

	var value interface{}
	if data, ok := input.Data.(map[string]interface{}); ok {
		value = data[field]
	}

	valueStr := fmt.Sprintf("%v", value)

	var nextNodes []string
	if caseNext, ok := cases[valueStr]; ok {
		if arr, ok := caseNext.([]interface{}); ok {
			for _, n := range arr {
				if s, ok := n.(string); ok {
					nextNodes = append(nextNodes, s)
				}
			}
		}
	} else if defaultNext != nil {
		for _, n := range defaultNext {
			if s, ok := n.(string); ok {
				nextNodes = append(nextNodes, s)
			}
		}
	}

	return &NodeOutput{
		Data:      input.Data,
		NextNodes: nextNodes,
	}, nil
}
