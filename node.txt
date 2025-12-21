Key findings
What Node.js developers want
Familiar patterns
Express-like middleware chains
Async/await-style syntax (even if Go doesn't have it natively)
Type safety with generics (reduce interface{})
Developer experience
Hot reload for development
Better error messages
Testing utilities
Quick start guides
Enterprise requirements
OpenTelemetry for distributed tracing
Structured logging (JSON, context)
Prometheus metrics
Security middleware (JWT, OAuth2, CORS, rate limiting)
Database abstractions (connection pooling, migrations)
Ecosystem integration
Redis/caching
Message queues (Kafka, RabbitMQ)
OpenAPI/Swagger generation
Cloud provider SDKs
Priority recommendations
P0 (Must have):
Enhanced structured logging
OpenTelemetry integration
Auth/security middleware
Database connection pooling
Prometheus metrics export
P1 (High value):
Type-safe generics throughout
Express-like middleware ecosystem
Testing utilities
OpenAPI support
The document (ENTERPRISE_REQUIREMENTS.md) includes:
Detailed requirements per category
Code examples showing desired APIs
Priority matrix
Migration path from Express.js/Nest.js
Specific recommendations for each area
Should I start implementing any of these? I recommend starting with:
Enhanced structured logging
Express-like middleware helpers (CORS, auth, rate limiting)
OpenTelemetry integration
Which should we prioritize?