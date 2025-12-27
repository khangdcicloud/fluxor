package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	handler := func(ctx *fasthttp.RequestCtx) {
		path := string(ctx.Path())
		ctx.SetContentType("application/json")

		switch path {
		case "/health":
			json.NewEncoder(ctx).Encode(map[string]interface{}{"status": "ok"})
		case "/ready":
			json.NewEncoder(ctx).Encode(map[string]interface{}{"ready": true})
		case "/api/status":
			json.NewEncoder(ctx).Encode(map[string]interface{}{
				"status": "ok",
				"time":   time.Now().Unix(),
			})
		case "/api/echo":
			var data map[string]interface{}
			if err := json.Unmarshal(ctx.PostBody(), &data); err != nil {
				ctx.SetStatusCode(400)
				json.NewEncoder(ctx).Encode(map[string]interface{}{"error": "invalid json"})
				return
			}
			json.NewEncoder(ctx).Encode(map[string]interface{}{"echo": data})
		default:
			ctx.SetStatusCode(404)
			json.NewEncoder(ctx).Encode(map[string]interface{}{"error": "not found"})
		}
	}

	server := &fasthttp.Server{
		Handler:         handler,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    10 * time.Second,
		MaxConnsPerIP:   100000,
		ReadBufferSize:  8192,
		WriteBufferSize: 8192,
	}

	fmt.Println("Starting simple FastHTTP server on :8080")
	if err := server.ListenAndServe(":8080"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
