# Todo API - Fluxor Example Application

A complete, production-ready Todo API built with Fluxor framework showcasing:

- ✅ Full CRUD operations for Todos
- ✅ PostgreSQL database with connection pooling
- ✅ JWT authentication
- ✅ Rate limiting
- ✅ CCU-based backpressure
- ✅ Prometheus metrics
- ✅ Health and readiness endpoints

## Features

### Authentication
- User registration and login
- JWT token-based authentication
- Protected API endpoints

### Todo Management
- Create, read, update, and delete todos
- Pagination support
- Filter by completion status
- User-scoped todos (users can only access their own todos)

### Performance & Reliability
- **CCU-based backpressure**: Automatically rejects requests when capacity is exceeded (returns 503)
- **Rate limiting**: 100 requests per minute per IP address
- **Connection pooling**: Efficient database connection management
- **Prometheus metrics**: Comprehensive metrics for monitoring

### Observability
- Request ID tracking across all operations
- Prometheus metrics endpoint at `/metrics`
- Health check endpoint at `/health`
- Readiness check endpoint at `/ready`

## Prerequisites

- Go 1.24+ 
- Docker and Docker Compose
- PostgreSQL client (optional, for manual database access)

## Quick Start

### 1. Start PostgreSQL Database

```bash
cd examples/todo-api
docker-compose up -d
```

This starts a PostgreSQL container on port 5432 with:
- Database: `todo_db`
- User: `todo_user`
- Password: `todo_password`

### 2. Set Environment Variables (Optional)

```bash
export DATABASE_URL="postgres://todo_user:todo_password@localhost:5432/todo_db?sslmode=disable"
export JWT_SECRET="your-super-secret-jwt-key-min-32-chars"
```

If not set, defaults will be used (not recommended for production).

### 3. Run the Application

```bash
go run main.go
```

The server will start on `http://localhost:8080`

## API Endpoints

### Authentication

#### Register User
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john",
    "email": "john@example.com",
    "password": "securepassword123"
  }'
```

#### Login
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "john",
    "password": "securepassword123"
  }'
```

Response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid",
    "username": "john",
    "email": "john@example.com",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-01T00:00:00Z"
  }
}
```

#### Get Profile
```bash
curl -X GET http://localhost:8080/api/auth/profile \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Todos

#### Create Todo
```bash
curl -X POST http://localhost:8080/api/todos \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Buy groceries",
    "description": "Milk, eggs, bread"
  }'
```

#### List Todos
```bash
# List all todos
curl -X GET "http://localhost:8080/api/todos" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# List with pagination
curl -X GET "http://localhost:8080/api/todos?page=1&page_size=10" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Filter by completion status
curl -X GET "http://localhost:8080/api/todos?completed=true" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Get Todo by ID
```bash
curl -X GET http://localhost:8080/api/todos/TODO_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Update Todo
```bash
curl -X PUT http://localhost:8080/api/todos/TODO_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Updated title",
    "completed": true
  }'
```

#### Delete Todo
```bash
curl -X DELETE http://localhost:8080/api/todos/TODO_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Monitoring Endpoints

#### Health Check
```bash
curl http://localhost:8080/health
```

#### Readiness Check
```bash
curl http://localhost:8080/ready
```

#### Prometheus Metrics
```bash
curl http://localhost:8080/metrics
```

## Architecture

### Components

1. **Handlers** (`handlers/`): HTTP request handlers
   - `auth_handler.go`: Authentication endpoints
   - `todo_handler.go`: Todo CRUD endpoints

2. **Services** (`services/`): Business logic layer
   - `user_service.go`: User management
   - `todo_service.go`: Todo operations

3. **Models** (`models/`): Data models
   - `user.go`: User model and DTOs
   - `todo.go`: Todo model and DTOs

4. **Middleware** (`middleware/`): Request middleware
   - `metrics.go`: Prometheus metrics collection

### Database Schema

**users**
- `id` (UUID, Primary Key)
- `username` (VARCHAR, Unique)
- `email` (VARCHAR, Unique)
- `password_hash` (VARCHAR)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

**todos**
- `id` (UUID, Primary Key)
- `user_id` (UUID, Foreign Key → users.id)
- `title` (VARCHAR)
- `description` (TEXT)
- `completed` (BOOLEAN)
- `created_at` (TIMESTAMP)
- `updated_at` (TIMESTAMP)

### Security Features

1. **JWT Authentication**
   - Tokens expire after 24 hours
   - Secret key configurable via `JWT_SECRET` environment variable
   - Protected routes require valid JWT token

2. **Rate Limiting**
   - 100 requests per minute per IP address
   - Returns 429 Too Many Requests when exceeded

3. **Password Security**
   - Passwords hashed using bcrypt
   - Never stored or returned in plain text

### Performance Features

1. **CCU-based Backpressure**
   - Server configured for 60% utilization (3000 CCU for 5000 max)
   - Automatically returns 503 when capacity exceeded
   - Prevents system overload

2. **Database Connection Pooling**
   - Max 25 open connections
   - 5 idle connections
   - Connection lifetime: 5 minutes
   - Idle timeout: 10 minutes

3. **Prometheus Metrics**
   - HTTP request counts, durations, sizes
   - Database query durations
   - Server capacity metrics
   - Custom metrics support

## Monitoring

### Prometheus Metrics

The application exposes Prometheus metrics at `/metrics`. Key metrics include:

- `fluxor_http_requests_total`: Total HTTP requests by method, path, status
- `fluxor_http_request_duration_seconds`: Request duration histogram
- `fluxor_database_query_duration_seconds`: Database query duration
- `fluxor_server_queued_requests`: Current queued requests
- `fluxor_server_rejected_requests_total`: Total rejected requests (503)
- `fluxor_server_current_ccu`: Current concurrent users
- `fluxor_server_ccu_utilization`: CCU utilization percentage

### Example Prometheus Query

```promql
# Request rate per second
rate(fluxor_http_requests_total[5m])

# 95th percentile request duration
histogram_quantile(0.95, fluxor_http_request_duration_seconds_bucket)

# Current CCU utilization
fluxor_server_ccu_utilization
```

## Production Considerations

1. **Environment Variables**
   - Set `JWT_SECRET` to a strong, random value
   - Configure `DATABASE_URL` for your production database
   - Use secrets management (e.g., Kubernetes secrets, AWS Secrets Manager)

2. **Database**
   - Use managed PostgreSQL service (RDS, Cloud SQL, etc.)
   - Enable SSL/TLS connections
   - Set up automated backups
   - Configure connection pooling appropriately

3. **Security**
   - Enable HTTPS/TLS
   - Use strong JWT secrets
   - Implement CORS if needed
   - Set up rate limiting per user (not just IP)

4. **Monitoring**
   - Set up Prometheus scraping
   - Configure alerting rules
   - Monitor CCU utilization
   - Track error rates

5. **Scaling**
   - Run multiple instances behind load balancer
   - Use shared database
   - Consider Redis for session storage (if needed)

## License

MIT
