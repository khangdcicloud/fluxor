package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github.com/fluxorio/fluxor/examples/todo-api/handlers"
	"github.com/fluxorio/fluxor/examples/todo-api/middleware"
	"github.com/fluxorio/fluxor/examples/todo-api/services"
	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/db"
	"github.com/fluxorio/fluxor/pkg/fx"
	"github.com/fluxorio/fluxor/pkg/observability/prometheus"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/fluxorio/fluxor/pkg/web/middleware/security"
	"github.com/valyala/fasthttp"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get configuration from environment
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production-min-32-chars"
		log.Println("WARNING: Using default JWT secret. Set JWT_SECRET environment variable in production!")
	}

	dbDSN := os.Getenv("DATABASE_URL")
	if dbDSN == "" {
		dbDSN = "postgres://todo_user:todo_password@localhost:5432/todo_db?sslmode=disable"
		log.Println("Using default database URL. Set DATABASE_URL environment variable to customize.")
	}

	// Create database pool
	poolConfig := db.DefaultPoolConfig(dbDSN, "postgres")
	poolConfig.MaxOpenConns = 25
	poolConfig.MaxIdleConns = 5
	poolConfig.ConnMaxLifetime = 5 * time.Minute
	poolConfig.ConnMaxIdleTime = 10 * time.Minute

	dbPool, err := db.NewPool(poolConfig)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}
	defer dbPool.Close()

	// Run migrations
	if err := runMigrations(dbPool.DB()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create Fluxor application with dependency injection
	app, err := fx.New(ctx,
		fx.Provide(fx.NewValueProvider("todo-api-config")),
		fx.Invoke(fx.NewInvoker(setupApplication(dbPool.DB(), jwtSecret))),
	)
	if err != nil {
		log.Fatalf("Failed to create Fluxor app: %v", err)
	}

	// Start the application
	if err := app.Start(); err != nil {
		log.Fatalf("Failed to start Fluxor app: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Todo API server started successfully!")
	log.Println("API endpoints:")
	log.Println("  POST   /api/auth/register - Register a new user")
	log.Println("  POST   /api/auth/login - Login and get JWT token")
	log.Println("  GET    /api/auth/profile - Get current user profile")
	log.Println("  GET    /api/todos - List todos (paginated)")
	log.Println("  POST   /api/todos - Create a new todo")
	log.Println("  GET    /api/todos/:id - Get a todo by ID")
	log.Println("  PUT    /api/todos/:id - Update a todo")
	log.Println("  DELETE /api/todos/:id - Delete a todo")
	log.Println("  GET    /metrics - Prometheus metrics")
	log.Println("  GET    /health - Health check")
	log.Println("  GET    /ready - Readiness check")

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	if err := app.Stop(); err != nil {
		log.Fatalf("Error stopping app: %v", err)
	}
}

func setupApplication(database *sql.DB, jwtSecret string) func(map[reflect.Type]interface{}) error {
	return func(deps map[reflect.Type]interface{}) error {
		vertx := deps[reflect.TypeOf((*core.GoCMD)(nil)).Elem()].(core.GoCMD)

		// Create services
		userService := services.NewUserService(database)
		todoService := services.NewTodoService(database)

		// Create handlers
		authHandler := handlers.NewAuthHandler(userService, jwtSecret)
		todoHandler := handlers.NewTodoHandler(todoService)

		// Create FastHTTP server with CCU-based backpressure
		// Configure for 60% utilization (leaves 40% headroom for spikes)
		maxCCU := 5000
		utilizationPercent := 60
		config := web.CCUBasedConfigWithUtilization(":8080", maxCCU, utilizationPercent)
		server := web.NewFastHTTPServer(vertx, config)

		// Setup routes
		router := server.FastRouter()

		// Create metrics middleware
		metricsMiddleware := middleware.MetricsMiddleware()

		// Public routes (no auth required) - wrap with metrics middleware
		router.POSTFast("/api/auth/register", metricsMiddleware(authHandler.Register))
		router.POSTFast("/api/auth/login", metricsMiddleware(authHandler.Login))

		// Protected routes (require JWT auth)
		// Apply JWT middleware to protected routes
		jwtConfig := auth.DefaultJWTConfig(jwtSecret)
		jwtConfig.SkipPaths = []string{"/api/auth/register", "/api/auth/login", "/health", "/ready", "/metrics"}
		jwtMiddleware := auth.JWT(jwtConfig)

		// Apply rate limiting to protected routes
		rateLimitConfig := security.DefaultRateLimitConfig()
		rateLimitConfig.RequestsPerMinute = 100 // 100 requests per minute per IP
		rateLimitMiddleware := security.RateLimit(rateLimitConfig)

		// Protected auth routes - wrap with metrics, jwt, and rate limit middleware
		router.GETFast("/api/auth/profile", metricsMiddleware(jwtMiddleware(rateLimitMiddleware(authHandler.GetProfile))))

		// Protected todo routes - wrap with metrics, jwt, and rate limit middleware
		router.GETFast("/api/todos", metricsMiddleware(jwtMiddleware(rateLimitMiddleware(todoHandler.ListTodos))))
		router.POSTFast("/api/todos", metricsMiddleware(jwtMiddleware(rateLimitMiddleware(todoHandler.CreateTodo))))
		router.GETFast("/api/todos/:id", metricsMiddleware(jwtMiddleware(rateLimitMiddleware(todoHandler.GetTodo))))
		router.PUTFast("/api/todos/:id", metricsMiddleware(jwtMiddleware(rateLimitMiddleware(todoHandler.UpdateTodo))))
		router.DELETEFast("/api/todos/:id", metricsMiddleware(jwtMiddleware(rateLimitMiddleware(todoHandler.DeleteTodo))))

		// Health and metrics endpoints - wrap with metrics middleware
		router.GETFast("/health", metricsMiddleware(func(ctx *web.FastRequestContext) error {
			return ctx.JSON(200, map[string]interface{}{
				"status": "UP",
				"service": "todo-api",
			})
		}))

		router.GETFast("/ready", metricsMiddleware(func(ctx *web.FastRequestContext) error {
			metrics := server.Metrics()
			ready := metrics.QueueUtilization < 90.0 && metrics.CCUUtilization < 90.0
			statusCode := 200
			if !ready {
				statusCode = 503
			}
			return ctx.JSON(statusCode, map[string]interface{}{
				"status": func() string {
					if ready {
						return "UP"
					}
					return "DOWN"
				}(),
				"metrics": metrics,
			})
		}))

		// Prometheus metrics endpoint
		prometheus.RegisterMetricsEndpoint(router, "/metrics")

		// Update server handler to use router
		server.SetHandler(func(ctx *fasthttp.RequestCtx) {
			reqCtx := &web.FastRequestContext{
				RequestCtx: ctx,
				GoCMD:      vertx,
				EventBus:   vertx.EventBus(),
				Params:     make(map[string]string),
			}
			router.ServeFastHTTP(reqCtx)
		})

		// Start server in goroutine
		go func() {
			log.Printf("Starting FastHTTP server on %s", config.Addr)
			if err := server.Start(); err != nil {
				log.Printf("Server error: %v", err)
			}
		}()

		return nil
	}
}

func runMigrations(db *sql.DB) error {
	migrationSQL := `
	-- Create users table
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(255) UNIQUE NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create todos table
	CREATE TABLE IF NOT EXISTS todos (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title VARCHAR(500) NOT NULL,
		description TEXT,
		completed BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_todos_user_id ON todos(user_id);
	CREATE INDEX IF NOT EXISTS idx_todos_completed ON todos(completed);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

	-- Create updated_at trigger function
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	-- Create triggers
	DROP TRIGGER IF EXISTS update_users_updated_at ON users;
	CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
		FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

	DROP TRIGGER IF EXISTS update_todos_updated_at ON todos;
	CREATE TRIGGER update_todos_updated_at BEFORE UPDATE ON todos
		FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := db.Exec(migrationSQL)
	return err
}
