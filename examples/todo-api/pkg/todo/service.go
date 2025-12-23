package todo

import (
	"context"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TodoVerticle struct {
	DB *pgxpool.Pool
}

func (v *TodoVerticle) Start(ctx core.FluxorContext) error {
	eb := ctx.EventBus()

	eb.Consumer("todo.create").Handler(v.createTodo)
	eb.Consumer("todo.list").Handler(v.listTodos)
	eb.Consumer("todo.get").Handler(v.getTodo)
	eb.Consumer("todo.update").Handler(v.updateTodo)
	eb.Consumer("todo.delete").Handler(v.deleteTodo)

	return nil
}

func (v *TodoVerticle) Stop(ctx core.FluxorContext) error {
	return nil
}

func (v *TodoVerticle) createTodo(ctx core.FluxorContext, msg core.Message) error {
	var req struct {
		UserID int    `json:"user_id"`
		Title  string `json:"title"`
	}
	if err := msg.DecodeBody(&req); err != nil {
		return msg.Reply(map[string]interface{}{"error": "invalid request"})
	}

	var todo Todo
	err := v.DB.QueryRow(context.Background(),
		"INSERT INTO todos (user_id, title) VALUES ($1, $2) RETURNING id, user_id, title, completed, created_at",
		req.UserID, req.Title).Scan(&todo.ID, &todo.UserID, &todo.Title, &todo.Completed, &todo.CreatedAt)

	if err != nil {
		return msg.Reply(map[string]interface{}{"error": "failed to create todo"})
	}

	return msg.Reply(todo)
}

func (v *TodoVerticle) listTodos(ctx core.FluxorContext, msg core.Message) error {
	var req struct {
		UserID int `json:"user_id"`
	}
	if err := msg.DecodeBody(&req); err != nil {
		return msg.Reply(map[string]interface{}{"error": "invalid request"})
	}

	rows, err := v.DB.Query(context.Background(),
		"SELECT id, user_id, title, completed, created_at FROM todos WHERE user_id = $1 ORDER BY created_at DESC",
		req.UserID)
	if err != nil {
		return msg.Reply(map[string]interface{}{"error": "failed to list todos"})
	}
	defer rows.Close()

	todos := make([]Todo, 0)
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt); err != nil {
			continue
		}
		todos = append(todos, t)
	}

	return msg.Reply(todos)
}

func (v *TodoVerticle) getTodo(ctx core.FluxorContext, msg core.Message) error {
	var req struct {
		UserID int `json:"user_id"`
		ID     int `json:"id"`
	}
	if err := msg.DecodeBody(&req); err != nil {
		return msg.Reply(map[string]interface{}{"error": "invalid request"})
	}

	var t Todo
	err := v.DB.QueryRow(context.Background(),
		"SELECT id, user_id, title, completed, created_at FROM todos WHERE id = $1 AND user_id = $2",
		req.ID, req.UserID).Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt)

	if err != nil {
		return msg.Reply(map[string]interface{}{"error": "todo not found"})
	}

	return msg.Reply(t)
}

func (v *TodoVerticle) updateTodo(ctx core.FluxorContext, msg core.Message) error {
	var req struct {
		UserID    int    `json:"user_id"`
		ID        int    `json:"id"`
		Title     string `json:"title"`
		Completed bool   `json:"completed"`
	}
	if err := msg.DecodeBody(&req); err != nil {
		return msg.Reply(map[string]interface{}{"error": "invalid request"})
	}

	var t Todo
	err := v.DB.QueryRow(context.Background(),
		"UPDATE todos SET title = $1, completed = $2 WHERE id = $3 AND user_id = $4 RETURNING id, user_id, title, completed, created_at",
		req.Title, req.Completed, req.ID, req.UserID).Scan(&t.ID, &t.UserID, &t.Title, &t.Completed, &t.CreatedAt)

	if err != nil {
		return msg.Reply(map[string]interface{}{"error": "failed to update todo"})
	}

	return msg.Reply(t)
}

func (v *TodoVerticle) deleteTodo(ctx core.FluxorContext, msg core.Message) error {
	var req struct {
		UserID int `json:"user_id"`
		ID     int `json:"id"`
	}
	if err := msg.DecodeBody(&req); err != nil {
		return msg.Reply(map[string]interface{}{"error": "invalid request"})
	}

	cmd, err := v.DB.Exec(context.Background(),
		"DELETE FROM todos WHERE id = $1 AND user_id = $2",
		req.ID, req.UserID)

	if err != nil {
		return msg.Reply(map[string]interface{}{"error": "failed to delete todo"})
	}

	if cmd.RowsAffected() == 0 {
		return msg.Reply(map[string]interface{}{"error": "todo not found"})
	}

	return msg.Reply(map[string]interface{}{"status": "deleted"})
}
