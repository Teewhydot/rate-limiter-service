# Global Rate Limiter as a Service

A high-availability, distributed rate limiting service designed to prevent API quota exhaustion and financial penalties when integrating with third-party services. Built with Go, Redis, and PostgreSQL for ultra-fast performance and accuracy across multiple service instances.

## 🎯 Problem Statement

When running microservices at scale with multiple instances, each instance managing its own rate limits leads to:
- Frequent `429 Too Many Requests` errors
- Unnecessary financial penalties
- Quota exhaustion due to lack of coordination

This service solves these problems with centralized, distributed rate limiting.

## ✨ Features

- **Distributed Rate Limiting**: Accurate limits across multiple service instances using Redis
- **Per-Client Configuration**: Different limits for different clients (e.g., Client A: 100 req/min, Client B: 5000 req/min)
- **API Key Authentication**: Secure client authentication with SHA-256 hashed API keys
- **Flutter Web Dashboard**: Clients login with API key and view their own statistics
- **Sub-Millisecond Latency**: Rate limit checks complete in microseconds
- **High Availability**: Runs in cluster mode with fail-safe strategy
- **Async Logging**: Non-blocking request logging for analytics and billing
- **Real-time Dashboard API**: Usage statistics, trends, and analytics
- **Sliding Window Algorithm**: Accurate rate limiting with no burst issues

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Client Applications                      │
│              (Banking, Logistics, AI Services)               │
└──────────────────┬───────────────┬───────────────────────────┘
                   │               │
                   ▼               ▼
         ┌─────────────────────────────────┐
         │   Load Balancer (Optional)       │
         └─────────────────────────────────┘
                   │
      ┌────────────┼────────────┐
      ▼            ▼            ▼
┌──────────┐ ┌──────────┐ ┌──────────┐
│ Service  │ │ Service  │ │ Service  │
│Instance 1│ │Instance 2│ │Instance 3│
└────┬─────┘ └────┬─────┘ └────┬─────┘
     │            │            │
     └────────────┼────────────┘
                  │
     ┌────────────┴────────────┐
     ▼                         ▼
┌──────────┐            ┌──────────────┐
│  Redis   │            │  PostgreSQL  │
│ (Shared  │            │  (Logs &     │
│  State)  │            │  Analytics)  │
└──────────┘            └──────────────┘
```

### Components

1. **Rate Limiter Service** (Go)
   - HTTP API for rate limit checks
   - Client management
   - Dashboard data endpoints

2. **Redis**
   - Distributed rate limiting state
   - Sliding window counter implementation
   - Sub-millisecond performance

3. **PostgreSQL**
   - Client configuration storage
   - Request logs for analytics
   - Historical data for dashboard

## 🚀 Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Flutter SDK (for dashboard)

### Run with Docker (Recommended)

```bash
# Start all services (3 instances + Redis + PostgreSQL)
docker-compose up --build

# The services will be available at:
# - Instance 1: http://localhost:8080
# - Instance 2: http://localhost:8081
# - Instance 3: http://localhost:8082
```

### Create Your First Client & Get API Key

```bash
# Create a client
curl -X POST http://localhost:8080/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-client",
    "name": "My Company",
    "limit": 1000,
    "window_sec": 60
  }'

# Response includes API key - save it securely!
{
  "id": "my-client",
  "api_key": "sk_live_abc123...",
  "message": "Save this API key securely, it won't be shown again!"
}
```

### Run Flutter Dashboard

#### Option 1: Using Flutter CLI (Local Development)

```bash
# Navigate to dashboard folder
cd flutter_dashboard

# Install dependencies (first time only)
flutter pub get

# Run on Chrome (port 3000)
flutter run -d chrome --web-port 3000 --dart-define=API_URL=http://localhost:8080

# Dashboard opens at http://localhost:3000
# Login with your API key to view your statistics
```

#### Option 2: Using Docker

```bash
# Navigate to dashboard folder
cd flutter_dashboard

# Build Flutter web app
flutter build web --release --dart-define=API_URL=http://localhost:8080

# Build Docker image
docker build -t flutter-dashboard .

# Run container
docker run -d -p 3000:80 --name dashboard flutter-dashboard

# Dashboard opens at http://localhost:3000
# Login with your API key to view your statistics
```

**Note**: You need to create a `Dockerfile` in the `flutter_dashboard` folder:

```dockerfile
# flutter_dashboard/Dockerfile
FROM nginx:alpine

# Copy built web files to nginx
COPY build/web /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
```

That's it! The entire system is running.

### Using Makefile

```bash
# View all available commands
make help

# Start services
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down

# Run tests
make test
```

## 📡 API Endpoints

### Rate Limit Check

```bash
POST /api/v1/ratelimit/check
Content-Type: application/json

{
  "client_id": "client-a",
  "resource": "/api/payment"
}
```

**Response (200 OK - Allowed):**
```json
{
  "allowed": true,
  "remaining": 95,
  "limit": 100,
  "reset_at": 1700000000
}
```

**Response (429 Too Many Requests - Blocked):**
```json
{
  "allowed": false,
  "remaining": 0,
  "limit": 100,
  "reset_at": 1700000000,
  "retry_after": 45
}
```

### Client Management

**Create Client (Returns API Key):**
```bash
POST /api/v1/clients
Content-Type: application/json

{
  "id": "client-d",
  "name": "Payment Gateway",
  "limit": 1000,
  "window_sec": 60
}
```

**Response:**
```json
{
  "id": "client-d",
  "name": "Payment Gateway",
  "rate_limit": 1000,
  "window_sec": 60,
  "api_key": "sk_live_abc123...",
  "message": "Save this API key securely, This key wont be shown again!"
}
```

**Get Client (supports "me" for authenticated client):**
```bash
# Get any client (unprotected)
GET /api/v1/clients/client-a

# Get authenticated client using API key
GET /api/v1/clients/me
X-API-Key: sk_live_abc123...
```

**List All Clients:**
```bash
GET /api/v1/clients
```

**Update Client:**
```bash
PUT /api/v1/clients/client-a
Content-Type: application/json

{
  "name": "Updated Name",
  "limit": 200,
  "window_sec": 60
}
```

**Revoke API Key:**
```bash
POST /api/v1/clients/:id/apikey/revoke
Content-Type: application/json

{
  "reason": "Key compromised"
}
```

### Dashboard API (Protected - Requires API Key)

**Get Usage Statistics:**
```bash
# Last 30 days (PROTECTED)
GET /api/v1/dashboard/usage/client-a?days=30
X-API-Key: sk_live_abc123...

# Custom date range (PROTECTED)
GET /api/v1/dashboard/usage/client-a?start_date=2024-01-01T00:00:00Z&end_date=2024-01-31T23:59:59Z
X-API-Key: sk_live_abc123...
```

**Response:**
```json
{
  "client_id": "client-a",
  "total_requests": 150000,
  "allowed_requests": 145000,
  "blocked_requests": 5000,
  "avg_response_time_ms": 2.5,
  "period_start": "2024-01-01T00:00:00Z",
  "period_end": "2024-01-31T23:59:59Z"
}
```

**Get Trend Data:**
```bash
# For graphing (PROTECTED)
GET /api/v1/dashboard/trends/client-a?days=10
X-API-Key: sk_live_abc123...
```

**Response:**
```json
{
  "client_id": "client-a",
  "period": {
    "start": "2024-01-21T00:00:00Z",
    "end": "2024-01-31T00:00:00Z",
    "days": 10
  },
  "interval_hours": 4,
  "data": [
    {
      "timestamp": "2024-01-21T00:00:00Z",
      "request_count": 1250,
      "blocked_count": 45,
      "avg_response_time_ms": 2.3
    }
  ]
}
```

**Get Real-time Stats:**
```bash
# Unprotected for demo/testing
GET /api/v1/stats/client-a
```

**Response:**
```json
{
  "client_id": "client-a",
  "limit": 100,
  "window_sec": 60,
  "current_used": 45,
  "remaining": 55
}
```

### Health Check

```bash
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-31T10:30:00Z",
  "redis": "healthy",
  "postgres": "healthy"
}
```

## 🧪 Testing

### Run All Tests

```bash
make test
```

### Run Specific Test Suites

```bash
# Unit tests only
make test-unit

# Race condition tests
make test-race

# Load and performance tests
make test-load
```

### Run Tests with Docker

```bash
# Start services
docker-compose up -d

# Wait for services to be ready
sleep 10

# Run tests against running services
go test -v ./tests/...
```

### Test Coverage

```bash
make test-coverage
# Opens coverage.html in browser
```

## 🔬 Test Scenarios Covered

### Unit Tests
- ✅ Basic rate limiting functionality
- ✅ Sliding window accuracy
- ✅ Multiple client isolation
- ✅ Window expiration behavior
- ✅ Edge cases (zero limit, etc.)

### Race Condition Tests
- ✅ Concurrent requests from single client
- ✅ Concurrent requests from multiple clients
- ✅ High contention scenarios (1000+ concurrent)
- ✅ Atomic operation verification
- ✅ Data race detection

### Load & Performance Tests
- ✅ 50,000+ requests throughput test
- ✅ Sustained load (30 seconds continuous)
- ✅ Latency percentiles (P50, P95, P99)
- ✅ Multi-instance simulation
- ✅ Stress test (100,000 requests, 500 concurrent)

### Expected Performance
- **Latency**: <5ms average, <10ms P95, <20ms P99
- **Throughput**: >5,000 requests/second
- **Accuracy**: 100% accurate even under high contention

## 🔧 Configuration

All configuration is done via environment variables. Copy `.env.example` to `.env` and modify:

```bash
# Server
SERVER_PORT=8080

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# PostgreSQL
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=ratelimiter

# Rate Limiting
DEFAULT_LIMIT=100
DEFAULT_WINDOW_SEC=60
FAILSAFE_MODE=true  # Allow requests when Redis is down

# Logging
LOG_LEVEL=info
ASYNC_LOG_BATCH_SIZE=100
ASYNC_LOG_INTERVAL=5
```

## 🛡️ Fail-Safe Strategy

When Redis becomes temporarily unavailable:

**Fail-Safe Mode Enabled (default):**
- Requests are **allowed** (fail-open)
- System remains available but rate limiting is disabled
- Logs warnings for monitoring

**Fail-Safe Mode Disabled:**
- Requests are **denied** (fail-closed)
- Ensures strict rate limiting even during outages
- May impact availability

Set via `FAILSAFE_MODE=true|false` environment variable.

## 🔍 Verifying Edge Cases

### Test 1: Rate Limit Accuracy Across Multiple Instances

```bash
# Terminal 1: Send requests to instance 1
for i in {1..60}; do
  curl -X POST http://localhost:8080/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"client-a"}' &
done
wait

# Terminal 2: Send requests to instance 2
for i in {1..60}; do
  curl -X POST http://localhost:8081/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"client-a"}' &
done
wait
```

**Expected:** Exactly 100 requests allowed total (client-a has 100 req/min limit), distributed across instances.

### Test 2: Fail-Safe Behavior

```bash
# Stop Redis
docker-compose stop redis

# Send request (should be allowed with FAILSAFE_MODE=true)
curl -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-a"}'

# Restart Redis
docker-compose start redis
```

### Test 3: Sliding Window Accuracy

```bash
# Send 100 requests (fills limit)
for i in {1..100}; do
  curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"client-a"}' | jq .allowed
done

# Next request should be blocked
curl -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-a"}' | jq .

# Wait 60 seconds
sleep 60

# Should be allowed again
curl -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-a"}' | jq .allowed
```

### Test 4: Performance Under Load

```bash
# Install Apache Bench
# macOS: brew install httpd
# Ubuntu: sudo apt-get install apache2-utils

# Create request body
echo '{"client_id":"client-b"}' > /tmp/payload.json

# Send 10,000 requests with 100 concurrent
ab -n 10000 -c 100 -p /tmp/payload.json \
  -T application/json \
  http://localhost:8080/api/v1/ratelimit/check

# Check results for latency and success rate
```

### Test 5: Dashboard Data Verification

```bash
# Generate some traffic
for i in {1..1000}; do
  curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"client-a"}' > /dev/null
done

# Check usage stats
curl http://localhost:8080/api/v1/dashboard/usage/client-a?days=1 | jq .

# Check trend data
curl http://localhost:8080/api/v1/dashboard/trends/client-a?days=1 | jq .
```

## 📊 Sample Clients

The system comes pre-configured with sample clients for testing:

| Client ID | Name | Limit | Window | Use Case |
|-----------|------|-------|--------|----------|
| client-a | Client A - Banking Service | 100 req | 60 sec | Banking API |
| client-b | Client B - Logistics Provider | 5000 req | 60 sec | Logistics API |
| client-c | Client C - AI Model Service | 1000 req | 60 sec | AI/ML API |

## 📱 Flutter Web Dashboard

### Features
- **API Key Login**: Clients authenticate with their API key
- **Personal Dashboard**: Each client sees only their own statistics
- **Usage Statistics**: Total requests, allowed/blocked counts, avg response time
- **Trend Graphs**: Visual charts for request patterns over time
- **System Health**: Monitor Redis and PostgreSQL status
- **Responsive Design**: Material Design 3 with dark/light theme support

### Running the Dashboard

#### Option 1: Using Flutter CLI (Development)

```bash
# Prerequisites: Flutter SDK installed
# Check: flutter --version

# Navigate to dashboard folder
cd flutter_dashboard

# Install dependencies (first time only)
flutter pub get

# Run on Chrome
flutter run -d chrome --web-port 3000 --dart-define=API_URL=http://localhost:8080

# Dashboard opens at: http://localhost:3000
```

#### Option 2: Using Docker (Production)

```bash
# Navigate to dashboard folder
cd flutter_dashboard

# Build Flutter web app
flutter build web --release --dart-define=API_URL=http://localhost:8080

# Build Docker image
docker build -t flutter-dashboard .

# Run container
docker run -d -p 3000:80 --name dashboard flutter-dashboard

# Dashboard opens at: http://localhost:3000

# Stop container
docker stop dashboard

# Remove container
docker rm dashboard
```

**Create Dockerfile** in `flutter_dashboard/Dockerfile`:

```dockerfile
FROM nginx:alpine

# Copy built web files to nginx
COPY build/web /usr/share/nginx/html

# Expose port 80
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
```

### Using the Dashboard

1. **Login**: Enter your API key (received when creating client)
2. **View Stats**: See your request counts, response times, allowed/blocked ratio
3. **Trends**: View graphs for last 7, 15, or 30 days
4. **Refresh**: Click refresh icon or it auto-updates every 5 seconds
5. **Logout**: Click logout to switch accounts

## 🐛 Troubleshooting

### Services won't start
```bash
# Check if ports are already in use
lsof -i :8080
lsof -i :5432
lsof -i :6379

# Clean up and restart
make docker-clean
make docker-up
```

### Redis connection errors
```bash
# Check Redis health
docker-compose exec redis redis-cli ping

# View Redis logs
docker-compose logs redis
```

### PostgreSQL connection errors
```bash
# Check PostgreSQL health
docker-compose exec postgres pg_isready

# View PostgreSQL logs
docker-compose logs postgres
```

### Tests failing
```bash
# Ensure services are running
docker-compose ps

# Check if Redis is accessible
redis-cli -h localhost -p 6379 ping

# Run tests with verbose output
go test -v ./tests/unit/
```

## 📦 Project Structure

```
rate-limiter-service/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers.go            # HTTP handlers (10 endpoints)
│   │   ├── middleware.go          # API key auth, logging, CORS
│   │   └── routes.go              # Route definitions
│   ├── auth/
│   │   └── api_key.go             # API key generation & hashing
│   ├── config/
│   │   └── config.go              # Configuration management
│   ├── logger/
│   │   └── async_logger.go        # Async request logger
│   ├── models/
│   │   └── models.go              # Data models
│   ├── ratelimiter/
│   │   └── limiter.go             # Rate limiting logic
│   └── storage/
│       ├── postgres.go            # PostgreSQL client (with API key methods)
│       └── redis.go               # Redis client (ZSET implementation)
├── migrations/
│   ├── 001_initial_schema.sql     # Clients + request_logs tables
│   └── 002_api_keys.sql           # API keys table
├── flutter_dashboard/              # Flutter web dashboard
│   ├── lib/
│   │   ├── main.dart              # App entry point
│   │   ├── models/                # Data models (Client, UsageStats, etc.)
│   │   ├── screens/               # Dashboard & client details screens
│   │   ├── services/              # API service with auth
│   │   └── widgets/               # Reusable UI components
│   ├── web/                       # Web-specific files
│   └── pubspec.yaml               # Flutter dependencies
├── tests/
│   ├── load/
│   │   └── load_test.go           # Load tests
│   └── unit/
│       ├── race_test.go           # Race condition tests
│       └── ratelimiter_test.go    # Unit tests
├── docs/                          # Architecture documentation
│   ├── 01-redis-zset-implementation.md
│   ├── 02-multi-instance-architecture.md
│   ├── 03-per-client-rate-limits.md
│   ├── 04-postgresql-implementation.md
│   ├── 05-api-key-authentication-guide.md
│   └── 06-database-indexes-explained.md
├── docker-compose.yml              # Docker orchestration
├── Dockerfile                      # Docker image
├── Makefile                        # Build commands
├── go.mod                          # Go dependencies
└── README.md                       # This file
```

## 🔐 Security Considerations

### API Key Authentication

The service implements API key authentication for sensitive endpoints:

**Protected Endpoints (Require API Key):**
- `GET /api/v1/dashboard/usage/:client_id` - Client usage statistics
- `GET /api/v1/dashboard/trends/:client_id` - Client trend data

**Unprotected Endpoints (For Performance):**
- `POST /api/v1/ratelimit/check` - Core rate limiting (microsecond latency critical)
- `GET /api/v1/clients` - Client list (admin access - add auth in production)
- `GET /api/v1/stats/:client_id` - Real-time stats (demo purposes)

**Security Features:**
- ✅ API keys are SHA-256 hashed (never stored in plain text)
- ✅ API key format: `sk_live_` + 32 random bytes
- ✅ Keys shown only once during client creation
- ✅ Last used timestamp tracking
- ✅ Key revocation support
- ✅ Foreign key cascade delete

**Production Recommendations:**
- Use strong passwords for PostgreSQL
- Enable Redis authentication (`requirepass`)
- Use TLS/SSL for database connections
- Add rate limiting to the API itself
- Implement admin authentication for client management
- Use HTTPS in production (add nginx reverse proxy)
- Monitor for suspicious patterns
- Rotate API keys periodically
- Implement IP whitelisting for admin endpoints

## 📈 Monitoring & Observability

The service provides:
- Health check endpoint (`/health`)
- Structured logging (JSON format)
- Request metrics in PostgreSQL
- Real-time statistics API

Integrate with your monitoring stack:
- Prometheus (add `/metrics` endpoint)
- Grafana dashboards
- ELK stack for log aggregation
- Sentry for error tracking

## 🚀 Production Deployment

### Scaling Considerations

1. **Horizontal Scaling**: Add more service instances behind a load balancer
2. **Redis Clustering**: Use Redis Cluster for higher throughput
3. **PostgreSQL**: Use read replicas for analytics queries
4. **Connection Pooling**: Tune connection pool sizes
5. **Caching**: Add caching layer for client configurations

### Recommended Infrastructure

```yaml
# Kubernetes deployment example
replicas: 5  # 5 service instances
redis:
  mode: cluster
  replicas: 3
postgres:
  replicas: 1 (primary) + 2 (read replicas)
```

## 📄 License

This project is created for the Abuja Tech Challenge qualification task.

## 👥 Author

**Tunde S. Mac**
- GitHub: [@tundesmac](https://github.com/tundesmac)

## 🙏 Acknowledgments

Built with:
- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [Redis](https://redis.io/) - In-memory data store
- [PostgreSQL](https://www.postgresql.org/) - Relational database
- [Zap](https://github.com/uber-go/zap) - Structured logging
