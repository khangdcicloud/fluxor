package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/fluxorio/fluxor/examples/todo-api/pkg/auth"
	"github.com/fluxorio/fluxor/examples/todo-api/pkg/todo"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/fx"
	"github.com/fluxorio/fluxor/pkg/observability/prometheus"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware"
	"github.com/fluxorio/fluxor/pkg/web/middleware/security"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize DB Pool
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		dbDSN = "postgres://todo:secret@localhost:5432/todo_db"
	}
	dbPool, err := pgxpool.New(ctx, dbDSN)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Create Fluxor app
	app, err := fx.New(ctx,
		fx.Provide(fx.NewValueProvider(dbPool)),
		fx.Invoke(fx.NewInvoker(setupApplication)),
	)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}

	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start app: %v", err)
	}

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down...")

	if err := app.Stop(); err != nil {
		log.Fatalf("Error stopping app: %v", err)
	}
}

func setupApplication(deps map[reflect.Type]interface{}) error {
	vertx := deps[reflect.TypeOf((*core.GoCMD)(nil)).Elem()].(core.GoCMD)
	dbPool := deps[reflect.TypeOf((*pgxpool.Pool)(nil))].(*pgxpool.Pool)

	// Deploy Verticles
	todoVerticle := &todo.TodoVerticle{DB: dbPool}
	if _, err := vertx.DeployVerticle(todoVerticle); err != nil {
		return fmt.Errorf("failed to deploy todo verticle: %w", err)
	}

	authVerticle := &auth.AuthVerticle{DB: dbPool}
	if _, err := vertx.DeployVerticle(authVerticle); err != nil {
		return fmt.Errorf("failed to deploy auth verticle: %w", err)
	}

	// Setup Server
	// Target 67% utilization for stability
	config := web.CCUBasedConfigWithUtilization(":8080", 5000, 67)
	server := web.NewFastHTTPServer(vertx, config)
	router := server.FastRouter()

	// Middleware Chain
	router.UseFast(middleware.Recovery(middleware.DefaultRecoveryConfig()))
	router.UseFast(middleware.Logging(middleware.DefaultLoggingConfig()))
	router.UseFast(prometheus.FastHTTPMetricsMiddleware())
	router.UseFast(security.CORS(security.CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
	}))
	// Rate limit: 1000 req/min
	router.UseFast(security.RateLimit(security.RateLimitConfig{
		RequestsPerMinute: 1000,
	}))

	// Routes
	authHandler := &auth.AuthHandler{EventBus: vertx.EventBus()}
	authHandler.RegisterRoutes(router)

	todoHandler := &todo.TodoHandler{EventBus: vertx.EventBus()}
	todoHandler.RegisterRoutes(router)

	// Observability Routes
	metricsHandler := fasthttpadaptor.NewFastHTTPHandler(promhttp.Handler())
	router.GETFast("/metrics", func(ctx *web.FastRequestContext) error {
		metricsHandler(ctx.RequestCtx)
		return nil
	})

	router.GETFast("/ready", func(ctx *web.FastRequestContext) error {
		metrics := server.Metrics()
		// Ready if queue < 90%, CCU < 90%, and DB is reachable
		dbErr := dbPool.Ping(ctx.Context())
		ready := metrics.QueueUtilization < 90 && metrics.CCUUtilization < 90 && dbErr == nil

		status := 200
		if !ready {
			status = 503
		}
		return ctx.JSON(status, map[string]interface{}{
			"ready": ready,
			"db":    dbErr == nil,
		})
	})

	router.GETFast("/live", func(ctx *web.FastRequestContext) error {
		return ctx.JSON(200, map[string]interface{}{"status": "up"})
	})

	// Start Metrics Updater
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			prometheus.UpdateServerMetrics(server)
		}
	}()

	// Start Server
	go func() {
		log.Printf("Starting Todo API on %s", config.Addr)
		if err := server.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	return nil
}
