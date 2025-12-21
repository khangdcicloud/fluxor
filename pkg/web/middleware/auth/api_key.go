package auth

import (
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
)

// APIKeyConfig configures API key authentication
type APIKeyConfig struct {
	// ValidateKey validates an API key
	ValidateKey func(key string) (map[string]interface{}, error)

	// KeyLookup is the key lookup pattern (default: "header:X-API-Key")
	// Format: "header:<name>", "query:<name>", "cookie:<name>"
	KeyLookup string

	// ClaimsKey is the key to store claims in request context
	ClaimsKey string

	// SkipPaths is a list of paths to skip authentication
	SkipPaths []string

	// OnError is called when authentication fails
	OnError func(ctx *web.FastRequestContext, err error) error
}

// DefaultAPIKeyConfig returns a default API key configuration
func DefaultAPIKeyConfig(validateKey func(string) (map[string]interface{}, error)) APIKeyConfig {
	return APIKeyConfig{
		ValidateKey: validateKey,
		KeyLookup:   "header:X-API-Key",
		ClaimsKey:   "user",
		SkipPaths:   []string{},
	}
}

// APIKey middleware validates API keys
func APIKey(config APIKeyConfig) web.FastMiddleware {
	if config.ValidateKey == nil {
		panic("APIKey: ValidateKey function must be provided")
	}

	// Default key lookup
	keyLookup := config.KeyLookup
	if keyLookup == "" {
		keyLookup = "header:X-API-Key"
	}

	// Parse key lookup
	lookupParts := strings.Split(keyLookup, ":")
	if len(lookupParts) != 2 {
		panic("APIKey: invalid KeyLookup format, expected 'source:name'")
	}
	lookupSource := lookupParts[0]
	lookupName := lookupParts[1]

	// Default error handler
	onError := config.OnError
	if onError == nil {
		onError = func(ctx *web.FastRequestContext, err error) error {
			ctx.RequestCtx.SetStatusCode(401)
			ctx.RequestCtx.SetContentType("application/json")
			ctx.RequestCtx.WriteString(fmt.Sprintf(`{"error":"unauthorized","message":"%s"}`, err.Error()))
			return nil
		}
	}

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Check if path should be skipped
			path := string(ctx.Path())
			for _, skipPath := range config.SkipPaths {
				if path == skipPath || strings.HasPrefix(path, skipPath) {
					return next(ctx)
				}
			}

			// Extract API key
			var apiKey string
			switch lookupSource {
			case "header":
				apiKey = string(ctx.RequestCtx.Request.Header.Peek(lookupName))
				if apiKey == "" {
					return onError(ctx, fmt.Errorf("API key header missing"))
				}
			case "query":
				apiKey = ctx.Query(lookupName)
				if apiKey == "" {
					return onError(ctx, fmt.Errorf("API key query parameter missing"))
				}
			case "cookie":
				cookieValue := ctx.RequestCtx.Request.Header.Cookie(lookupName)
				if len(cookieValue) == 0 {
					return onError(ctx, fmt.Errorf("API key cookie missing"))
				}
				apiKey = string(cookieValue)
			default:
				return onError(ctx, fmt.Errorf("unsupported key lookup source: %s", lookupSource))
			}

			// Validate API key
			claims, err := config.ValidateKey(apiKey)
			if err != nil {
				return onError(ctx, fmt.Errorf("invalid API key: %w", err))
			}

			// Store claims in context
			ctx.Set(config.ClaimsKey, claims)

			return next(ctx)
		}
	}
}

// SimpleAPIKeyValidator creates a simple API key validator from a map
func SimpleAPIKeyValidator(validKeys map[string]map[string]interface{}) func(string) (map[string]interface{}, error) {
	return func(key string) (map[string]interface{}, error) {
		claims, ok := validKeys[key]
		if !ok {
			return nil, fmt.Errorf("invalid API key")
		}
		return claims, nil
	}
}

