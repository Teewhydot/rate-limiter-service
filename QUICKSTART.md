# Quick Start Guide

Get the Rate Limiter service running in 5 minutes!

## Prerequisites

- **Docker Desktop** installed ([Download](https://www.docker.com/products/docker-desktop))
- **curl** (usually pre-installed on Mac/Linux)
- **jq** for JSON formatting (optional): `brew install jq` on macOS

## Step 1: Start the Services

```bash
# Clone or navigate to the project directory
cd rate-limiter-service

# Start all services (3 instances + Redis + PostgreSQL)
docker-compose up --build -d

# Wait for services to be ready (~10-15 seconds)
sleep 15
```

That's it! The entire system is now running.

## Step 2: Verify Services

```bash
# Check if all containers are running
docker-compose ps

# Expected output:
# NAME                       STATUS
# ratelimiter-service-1      Up
# ratelimiter-service-2      Up
# ratelimiter-service-3      Up
# ratelimiter-postgres       Up (healthy)
# ratelimiter-redis          Up (healthy)
```

## Step 3: Test the System

### Quick Health Check

```bash
curl http://localhost:8080/health | jq .
```

**Expected output:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-31T10:30:00Z",
  "redis": "healthy",
  "postgres": "healthy"
}
```

### Test Rate Limiting

```bash
# Send a rate limit check request
curl -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-a","resource":"/api/test"}' | jq .
```

**Expected output:**
```json
{
  "allowed": true,
  "remaining": 99,
  "limit": 100,
  "reset_at": 1706702400
}
```

### Run Automated Test Suite

```bash
# Make the test script executable (if not already)
chmod +x test-system.sh

# Run comprehensive tests
./test-system.sh
```

This will test:
- Health checks
- Rate limit allowing/blocking
- Limit exhaustion
- Multi-instance consistency
- Dashboard APIs

## Step 4: Explore the API

### Pre-configured Clients

The system comes with 3 sample clients:

| Client ID | Name | Limit | Window |
|-----------|------|-------|--------|
| client-a | Banking Service | 100 req/min | 60 sec |
| client-b | Logistics Provider | 5000 req/min | 60 sec |
| client-c | AI Model Service | 1000 req/min | 60 sec |

### List All Clients

```bash
curl http://localhost:8080/api/v1/clients | jq .
```

### Get Usage Statistics (for Dashboard)

```bash
# Last 30 days
curl "http://localhost:8080/api/v1/dashboard/usage/client-a?days=30" | jq .

# Last 10 days
curl "http://localhost:8080/api/v1/dashboard/usage/client-a?days=10" | jq .
```

### Get Real-time Statistics

```bash
curl http://localhost:8080/api/v1/stats/client-a | jq .
```

## Step 5: Test Multi-Instance Behavior

```bash
# Send requests to all 3 instances
curl -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-a"}' | jq '.remaining'

curl -X POST http://localhost:8081/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-a"}' | jq '.remaining'

curl -X POST http://localhost:8082/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-a"}' | jq '.remaining'
```

Notice how the `remaining` count decreases across all instances - they share the same rate limit state!

## Step 6: Create Your Own Client

```bash
curl -X POST http://localhost:8080/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{
    "id": "my-client",
    "name": "My Custom Service",
    "limit": 500,
    "window_sec": 60
  }' | jq .
```

Now test it:

```bash
curl -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"my-client"}' | jq .
```

## Running Tests

### Unit Tests

```bash
# Run tests inside Docker container
docker-compose exec ratelimiter-1 go test -v ./tests/unit/

# Or use make commands
make test-unit
```

### Race Condition Tests

```bash
docker-compose exec ratelimiter-1 go test -race -v ./tests/unit/race_test.go
```

### Load Tests

```bash
docker-compose exec ratelimiter-1 go test -v ./tests/load/ -timeout 30m
```

## Viewing Logs

```bash
# View all logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f ratelimiter-1
docker-compose logs -f redis
docker-compose logs -f postgres
```

## Stopping the Services

```bash
# Stop services but keep data
docker-compose down

# Stop services and remove all data
docker-compose down -v
```

## Restarting Services

```bash
# Restart all services
docker-compose restart

# Restart specific service
docker-compose restart ratelimiter-1
```

## Troubleshooting

### Services won't start

```bash
# Check if ports are already in use
lsof -i :8080
lsof -i :5432
lsof -i :6379

# Clean up completely and restart
docker-compose down -v
docker system prune -f
docker-compose up --build -d
```

### Can't connect to services

```bash
# Check if containers are running
docker-compose ps

# Check container logs
docker-compose logs ratelimiter-1
docker-compose logs redis
docker-compose logs postgres

# Check Docker network
docker network ls
docker network inspect rate-limiter-service_ratelimiter-network
```

### Tests failing

```bash
# Ensure services are healthy
curl http://localhost:8080/health

# Check Redis
docker-compose exec redis redis-cli ping

# Check PostgreSQL
docker-compose exec postgres pg_isready -U postgres
```

## Performance Testing

### Simple Load Test

```bash
# Send 1000 requests sequentially
for i in {1..1000}; do
  curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"client-b"}' > /dev/null
done
```

### Concurrent Load Test

```bash
# Send 500 concurrent requests
for i in {1..500}; do
  curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"client-b"}' > /dev/null &
done
wait
```

## Next Steps

1. **Read the full documentation:** Check out [README.md](README.md) for detailed API documentation
2. **Review architecture:** See [ARCHITECTURE.md](ARCHITECTURE.md) for system design
3. **API examples:** Explore [api-examples.md](api-examples.md) for more API usage examples
4. **Build your Flutter dashboard:** Use the dashboard APIs to create visualizations
5. **Deploy to production:** See deployment considerations in README.md

## Service URLs

When running locally:

- **Instance 1:** http://localhost:8080
- **Instance 2:** http://localhost:8081
- **Instance 3:** http://localhost:8082
- **Redis:** localhost:6379
- **PostgreSQL:** localhost:5432

## Default Credentials

**PostgreSQL:**
- Username: `postgres`
- Password: `postgres`
- Database: `ratelimiter`

**Redis:**
- No password (development mode)

## Important Files

- `docker-compose.yml` - Service orchestration
- `.env.example` - Configuration template
- `migrations/001_initial_schema.sql` - Database schema
- `test-system.sh` - Automated test script
- `Makefile` - Build and test commands

## Support

If you encounter issues:

1. Check the [Troubleshooting](#troubleshooting) section above
2. Review logs: `docker-compose logs -f`
3. Verify prerequisites are installed
4. Ensure Docker has enough resources (4GB+ RAM recommended)

## Summary of Commands

```bash
# Start services
docker-compose up --build -d

# Run tests
./test-system.sh

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Clean up everything
docker-compose down -v
```

That's it! You now have a fully functional, highly available rate limiting service running locally. 🚀
