# Load Balancing Example

This example demonstrates a Master-Worker pattern with Fluxor.

## Components

1.  **Master Verticle**:
    *   Starts a TCP Server on port `:9090`
    *   Starts an HTTP Server on port `:8080`
    *   Distributes requests to workers using Round Robin

2.  **Worker Verticles**:
    *   Process requests
    *   2 instances running

## Running

```bash
go run examples/load-balancing/main.go
```

## Testing

### HTTP

```bash
curl "http://localhost:8080/process?data=hello"
```

### TCP

```bash
echo "tcp-data" | nc localhost 9090
```
