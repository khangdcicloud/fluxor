package auth

import (
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig configures JWT authentication
type JWTConfig struct {
	// SecretKey is the secret key for signing/verifying tokens
	SecretKey string

	// SecretKeyFunc is a function that returns the secret key (alternative to SecretKey)
	SecretKeyFunc func(token *jwt.Token) (interface{}, error)

	// ClaimsKey is the key to store claims in request context
	ClaimsKey string

	// TokenLookup is the token lookup pattern (default: "header:Authorization")
	// Format: "header:<name>", "query:<name>", "cookie:<name>"
	TokenLookup string

	// AuthScheme is the authorization scheme (default: "Bearer")
	AuthScheme string

	// SkipPaths is a list of paths to skip authentication
	SkipPaths []string

	// OnError is called when authentication fails
	OnError func(ctx *web.FastRequestContext, err error) error
}

// DefaultJWTConfig returns a default JWT configuration
func DefaultJWTConfig(secretKey string) JWTConfig {
	return JWTConfig{
		SecretKey:  secretKey,
		ClaimsKey:  "user",
		TokenLookup: "header:Authorization",
		AuthScheme: "Bearer",
		SkipPaths:  []string{},
	}
}

// JWT middleware validates JWT tokens
func JWT(config JWTConfig) web.FastMiddleware {
	if config.SecretKey == "" && config.SecretKeyFunc == nil {
		panic("JWT: SecretKey or SecretKeyFunc must be provided")
	}

	// Default secret key function
	keyFunc := config.SecretKeyFunc
	if keyFunc == nil {
		keyFunc = func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(config.SecretKey), nil
		}
	}

	// Default token lookup
	tokenLookup := config.TokenLookup
	if tokenLookup == "" {
		tokenLookup = "header:Authorization"
	}

	// Parse token lookup
	lookupParts := strings.Split(tokenLookup, ":")
	if len(lookupParts) != 2 {
		panic("JWT: invalid TokenLookup format, expected 'source:name'")
	}
	lookupSource := lookupParts[0]
	lookupName := lookupParts[1]

	// Default auth scheme
	authScheme := config.AuthScheme
	if authScheme == "" {
		authScheme = "Bearer"
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
				// Extract token from "Bearer <token>"
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

			// Parse and validate token
			token, err := jwt.Parse(tokenString, keyFunc)
			if err != nil {
				return onError(ctx, fmt.Errorf("invalid token: %w", err))
			}

			if !token.Valid {
				return onError(ctx, fmt.Errorf("token is not valid"))
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return onError(ctx, fmt.Errorf("invalid token claims"))
			}

			// Store claims in context
			ctx.Set(config.ClaimsKey, claims)

			return next(ctx)
		}
	}
}

// GetClaims extracts JWT claims from request context
func GetClaims(ctx *web.FastRequestContext, key string) (jwt.MapClaims, error) {
	claims, ok := ctx.Get(key).(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("claims not found in context")
	}
	return claims, nil
}

// GetUserID extracts user ID from JWT claims
func GetUserID(ctx *web.FastRequestContext, key string) (string, error) {
	claims, err := GetClaims(ctx, key)
	if err != nil {
		return "", err
	}

	// Try common claim names
	userID, ok := claims["user_id"].(string)
	if ok {
		return userID, nil
	}

	userID, ok = claims["sub"].(string) // JWT standard "subject"
	if ok {
		return userID, nil
	}

	userID, ok = claims["id"].(string)
	if ok {
		return userID, nil
	}

	return "", fmt.Errorf("user ID not found in claims")
}

