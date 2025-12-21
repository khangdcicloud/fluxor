package security

import (
	"fmt"

	"github.com/fluxorio/fluxor/pkg/web"
)

// HeadersConfig configures security headers
type HeadersConfig struct {
	// HSTS (HTTP Strict Transport Security)
	HSTS           bool
	HSTSMaxAge     int // in seconds, default 31536000 (1 year)
	HSTSIncludeSub bool

	// CSP (Content Security Policy)
	CSP string

	// X-Frame-Options
	XFrameOptions string // DENY, SAMEORIGIN, or ALLOW-FROM uri

	// X-Content-Type-Options
	XContentTypeOptions bool // nosniff

	// X-XSS-Protection
	XXSSProtection string // 1; mode=block

	// Referrer-Policy
	ReferrerPolicy string // no-referrer, no-referrer-when-downgrade, origin, etc.

	// Permissions-Policy (formerly Feature-Policy)
	PermissionsPolicy string

	// Custom headers
	CustomHeaders map[string]string
}

// DefaultHeadersConfig returns a default security headers configuration
func DefaultHeadersConfig() HeadersConfig {
	return HeadersConfig{
		HSTS:                true,
		HSTSMaxAge:          31536000, // 1 year
		HSTSIncludeSub:      true,
		XContentTypeOptions: true,
		XXSSProtection:      "1; mode=block",
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		XFrameOptions:       "DENY",
		CustomHeaders:       make(map[string]string),
	}
}

// Headers middleware adds security headers to responses
func Headers(config HeadersConfig) web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// HSTS
			if config.HSTS {
				hstsValue := "max-age="
				if config.HSTSMaxAge > 0 {
					hstsValue += fmt.Sprintf("%d", config.HSTSMaxAge)
				} else {
					hstsValue += "31536000"
				}
				if config.HSTSIncludeSub {
					hstsValue += "; includeSubDomains"
				}
				ctx.RequestCtx.Response.Header.Set("Strict-Transport-Security", hstsValue)
			}

			// CSP
			if config.CSP != "" {
				ctx.RequestCtx.Response.Header.Set("Content-Security-Policy", config.CSP)
			}

			// X-Frame-Options
			if config.XFrameOptions != "" {
				ctx.RequestCtx.Response.Header.Set("X-Frame-Options", config.XFrameOptions)
			}

			// X-Content-Type-Options
			if config.XContentTypeOptions {
				ctx.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
			}

			// X-XSS-Protection
			if config.XXSSProtection != "" {
				ctx.RequestCtx.Response.Header.Set("X-XSS-Protection", config.XXSSProtection)
			}

			// Referrer-Policy
			if config.ReferrerPolicy != "" {
				ctx.RequestCtx.Response.Header.Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// Permissions-Policy
			if config.PermissionsPolicy != "" {
				ctx.RequestCtx.Response.Header.Set("Permissions-Policy", config.PermissionsPolicy)
			}

			// Custom headers
			for key, value := range config.CustomHeaders {
				ctx.RequestCtx.Response.Header.Set(key, value)
			}

			return next(ctx)
		}
	}
}

