# MyApp - Fluxor Application

A simple Fluxor application starter template.

## Getting Started

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Run the application:**
   ```bash
   go run .
   ```

3. **Test the endpoints:**
   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/api/hello
   ```

## Project Structure

```
myapp/
├── main.go          # Application entry point
├── config.json      # Configuration file
├── go.mod          # Go module file
└── README.md       # This file
```

## Next Steps

- Add more verticles in separate files (see `verticles/` directory pattern)
- Create event contracts in `contracts/` directory
- Add database connections using `pkg/db`
- Add middleware for authentication, logging, etc.

## Resources

- [Fluxor Documentation](../DOCUMENTATION.md)
- [Primary Pattern Guide](../docs/PRIMARY_PATTERN.md)
- [Examples](../examples/)

