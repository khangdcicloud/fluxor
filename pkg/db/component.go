package db

import (
	"context"
	"database/sql"

	"github.com/fluxorio/fluxor/pkg/core"
)

// DatabaseComponent provides database connection pooling using Premium Pattern
// Similar to HikariCP DataSource but integrated with Fluxor
type DatabaseComponent struct {
	*core.BaseComponent
	config PoolConfig
	pool   *Pool
}

// NewDatabaseComponent creates a new database component with connection pooling
// Fail-fast: Validates configuration
func NewDatabaseComponent(config PoolConfig) *DatabaseComponent {
	// Fail-fast: Validate configuration
	if config.DSN == "" {
		panic("DSN cannot be empty")
	}
	if config.DriverName == "" {
		panic("DriverName cannot be empty")
	}
	if config.MaxOpenConns <= 0 {
		panic("MaxOpenConns must be positive")
	}

	return &DatabaseComponent{
		BaseComponent: core.NewBaseComponent("database"),
		config:        config,
	}
}

// doStart initializes the connection pool (similar to HikariDataSource initialization)
// Fail-fast: Validates state and configuration before starting
func (c *DatabaseComponent) doStart(ctx core.FluxorContext) error {
	// Fail-fast: Validate context
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "FluxorContext cannot be nil"}
	}

	// Fail-fast: Validate configuration (should have been validated in NewDatabaseComponent)
	if c.config.DSN == "" {
		return &core.EventBusError{Code: "INVALID_CONFIG", Message: "DSN cannot be empty"}
	}
	if c.config.DriverName == "" {
		return &core.EventBusError{Code: "INVALID_CONFIG", Message: "DriverName cannot be empty"}
	}

	// Create connection pool (NewPool also validates config)
	pool, err := NewPool(c.config)
	if err != nil {
		return err
	}

	c.pool = pool

	// Notify via EventBus (Premium Pattern integration)
	eventBus := c.EventBus()
	if eventBus != nil {
		if err := eventBus.Publish("database.ready", map[string]interface{}{
			"component":      c.Name(),
			"max_open_conns": c.config.MaxOpenConns,
			"max_idle_conns": c.config.MaxIdleConns,
		}); err != nil {
			// Best-effort notification; ignore on error.
		}
	}

	return nil
}

// doStop closes the connection pool
func (c *DatabaseComponent) doStop(ctx core.FluxorContext) error {
	if c.pool != nil {
		return c.pool.Close()
	}
	return nil
}

// Pool returns the connection pool
// Fail-fast: Panics if component not started
func (c *DatabaseComponent) Pool() *Pool {
	if c == nil {
		panic("DatabaseComponent cannot be nil")
	}
	if c.pool == nil {
		panic("database component not started - call Start() first")
	}
	return c.pool
}

// DB returns the underlying *sql.DB
// Fail-fast: Panics if component not started
func (c *DatabaseComponent) DB() *sql.DB {
	if c == nil {
		panic("DatabaseComponent cannot be nil")
	}
	if c.pool == nil {
		panic("database component not started - call Start() first")
	}
	return c.pool.DB()
}

// Query executes a query that returns rows
// Fail-fast: Validates state and inputs before querying
func (c *DatabaseComponent) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if c == nil {
		return nil, &core.EventBusError{Code: "INVALID_STATE", Message: "DatabaseComponent cannot be nil"}
	}
	if c.pool == nil {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "database component not started - call Start() first"}
	}
	if ctx == nil {
		return nil, &core.EventBusError{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	if query == "" {
		return nil, &core.EventBusError{Code: "INVALID_INPUT", Message: "query cannot be empty"}
	}
	return c.pool.Query(ctx, query, args...)
}

// QueryRow executes a query that returns a single row
// Fail-fast: Validates state and inputs before querying
func (c *DatabaseComponent) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if c == nil {
		panic("DatabaseComponent cannot be nil")
	}
	if c.pool == nil {
		panic("database component not started - call Start() first")
	}
	if ctx == nil {
		panic("context cannot be nil")
	}
	if query == "" {
		panic("query cannot be empty")
	}
	return c.pool.QueryRow(ctx, query, args...)
}

// Exec executes a command
// Fail-fast: Validates state and inputs before executing
func (c *DatabaseComponent) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if c == nil {
		return nil, &core.EventBusError{Code: "INVALID_STATE", Message: "DatabaseComponent cannot be nil"}
	}
	if c.pool == nil {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "database component not started - call Start() first"}
	}
	if ctx == nil {
		return nil, &core.EventBusError{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	if query == "" {
		return nil, &core.EventBusError{Code: "INVALID_INPUT", Message: "query cannot be empty"}
	}
	return c.pool.Exec(ctx, query, args...)
}

// Begin starts a transaction
// Fail-fast: Validates state and inputs before beginning transaction
func (c *DatabaseComponent) Begin(ctx context.Context) (*sql.Tx, error) {
	if c == nil {
		return nil, &core.EventBusError{Code: "INVALID_STATE", Message: "DatabaseComponent cannot be nil"}
	}
	if c.pool == nil {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "database component not started - call Start() first"}
	}
	if ctx == nil {
		return nil, &core.EventBusError{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	return c.pool.Begin(ctx)
}

// BeginTx starts a transaction with options
// Fail-fast: Validates state and inputs before beginning transaction
func (c *DatabaseComponent) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if c == nil {
		return nil, &core.EventBusError{Code: "INVALID_STATE", Message: "DatabaseComponent cannot be nil"}
	}
	if c.pool == nil {
		return nil, &core.EventBusError{Code: "NOT_STARTED", Message: "database component not started - call Start() first"}
	}
	if ctx == nil {
		return nil, &core.EventBusError{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	return c.pool.BeginTx(ctx, opts)
}

// Stats returns pool statistics (similar to HikariPoolMXBean)
// Fail-fast: Returns empty stats if not started (safe, no panic)
func (c *DatabaseComponent) Stats() sql.DBStats {
	if c == nil || c.pool == nil {
		return sql.DBStats{}
	}
	return c.pool.Stats()
}

// Ping tests the connection
// Fail-fast: Validates state and inputs before pinging
func (c *DatabaseComponent) Ping(ctx context.Context) error {
	if c == nil {
		return &core.EventBusError{Code: "INVALID_STATE", Message: "DatabaseComponent cannot be nil"}
	}
	if c.pool == nil {
		return &core.EventBusError{Code: "NOT_STARTED", Message: "database component not started - call Start() first"}
	}
	if ctx == nil {
		return &core.EventBusError{Code: "INVALID_INPUT", Message: "context cannot be nil"}
	}
	return c.pool.Ping(ctx)
}
