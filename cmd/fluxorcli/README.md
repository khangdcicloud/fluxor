# Fluxor CLI

Command-line tool for creating and managing Fluxor applications.

## Installation

Build from source:
```bash
go build -o bin/fluxorcli ./cmd/fluxorcli
```

Or install globally:
```bash
go install ./cmd/fluxorcli
```

## Usage

### Create a new application

```bash
fluxorcli new myapp
```

This will create a new Fluxor application with:
- `main.go` - Application entry point with API verticle
- `config.json` - Configuration file
- `go.mod` - Go module file
- `README.md` - Documentation

### Show version

```bash
fluxorcli version
```

## Next Steps

After creating a new application:

1. Navigate to the app directory:
   ```bash
   cd myapp
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Run the application:
   ```bash
   go run .
   ```

4. Test the endpoints:
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/api/hello
   ```

