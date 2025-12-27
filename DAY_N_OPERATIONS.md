# Day N Operations Guide

**Day N** refers to ongoing operational tasks after initial deployment (Day 1) and observability setup (Day 2). This guide covers production operations, scaling, troubleshooting, and maintenance.

## Table of Contents

1. [Daily Operations](#daily-operations)
2. [Scaling Strategies](#scaling-strategies)
3. [Troubleshooting](#troubleshooting)
4. [Incident Response](#incident-response)
5. [Capacity Planning](#capacity-planning)
6. [Backup and Recovery](#backup-and-recovery)
7. [Performance Optimization](#performance-optimization)
8. [Configuration Management](#configuration-management)
9. [Monitoring and Alerting](#monitoring-and-alerting)
10. [Runbooks](#runbooks)

---

## Daily Operations

### Health Checks

**Automated Health Monitoring:**

```bash
# Check application health
curl http://localhost:8080/health

# Check readiness (includes metrics)
curl http://localhost:8080/ready

# Detailed health check
curl http://localhost:8080/health/detailed
```

**Expected Responses:**

```json
// /health
{
  "status": "UP"
}

// /ready
{
  "ready": true,
  "metrics": {
    "queue_utilization": 45.2,
    "ccu_utilization": 62.1,
    "active_connections": 2100
  }
}
```

### Key Metrics to Monitor Daily

1. **Request Rate (RPS)**
   - Normal: < 50,000 RPS
   - Warning: > 50,000 RPS
   - Critical: > 100,000 RPS

2. **Latency (P95)**
   - Normal: < 50ms
   - Warning: 50-200ms
   - Critical: > 200ms

3. **Error Rate**
   - Normal: < 0.1%
   - Warning: 0.1-1%
   - Critical: > 1%

4. **CCU Utilization**
   - Normal: < 67%
   - Warning: 67-80%
   - Critical: > 80%

5. **Queue Utilization**
   - Normal: < 50%
   - Warning: 50-80%
   - Critical: > 80%

### Daily Checklist

- [ ] Review application logs for errors
- [ ] Check metrics dashboard
- [ ] Verify health endpoints
- [ ] Review database connection pool usage
- [ ] Check disk space and memory usage
- [ ] Review error rates and latency
- [ ] Verify backup completion
- [ ] Check for security alerts

---

## Scaling Strategies

### When to Scale

**Scale Up (Vertical):**
- CPU utilization consistently > 70%
- Memory utilization > 80%
- CCU utilization > 80%
- Single instance bottleneck

**Scale Out (Horizontal):**
- Need high availability
- Traffic exceeds single instance capacity
- Want fault tolerance
- Cost-effective at scale

### Vertical Scaling

**Configuration Changes:**

```go
// Increase maxCCU
maxCCU := 15000  // from 5000
utilizationPercent := 67

// Increase database connections
dbConfig.MaxOpenConns = 200  // from 100
dbConfig.MaxIdleConns = 50   // from 10

// Increase worker pool
config.Workers = 100  // from 50
```

**Infrastructure Changes:**
- Increase CPU cores (4 → 8)
- Increase memory (4GB → 8GB)
- Increase network bandwidth
- Optimize database instance

### Horizontal Scaling

**Kubernetes Deployment:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fluxor-app
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    spec:
      containers:
      - name: fluxor
        resources:
          requests:
            cpu: "2"
            memory: "2Gi"
          limits:
            cpu: "4"
            memory: "4Gi"
```

**Auto-Scaling Configuration:**

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: fluxor-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: fluxor-app
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

**Load Balancer Configuration:**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: fluxor-service
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
  selector:
    app: fluxor
  sessionAffinity: None  # Stateless application
```

### Scaling Decision Matrix

| Metric | Threshold | Action |
|--------|-----------|--------|
| CPU > 70% | 5 minutes | Scale up/out |
| Memory > 80% | 5 minutes | Scale up/out |
| CCU > 80% | 5 minutes | Scale up/out |
| Queue > 80% | 5 minutes | Scale up/out |
| Error rate > 1% | 1 minute | Investigate + scale |
| P95 > 200ms | 5 minutes | Scale up/out |

---

## Troubleshooting

### High Latency

**Symptoms:**
- P95 latency > 200ms
- Slow response times
- User complaints

**Diagnosis:**

```bash
# Check current metrics
curl http://localhost:8080/api/metrics

# Check database performance
# - Slow queries
# - Connection pool exhaustion
# - Missing indexes

# Profile application
go tool pprof http://localhost:6060/debug/pprof/profile
```

**Solutions:**
1. Increase worker pool size
2. Add database indexes
3. Implement caching (Redis)
4. Optimize slow queries
5. Scale horizontally
6. Review external service dependencies

### High Error Rate

**Symptoms:**
- Error rate > 0.1%
- 503 Service Unavailable responses
- Connection timeouts

**Diagnosis:**

```bash
# Check logs
grep "ERROR" logs/app.log | tail -100

# Check metrics
curl http://localhost:8080/api/metrics | jq '.error_rate'

# Common causes:
# - Backpressure (503) - capacity exceeded
# - Database errors
# - External service failures
# - Memory issues
```

**Solutions:**
1. Increase capacity (maxCCU)
2. Fix database issues
3. Add retry logic with exponential backoff
4. Implement circuit breakers
5. Scale horizontally
6. Review and fix application bugs

### Resource Exhaustion

**Symptoms:**
- High CPU (> 90%)
- High memory (> 90%)
- OOM (Out of Memory) kills
- Slow responses

**Diagnosis:**

```bash
# Check resource usage
top
free -h
df -h

# Check application metrics
curl http://localhost:8080/api/metrics

# Profile memory
go tool pprof http://localhost:6060/debug/pprof/heap
```

**Solutions:**
1. Optimize CPU-intensive code
2. Fix memory leaks
3. Increase resource limits
4. Add more instances
5. Implement caching
6. Review garbage collection settings

### Database Connection Issues

**Symptoms:**
- "too many connections" errors
- Slow database queries
- Connection timeouts

**Diagnosis:**

```bash
# Check connection pool metrics
curl http://localhost:8080/api/metrics | jq '.database'

# Check database connections
# PostgreSQL: SELECT count(*) FROM pg_stat_activity;
# MySQL: SHOW PROCESSLIST;
```

**Solutions:**
1. Increase connection pool size
2. Optimize query performance
3. Add connection pooling middleware
4. Review connection lifecycle
5. Implement connection retry logic

---

## Incident Response

### Incident Severity Levels

**P0 - Critical:**
- Service completely down
- Data loss or corruption
- Security breach
- Response time: Immediate

**P1 - High:**
- Major feature broken
- High error rate (> 5%)
- Performance degradation (> 50%)
- Response time: < 15 minutes

**P2 - Medium:**
- Minor feature broken
- Moderate error rate (1-5%)
- Performance degradation (20-50%)
- Response time: < 1 hour

**P3 - Low:**
- Cosmetic issues
- Low error rate (< 1%)
- Minor performance issues
- Response time: < 4 hours

### Incident Response Process

1. **Detection**
   - Monitor alerts
   - Check dashboards
   - Review logs

2. **Assessment**
   - Determine severity
   - Identify affected systems
   - Assess impact

3. **Containment**
   - Isolate affected systems
   - Enable circuit breakers
   - Scale up if needed

4. **Resolution**
   - Fix root cause
   - Verify fix
   - Monitor recovery

5. **Post-Incident**
   - Document incident
   - Root cause analysis
   - Update runbooks
   - Implement preventive measures

### Incident Runbook Template

```markdown
## Incident: [Title]

**Severity:** P0/P1/P2/P3
**Detected:** [Timestamp]
**Resolved:** [Timestamp]
**Duration:** [Duration]

### Symptoms
- [Symptom 1]
- [Symptom 2]

### Root Cause
[Description of root cause]

### Resolution Steps
1. [Step 1]
2. [Step 2]
3. [Step 3]

### Prevention
- [Preventive measure 1]
- [Preventive measure 2]

### Lessons Learned
[Key learnings]
```

---

## Capacity Planning

### Current Capacity

**Single Instance:**
- Normal capacity: 3,350 CCU (67% of 5,000)
- Peak capacity: 5,000 CCU
- Target RPS: 50,000+
- Target P95: < 50ms

**3 Instances (Horizontal):**
- Normal capacity: 10,050 CCU
- Peak capacity: 15,000 CCU
- Target RPS: 150,000+
- Target P95: < 50ms

### Capacity Planning Formula

```
Required Instances = (Peak Traffic / Instance Capacity) * Safety Factor

Where:
- Peak Traffic = Peak concurrent users or RPS
- Instance Capacity = Normal CCU capacity per instance
- Safety Factor = 1.2-1.5 (20-50% headroom)
```

### Growth Projections

**Monthly Growth:**
- Current: 3,350 CCU
- Month 1: 4,000 CCU (+20%)
- Month 3: 5,000 CCU (+50%)
- Month 6: 7,000 CCU (+100%)

**Scaling Timeline:**
- Month 1-2: Single instance (vertical scaling)
- Month 3-4: 2-3 instances (horizontal scaling)
- Month 5+: Auto-scaling (3-10 instances)

---

## Backup and Recovery

### Backup Strategy

**Database Backups:**
```bash
# Daily full backup
pg_dump -h localhost -U user -d fluxor > backup_$(date +%Y%m%d).sql

# Hourly incremental backup (if supported)
# Use database-specific tools for incremental backups
```

**Configuration Backups:**
```bash
# Backup configuration files
tar -czf config_backup_$(date +%Y%m%d).tar.gz config.json config.yaml

# Store in version control
git add config/
git commit -m "Config backup $(date +%Y%m%d)"
```

**Application State:**
- Stateless application (no state backup needed)
- EventBus state (if using persistent EventBus)
- Workflow state (if using persistent workflows)

### Recovery Procedures

**Database Recovery:**
```bash
# Restore from backup
psql -h localhost -U user -d fluxor < backup_20251227.sql

# Verify restoration
psql -h localhost -U user -d fluxor -c "SELECT count(*) FROM users;"
```

**Application Recovery:**
```bash
# Redeploy application
kubectl rollout restart deployment/fluxor-app

# Verify health
curl http://localhost:8080/health
```

**Disaster Recovery:**
1. Restore database from latest backup
2. Redeploy application
3. Verify all services
4. Monitor for issues
5. Document recovery process

---

## Performance Optimization

### Optimization Checklist

**Application Level:**
- [ ] Profile hot paths
- [ ] Optimize database queries
- [ ] Implement caching
- [ ] Reduce allocations
- [ ] Optimize serialization

**Infrastructure Level:**
- [ ] Tune JVM/Go runtime
- [ ] Optimize network settings
- [ ] Configure load balancer
- [ ] Tune database
- [ ] Optimize storage

**Monitoring:**
- [ ] Set up performance baselines
- [ ] Track performance trends
- [ ] Identify regressions
- [ ] Measure optimization impact

### Performance Tuning

**Go Runtime:**
```bash
# Set GOMAXPROCS
export GOMAXPROCS=4

# GC tuning (if needed)
export GOGC=100  # Default, adjust if needed
```

**Database:**
```sql
-- Add indexes for frequently queried columns
CREATE INDEX idx_user_email ON users(email);
CREATE INDEX idx_order_created ON orders(created_at);

-- Analyze query performance
EXPLAIN ANALYZE SELECT * FROM users WHERE email = 'user@example.com';
```

**Caching:**
```go
// Implement Redis caching
cache := redis.NewClient(&redis.Options{
    Addr: "localhost:6379",
})

// Cache frequently accessed data
val, err := cache.Get(ctx, "user:123").Result()
if err == redis.Nil {
    // Fetch from database and cache
}
```

---

## Configuration Management

### Configuration Best Practices

1. **Environment-Specific Configs:**
   - `config.dev.json` - Development
   - `config.staging.json` - Staging
   - `config.prod.json` - Production

2. **Environment Variables:**
   ```bash
   # Override config with environment variables
   export APP_DATABASE_URL="postgres://user:pass@localhost/db"
   export APP_HTTP_PORT="8080"
   ```

3. **Secrets Management:**
   - Use environment variables for secrets
   - Never commit secrets to version control
   - Use secret management tools (Vault, AWS Secrets Manager)

4. **Configuration Validation:**
   ```go
   // Validate configuration on startup
   if err := config.Validate(); err != nil {
       log.Fatal("Invalid configuration:", err)
   }
   ```

### Configuration Changes

**Process:**
1. Update configuration file
2. Validate configuration
3. Test in staging
4. Deploy to production
5. Monitor for issues
6. Rollback if needed

**Rollback Procedure:**
```bash
# Revert configuration
git checkout HEAD~1 config.json

# Restart application
kubectl rollout restart deployment/fluxor-app

# Verify
curl http://localhost:8080/health
```

---

## Monitoring and Alerting

### Key Metrics to Alert On

**Critical Alerts:**
- Service down (health check fails)
- Error rate > 5%
- P95 latency > 500ms
- CCU utilization > 95%
- Database connection failures

**Warning Alerts:**
- Error rate > 1%
- P95 latency > 200ms
- CCU utilization > 80%
- Queue utilization > 80%
- High memory usage (> 85%)

### Alert Configuration

**Prometheus Alerts:**
```yaml
groups:
- name: fluxor_alerts
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.01
    for: 5m
    annotations:
      summary: "High error rate detected"
      
  - alert: HighLatency
    expr: histogram_quantile(0.95, http_request_duration_seconds_bucket) > 0.2
    for: 5m
    annotations:
      summary: "High latency detected"
```

### Dashboard Configuration

**Grafana Dashboard:**
- Request rate (RPS)
- Latency (P50, P95, P99)
- Error rate
- CCU utilization
- Queue utilization
- Database connections
- Memory usage
- CPU usage

---

## Runbooks

### Runbook: Service Restart

**When to Use:**
- After configuration changes
- After deployment
- When service is unresponsive

**Steps:**
1. Check current health: `curl http://localhost:8080/health`
2. Graceful shutdown: `kubectl rollout restart deployment/fluxor-app`
3. Wait for new pods: `kubectl get pods -w`
4. Verify health: `curl http://localhost:8080/health`
5. Check metrics: `curl http://localhost:8080/api/metrics`

### Runbook: Scale Up

**When to Use:**
- High utilization (> 80%)
- High latency (> 200ms)
- High error rate (> 1%)

**Steps:**
1. Check current metrics
2. Update deployment: `kubectl scale deployment fluxor-app --replicas=5`
3. Monitor scaling: `kubectl get pods -w`
4. Verify metrics improve
5. Update auto-scaling if needed

### Runbook: Database Connection Issues

**When to Use:**
- "too many connections" errors
- Slow database queries
- Connection timeouts

**Steps:**
1. Check connection pool metrics
2. Check database connections: `SELECT count(*) FROM pg_stat_activity;`
3. Increase connection pool size in config
4. Restart application
5. Monitor connection usage

### Runbook: High Latency

**When to Use:**
- P95 latency > 200ms
- User complaints
- Slow response times

**Steps:**
1. Check current metrics
2. Identify bottleneck (database, external service, application)
3. Profile application: `go tool pprof http://localhost:6060/debug/pprof/profile`
4. Optimize identified bottleneck
5. Scale if needed
6. Monitor improvement

---

## Best Practices

1. **Monitor Continuously**
   - Set up comprehensive monitoring
   - Review metrics daily
   - Respond to alerts promptly

2. **Plan for Growth**
   - Project capacity needs
   - Plan scaling strategy
   - Test scaling procedures

3. **Document Everything**
   - Document runbooks
   - Update after incidents
   - Share knowledge with team

4. **Test Recovery Procedures**
   - Test backups regularly
   - Practice disaster recovery
   - Verify runbooks work

5. **Automate Operations**
   - Automate scaling
   - Automate backups
   - Automate health checks

6. **Continuous Improvement**
   - Review incidents
   - Optimize performance
   - Update procedures

---

## References

- [Observability Guide](OBSERVABILITY.md) - Day 2 features
- [Performance Guide](PERFORMANCE.md) - Performance optimization
- [Security Guide](SECURITY.md) - Security operations
- [Production Ready Guide](PRODUCTION_READY.md) - Production checklist

---

**Last Updated**: 2025-12-27  
**Status**: ✅ Production-ready operations guide

