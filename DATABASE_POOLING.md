# Database Connection Pooling in Go (HikariCP Equivalent)

Hướng dẫn về connection pooling trong Go và các thư viện tương tự HikariCP.

## Tổng quan

Trong Java, **HikariCP** là thư viện connection pool phổ biến nhất. Trong Go, có nhiều lựa chọn tương tự:

1. **`database/sql`** (Standard Library) - Built-in pooling
2. **`pgxpool`** - PostgreSQL optimized pool
3. **`sqlx`** - Extended database/sql with better features
4. **Custom pools** - Build your own

---

## 1. database/sql (Standard Library) - Tương tự HikariCP

Go's `database/sql` package **đã có built-in connection pooling** giống HikariCP!

### So sánh với HikariCP

| HikariCP (Java) | database/sql (Go) | Notes |
|----------------|-------------------|-------|
| `HikariConfig` | `sql.DB` configuration | Both support pooling |
| `maximumPoolSize` | `SetMaxOpenConns()` | Max connections |
| `minimumIdle` | `SetMaxIdleConns()` | Min idle connections |
| `connectionTimeout` | `SetConnMaxLifetime()` | Connection lifetime |
| `idleTimeout` | `SetConnMaxIdleTime()` | Idle timeout |
| Auto pool management | ✅ Built-in | Automatic |

### Ví dụ sử dụng

```go
import (
    "database/sql"
    _ "github.com/lib/pq" // PostgreSQL driver
    "time"
)

// Tạo connection pool (giống HikariDataSource)
func NewDatabasePool(dsn string) (*sql.DB, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }
    
    // Configure pool (giống HikariConfig)
    db.SetMaxOpenConns(25)        // maximumPoolSize = 25
    db.SetMaxIdleConns(5)         // minimumIdle = 5
    db.SetConnMaxLifetime(5 * time.Minute)  // connectionTimeout
    db.SetConnMaxIdleTime(10 * time.Minute) // idleTimeout
    
    // Test connection
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return db, nil
}

// Sử dụng pool
func main() {
    db, err := NewDatabasePool("postgres://user:pass@localhost/db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Pool tự động quản lý connections
    rows, err := db.Query("SELECT * FROM users")
    // Connection được tự động lấy từ pool và trả về pool sau khi dùng
}
```

### Ưu điểm

- ✅ **Built-in**: Không cần thư viện bên ngoài
- ✅ **Tự động quản lý**: Pool tự động tạo/đóng connections
- ✅ **Thread-safe**: An toàn cho concurrent access
- ✅ **Tối ưu**: Hiệu suất tốt, tương tự HikariCP

### Nhược điểm

- ⚠️ **Cơ bản**: Ít tính năng advanced hơn HikariCP
- ⚠️ **No metrics**: Không có metrics built-in (cần tự implement)

---

## 2. pgxpool - PostgreSQL Optimized (Tốt nhất cho PostgreSQL)

**`pgxpool`** là connection pool được tối ưu đặc biệt cho PostgreSQL, tương tự HikariCP nhưng tốt hơn cho PostgreSQL.

### Cài đặt

```bash
go get github.com/jackc/pgx/v5/pgxpool
```

### Ví dụ sử dụng

```go
import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
)

// Tạo pool (giống HikariDataSource)
func NewPgxPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
    config, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, err
    }
    
    // Configure pool (giống HikariConfig)
    config.MaxConns = 25              // maximumPoolSize
    config.MinConns = 5                // minimumIdle
    config.MaxConnLifetime = 5 * time.Minute
    config.MaxConnIdleTime = 10 * time.Minute
    
    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, err
    }
    
    // Test connection
    if err := pool.Ping(ctx); err != nil {
        return nil, err
    }
    
    return pool, nil
}

// Sử dụng
func main() {
    ctx := context.Background()
    pool, err := NewPgxPool(ctx, "postgres://user:pass@localhost/db")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()
    
    // Query với pool
    rows, err := pool.Query(ctx, "SELECT * FROM users")
    // Connection tự động được quản lý
}
```

### Ưu điểm

- ✅ **Tối ưu cho PostgreSQL**: Hiệu suất tốt hơn `database/sql`
- ✅ **Binary protocol**: Sử dụng PostgreSQL binary protocol
- ✅ **Better error handling**: Error messages chi tiết hơn
- ✅ **Metrics**: Có sẵn metrics (pool stats)

### So sánh với HikariCP

| HikariCP | pgxpool | Notes |
|----------|---------|-------|
| Generic (all DBs) | PostgreSQL only | pgxpool tốt hơn cho PostgreSQL |
| JDBC driver | Native protocol | pgxpool nhanh hơn |
| Metrics | ✅ Built-in | Cả hai đều có |

---

## 3. sqlx - Extended database/sql

**`sqlx`** mở rộng `database/sql` với các tính năng bổ sung, vẫn sử dụng built-in pooling.

### Cài đặt

```bash
go get github.com/jmoiron/sqlx
```

### Ví dụ sử dụng

```go
import (
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
)

// Tạo pool với sqlx
func NewSqlxPool(dsn string) (*sqlx.DB, error) {
    db, err := sqlx.Connect("postgres", dsn)
    if err != nil {
        return nil, err
    }
    
    // Configure pool (giống database/sql)
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    return db, nil
}

// Sử dụng với struct scanning (tiện lợi hơn)
type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
}

func GetUsers(db *sqlx.DB) ([]User, error) {
    var users []User
    err := db.Select(&users, "SELECT id, name FROM users")
    return users, err
}
```

### Ưu điểm

- ✅ **Dễ sử dụng**: Struct scanning tự động
- ✅ **Named queries**: Hỗ trợ named parameters
- ✅ **Built-in pooling**: Sử dụng database/sql pooling
- ✅ **Backward compatible**: Tương thích với database/sql

---

## 4. Fluxor Integration - Premium Pattern

Fluxor cung cấp package `pkg/db` với connection pooling tích hợp sẵn!

### Sử dụng DatabaseComponent (Đã có sẵn!)

```go
import "github.com/fluxorio/fluxor/pkg/db"

// Tạo database component với pooling
component := db.NewDatabaseComponent(
    db.DefaultPoolConfig(
        "postgres://user:pass@localhost/dbname",
        "postgres",
    ),
)

// Trong verticle's doStart
func (v *MyVerticle) doStart(ctx core.FluxorContext) error {
    component.SetParent(v.BaseVerticle)
    return component.Start(ctx)
}

// Sử dụng component
rows, err := component.Query(ctx.Context(), "SELECT * FROM users")
```

**Package `pkg/db` đã implement đầy đủ:**
- ✅ `Pool` - Connection pool (giống HikariDataSource)
- ✅ `PoolConfig` - Configuration (giống HikariConfig)
- ✅ `DatabaseComponent` - Premium Pattern integration
- ✅ Pool statistics và monitoring
- ✅ Full context support

Xem chi tiết: `pkg/db/README.md`

### Sử dụng trong Service (Package `pkg/db`)

```go
import "github.com/fluxorio/fluxor/pkg/db"

type UserService struct {
    *core.BaseService
    db *db.DatabaseComponent
}

func NewUserService() *UserService {
    return &UserService{
        BaseService: core.NewBaseService("user-service", "user.service"),
        db: db.NewDatabaseComponent(
            db.DefaultPoolConfig(
                "postgres://user:pass@localhost/db",
                "postgres",
            ),
        ),
    }
}

func (s *UserService) doStart(ctx core.FluxorContext) error {
    // Initialize database component
    s.db.SetParent(s.BaseVerticle)
    if err := s.db.Start(ctx); err != nil {
        return err
    }
    return nil
}

func (s *UserService) doStop(ctx core.FluxorContext) error {
    return s.db.Stop(ctx)
}

func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    // Get user from database (pool tự động quản lý connection)
    userID := msg.Body().(string)
    
    var user map[string]interface{}
    err := s.db.QueryRow(
        ctx.Context(),
        "SELECT id, name FROM users WHERE id = $1",
        userID,
    ).Scan(&user["id"], &user["name"])
    
    if err != nil {
        return s.Fail(msg, 500, err.Error())
    }
    
    return s.Reply(msg, user)
}
```

---

## 5. pgxpool Integration với Fluxor

Nếu dùng PostgreSQL, nên dùng `pgxpool`:

```go
package database

import (
    "context"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/fluxorio/fluxor/pkg/core"
)

type PgxPoolComponent struct {
    *core.BaseComponent
    config *pgxpool.Config
    pool   *pgxpool.Pool
}

func NewPgxPoolComponent(dsn string) (*PgxPoolComponent, error) {
    config, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, err
    }
    
    // HikariCP-like configuration
    config.MaxConns = 25
    config.MinConns = 5
    config.MaxConnLifetime = 5 * time.Minute
    config.MaxConnIdleTime = 10 * time.Minute
    
    return &PgxPoolComponent{
        BaseComponent: core.NewBaseComponent("pgxpool"),
        config:        config,
    }, nil
}

func (c *PgxPoolComponent) doStart(ctx core.FluxorContext) error {
    pool, err := pgxpool.NewWithConfig(ctx.Context(), c.config)
    if err != nil {
        return err
    }
    
    // Test connection
    if err := pool.Ping(ctx.Context()); err != nil {
        return err
    }
    
    c.pool = pool
    return nil
}

func (c *PgxPoolComponent) doStop(ctx core.FluxorContext) error {
    if c.pool != nil {
        c.pool.Close()
    }
    return nil
}

func (c *PgxPoolComponent) Pool() *pgxpool.Pool {
    return c.pool
}

// Stats returns pool statistics
func (c *PgxPoolComponent) Stats() *pgxpool.Stat {
    return c.pool.Stat()
}
```

---

## 6. So sánh các lựa chọn

| Library | Database | Performance | Features | Recommendation |
|---------|----------|-------------|----------|----------------|
| `database/sql` | All | ⭐⭐⭐⭐ | ⭐⭐⭐ | ✅ **Best for most cases** |
| `pgxpool` | PostgreSQL | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ✅ **Best for PostgreSQL** |
| `sqlx` | All | ⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ✅ **Best for convenience** |
| Custom pool | All | ⭐⭐⭐ | ⭐⭐ | ⚠️ Only if needed |

### Khi nào dùng gì?

**Dùng `database/sql` khi:**
- ✅ Cần support nhiều database (PostgreSQL, MySQL, SQLite, etc.)
- ✅ Muốn dùng standard library
- ✅ Đơn giản, không cần advanced features

**Dùng `pgxpool` khi:**
- ✅ Chỉ dùng PostgreSQL
- ✅ Cần performance tối đa
- ✅ Cần metrics và monitoring

**Dùng `sqlx` khi:**
- ✅ Cần struct scanning tiện lợi
- ✅ Cần named queries
- ✅ Vẫn muốn dùng standard pooling

---

## 7. Best Practices

### 1. Pool Configuration (HikariCP-like)

```go
// ✅ Good: Configure pool properly
db.SetMaxOpenConns(25)        // Based on DB max_connections
db.SetMaxIdleConns(5)         // Keep some connections ready
db.SetConnMaxLifetime(5 * time.Minute)  // Prevent stale connections
db.SetConnMaxIdleTime(10 * time.Minute) // Close idle connections

// ❌ Bad: Default settings (may not be optimal)
db, _ := sql.Open("postgres", dsn)
// No configuration = default settings may not be optimal
```

### 2. Context Usage

```go
// ✅ Good: Always use context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

rows, err := db.QueryContext(ctx, "SELECT * FROM users")

// ❌ Bad: No context (no timeout, no cancellation)
rows, err := db.Query("SELECT * FROM users")
```

### 3. Connection Lifecycle

```go
// ✅ Good: Close resources properly
rows, err := db.Query("SELECT * FROM users")
if err != nil {
    return err
}
defer rows.Close() // Important!

// ❌ Bad: Leak connections
rows, _ := db.Query("SELECT * FROM users")
// Forgot to close rows = connection leak
```

### 4. Pool Monitoring

```go
// ✅ Good: Monitor pool stats
stats := db.Stats()
logger.Infof("Pool stats: Open=%d, InUse=%d, Idle=%d",
    stats.OpenConnections,
    stats.InUse,
    stats.Idle,
)

// Check for issues
if stats.WaitCount > 0 {
    logger.Warnf("Connections waiting: %d", stats.WaitCount)
}
```

---

## 8. Migration từ HikariCP

### Java HikariCP → Go database/sql

**Java:**
```java
HikariConfig config = new HikariConfig();
config.setJdbcUrl("jdbc:postgresql://localhost/db");
config.setMaximumPoolSize(25);
config.setMinimumIdle(5);
config.setConnectionTimeout(30000);

HikariDataSource ds = new HikariDataSource(config);
Connection conn = ds.getConnection();
```

**Go:**
```go
db, err := sql.Open("postgres", "postgres://localhost/db")
db.SetMaxOpenConns(25)        // maximumPoolSize
db.SetMaxIdleConns(5)         // minimumIdle
db.SetConnMaxLifetime(30 * time.Second) // connectionTimeout

conn, err := db.Conn(context.Background())
```

### Key Differences

| HikariCP | Go database/sql |
|----------|----------------|
| `HikariDataSource` | `sql.DB` |
| `getConnection()` | `Conn()` or `Query()` |
| `close()` | `Close()` or `rows.Close()` |
| Auto pool management | ✅ Same |
| Thread-safe | ✅ Same |

---

## 9. Fluxor Premium Pattern Example

Complete example với connection pooling (sử dụng package `pkg/db`):

```go
package main

import (
    "github.com/fluxorio/fluxor/pkg/core"
    "github.com/fluxorio/fluxor/pkg/db"
)

type UserService struct {
    *core.BaseService
    db *db.DatabaseComponent
}

func NewUserService() *UserService {
    return &UserService{
        BaseService: core.NewBaseService("user-service", "user.service"),
        db: db.NewDatabaseComponent(
            db.DefaultPoolConfig(
                "postgres://user:pass@localhost/db",
                "postgres",
            ),
        ),
    }
}

func (s *UserService) doStart(ctx core.FluxorContext) error {
    s.db.SetParent(s.BaseVerticle)
    return s.db.Start(ctx)
}

func (s *UserService) doStop(ctx core.FluxorContext) error {
    return s.db.Stop(ctx)
}

func (s *UserService) doHandleRequest(ctx core.FluxorContext, msg core.Message) error {
    userID := msg.Body().(string)
    
    // Pool tự động quản lý connection
    var name string
    err := s.db.QueryRow(
        ctx.Context(),
        "SELECT name FROM users WHERE id = $1",
        userID,
    ).Scan(&name)
    
    if err != nil {
        return s.Fail(msg, 500, err.Error())
    }
    
    return s.Reply(msg, map[string]interface{}{
        "id":   userID,
        "name": name,
    })
}
```

---

## Summary

**Go có built-in connection pooling** trong `database/sql` - tương tự HikariCP!

**Recommendations:**

1. **Most cases**: Dùng `database/sql` (standard library)
2. **PostgreSQL only**: Dùng `pgxpool` (tốt nhất)
3. **Convenience**: Dùng `sqlx` (dễ sử dụng hơn)
4. **Fluxor**: Tích hợp vào `BaseComponent` (Premium Pattern)

**Key Points:**
- ✅ Go's `database/sql` đã có pooling built-in
- ✅ Không cần thư viện bên ngoài cho basic pooling
- ✅ `pgxpool` tốt hơn cho PostgreSQL
- ✅ Fluxor Premium Pattern giúp tích hợp dễ dàng

**Migration từ HikariCP:**
- `HikariDataSource` → `sql.DB`
- `maximumPoolSize` → `SetMaxOpenConns()`
- `minimumIdle` → `SetMaxIdleConns()`
- Pool management → ✅ Automatic (giống HikariCP)

