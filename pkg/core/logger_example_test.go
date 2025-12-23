package core_test

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core"
)

// demoLoggerWithFields demonstrates adding structured fields to logs.
// This is a documentation example, not a runnable test.
func demoLoggerWithFields() {
	logger := core.NewDefaultLogger()

	// Add structured fields
	loggerWithFields := logger.WithFields(map[string]interface{}{
		"user_id": "123",
		"action":  "login",
		"ip":      "192.168.1.1",
	})

	// Log with fields included
	loggerWithFields.Info("User logged in")
	// Outputs: [INFO] 2025/12/23 15:05:00 User logged in map[action:login ip:192.168.1.1 user_id:123]
}

// demoLoggerWithContext demonstrates logging with request context.
// This is a documentation example, not a runnable test.
func demoLoggerWithContext() {
	logger := core.NewDefaultLogger()

	// Create context with request ID
	requestID := core.GenerateRequestID()
	ctx := core.WithRequestID(context.Background(), requestID)

	// Create logger with context (automatically extracts request ID)
	loggerWithContext := logger.WithContext(ctx)

	// Log with request ID included
	loggerWithContext.Info("Request processed")
	// Outputs: [INFO] Request processed map[request_id:...]
}

// demoNewJSONLogger demonstrates structured JSON logging.
// This is a documentation example, not a runnable test.
func demoNewJSONLogger() {
	// Create JSON logger for structured logging
	logger := core.NewJSONLogger()

	// Add fields
	loggerWithFields := logger.WithFields(map[string]interface{}{
		"service": "user-service",
		"version": "1.0.0",
	})

	// Log messages will be in JSON format
	loggerWithFields.WithContext(context.Background()).Info("Service started")
	// Outputs: {"timestamp":"...","level":"INFO","message":"Service started","fields":{"service":"user-service","version":"1.0.0"}}
}

// demoLoggerEnterpriseUsage demonstrates enterprise-grade logging patterns.
// This is a documentation example, not a runnable test.
func demoLoggerEnterpriseUsage() {
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

	// Outputs:
	// {"timestamp":"...","level":"INFO","message":"Request received","fields":{"endpoint":"/api/users","method":"GET","request_id":"..."}}
	// {"timestamp":"...","level":"INFO","message":"Request completed","fields":{"endpoint":"/api/users","method":"GET","request_id":"...","status":"success","user_id":"user-123"}}
}

// demoLoggerErrorLogging demonstrates error logging with context.
// This is a documentation example, not a runnable test.
func demoLoggerErrorLogging() {
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

	// Outputs: [ERROR] Validation failed map[error_code:VALIDATION_ERROR field:email request_id:... value:invalid-email]
}

// Ensure demo functions are used to avoid unused function warnings
var _ = demoLoggerWithFields
var _ = demoLoggerWithContext
var _ = demoNewJSONLogger
var _ = demoLoggerEnterpriseUsage
var _ = demoLoggerErrorLogging

