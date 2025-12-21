package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/web"
)

// OAuth2Config configures OAuth2/OIDC authentication
type OAuth2Config struct {
	// IntrospectionURL is the OAuth2 token introspection endpoint
	IntrospectionURL string

	// ClientID is the OAuth2 client ID
	ClientID string

	// ClientSecret is the OAuth2 client secret
	ClientSecret string

	// TokenLookup is the token lookup pattern (default: "header:Authorization")
	TokenLookup string

	// AuthScheme is the authorization scheme (default: "Bearer")
	AuthScheme string

	// ClaimsKey is the key to store claims in request context
	ClaimsKey string

	// SkipPaths is a list of paths to skip authentication
	SkipPaths []string

	// OnError is called when authentication fails
	OnError func(ctx *web.FastRequestContext, err error) error

	// HTTPClient is the HTTP client for introspection (optional)
	HTTPClient *http.Client
}

// DefaultOAuth2Config returns a default OAuth2 configuration
func DefaultOAuth2Config(introspectionURL, clientID, clientSecret string) OAuth2Config {
	return OAuth2Config{
		IntrospectionURL: introspectionURL,
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		TokenLookup:      "header:Authorization",
		AuthScheme:       "Bearer",
		ClaimsKey:        "user",
		SkipPaths:        []string{},
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// OAuth2 middleware validates OAuth2 tokens via introspection
func OAuth2(config OAuth2Config) web.FastMiddleware {
	if config.IntrospectionURL == "" {
		panic("OAuth2: IntrospectionURL must be provided")
	}
	if config.ClientID == "" {
		panic("OAuth2: ClientID must be provided")
	}
	if config.ClientSecret == "" {
		panic("OAuth2: ClientSecret must be provided")
	}

	// Default token lookup
	tokenLookup := config.TokenLookup
	if tokenLookup == "" {
		tokenLookup = "header:Authorization"
	}

	// Parse token lookup
	lookupParts := strings.Split(tokenLookup, ":")
	if len(lookupParts) != 2 {
		panic("OAuth2: invalid TokenLookup format, expected 'source:name'")
	}
	lookupSource := lookupParts[0]
	lookupName := lookupParts[1]

	// Default auth scheme
	authScheme := config.AuthScheme
	if authScheme == "" {
		authScheme = "Bearer"
	}

	// Default HTTP client
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

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

			// Extract token
			var tokenString string
			switch lookupSource {
			case "header":
				authHeader := string(ctx.RequestCtx.Request.Header.Peek(lookupName))
				if authHeader == "" {
					return onError(ctx, fmt.Errorf("authorization header missing"))
				}
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || parts[0] != authScheme {
					return onError(ctx, fmt.Errorf("invalid authorization header format"))
				}
				tokenString = parts[1]
			case "query":
				tokenString = ctx.Query(lookupName)
				if tokenString == "" {
					return onError(ctx, fmt.Errorf("token query parameter missing"))
				}
			case "cookie":
				cookieValue := ctx.RequestCtx.Request.Header.Cookie(lookupName)
				if len(cookieValue) == 0 {
					return onError(ctx, fmt.Errorf("token cookie missing"))
				}
				tokenString = string(cookieValue)
			default:
				return onError(ctx, fmt.Errorf("unsupported token lookup source: %s", lookupSource))
			}

			// Introspect token
			claims, err := introspectToken(httpClient, config.IntrospectionURL, config.ClientID, config.ClientSecret, tokenString)
			if err != nil {
				return onError(ctx, fmt.Errorf("token introspection failed: %w", err))
			}

			// Check if token is active
			active, ok := claims["active"].(bool)
			if !ok || !active {
				return onError(ctx, fmt.Errorf("token is not active"))
			}

			// Store claims in context
			ctx.Set(config.ClaimsKey, claims)

			return next(ctx)
		}
	}
}

// introspectToken performs OAuth2 token introspection
func introspectToken(client *http.Client, url, clientID, clientSecret, token string) (map[string]interface{}, error) {
	// Create introspection request
	reqBody := fmt.Sprintf("token=%s&token_type_hint=access_token", token)
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(reqBody))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("introspection failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var claims map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("failed to decode introspection response: %w", err)
	}

	return claims, nil
}

