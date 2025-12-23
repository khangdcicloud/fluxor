# Build and Test Guide

This guide explains how to build, test, and run Fluxor applications.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Building](#building)
3. [Running Tests](#running-tests)
4. [Running Examples](#running-examples)
5. [Integration Testing](#integration-testing)
6. [Test Coverage](#test-coverage)
7. [Continuous Integration](#continuous-integration)

---

## Prerequisites

### Required

- **Go 1.21+**: [Download Go](https://go.dev/dl/)
- **Git**: For cloning the repository

### Optional (for Day2 features)

- **Prometheus**: For metrics scraping
- **Jaeger/Zipkin**: For distributed tracing
- **PostgreSQL**: For database examples

---

## Building

### Build All Packages

```bash
# Build all packages (no output = success)
go build ./...

# Build with verbose output
go build -v ./...

# Build specific package
go build ./pkg/core
go build ./pkg/web
go build ./pkg/config
go build ./pkg/observability/prometheus
```

### Build Example Application

```bash
# Build example application
go build -o bin/example ./cmd/example

# Run example
./bin/example
```

### Build with Tags

```bash
# Build with specific build tags
go build -tags "debug" ./...

# Build for specific OS/Architecture
GOOS=linux GOARCH=amd64 go build ./...
```

### Install to GOPATH/bin

```bash
# Install to $GOPATH/bin or $HOME/go/bin
go install ./cmd/example

# Run from anywhere
example
```

---

## Running Tests

### Run All Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run with race detector
go test -race ./...

# Run with coverage
go test -cover ./...
```

### Run Tests for Specific Package

```bash
# Test specific package
go test ./pkg/core
go test ./pkg/web
go test ./pkg/config
go test ./pkg/observability/prometheus

# Test with verbose output
go test -v ./pkg/web/middleware/auth
```

### Run Specific Test

```bash
# Run specific test function
go test -v ./pkg/config -run TestLoadYAML

# Run tests matching pattern
go test -v ./pkg/web -run TestFastRequestContext
```

### Run Tests with Timeout

```bash
# Set test timeout (default: 10 minutes)
go test -timeout 30s ./pkg/core
go test -timeout 5m ./pkg/web
```

### Run Tests in Parallel

```bash
# Run tests in parallel (default: GOMAXPROCS)
go test -parallel 4 ./...

# Disable parallel execution
go test -parallel 1 ./...
```

---

## Test Coverage

### Generate Coverage Report

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Open HTML report (platform-specific)
# Windows: start coverage.html
# macOS: open coverage.html
# Linux: xdg-open coverage.html
```

### Coverage by Package

```bash
# Show coverage summary
go test -cover ./...

# Coverage for specific package
go test -cover ./pkg/config
```

### Coverage Threshold

```bash
# Check if coverage meets threshold
go test -cover ./... | grep -E "coverage: [0-9]+\.[0-9]+%" | \
  awk '{if ($3+0 < 80.0) exit 1}'
```

---

## Running Examples

### Basic Example

```bash
# Run basic example
go run ./cmd/example/main.go

# Or build and run
go build -o bin/example ./cmd/example
./bin/example
```

### Example with Configuration

```bash
# Create config file
cat > config.yaml << EOF
server:
  port: 8080
  host: "localhost"
database:
  dsn: "postgres://user:pass@localhost/db"
  max_conns: 25
EOF

# Run with config
go run ./cmd/example/main.go -config config.yaml
```

### Example with Environment Variables

```bash
# Set environment variables
export APP_SERVER_PORT=9090
export APP_DATABASE_DSN="postgres://user:pass@localhost/db"

# Run example
go run ./cmd/example/main.go
```

---

## Integration Testing

### Run Integration Tests

```bash
# Run integration tests (if tagged)
go test -tags=integration ./...

# Run specific integration test
go test -tags=integration -v ./pkg/config -run TestConfigWithEnvOverrides
```

### Integration Test Setup

Integration tests may require external services:

```bash
# Start PostgreSQL (Docker)
docker run -d --name postgres-test \
  -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=testdb \
  -p 5432:5432 \
  postgres:15

# Start Prometheus (Docker)
docker run -d --name prometheus \
  -p 9090:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus

# Start Jaeger (Docker)
docker run -d --name jaeger \
  -p 14268:14268 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

### Test with External Services

```bash
# Set test environment variables
export TEST_DB_DSN="postgres://postgres:test@localhost:5432/testdb?sslmode=disable"
export TEST_PROMETHEUS_URL="http://localhost:9090"
export TEST_JAEGER_URL="http://localhost:14268/api/traces"

# Run integration tests
go test -tags=integration -v ./...
```

---

## Test Structure

### Unit Tests

Unit tests are in the same package with `_test.go` suffix:

```
pkg/
  config/
    config.go
    config_test.go          # Unit tests
    config_integration_test.go  # Integration tests
```

### Test Naming Convention

```go
// Test function name starts with "Test"
func TestLoadYAML(t *testing.T) { ... }

// Benchmark tests start with "Benchmark"
func BenchmarkLoadYAML(b *testing.B) { ... }

// Example tests start with "Example"
func ExampleLoadYAML() { ... }
```

### Test Helpers

```go
// Helper function (not run as test)
func setupTestConfig(t *testing.T) *Config {
    t.Helper() // Marks this as a test helper
    // ... setup code
    return config
}

// Cleanup
func teardownTestConfig(t *testing.T) {
    t.Helper()
    // ... cleanup code
}
```

---

## Benchmarking

### Run Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkLoadYAML ./pkg/config

# Run with memory profiling
go test -bench=. -benchmem ./...

# Compare benchmarks
go test -bench=. -benchmem ./pkg/core > old.txt
# ... make changes ...
go test -bench=. -benchmem ./pkg/core > new.txt
benchcmp old.txt new.txt
```

### Benchmark Example

```go
func BenchmarkEventBusPublish(b *testing.B) {
    ctx := context.Background()
    vertx := core.NewVertx(ctx)
    defer vertx.Close()
    
    eventBus := vertx.EventBus()
    eventBus.Consumer("test").Handler(func(ctx core.FluxorContext, msg core.Message) error {
        return nil
    })
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        eventBus.Publish("test", map[string]interface{}{"id": i})
    }
}
```

---

## Continuous Integration

### GitHub Actions Example

Create `.github/workflows/test.yml`:

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

### GitLab CI Example

Create `.gitlab-ci.yml`:

```yaml
test:
  image: golang:1.21
  script:
    - go test -v -race -coverprofile=coverage.out ./...
    - go tool cover -func=coverage.out
  coverage: '/total:\s+\(statements\)\s+(\d+\.\d+)%/'
```

---

## Common Build Issues

### Module Not Found

```bash
# Download dependencies
go mod download

# Tidy dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Import Path Issues

```bash
# Check module path
cat go.mod | grep "^module"

# Ensure imports match module path
# go.mod: module github.com/fluxorio/fluxor
# import: github.com/fluxorio/fluxor/pkg/core
```

### CGO Issues

```bash
# Disable CGO if not needed
CGO_ENABLED=0 go build ./...

# Or enable CGO
CGO_ENABLED=1 go build ./...
```

---

## Testing Best Practices

### 1. Use Table-Driven Tests

```go
func TestLoadYAML(t *testing.T) {
    tests := []struct {
        name    string
        content string
        want    string
        wantErr bool
    }{
        {
            name:    "valid yaml",
            content: "key: value",
            want:    "value",
            wantErr: false,
        },
        {
            name:    "invalid yaml",
            content: "key: [",
            want:    "",
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 2. Use Test Helpers

```go
func setupTestServer(t *testing.T) *web.FastHTTPServer {
    t.Helper()
    ctx := context.Background()
    vertx := core.NewVertx(ctx)
    config := web.CCUBasedConfigWithUtilization(":0", 100, 67)
    return web.NewFastHTTPServer(vertx, config)
}
```

### 3. Clean Up Resources

```go
func TestSomething(t *testing.T) {
    server := setupTestServer(t)
    defer server.Stop() // Always clean up
    
    // Test code
}
```

### 4. Test Error Cases

```go
func TestInvalidInput(t *testing.T) {
    _, err := someFunction("")
    if err == nil {
        t.Error("expected error for empty input")
    }
}
```

### 5. Use Subtests

```go
func TestMultipleScenarios(t *testing.T) {
    t.Run("scenario1", func(t *testing.T) {
        // Test scenario 1
    })
    
    t.Run("scenario2", func(t *testing.T) {
        // Test scenario 2
    })
}
```

---

## Quick Reference

### Build Commands

```bash
# Build all
go build ./...

# Build example
go build -o bin/example ./cmd/example

# Install
go install ./cmd/example
```

### Test Commands

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific test
go test -v -run TestName ./pkg/package

# Run benchmarks
go test -bench=. ./...
```

### Run Commands

```bash
# Run example
go run ./cmd/example/main.go

# Run with config
go run ./cmd/example/main.go -config config.yaml
```

---

## Example: Complete Test Workflow

```bash
# 1. Clone repository
git clone https://github.com/fluxorio/fluxor.git
cd fluxor

# 2. Download dependencies
go mod download

# 3. Run tests
go test -v ./...

# 4. Check coverage
go test -cover ./...

# 5. Build example
go build -o bin/example ./cmd/example

# 6. Run example
./bin/example

# 7. Run with integration tests (if services available)
go test -tags=integration -v ./...
```

---

## Summary

This guide covers:

- ✅ **Building**: All packages, examples, and custom builds
- ✅ **Testing**: Unit tests, integration tests, benchmarks
- ✅ **Coverage**: Generating and viewing coverage reports
- ✅ **Examples**: Running example applications
- ✅ **CI/CD**: Continuous integration setup
- ✅ **Best Practices**: Testing patterns and conventions

For more details, see:
- **[DOCUMENTATION.md](DOCUMENTATION.md)** - API reference
- **[CORE_COMPONENTS.md](CORE_COMPONENTS.md)** - Component architecture
- **[COMPONENT_FLOW.md](COMPONENT_FLOW.md)** - Data flow diagrams

