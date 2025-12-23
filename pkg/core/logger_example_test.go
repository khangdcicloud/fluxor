package core_test

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core"
)

func ExampleLogger_WithFields() {
	logger := core.NewDefaultLogger()

	// Add structured fields
	loggerWithFields := logger.WithFields(map[string]interface{}{
		"user_id": "123",
		"action":  "login",
		"ip":      "192.168.1.1",
	})

	// Log with fields included
	loggerWithFields.Info("User logged in")
	// Output: [INFO] User logged in map[action:login ip:192.168.1.1 user_id:123]
}

func ExampleLogger_WithContext() {
	logger := core.NewDefaultLogger()

	// Create context with request ID
	requestID := core.GenerateRequestID()
	ctx := core.WithRequestID(context.Background(), requestID)

	// Create logger with context (automatically extracts request ID)
	loggerWithContext := logger.WithContext(ctx)

	// Log with request ID included
	loggerWithContext.Info("Request processed")
	// Output: [INFO] Request processed map[request_id:...]
}

func ExampleNewJSONLogger() {
	// Create JSON logger for structured logging
	logger := core.NewJSONLogger()

	// Add fields
	loggerWithFields := logger.WithFields(map[string]interface{}{
		"service": "user-service",
		"version": "1.0.0",
	})

	// Log messages will be in JSON format
	loggerWithFields.WithContext(context.Background()).Info("Service started")
	// Output: {"timestamp":"2025-12-20T23:20:08Z","level":"INFO","message":"Service started","fields":{"service":"user-service","version":"1.0.0"}}
}

func ExampleLogger_enterpriseUsage() {
	// Enterprise example: Structured logging with context
	logger := core.NewJSONLogger()

	// In HTTP handler
	requestID := core.GenerateRequestID()
	ctx := core.WithRequestID(context.Background(), requestID)

	// Create logger with request context
	reqLogger := logger.WithContext(ctx).WithFields(map[string]interface{}{
		"endpoint": "/api/users",
		"method":   "GET",
	})

	// Log request
	reqLogger.Info("Request received")

	// Process request...
	userID := "user-123"

	// Log with additional context
	reqLogger.WithFields(map[string]interface{}{
		"user_id": userID,
		"status":  "success",
	}).Info("Request completed")

	// Output:
	// {"timestamp":"...","level":"INFO","message":"Request received","fields":{"endpoint":"/api/users","method":"GET","request_id":"..."}}
	// {"timestamp":"...","level":"INFO","message":"Request completed","fields":{"endpoint":"/api/users","method":"GET","request_id":"...","status":"success","user_id":"user-123"}}
}

func ExampleLogger_errorLogging() {
	logger := core.NewDefaultLogger()

	// Error logging with context
	ctx := core.WithRequestID(context.Background(), core.GenerateRequestID())
	loggerWithContext := logger.WithContext(ctx)

	// Log error with fields
	loggerWithContext.WithFields(map[string]interface{}{
		"error_code": "VALIDATION_ERROR",
		"field":      "email",
		"value":      "invalid-email",
	}).Error("Validation failed")

	// Output: [ERROR] Validation failed map[error_code:VALIDATION_ERROR field:email request_id:... value:invalid-email]
}
