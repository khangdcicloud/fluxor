# fluxor-project (multi-module example)

This folder shows how to build multiple services (`api-gateway`, `payment-service`) that communicate via a **cluster EventBus** backed by **NATS**, while still using Fluxor's "main-like" bootstrap API:

- `fluxor.NewMainVerticleWithOptions(...)`
- `DeployVerticle(...)`
- `Start()` (waits for SIGINT/SIGTERM)

## Prerequisite

Run a local NATS server (example):

```bash
nats-server -p 4222
```

## Run services (separate processes)

In one terminal:

```bash
cd fluxor-project/payment-service
go run .
```

In another terminal:

```bash
cd fluxor-project/api-gateway
go run .
```

Call API:

```bash
curl -sS -X POST http://127.0.0.1:8080/payments/authorize \
  -H 'content-type: application/json' \
  -d '{"paymentId":"p_123","userId":"u_1","amount":1200,"currency":"VND"}'
```

## Run all-in-one (single process)

```bash
cd fluxor-project/all-in-one
go run .
```

