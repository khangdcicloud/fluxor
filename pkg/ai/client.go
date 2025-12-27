package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// AIClient is the base implementation of the Client interface
type AIClient struct {
	provider Provider
	config   Config
	client   *http.Client
	baseURL  string
	apiKey   string
}

// NewClient creates a new AI client
// Fail-fast: Validates configuration
func NewClient(config Config) (Client, error) {
	// Fail-fast: Validate provider
	if config.Provider == "" {
		config.Provider = ProviderOpenAI
	}

	// Fail-fast: Get API key
	apiKey := config.APIKey
	if apiKey == "" {
		envVar := getProviderEnvVar(config.Provider)
		apiKey = os.Getenv(envVar)
		if apiKey == "" {
			return nil, fmt.Errorf("ai client requires 'apiKey' config or %s env var", envVar)
		}
	}

	// Get base URL
	baseURL := getProviderBaseURL(config.Provider)
	if config.BaseURL != "" {
		baseURL = config.BaseURL
	}

	// Get timeout
	timeout := 60 * time.Second
	if config.Timeout != "" {
		if d, err := time.ParseDuration(config.Timeout); err == nil {
			timeout = d
		}
	}

	return &AIClient{
		provider: config.Provider,
		config:   config,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		apiKey:  apiKey,
	}, nil
}

// Provider returns the provider type
func (c *AIClient) Provider() Provider {
	return c.provider
}

// Chat sends a chat completion request
func (c *AIClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Fail-fast: Validate request
	if req == nil {
		return nil, fmt.Errorf("chat request cannot be nil")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("chat request must have at least one message")
	}

	// Get model
	model := req.Model
	if model == "" {
		model = c.config.Model
		if model == "" {
			model = getProviderDefaultModel(c.provider)
		}
	}

	// Build request body
	requestBody := map[string]interface{}{
		"model":    model,
		"messages": req.Messages,
	}

	if req.Temperature > 0 {
		requestBody["temperature"] = req.Temperature
	} else {
		requestBody["temperature"] = 1.0
	}

	if req.MaxTokens > 0 {
		requestBody["max_tokens"] = req.MaxTokens
	} else {
		requestBody["max_tokens"] = 1000
	}

	if req.TopP > 0 {
		requestBody["top_p"] = req.TopP
	} else {
		requestBody["top_p"] = 1.0
	}

	if req.Stream {
		requestBody["stream"] = true
	}

	// Get endpoint
	endpoint := getProviderEndpoint(c.provider, model)

	// Marshal request body
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	reqCtx, cancel := context.WithTimeout(ctx, c.client.Timeout)
	defer cancel()

	url := c.baseURL + endpoint
	httpReq, err := http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	setProviderHeaders(httpReq, c.provider, c.apiKey)

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if errorMsg, ok := errorResp["error"].(map[string]interface{}); ok {
				if message, ok := errorMsg["message"].(string); ok {
					return nil, fmt.Errorf("ai API error: %s", message)
				}
			}
		}
		return nil, fmt.Errorf("ai API error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var responseData ChatResponse
	if err := json.Unmarshal(respBody, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &responseData, nil
}

// ChatSimple sends a simple chat message and returns the response text
func (c *AIClient) ChatSimple(ctx context.Context, prompt string) (string, error) {
	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	resp, err := c.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}

// getProviderBaseURL returns the base URL for a provider
func getProviderBaseURL(provider Provider) string {
	switch provider {
	case ProviderOpenAI:
		return "https://api.openai.com/v1"
	case ProviderCursor:
		return "https://api.openai.com/v1" // Cursor uses OpenAI-compatible API
	case ProviderAnthropic:
		return "https://api.anthropic.com/v1"
	case ProviderCustom:
		return "" // Must be provided in config
	default:
		return "https://api.openai.com/v1"
	}
}

// getProviderEndpoint returns the endpoint for a provider and model
func getProviderEndpoint(provider Provider, model string) string {
	switch provider {
	case ProviderOpenAI, ProviderCursor:
		if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1-") {
			return "/chat/completions"
		}
		return "/completions"
	case ProviderAnthropic:
		return "/messages"
	default:
		return "/chat/completions"
	}
}

// getProviderDefaultModel returns the default model for a provider
func getProviderDefaultModel(provider Provider) string {
	switch provider {
	case ProviderOpenAI:
		return "gpt-3.5-turbo"
	case ProviderCursor:
		return "gpt-4"
	case ProviderAnthropic:
		return "claude-3-sonnet-20240229"
	default:
		return "gpt-3.5-turbo"
	}
}

// getProviderEnvVar returns the environment variable name for a provider's API key
func getProviderEnvVar(provider Provider) string {
	switch provider {
	case ProviderOpenAI:
		return "OPENAI_API_KEY"
	case ProviderCursor:
		return "CURSOR_API_KEY"
	case ProviderAnthropic:
		return "ANTHROPIC_API_KEY"
	default:
		return "AI_API_KEY"
	}
}

// setProviderHeaders sets provider-specific headers
func setProviderHeaders(req *http.Request, provider Provider, apiKey string) {
	switch provider {
	case ProviderOpenAI, ProviderCursor:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	case ProviderAnthropic:
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	default:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}
