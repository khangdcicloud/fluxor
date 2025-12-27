package handlers

import (
	"strconv"

	"github.com/fluxorio/fluxor/examples/todo-api/models"
	"github.com/fluxorio/fluxor/examples/todo-api/services"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/google/uuid"
)

// TodoHandler handles todo-related requests
type TodoHandler struct {
	todoService *services.TodoService
}

// NewTodoHandler creates a new todo handler
func NewTodoHandler(todoService *services.TodoService) *TodoHandler {
	return &TodoHandler{
		todoService: todoService,
	}
}

// getUserID extracts user ID from JWT claims
func (h *TodoHandler) getUserID(ctx *web.FastRequestContext) (uuid.UUID, error) {
	userIDStr, err := auth.GetUserID(ctx, "user")
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(userIDStr)
}

// CreateTodo handles POST /api/todos
func (h *TodoHandler) CreateTodo(ctx *web.FastRequestContext) error {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return ctx.JSON(401, map[string]interface{}{
			"error":   "unauthorized",
			"message": "User not authenticated",
		})
	}

	var req models.CreateTodoRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	if req.Title == "" {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "validation_error",
			"message": "Title is required",
		})
	}

	todo, err := h.todoService.CreateTodo(ctx.Context(), userID, req)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error":   "creation_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(201, todo)
}

// GetTodo handles GET /api/todos/:id
func (h *TodoHandler) GetTodo(ctx *web.FastRequestContext) error {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return ctx.JSON(401, map[string]interface{}{
			"error":   "unauthorized",
			"message": "User not authenticated",
		})
	}

	todoIDStr := ctx.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_id",
			"message": "Invalid todo ID",
		})
	}

	todo, err := h.todoService.GetTodoByID(ctx.Context(), todoID, userID)
	if err != nil {
		return ctx.JSON(404, map[string]interface{}{
			"error":   "not_found",
			"message": "Todo not found",
		})
	}

	return ctx.JSON(200, todo)
}

// ListTodos handles GET /api/todos
func (h *TodoHandler) ListTodos(ctx *web.FastRequestContext) error {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return ctx.JSON(401, map[string]interface{}{
			"error":   "unauthorized",
			"message": "User not authenticated",
		})
	}

	// Parse query parameters
	page := 1
	pageSize := 20
	var completed *bool

	if pageStr := ctx.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if sizeStr := ctx.Query("page_size"); sizeStr != "" {
		if s, err := strconv.Atoi(sizeStr); err == nil && s > 0 && s <= 100 {
			pageSize = s
		}
	}

	if completedStr := ctx.Query("completed"); completedStr != "" {
		if c, err := strconv.ParseBool(completedStr); err == nil {
			completed = &c
		}
	}

	result, err := h.todoService.ListTodos(ctx.Context(), userID, page, pageSize, completed)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error":   "list_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(200, result)
}

// UpdateTodo handles PUT /api/todos/:id
func (h *TodoHandler) UpdateTodo(ctx *web.FastRequestContext) error {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return ctx.JSON(401, map[string]interface{}{
			"error":   "unauthorized",
			"message": "User not authenticated",
		})
	}

	todoIDStr := ctx.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_id",
			"message": "Invalid todo ID",
		})
	}

	var req models.UpdateTodoRequest
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	todo, err := h.todoService.UpdateTodo(ctx.Context(), todoID, userID, req)
	if err != nil {
		return ctx.JSON(500, map[string]interface{}{
			"error":   "update_failed",
			"message": err.Error(),
		})
	}

	return ctx.JSON(200, todo)
}

// DeleteTodo handles DELETE /api/todos/:id
func (h *TodoHandler) DeleteTodo(ctx *web.FastRequestContext) error {
	userID, err := h.getUserID(ctx)
	if err != nil {
		return ctx.JSON(401, map[string]interface{}{
			"error":   "unauthorized",
			"message": "User not authenticated",
		})
	}

	todoIDStr := ctx.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_id",
			"message": "Invalid todo ID",
		})
	}

	if err := h.todoService.DeleteTodo(ctx.Context(), todoID, userID); err != nil {
		return ctx.JSON(404, map[string]interface{}{
			"error":   "not_found",
			"message": "Todo not found",
		})
	}

	return ctx.JSON(200, map[string]interface{}{
		"message": "Todo deleted successfully",
	})
}
