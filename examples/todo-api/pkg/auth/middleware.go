package auth

import (
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
)

// AuthMiddleware creates a middleware that validates JWT tokens
func AuthMiddleware() web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			authHeader := string(ctx.RequestCtx.Request.Header.Peek("Authorization"))
			if authHeader == "" {
				return ctx.JSON(401, map[string]string{"error": "missing authorization header"})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				return ctx.JSON(401, map[string]string{"error": "invalid authorization format"})
			}

			claims, err := ValidateToken(parts[1])
			if err != nil {
				return ctx.JSON(401, map[string]string{"error": "invalid token"})
			}

			// Store user ID in params for handlers to access
			ctx.Params["user_id"] = fmt.Sprintf("%d", claims.UserID)
			
			return next(ctx)
		}
	}
}
