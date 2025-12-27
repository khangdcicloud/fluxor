//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Println("Starting debug server")

	// Test if port 8080 is available
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Port 8080 not available: %v", err)
	}
	ln.Close()
	log.Println("Port 8080 is available")

	ctx := context.Background()
	log.Println("Creating Vertx...")
	vertx := core.NewVertx(ctx)
	log.Println("Vertx created")

	log.Println("Creating server config...")
	config := web.DefaultFastHTTPServerConfig(":8080")
	log.Printf("Config: %+v", config)

	log.Println("Creating FastHTTP server...")
	server := web.NewFastHTTPServer(vertx, config)
	log.Println("Server created")

	router := server.FastRouter()
	log.Println("Router obtained")

	router.GETFast("/health", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"status": "ok"})
	})

	router.GETFast("/ready", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"ready": true})
	})

	router.GETFast("/api/status", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"status": "ok", "time": time.Now().Unix()})
	})

	router.POSTFast("/api/echo", func(ctx *web.FastRequestContext) error {
		var data map[string]interface{}
		if err := ctx.BindJSON(&data); err != nil {
			return ctx.JSON(400, map[string]interface{}{"error": "invalid json"})
		}
		return ctx.JSON(200, map[string]interface{}{"echo": data})
	})
	log.Println("Routes registered")

	server.SetHandler(func(ctx *fasthttp.RequestCtx) {
		reqCtx := &web.FastRequestContext{
			RequestCtx: ctx,
			Vertx:      vertx,
			EventBus:   vertx.EventBus(),
			Params:     make(map[string]string),
		}
		router.ServeFastHTTP(reqCtx)
	})
	log.Println("Handler set")

	fmt.Println("Starting server on :8080...")
	log.Println("About to call server.Start()")

	// Start in goroutine like the main app does
	errCh := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			errCh <- err
		}
	}()

	// Give it time to start
	time.Sleep(2 * time.Second)

	// Check for error
	select {
	case err := <-errCh:
		log.Fatalf("Server.Start() returned error: %v", err)
	default:
		log.Println("Server.Start() running (no error)")
	}

	// Test if the server is actually listening
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
	} else {
		log.Println("Successfully connected to server!")
		conn.Close()
	}

	// Keep running
	select {}
}
