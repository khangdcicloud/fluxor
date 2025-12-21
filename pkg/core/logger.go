package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// Logger provides structured logging capabilities
// This abstraction allows swapping logging implementations
type Logger interface {
	// Error logs an error message
	Error(args ...interface{})
	
	// Errorf logs a formatted error message
	Errorf(format string, args ...interface{})
	
	// Warn logs a warning message
	Warn(args ...interface{})
	
	// Warnf logs a formatted warning message
	Warnf(format string, args ...interface{})
	
	// Info logs an informational message
	Info(args ...interface{})
	
	// Infof logs a formatted informational message
	Infof(format string, args ...interface{})
	
	// Debug logs a debug message
	Debug(args ...interface{})
	
	// Debugf logs a formatted debug message
	Debugf(format string, args ...interface{})
	
	// WithFields returns a new logger with structured fields
	// This enables structured logging with key-value pairs
	WithFields(fields map[string]interface{}) Logger
	
	// WithContext returns a new logger with context values
	// Extracts request ID and other context values automatically
	WithContext(ctx context.Context) Logger
}

// LoggerConfig configures logger behavior
type LoggerConfig struct {
	// JSONOutput enables JSON structured output
	JSONOutput bool
	// Level sets the minimum log level (DEBUG, INFO, WARN, ERROR)
	Level string
}

// defaultLogger implements Logger using Go's standard log package
// Can be swapped with other logging implementations (e.g., structured loggers)
type defaultLogger struct {
	errorLogger *log.Logger
	warnLogger  *log.Logger
	infoLogger  *log.Logger
	debugLogger *log.Logger
	config      LoggerConfig
	fields      map[string]interface{} // Structured fields
}

// NewDefaultLogger creates a new default logger implementation
func NewDefaultLogger() Logger {
	return NewLogger(LoggerConfig{
		JSONOutput: false,
		Level:      "DEBUG",
	})
}

// NewLogger creates a new logger with configuration
func NewLogger(config LoggerConfig) Logger {
	return &defaultLogger{
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
		warnLogger:  log.New(os.Stderr, "[WARN] ", log.LstdFlags|log.Lshortfile),
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lshortfile),
		debugLogger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		config:      config,
		fields:      make(map[string]interface{}),
	}
}

// NewJSONLogger creates a logger with JSON output enabled
func NewJSONLogger() Logger {
	return NewLogger(LoggerConfig{
		JSONOutput: true,
		Level:      "DEBUG",
	})
}

// logEntry represents a structured log entry
type logEntry struct {
	Timestamp string                 `json:"timestamp,omitempty"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// log writes a log entry with structured fields
func (l *defaultLogger) log(level string, logger *log.Logger, message string) {
	if l.config.JSONOutput {
		entry := logEntry{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Level:     level,
			Message:   message,
		}
		if len(l.fields) > 0 {
			entry.Fields = make(map[string]interface{})
			for k, v := range l.fields {
				entry.Fields[k] = v
			}
		}
		jsonData, err := json.Marshal(entry)
		if err == nil {
			logger.Output(3, string(jsonData))
		} else {
			// Fallback to plain text if JSON marshal fails
			logger.Output(3, fmt.Sprintf("[%s] %s %v", level, message, l.fields))
		}
	} else {
		// Plain text output with fields appended
		if len(l.fields) > 0 {
			logger.Output(3, fmt.Sprintf("%s %v", message, l.fields))
		} else {
			logger.Output(3, message)
		}
	}
}

// Error logs an error message
func (l *defaultLogger) Error(args ...interface{}) {
	l.log("ERROR", l.errorLogger, fmt.Sprint(args...))
}

// Errorf logs a formatted error message
func (l *defaultLogger) Errorf(format string, args ...interface{}) {
	l.log("ERROR", l.errorLogger, fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func (l *defaultLogger) Warn(args ...interface{}) {
	l.log("WARN", l.warnLogger, fmt.Sprint(args...))
}

// Warnf logs a formatted warning message
func (l *defaultLogger) Warnf(format string, args ...interface{}) {
	l.log("WARN", l.warnLogger, fmt.Sprintf(format, args...))
}

// Info logs an informational message
func (l *defaultLogger) Info(args ...interface{}) {
	l.log("INFO", l.infoLogger, fmt.Sprint(args...))
}

// Infof logs a formatted informational message
func (l *defaultLogger) Infof(format string, args ...interface{}) {
	l.log("INFO", l.infoLogger, fmt.Sprintf(format, args...))
}

// Debug logs a debug message
func (l *defaultLogger) Debug(args ...interface{}) {
	l.log("DEBUG", l.debugLogger, fmt.Sprint(args...))
}

// Debugf logs a formatted debug message
func (l *defaultLogger) Debugf(format string, args ...interface{}) {
	l.log("DEBUG", l.debugLogger, fmt.Sprintf(format, args...))
}

// WithFields returns a new logger with structured fields
// Fields are included in all subsequent log entries
func (l *defaultLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	// Copy existing fields
	for k, v := range l.fields {
		newFields[k] = v
	}
	// Merge new fields (new fields override existing ones)
	for k, v := range fields {
		newFields[k] = v
	}
	return &defaultLogger{
		errorLogger: l.errorLogger,
		warnLogger:  l.warnLogger,
		infoLogger:  l.infoLogger,
		debugLogger: l.debugLogger,
		config:      l.config,
		fields:      newFields,
	}
}

// WithContext returns a new logger with context values
// Automatically extracts request ID and other context values
func (l *defaultLogger) WithContext(ctx context.Context) Logger {
	fields := make(map[string]interface{})
	
	// Extract request ID from context
	if requestID := GetRequestID(ctx); requestID != "" {
		fields["request_id"] = requestID
	}
	
	// Copy existing fields
	for k, v := range l.fields {
		fields[k] = v
	}
	
	return &defaultLogger{
		errorLogger: l.errorLogger,
		warnLogger:  l.warnLogger,
		infoLogger:  l.infoLogger,
		debugLogger: l.debugLogger,
		config:      l.config,
		fields:      fields,
	}
}

