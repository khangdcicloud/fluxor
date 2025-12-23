# Fluxor Load Testing

This directory contains k6 load testing scripts for the Fluxor framework.

## Prerequisites

Install k6:
```bash
# macOS
brew install k6

# Linux
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6
```

## Running Tests

### 1. Start the server
```bash
cd /workspace
go run cmd/enterprise/main.go
```

### 2. Run load tests

#### Load Test (10k concurrent users)
```bash
k6 run loadtest/load-test.js
```

#### Spike Test (sudden traffic burst)
```bash
k6 run loadtest/spike-test.js
```

#### Stress Test (find breaking point)
```bash
k6 run loadtest/stress-test.js
```

### 3. Run with custom base URL
```bash
BASE_URL=http://localhost:8080 k6 run loadtest/load-test.js
```

### 4. Run with results output
```bash
k6 run --out json=results.json loadtest/load-test.js
```

## Test Scenarios

### Load Test (`load-test.js`)
- **Ramp up**: 1k → 5k → 10k users over 5 minutes
- **Sustain**: 10k users for 3 minutes
- **Ramp down**: 10k → 0 over 2 minutes
- **Endpoints tested**:
  - 30% - `/health` (lightweight)
  - 20% - `/ready` (with metrics)
  - 20% - `/api/users` (simulated API)
  - 20% - `/api/echo` (heavy JSON payload)
  - 10% - `/api/metrics`
- **Thresholds**:
  - P95 latency < 200ms
  - Error rate < 0.1%

### Spike Test (`spike-test.js`)
- **Purpose**: Test sudden traffic bursts
- **Pattern**: 100 → 5000 → 100 users
- **Duration**: 2.5 minutes total
- **Thresholds**:
  - P95 latency < 500ms
  - Error rate < 1%

### Stress Test (`stress-test.js`)
- **Purpose**: Find breaking point
- **Pattern**: Ramp up to 20k users
- **Duration**: 8 minutes total
- **Thresholds**:
  - P95 latency < 1000ms
  - Error rate < 5%

## Interpreting Results

### Good Results
```
✓ http_req_duration.............: avg=45ms  min=1ms   med=32ms  max=500ms p(95)=120ms
✓ errors.........................: 0.05%
✓ http_req_failed................: 0.03%
✓ http_reqs......................: 125000 (2083/s)
```

### Warning Signs
- P95 latency > 200ms
- Error rate > 0.1%
- High number of 503 responses (backpressure active)
- Increasing response times over time

### Critical Issues
- P95 latency > 1000ms
- Error rate > 1%
- Connection timeouts
- Server crashes

## Tuning Based on Results

### If P95 latency is high
1. Increase worker pool size
2. Reduce queue size (fail fast)
3. Add more server instances
4. Optimize database queries

### If error rate is high
1. Check backpressure settings
2. Review rate limiting configuration
3. Check database connection pool
4. Review logs for errors

### If 503 responses are common
1. Adjust CCU utilization target (currently 67%)
2. Increase worker pool size
3. Increase queue size
4. Add horizontal scaling

## CI/CD Integration

Load tests run automatically on:
- Push to `main` branch
- Pull requests to `main`
- Nightly builds

Results are uploaded as artifacts and can be viewed in GitHub Actions.

## Advanced Usage

### Run with custom VUs
```bash
k6 run --vus 1000 --duration 30s loadtest/load-test.js
```

### Generate HTML report
```bash
k6 run --out json=results.json loadtest/load-test.js
k6 report results.json --out html=report.html
```

### Run in Docker
```bash
docker run --network=host -v $PWD/loadtest:/scripts grafana/k6 run /scripts/load-test.js
```

## Monitoring During Tests

Monitor these metrics during load tests:
1. **Response times**: P50, P95, P99
2. **Error rates**: 4xx, 5xx responses
3. **Throughput**: Requests per second
4. **Resource usage**: CPU, memory, connections
5. **Queue utilization**: From `/api/metrics`
6. **CCU utilization**: From `/api/metrics`

## Troubleshooting

### Connection refused
- Ensure server is running
- Check port (default: 8080)
- Verify firewall settings

### High error rates
- Check server logs
- Review rate limiting settings
- Verify database connectivity
- Check resource limits (ulimit, file descriptors)

### Timeout errors
- Increase server timeout settings
- Check network latency
- Review slow endpoints
- Optimize database queries

## Best Practices

1. **Baseline first**: Run tests with low load to establish baseline
2. **Incremental load**: Gradually increase load to find limits
3. **Monitor resources**: Watch CPU, memory, connections
4. **Realistic scenarios**: Match production traffic patterns
5. **Repeat tests**: Run multiple times for consistency
6. **Document results**: Keep records of all test runs
7. **Version control**: Track changes to test scripts
8. **Production-like**: Test against production-like environment

## Performance Targets

Based on Fluxor configuration (67% utilization, 5000 max CCU):

- **Normal load**: Up to 3,350 CCU
- **Peak load**: 5,000 CCU (backpressure may activate)
- **Target RPS**: 100,000+ requests per second
- **Target P95**: < 10ms under normal load
- **Target P99**: < 50ms under normal load
- **Error rate**: < 0.1% under normal load

## References

- [k6 Documentation](https://k6.io/docs/)
- [k6 Test Types](https://k6.io/docs/test-types/)
- [k6 Metrics](https://k6.io/docs/using-k6/metrics/)
- [k6 Thresholds](https://k6.io/docs/using-k6/thresholds/)
