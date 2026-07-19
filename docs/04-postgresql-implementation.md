# PostgreSQL Implementation

## Decision: Why PostgreSQL Alongside Redis?

Redis is perfect for fast rate limiting, but we also need:
1. **Persistent storage** (survive restarts)
2. **Client configuration** (limits, window sizes)
3. **Historical analytics** (usage statistics)
4. **Audit logs** (compliance, debugging)

**Solution:** PostgreSQL for persistence, Redis for speed.

---

## Two-Database Architecture

```
                Request Flow
                     ↓
         ┌───────────┴───────────┐
         ↓                       ↓
    PostgreSQL               Redis
    (Slow, Persistent)       (Fast, Volatile)
         ↓                       ↓
    Configuration            Rate Limit State
    Historical Data          Current Counts
```

### Division of Responsibility

| Data Type | Storage | Why |
|-----------|---------|-----|
| Client config | PostgreSQL | Needs persistence |
| Rate limit state | Redis | Needs speed |
| Request logs | PostgreSQL | Needs persistence |
| Current counts | Redis | Needs speed |
| Analytics | PostgreSQL | Complex queries |

---

## Database Schema

**Location:** `migrations/001_initial_schema.sql`

### Table 1: clients (Configuration)
```sql
CREATE TABLE clients (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    rate_limit INTEGER NOT NULL,
    window_sec INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_clients_id ON clients(id);
```

**Purpose:** Store per-client rate limit configuration

**Sample Data:**
```sql
id         | name              | rate_limit | window_sec
-----------|-------------------|------------|------------
client-a   | Banking Service   | 100        | 60
client-b   | Logistics         | 5000       | 60
```

### Table 2: request_logs (Analytics)
```sql
CREATE TABLE request_logs (
    id SERIAL PRIMARY KEY,
    client_id VARCHAR(255) NOT NULL,
    resource VARCHAR(255),
    allowed BOOLEAN NOT NULL,
    response_time_ms BIGINT NOT NULL,
    timestamp TIMESTAMP NOT NULL
);

CREATE INDEX idx_request_logs_client ON request_logs(client_id);
CREATE INDEX idx_request_logs_timestamp ON request_logs(timestamp);
CREATE INDEX idx_request_logs_client_timestamp ON request_logs(client_id, timestamp);
```

**Purpose:** Store request history for analytics

**Sample Data:**
```sql
id | client_id | resource      | allowed | response_time_ms | timestamp
---|-----------|---------------|---------|------------------|------------------
1  | client-a  | /api/endpoint | true    | 2                | 2026-07-17 10:00
2  | client-a  | /api/endpoint | true    | 3                | 2026-07-17 10:01
3  | client-a  | /api/endpoint | false   | 1                | 2026-07-17 10:02
```

---

## Connection Pool Configuration

**Location:** `internal/storage/postgres.go`

```go
func NewPostgresClient(cfg *config.Config) (*PostgresClient, error) {
    db, err := sql.Open("postgres", cfg.PostgresDSN())
    
    // Configure connection pool
    db.SetMaxOpenConns(25)           // Max 25 concurrent connections
    db.SetMaxIdleConns(5)            // Keep 5 connections ready
    db.SetConnMaxLifetime(5 * time.Minute) // Recycle after 5 min
    
    // Test connection
    if err := db.Ping(); err != nil {
        return nil, err
    }
    
    return &PostgresClient{db: db}, nil
}
```

### Why Connection Pooling?

**Without Pool (Bad):**
```
Request 1: Open connection → Query → Close connection
Request 2: Open connection → Query → Close connection
Request 3: Open connection → Query → Close connection

Opening connection: ~50ms each time
Total overhead: 150ms for 3 requests
```

**With Pool (Good):**
```
Request 1: Get from pool → Query → Return to pool
Request 2: Reuse connection → Query → Return to pool
Request 3: Reuse connection → Query → Return to pool

Getting from pool: ~0.1ms
Total overhead: 0.3ms for 3 requests
```

### Pool Settings Explained

```go
SetMaxOpenConns(25)
```
- **Maximum 25 simultaneous queries**
- Prevents overwhelming PostgreSQL
- With 3 instances: 75 max connections total

```go
SetMaxIdleConns(5)
```
- **Keep 5 connections ready** (not closed)
- Faster response for next request
- Balance between speed and resources

```go
SetConnMaxLifetime(5 * time.Minute)
```
- **Recycle connections** after 5 minutes
- Prevents stale connections
- Allows PostgreSQL to rebalance

---

## CRUD Operations

### Create Client
**Location:** `internal/storage/postgres.go`

```go
func (p *PostgresClient) CreateClient(client *models.Client) error {
    query := `
        INSERT INTO clients (id, name, rate_limit, window_sec, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6)
    `
    
    now := time.Now()
    _, err := p.db.Exec(query,
        client.ID,
        client.Name,
        client.Limit,
        client.WindowSec,
        now,
        now,
    )
    
    return err
}
```

**Used when:** Creating new client via API or dashboard

### Get Client
```go
func (p *PostgresClient) GetClient(clientID string) (*models.Client, error) {
    query := `
        SELECT id, name, rate_limit, window_sec, created_at, updated_at
        FROM clients
        WHERE id = $1
    `
    
    var client models.Client
    err := p.db.QueryRow(query, clientID).Scan(
        &client.ID,
        &client.Name,
        &client.Limit,
        &client.WindowSec,
        &client.CreatedAt,
        &client.UpdatedAt,
    )
    
    if err == sql.ErrNoRows {
        return nil, nil // Client not found
    }
    
    return &client, err
}
```

**Used when:** Every rate limit check (to get client's limits)

### Update Client
```go
func (p *PostgresClient) UpdateClient(client *models.Client) error {
    query := `
        UPDATE clients
        SET name = $2, rate_limit = $3, window_sec = $4, updated_at = $5
        WHERE id = $1
    `
    
    _, err := p.db.Exec(query,
        client.ID,
        client.Name,
        client.Limit,
        client.WindowSec,
        time.Now(),
    )
    
    return err
}
```

**Used when:** Updating client limits via API

### List All Clients
```go
func (p *PostgresClient) ListClients() ([]models.Client, error) {
    query := `
        SELECT id, name, rate_limit, window_sec, created_at, updated_at
        FROM clients
        ORDER BY created_at DESC
    `
    
    rows, err := p.db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var clients []models.Client
    for rows.Next() {
        var client models.Client
        err := rows.Scan(
            &client.ID,
            &client.Name,
            &client.Limit,
            &client.WindowSec,
            &client.CreatedAt,
            &client.UpdatedAt,
        )
        if err != nil {
            return nil, err
        }
        clients = append(clients, client)
    }
    
    return clients, nil
}
```

**Used when:** Dashboard shows client list

---

## Async Logging Pattern

### The Problem: Synchronous Logging Blocks
```go
// BAD: Synchronous logging
func CheckRateLimit() {
    // Check Redis (1ms)
    allowed := redis.CheckLimit()
    
    // Log to PostgreSQL (5ms) ← BLOCKS HERE
    postgres.LogRequest(...)
    
    // Return response
    return allowed
}

Total time: 6ms (1ms + 5ms)
```

### The Solution: Async Logging
```go
// GOOD: Asynchronous logging
func CheckRateLimit() {
    // Check Redis (1ms)
    allowed := redis.CheckLimit()
    
    // Queue log (instant, non-blocking)
    asyncLogger.Log(...)
    
    // Return response immediately
    return allowed
}

Total time: 1ms (PostgreSQL write happens in background)
```

### Implementation
**Location:** `internal/logger/async_logger.go`

```go
type AsyncLogger struct {
    logChan  chan models.RequestLog
    batch    []models.RequestLog
    postgres *storage.PostgresClient
}

func (al *AsyncLogger) Log(log models.RequestLog) {
    // Send to buffered channel (non-blocking)
    select {
    case al.logChan <- log:
        // Sent successfully
    default:
        // Channel full, drop log (don't block request!)
    }
}

func (al *AsyncLogger) processLogs() {
    ticker := time.NewTicker(5 * time.Second)
    
    for {
        select {
        case log := <-al.logChan:
            al.batch = append(al.batch, log)
            
            // Flush when batch full
            if len(al.batch) >= 100 {
                al.flushBatch()
            }
            
        case <-ticker.C:
            // Flush every 5 seconds
            if len(al.batch) > 0 {
                al.flushBatch()
            }
        }
    }
}

func (al *AsyncLogger) flushBatch() {
    // Bulk insert all logs at once
    al.postgres.LogRequestBatch(al.batch)
    al.batch = nil
}
```

### Batch Insert Performance
```go
func (p *PostgresClient) LogRequestBatch(logs []models.RequestLog) error {
    tx, _ := p.db.Begin()
    defer tx.Rollback()
    
    stmt, _ := tx.Prepare(`
        INSERT INTO request_logs (client_id, resource, allowed, response_time_ms, timestamp)
        VALUES ($1, $2, $3, $4, $5)
    `)
    defer stmt.Close()
    
    for _, log := range logs {
        stmt.Exec(log.ClientID, log.Resource, log.Allowed, log.ResponseTime, log.Timestamp)
    }
    
    return tx.Commit()
}
```

**Performance:**
- **Individual inserts**: 100 logs × 5ms = 500ms
- **Batch insert**: 100 logs = 20ms
- **Speedup**: 25x faster!

---

## Analytics Queries

### Usage Statistics
```go
func (p *PostgresClient) GetUsageStats(clientID string, startDate, endDate time.Time) (*models.UsageStats, error) {
    query := `
        SELECT
            client_id,
            COUNT(*) as total_requests,
            SUM(CASE WHEN allowed = true THEN 1 ELSE 0 END) as allowed_requests,
            SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) as blocked_requests,
            AVG(response_time_ms) as avg_response_time
        FROM request_logs
        WHERE client_id = $1 AND timestamp BETWEEN $2 AND $3
        GROUP BY client_id
    `
    
    var stats models.UsageStats
    err := p.db.QueryRow(query, clientID, startDate, endDate).Scan(
        &stats.ClientID,
        &stats.TotalRequests,
        &stats.AllowedRequests,
        &stats.BlockedRequests,
        &stats.AvgResponseTime,
    )
    
    return &stats, err
}
```

**Used for:** Dashboard "Usage Statistics" card

**Example result:**
```json
{
  "client_id": "client-a",
  "total_requests": 10000,
  "allowed_requests": 9800,
  "blocked_requests": 200,
  "avg_response_time_ms": 2.5
}
```

### Trend Data (Time-Series)
```go
func (p *PostgresClient) GetTrendData(clientID string, startDate, endDate time.Time, intervalHours int) ([]models.TrendData, error) {
    query := `
        SELECT
            DATE_TRUNC('hour', timestamp) + 
            INTERVAL '1 hour' * (EXTRACT(hour FROM timestamp)::int / $4) * $4 as time_bucket,
            COUNT(*) as request_count,
            SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) as blocked_count,
            AVG(response_time_ms) as avg_response_time
        FROM request_logs
        WHERE client_id = $1 AND timestamp BETWEEN $2 AND $3
        GROUP BY time_bucket
        ORDER BY time_bucket
    `
    
    rows, _ := p.db.Query(query, clientID, startDate, endDate, intervalHours)
    defer rows.Close()
    
    var trends []models.TrendData
    for rows.Next() {
        var trend models.TrendData
        rows.Scan(&trend.Timestamp, &trend.RequestCount, &trend.BlockedCount, &trend.AvgResponseTime)
        trends = append(trends, trend)
    }
    
    return trends, nil
}
```

**Used for:** Dashboard charts showing request trends over time

**Example result:**
```json
[
  {
    "timestamp": "2026-07-17T10:00:00Z",
    "request_count": 450,
    "blocked_count": 10,
    "avg_response_time_ms": 2.1
  },
  {
    "timestamp": "2026-07-17T11:00:00Z",
    "request_count": 520,
    "blocked_count": 15,
    "avg_response_time_ms": 2.3
  }
]
```

---

## Index Strategy

### Why Indexes Matter
```sql
-- WITHOUT INDEX
SELECT * FROM request_logs WHERE client_id = 'client-a';
-- Scans entire table: 1,000,000 rows → 500ms

-- WITH INDEX
SELECT * FROM request_logs WHERE client_id = 'client-a';
-- Uses index: Finds rows directly → 5ms
```

### Indexes Created
```sql
CREATE INDEX idx_request_logs_client ON request_logs(client_id);
CREATE INDEX idx_request_logs_timestamp ON request_logs(timestamp);
CREATE INDEX idx_request_logs_client_timestamp ON request_logs(client_id, timestamp);
```

**Usage:**
- `idx_request_logs_client` → Fast lookup by client
- `idx_request_logs_timestamp` → Fast time-range queries
- `idx_request_logs_client_timestamp` → Composite for analytics queries

---

## Data Retention Strategy

### Current: No Automatic Cleanup
All logs are kept indefinitely.

### Future: Partition by Time
```sql
-- Create partitions by month
CREATE TABLE request_logs_2026_07 PARTITION OF request_logs
FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');

CREATE TABLE request_logs_2026_08 PARTITION OF request_logs
FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');

-- Drop old partitions
DROP TABLE request_logs_2025_01;
```

**Benefits:**
- Easy cleanup (drop partition vs DELETE millions of rows)
- Better query performance (scan only relevant partition)
- Maintain recent data, discard old data

---

## Backup Strategy

### pg_dump (Simple)
```bash
# Backup entire database
pg_dump -h localhost -U ratelimiter ratelimiter > backup.sql

# Restore
psql -h localhost -U ratelimiter ratelimiter < backup.sql
```

### Continuous Archiving (Production)
```bash
# PostgreSQL WAL archiving
archive_mode = on
archive_command = 'cp %p /var/lib/postgresql/archive/%f'

# Point-in-time recovery possible
```

---

## Monitoring Queries

### Active Connections
```sql
SELECT count(*) FROM pg_stat_activity 
WHERE datname = 'ratelimiter';
```

### Slow Queries
```sql
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
WHERE query LIKE '%request_logs%'
ORDER BY mean_exec_time DESC
LIMIT 10;
```

### Table Size
```sql
SELECT 
    pg_size_pretty(pg_total_relation_size('request_logs')) as total_size,
    pg_size_pretty(pg_relation_size('request_logs')) as table_size,
    pg_size_pretty(pg_indexes_size('request_logs')) as indexes_size;
```

---

## Trade-offs

### ✅ Advantages
1. **Persistence**: Data survives crashes
2. **Complex queries**: Analytics, reporting
3. **ACID compliance**: Transactional guarantees
4. **Mature ecosystem**: Tools, monitoring, backups

### ⚠️ Disadvantages
1. **Slower than Redis**: 2-5ms vs <1ms
2. **Disk I/O**: Limited by disk speed
3. **Requires maintenance**: Vacuuming, backups

### Why Trade-offs Are Acceptable
- **Client config**: Queried once per rate limit check (acceptable 2-5ms)
- **Request logs**: Written async (doesn't block)
- **Analytics**: Queried infrequently (dashboard loads)

---

## Summary

**PostgreSQL provides:**
- ✅ Persistent client configuration
- ✅ Historical request logs for analytics
- ✅ Complex query capabilities
- ✅ Transactional guarantees
- ✅ Backup and recovery

**Key patterns:**
- Connection pooling (25 connections per instance)
- Async logging (non-blocking writes)
- Batch inserts (25x faster)
- Strategic indexes (fast queries)
- Prepared statements (SQL injection prevention)

**Together with Redis:**
- Redis: Speed (rate limit checks)
- PostgreSQL: Persistence (configuration, analytics)
- Best of both worlds!
