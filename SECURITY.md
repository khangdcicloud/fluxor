# Security in Fluxor

Fluxor provides enterprise-grade security features including authentication, authorization, security headers, CORS, and rate limiting.

## Table of Contents

1. [Authentication](#authentication)
2. [Authorization](#authorization)
3. [Security Headers](#security-headers)
4. [CORS](#cors)
5. [Rate Limiting](#rate-limiting)
6. [Best Practices](#best-practices)

---

## Authentication

Fluxor supports multiple authentication methods: JWT, OAuth2/OIDC, and API keys.

### JWT Authentication

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth"

// Configure JWT middleware
router.UseFast(auth.JWT(auth.JWTConfig{
    SecretKey:  "your-secret-key",
    ClaimsKey:  "user",
    TokenLookup: "header:Authorization",
    AuthScheme: "Bearer",
}))

// Access user claims in handler
router.GETFast("/api/profile", func(ctx *web.FastRequestContext) error {
    claims, err := auth.GetClaims(ctx, "user")
    if err != nil {
        return ctx.JSON(401, map[string]string{"error": "unauthorized"})
    }
    
    userID, _ := auth.GetUserID(ctx, "user")
    
    return ctx.JSON(200, map[string]interface{}{
        "user_id": userID,
        "claims":  claims,
    })
})
```

### OAuth2/OIDC Authentication

```go
// Configure OAuth2 middleware
router.UseFast(auth.OAuth2(auth.OAuth2Config{
    IntrospectionURL: "https://auth.example.com/oauth2/introspect",
    ClientID:         "your-client-id",
    ClientSecret:     "your-client-secret",
    ClaimsKey:        "user",
}))

// Access user claims in handler
router.GETFast("/api/profile", func(ctx *web.FastRequestContext) error {
    claims := ctx.Get("user").(map[string]interface{})
    return ctx.JSON(200, claims)
})
```

### API Key Authentication

```go
// Simple API key validator
validKeys := map[string]map[string]interface{}{
    "api-key-123": {
        "user_id": "123",
        "roles":   []string{"api_user"},
    },
}

router.UseFast(auth.APIKey(auth.APIKeyConfig{
    ValidateKey: auth.SimpleAPIKeyValidator(validKeys),
    KeyLookup:   "header:X-API-Key",
    ClaimsKey:   "user",
}))

// Custom API key validator
router.UseFast(auth.APIKey(auth.APIKeyConfig{
    ValidateKey: func(key string) (map[string]interface{}, error) {
        // Validate against database or external service
        user, err := validateAPIKey(key)
        if err != nil {
            return nil, err
        }
        return map[string]interface{}{
            "user_id": user.ID,
            "roles":   user.Roles,
        }, nil
    },
}))
```

---

## Authorization

Fluxor provides Role-Based Access Control (RBAC) middleware.

### Require Role

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/auth"

// Require specific role
router.GETFast("/admin/users", 
    auth.RequireRole("admin"),
    func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, users)
    },
)

// Require any of multiple roles
router.GETFast("/moderator/content",
    auth.RequireAnyRole("admin", "moderator"),
    func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, content)
    },
)

// Require all roles
router.GETFast("/super-admin",
    auth.RequireAllRoles("admin", "super_admin"),
    func(ctx *web.FastRequestContext) error {
        return ctx.JSON(200, data)
    },
)
```

### Custom Authorization

```go
// Custom authorization middleware
func RequirePermission(permission string) web.FastMiddleware {
    return func(next web.FastRequestHandler) web.FastRequestHandler {
        return func(ctx *web.FastRequestContext) error {
            user := ctx.Get("user").(*auth.User)
            
            if !user.HasPermission(permission) {
                ctx.RequestCtx.SetStatusCode(403)
                return ctx.JSON(403, map[string]string{
                    "error": "forbidden",
                    "message": "insufficient permissions",
                })
            }
            
            return next(ctx)
        }
    }
}
```

---

## Security Headers

Fluxor provides security headers middleware to protect against common attacks.

### Basic Security Headers

```go
import "github.com/fluxorio/fluxor/pkg/web/middleware/security"

// Default security headers
router.UseFast(security.Headers(security.DefaultHeadersConfig()))

// Custom security headers
router.UseFast(security.Headers(security.HeadersConfig{
    HSTS:                true,
    HSTSMaxAge:          31536000, // 1 year
    HSTSIncludeSub:      true,
    CSP:                 "default-src 'self'; script-src 'self' 'unsafe-inline'",
    XFrameOptions:       "DENY",
    XContentTypeOptions: true,
    XXSSProtection:      "1; mode=block",
    ReferrerPolicy:      "strict-origin-when-cross-origin",
    PermissionsPolicy:   "geolocation=(), microphone=()",
    CustomHeaders: map[string]string{
        "X-Custom-Header": "value",
    },
}))
```

### Security Headers Explained

- **HSTS (HTTP Strict Transport Security)**: Forces HTTPS connections
- **CSP (Content Security Policy)**: Prevents XSS attacks
- **X-Frame-Options**: Prevents clickjacking
- **X-Content-Type-Options**: Prevents MIME type sniffing
- **X-XSS-Protection**: Enables browser XSS protection
- **Referrer-Policy**: Controls referrer information
- **Permissions-Policy**: Controls browser features

---

## CORS

Fluxor provides CORS middleware for cross-origin requests.

### Basic CORS

```go
// Default CORS (allows all origins)
router.UseFast(security.CORS(security.DefaultCORSConfig()))

// Custom CORS
router.UseFast(security.CORS(security.CORSConfig{
    AllowedOrigins:   []string{"https://example.com", "https://app.example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID"},
    ExposedHeaders:   []string{"X-Request-ID", "X-Total-Count"},
    AllowCredentials: true,
    MaxAge:          86400, // 24 hours
}))
```

### CORS for Specific Routes

```go
// Apply CORS only to API routes
apiRouter := router.Group("/api")
apiRouter.UseFast(security.CORS(security.CORSConfig{
    AllowedOrigins: []string{"https://example.com"},
}))
```

---

## Rate Limiting

Fluxor provides rate limiting middleware to prevent abuse.

### Basic Rate Limiting

```go
// Rate limit: 100 requests per minute per IP
router.UseFast(security.RateLimit(security.RateLimitConfig{
    RequestsPerMinute: 100,
}))

// Rate limit with custom key function
router.UseFast(security.RateLimit(security.RateLimitConfig{
    RequestsPerMinute: 50,
    KeyFunc: func(ctx *web.FastRequestContext) string {
        // Use user ID instead of IP
        userID := ctx.Get("user_id")
        if userID != nil {
            return userID.(string)
        }
        return ctx.RequestCtx.RemoteIP().String()
    },
}))

// Custom error handler
router.UseFast(security.RateLimit(security.RateLimitConfig{
    RequestsPerMinute: 100,
    OnLimitReached: func(ctx *web.FastRequestContext) error {
        ctx.RequestCtx.SetStatusCode(429)
        return ctx.JSON(429, map[string]string{
            "error": "rate_limit_exceeded",
            "message": "Too many requests, please try again later",
            "retry_after": "60",
        })
    },
}))
```

### Skip Rate Limiting for Specific Paths

```go
// Apply rate limiting globally
router.UseFast(security.RateLimit(security.RateLimitConfig{
    RequestsPerMinute: 100,
}))

// Health checks are automatically excluded from rate limiting
// (handled by middleware skip logic)
```

---

## Complete Security Setup

```go
func setupSecurity(router *web.FastRouter) {
    // Security headers (first)
    router.UseFast(security.Headers(security.DefaultHeadersConfig()))
    
    // CORS
    router.UseFast(security.CORS(security.CORSConfig{
        AllowedOrigins: []string{"https://example.com"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
        AllowCredentials: true,
    }))
    
    // Rate limiting
    router.UseFast(security.RateLimit(security.RateLimitConfig{
        RequestsPerMinute: 100,
    }))
    
    // Authentication (JWT)
    router.UseFast(auth.JWT(auth.JWTConfig{
        SecretKey: os.Getenv("JWT_SECRET"),
        ClaimsKey: "user",
        SkipPaths: []string{"/health", "/ready", "/metrics"},
    }))
    
    // Protected routes
    router.GETFast("/api/users",
        auth.RequireRole("admin"),
        getUserHandler,
    )
}
```

---

## Best Practices

### 1. Always Use Security Headers

```go
// ✅ Good: Security headers on all routes
router.UseFast(security.Headers(security.DefaultHeadersConfig()))

// ❌ Bad: No security headers
```

### 2. Configure CORS Properly

```go
// ✅ Good: Specific origins
router.UseFast(security.CORS(security.CORSConfig{
    AllowedOrigins: []string{"https://example.com"},
}))

// ❌ Bad: Allow all origins in production
router.UseFast(security.CORS(security.CORSConfig{
    AllowedOrigins: []string{"*"}, // Only for development!
}))
```

### 3. Use Rate Limiting

```go
// ✅ Good: Rate limiting enabled
router.UseFast(security.RateLimit(security.RateLimitConfig{
    RequestsPerMinute: 100,
}))

// ❌ Bad: No rate limiting
```

### 4. Validate JWT Tokens

```go
// ✅ Good: Validate JWT with proper secret
router.UseFast(auth.JWT(auth.JWTConfig{
    SecretKey: os.Getenv("JWT_SECRET"), // From environment
}))

// ❌ Bad: Hardcoded secret
router.UseFast(auth.JWT(auth.JWTConfig{
    SecretKey: "hardcoded-secret", // Never do this!
}))
```

### 5. Skip Authentication for Public Endpoints

```go
// ✅ Good: Skip authentication for health checks
router.UseFast(auth.JWT(auth.JWTConfig{
    SecretKey: secret,
    SkipPaths: []string{"/health", "/ready", "/metrics"},
}))

// ❌ Bad: Require authentication for health checks
```

---

## Security Checklist

- [ ] Security headers enabled
- [ ] CORS configured with specific origins
- [ ] Rate limiting enabled
- [ ] JWT/OAuth2 authentication configured
- [ ] RBAC authorization for protected routes
- [ ] Secrets stored in environment variables
- [ ] Health checks excluded from authentication
- [ ] Error messages don't leak sensitive information
- [ ] Request validation enabled
- [ ] HTTPS enforced (via HSTS header)

---

## Summary

Fluxor provides enterprise-grade security:

- ✅ **Authentication**: JWT, OAuth2/OIDC, API keys
- ✅ **Authorization**: Role-Based Access Control (RBAC)
- ✅ **Security Headers**: HSTS, CSP, X-Frame-Options, etc.
- ✅ **CORS**: Cross-Origin Resource Sharing
- ✅ **Rate Limiting**: Token bucket rate limiting

All security features work together to protect your application from common attacks.

