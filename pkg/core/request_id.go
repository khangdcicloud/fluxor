package core

import (
	"context"
	"github.com/google/uuid"
)

// RequestIDKey is the context key for request ID
type requestIDKey struct{}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// GenerateRequestID generates a new request ID
func GenerateRequestID() string {
	return uuid.New().String()
}

// WithNewRequestID adds a new request ID to the context
func WithNewRequestID(ctx context.Context) context.Context {
	return WithRequestID(ctx, GenerateRequestID())
}
