package auth

import (
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
)

type AuthHandler struct {
	EventBus core.EventBus
}

func (h *AuthHandler) RegisterRoutes(router *web.FastRouter) {
	// The fast router doesn't implement Group yet, so we register directly
	router.POSTFast("/api/auth/register", h.Register)
	router.POSTFast("/api/auth/login", h.Login)
}

func (h *AuthHandler) Register(ctx *web.FastRequestContext) error {
	var req map[string]interface{}
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]string{"error": "invalid json"})
	}

	reply, err := h.EventBus.Request("auth.register", req, 5*time.Second)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	var res map[string]interface{}
	reply.DecodeBody(&res)

	if _, ok := res["error"]; ok {
		return ctx.JSON(400, res)
	}

	return ctx.JSON(201, res)
}

func (h *AuthHandler) Login(ctx *web.FastRequestContext) error {
	var req map[string]interface{}
	if err := ctx.BindJSON(&req); err != nil {
		return ctx.JSON(400, map[string]string{"error": "invalid json"})
	}

	reply, err := h.EventBus.Request("auth.login", req, 5*time.Second)
	if err != nil {
		return ctx.JSON(500, map[string]string{"error": err.Error()})
	}

	var res map[string]interface{}
	reply.DecodeBody(&res)

	if _, ok := res["error"]; ok {
		return ctx.JSON(401, res)
	}

	return ctx.JSON(200, res)
}
