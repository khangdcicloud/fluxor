package handlers

import (
	"time"

	"github.com/fluxorio/fluxor/examples/todo-api/models"
	"github.com/fluxorio/fluxor/examples/todo-api/services"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	userService *services.UserService
	jwtSecret   string
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(userService *services.UserService, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(ctx *web.FastRequestContext) error {
	var req models.CreateUserRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Username, email, and password are required",
		})
	}

	// Create user
	user, err := h.userService.CreateUser(ctx.Context(), req)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "user_creation_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(201, map[string]interface{}{
		"message": "User created successfully",
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

// Login handles user login and returns JWT token
func (h *AuthHandler) Login(ctx *web.FastRequestContext) error {
	var req models.LoginRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	// Get user
	user, err := h.userService.GetUserByUsername(ctx.Context(), req.Username)
	if err != nil {
		return ctx.JSON(401, map[string]interface{}{
			"error":   "authentication_failed",
			"message": "Invalid username or password",
		})
	}

	// Verify password
	if !h.userService.VerifyPassword(user.PasswordHash, req.Password) {
		return ctx.JSON(401, map[string]interface{}{
			"error":   "authentication_failed",
			"message": "Invalid username or password",
		})
	}

	// Generate JWT token
	token, err := h.generateToken(user.ID, user.Username)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error":   "token_generation_failed",
			"message": "Failed to generate token",
		})
	}

	return ctx.JSON(200, models.LoginResponse{
		Token: token,
		User: models.User{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	})
}

// generateToken generates a JWT token for a user
func (h *AuthHandler) generateToken(userID uuid.UUID, username string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID.String(),
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(ctx *web.FastRequestContext) error {
	// Get user ID from JWT claims (set by JWT middleware)
	userIDStr, err := auth.GetUserID(ctx, "user")
	if err != nil {
		return ctx.JSON(401, map[string]interface{}{
			"error":   "unauthorized",
			"message": "User not authenticated",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_user_id",
			"message": "Invalid user ID",
		})
	}

	user, err := h.userService.GetUserByID(ctx.Context(), userID)
	if err != nil {
		return ctx.JSON(404, map[string]interface{}{
			"error":   "user_not_found",
			"message": "User not found",
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"created_at": user.CreatedAt,
		"updated_at": user.UpdatedAt,
	})
}
