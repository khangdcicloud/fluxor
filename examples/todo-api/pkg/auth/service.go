package auth

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthVerticle struct {
	DB *pgxpool.Pool
}

func (v *AuthVerticle) Start(ctx core.FluxorContext) error {
	eb := ctx.EventBus()

	eb.Consumer("auth.register").Handler(v.handleRegister)
	eb.Consumer("auth.login").Handler(v.handleLogin)

	return nil
}

func (v *AuthVerticle) Stop(ctx core.FluxorContext) error {
	return nil
}

func (v *AuthVerticle) handleRegister(ctx core.FluxorContext, msg core.Message) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := msg.DecodeBody(&req); err != nil {
		return msg.Reply(map[string]interface{}{"error": "invalid request"})
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return msg.Reply(map[string]interface{}{"error": "internal error"})
	}

	var userID int
	err = v.DB.QueryRow(context.Background(),
		"INSERT INTO users (username, password_hash) VALUES ($1, $2) RETURNING id",
		req.Username, string(hash)).Scan(&userID)

	if err != nil {
		return msg.Reply(map[string]interface{}{"error": "username already exists"})
	}

	return msg.Reply(map[string]interface{}{
		"id":       userID,
		"username": req.Username,
	})
}

func (v *AuthVerticle) handleLogin(ctx core.FluxorContext, msg core.Message) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := msg.DecodeBody(&req); err != nil {
		return msg.Reply(map[string]interface{}{"error": "invalid request"})
	}

	var id int
	var hash string
	err := v.DB.QueryRow(context.Background(),
		"SELECT id, password_hash FROM users WHERE username = $1",
		req.Username).Scan(&id, &hash)

	if err == pgx.ErrNoRows {
		return msg.Reply(map[string]interface{}{"error": "invalid credentials"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		return msg.Reply(map[string]interface{}{"error": "invalid credentials"})
	}

	token, err := GenerateToken(id, req.Username)
	if err != nil {
		return msg.Reply(map[string]interface{}{"error": "failed to generate token"})
	}

	return msg.Reply(map[string]interface{}{
		"token": token,
	})
}
