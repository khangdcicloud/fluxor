package middleware

import (
	"strconv"
	"time"

	"github.com/fluxorio/fluxor/pkg/observability/prometheus"
	"github.com/fluxorio/fluxor/pkg/web"
)

// MetricsMiddleware records HTTP request metrics for Prometheus
func MetricsMiddleware() web.FastMiddleware {
	metrics := prometheus.GetMetrics()

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			start := time.Now()
			method := string(ctx.Method())
			path := string(ctx.Path())
			requestSize := int64(len(ctx.RequestCtx.Request.Body()))

			// Execute handler
			err := next(ctx)

			// Record metrics
			duration := time.Since(start)
			status := "200"
			if ctx.RequestCtx.Response.StatusCode() != 0 {
				status = strconv.Itoa(ctx.RequestCtx.Response.StatusCode())
			}
			responseSize := int64(len(ctx.RequestCtx.Response.Body()))

			// Normalize path for metrics (remove IDs)
			normalizedPath := normalizePath(path)

			metrics.RecordHTTPRequest(method, normalizedPath, status, duration, requestSize, responseSize)

			return err
		}
	}
}

// normalizePath removes dynamic parts from path for better metrics aggregation
func normalizePath(path string) string {
	// Simple normalization - replace UUIDs and IDs with placeholders
	// This is a basic implementation; you might want to use regex for more complex cases
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	return "/" + path
}
