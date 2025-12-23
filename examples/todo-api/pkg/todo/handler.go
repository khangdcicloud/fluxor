package todo

import (
	"strconv"
	"time"

	"github.com/fluxorio/fluxor/examples/todo-api/pkg/auth"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

type TodoHandler struct {
	EventBus core.EventBus
}

func (h *TodoHandler) RegisterRoutes(router *web.FastRouter) {
	// Protected routes
	authMw := auth.AuthMiddleware()
	
	// Create middleware chain manually since fast router is simple
	// In a real app we'd have Router.Group()
	
	router.POSTFast("/api/todos", h.protect(authMw, h.Create))
	router.GETFast("/api/todos", h.protect(authMw, h.List))
	router.GETFast("/api/todos/:id", h.protect(authMw, h.Get))
	router.PUTFast("/api/todos/:id", h.protect(authMw, h.Update))
	router.DELETEFast("/api/todos/:id", h.protect(authMw, h.Delete))
}

// protect wraps a handler with auth middleware
func (h *TodoHandler) protect(mw web.FastMiddleware, handler web.FastRequestHandler) web.FastRequestHandler {
	return mw(handler)
}

func (h *TodoHandler) Create(ctx *web.FastRequestContext) error {
	userID, _ := strconv.Atoi(ctx.Params["user_id"])
	
	var req struct {
		Title string `json:"title"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]string{"error": "invalid json"})
	}

	payload := map[string]interface{}{
		"user_id": userID,
		"title":   req.Title,
	}

	reply, err := h.EventBus.Request("todo.create", payload, 5*time.Second)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	var res interface{}
	reply.DecodeBody(&res)
	return ctx.JSON(201, res)
}

func (h *TodoHandler) List(ctx *web.FastRequestContext) error {
	userID, _ := strconv.Atoi(ctx.Params["user_id"])

	payload := map[string]interface{}{
		"user_id": userID,
	}

	reply, err := h.EventBus.Request("todo.list", payload, 5*time.Second)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	var res interface{}
	reply.DecodeBody(&res)
	return ctx.JSON(200, res)
}

func (h *TodoHandler) Get(ctx *web.FastRequestContext) error {
	userID, _ := strconv.Atoi(ctx.Params["user_id"])
	id, _ := strconv.Atoi(ctx.Param("id"))

	payload := map[string]interface{}{
		"user_id": userID,
		"id":      id,
	}

	reply, err := h.EventBus.Request("todo.get", payload, 5*time.Second)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	var res map[string]interface{}
	reply.DecodeBody(&res)
	
	if _, ok := res["error"]; ok {
		return ctx.JSON(404, res)
	}

	return ctx.JSON(200, res)
}

func (h *TodoHandler) Update(ctx *web.FastRequestContext) error {
	userID, _ := strconv.Atoi(ctx.Params["user_id"])
	id, _ := strconv.Atoi(ctx.Param("id"))

	var req struct {
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	}
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]string{"error": "invalid json"})
	}

	payload := map[string]interface{}{
		"user_id":   userID,
		"id":        id,
		"title":     req.Title,
		"completed": req.Completed,
	}

	reply, err := h.EventBus.Request("todo.update", payload, 5*time.Second)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	var res map[string]interface{}
	reply.DecodeBody(&res)

	if _, ok := res["error"]; ok {
		return ctx.JSON(404, res)
	}

	return ctx.JSON(200, res)
}

func (h *TodoHandler) Delete(ctx *web.FastRequestContext) error {
	userID, _ := strconv.Atoi(ctx.Params["user_id"])
	id, _ := strconv.Atoi(ctx.Param("id"))

	payload := map[string]interface{}{
		"user_id": userID,
		"id":      id,
	}

	reply, err := h.EventBus.Request("todo.delete", payload, 5*time.Second)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	var res map[string]interface{}
	reply.DecodeBody(&res)

	if _, ok := res["error"]; ok {
		return ctx.JSON(404, res)
	}

	return ctx.JSON(200, res)
}
