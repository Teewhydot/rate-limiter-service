# Architecture Documentation

## System Architecture

### High-Level Design

The Global Rate Limiter service follows a distributed architecture pattern with the following key components:

```
                    ┌─────────────────────────────────────┐
                    │    External Client Applications     │
                    │  (Banking, Logistics, AI Services)  │
                    └──────────────┬──────────────────────┘
                                   │
                                   │ HTTP/HTTPS
                                   ▼
                    ┌─────────────────────────────────────┐
                    │       Load Balancer (Optional)       │
                    │         (Nginx, HAProxy, ALB)        │
                    └──────────────┬──────────────────────┘
                                   │
                ┌──────────────────┼──────────────────┐
                │                  │                  │
                ▼                  ▼                  ▼
        ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
        │Rate Limiter  │  │Rate Limiter  │  │Rate Limiter  │
        │  Instance 1  │  │  Instance 2  │  │  Instance 3  │
        │  (Port 8080) │  │  (Port 8081) │  │  (Port 8082) │
        └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
               │                 │                  │
               │    Shared       │   Shared         │
               │    State        │   State          │
               │                 │                  │
               └─────────┬───────┴──────────┬───────┘
                         │                  │
                         ▼                  ▼
                  ┌─────────────┐    ┌──────────────────┐
                  │    Redis    │    │   PostgreSQL     │
                  │  (Cluster)  │    │   (Primary +     │
                  │             │    │   Read Replicas) │
                  │  - Rate     │    │                  │
                  │    Limit    │    │  - Client Config │
                  │    State    │    │  - Request Logs  │
                  │  - Sliding  │    │  - Analytics     │
                  │    Window   │    │    Data          │
                  │    Counters │    │                  │
                  └─────────────┘    └──────────────────┘
```

## Component Details

### 1. Rate Limiter Service Instances

**Technology:** Go (Golang)

**Responsibilities:**
- Accept incoming rate limit check requests
- Validate client credentials
- Query Redis for current rate limit state
- Apply sliding window algorithm
- Return allow/deny decisions in <5ms
- Asynchronously log requests to PostgreSQL
- Serve dashboard API endpoints

**Key Features:**
- Stateless design (all state in Redis)
- Horizontally scalable
- Graceful shutdown handling
- Health check endpoints
- Structured logging

### 2. Redis (Distributed State Store)

**Purpose:** Centralized rate limiting state

**Data Structures Used:**
- **Sorted Sets (ZSET)**: Track request timestamps
  - Key: `ratelimit:{client_id}`
  - Score: Request timestamp (nanoseconds)
  - Member: Unique request identifier

**Operations:**
- `ZREMRANGEBYSCORE`: Remove expired requests
- `ZCOUNT`: Count requests in window
- `ZADD`: Add new request
- `EXPIRE`: Set TTL on keys

**Why Redis?**
- Atomic operations prevent race conditions
- Sub-millisecond latency
- Built-in data expiration
- High availability with clustering
- Supports distributed systems

### 3. PostgreSQL (Persistent Storage)

**Purpose:** Client configuration and analytics

**Tables:**

```sql
-- Client Configuration
clients (
  id VARCHAR PRIMARY KEY,
  name VARCHAR,
  limit INTEGER,
  window_sec INTEGER,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)

-- Request Logs for Analytics
request_logs (
  id BIGSERIAL PRIMARY KEY,
  client_id VARCHAR,
  resource VARCHAR,
  allowed BOOLEAN,
  response_time_ms BIGINT,
  timestamp TIMESTAMP
)
```

**Indexes:**
- `idx_request_logs_client_id`
- `idx_request_logs_timestamp`
- `idx_request_logs_client_timestamp`
- `idx_request_logs_analytics`

### 4. Async Logger

**Purpose:** Non-blocking request logging

**Design:**
- Buffered channel for log entries
- Batch writes to database
- Periodic flushing (configurable interval)
- Graceful shutdown with drain

**Benefits:**
- Rate limit checks remain fast
- Efficient database writes
- No log loss on shutdown
- Configurable trade-offs

## Algorithms

### Sliding Window Counter

**Advantages over Token Bucket:**
- No burst allowance (strict enforcement)
- More accurate per-second/minute limits
- Fair distribution of requests

**Implementation:**

```
1. Current time: T
2. Window start: T - window_sec
3. Remove requests older than window start
4. Count requests in [window_start, T]
5. If count < limit:
     - Add current request
     - Return ALLOWED
   Else:
     - Return DENIED
6. Set expiration on data structure
```

**Time Complexity:** O(log N) per request
**Space Complexity:** O(limit) per client

## Data Flow

### Rate Limit Check Flow

```
1. Client sends POST /api/v1/ratelimit/check
   ├─> Client ID: "client-a"
   └─> Resource: "/api/payment"

2. Handler receives request
   └─> Start timer (for latency tracking)

3. Query PostgreSQL for client config
   ├─> Hit (cached): Use config
   └─> Miss: Use default config

4. Query Redis with sliding window
   ├─> ZREMRANGEBYSCORE (cleanup)
   ├─> ZCOUNT (count requests)
   ├─> ZADD (add current request)
   └─> EXPIRE (set TTL)

5. Evaluate result
   ├─> Count < Limit: ALLOWED
   └─> Count >= Limit: DENIED

6. Queue log entry (async)
   └─> {client_id, allowed, latency, timestamp}

7. Return response to client
   └─> {allowed, remaining, reset_at}

Total time: < 5ms
```

### Dashboard Query Flow

```
1. Flutter app requests usage stats
   └─> GET /api/v1/dashboard/usage/client-a?days=30

2. Handler queries PostgreSQL
   └─> SELECT COUNT(*), AVG(response_time_ms), ...
       FROM request_logs
       WHERE client_id = 'client-a'
       AND timestamp BETWEEN start AND end

3. Aggregate and format results
   └─> {total, allowed, blocked, avg_latency}

4. Return JSON to Flutter app
```

## High Availability Design

### Instance Failures

**Scenario:** One service instance crashes

**Impact:** None - other instances continue serving

**Recovery:** 
- Automatic restart (Docker restart policy)
- Load balancer removes unhealthy instance
- Requests routed to healthy instances

### Redis Failures

**Scenario:** Redis becomes unavailable

**Impact:** Depends on fail-safe mode

**Fail-Safe Mode ON (default):**
- Requests are ALLOWED
- Warnings logged
- System remains available

**Fail-Safe Mode OFF:**
- Requests are DENIED
- Strict rate limiting maintained
- May impact availability

**Recovery:**
- Redis reconnect logic
- Automatic retry with exponential backoff
- State rebuilds from request logs

### PostgreSQL Failures

**Scenario:** Database connection lost

**Impact:**
- Client config queries fail → use defaults
- Log writes fail → entries dropped (logged)
- Dashboard queries fail → error response

**Recovery:**
- Connection pool auto-reconnect
- Read replica failover (if configured)
- Log buffer retention during outage

## Performance Characteristics

### Latency Targets

| Metric | Target | Typical |
|--------|--------|---------|
| P50 Latency | <5ms | 2-3ms |
| P95 Latency | <10ms | 5-8ms |
| P99 Latency | <20ms | 10-15ms |

### Throughput

- **Single Instance:** 5,000-10,000 req/sec
- **Three Instances:** 15,000-30,000 req/sec
- **Bottleneck:** Redis (can scale with clustering)

### Resource Usage

**Per Instance:**
- Memory: 50-100 MB
- CPU: 0.1-0.5 cores (idle/loaded)
- Network: 1-5 Mbps

**Redis:**
- Memory: ~1 KB per active client
- Scales linearly with number of clients

**PostgreSQL:**
- Grows with log retention
- Use partitioning for large datasets
- Archive old logs periodically

## Scalability

### Horizontal Scaling

**Service Instances:**
- Add more instances behind load balancer
- No coordination needed between instances
- Linear scaling up to Redis limits

**Redis:**
- Use Redis Cluster for >100k ops/sec
- 3-6 node cluster typical
- Shard by client_id for distribution

**PostgreSQL:**
- Primary for writes
- Read replicas for dashboard queries
- Partition request_logs by date

### Vertical Scaling

**When to scale:**
- Redis memory exhaustion
- PostgreSQL query performance degradation
- CPU saturation on instances

**Recommendations:**
- Redis: More memory for more clients
- PostgreSQL: More CPU/memory for analytics
- Service: More CPU cores for higher concurrency

## Security Considerations

### Data Protection

- Rate limit data expires automatically (TTL)
- No sensitive data stored in Redis
- PostgreSQL connections encrypted (TLS)
- Logs contain only client_id, not payload

### Access Control

- API authentication (add JWT/API keys)
- Network segmentation
- Redis password protection
- PostgreSQL role-based access

### Monitoring

- Anomaly detection for unusual patterns
- Rate limit the rate limiter (meta-limiting)
- Alert on Redis/PostgreSQL failures
- Track suspicious client behavior

## Future Enhancements

1. **Distributed Tracing:** Add OpenTelemetry
2. **Metrics Export:** Prometheus /metrics endpoint
3. **Dynamic Configuration:** Hot-reload client configs
4. **Geo-Distribution:** Multi-region deployment
5. **Advanced Algorithms:** Token bucket, leaky bucket options
6. **Quota Management:** Daily/monthly quotas
7. **Client Prioritization:** VIP client handling
8. **Circuit Breaker:** Protection for downstream APIs

## Technology Choices Rationale

### Why Go?

- Native concurrency (goroutines)
- Fast compilation and execution
- Small binary size (<20 MB)
- Excellent HTTP performance
- Strong standard library

### Why Redis?

- Atomic operations (no race conditions)
- Sub-millisecond latency
- Proven at scale (Twitter, GitHub)
- Simple data structures
- Active community

### Why PostgreSQL?

- ACID compliance for logs
- Rich query capabilities (analytics)
- Time-series optimization (indexes)
- Reliability and maturity
- Easy backup/restore

### Why Sliding Window?

- Most accurate algorithm
- No burst problems
- Fair request distribution
- Simple to understand
- Efficient implementation

## Conclusion

This architecture provides:
- ✅ Sub-millisecond latency
- ✅ Accuracy across multiple instances
- ✅ High availability with fail-safe
- ✅ Horizontal scalability
- ✅ Rich analytics capabilities
- ✅ Production-ready robustness
