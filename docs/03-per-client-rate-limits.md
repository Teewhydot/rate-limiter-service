# Per-Client Rate Limit Configuration

## Decision: Why Different Limits Per Client?

Not all clients have the same needs:
- Banking API: 100 requests/min (critical, but moderate traffic)
- Logistics service: 5,000 requests/min (high volume tracking)
- AI model service: 1,000 requests/min (balanced needs)

**Solution:** Store client-specific configuration in PostgreSQL, apply dynamically.

---

## Architecture

```
Request comes in
     ↓
Handler gets client_id
     ↓
Query PostgreSQL for client config
     ↓
Apply client's specific limit
     ↓
Check against Redis
```

---

## Database Schema

**Location:** `migrations/001_initial_schema.sql`

```sql
CREATE TABLE clients (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    rate_limit INTEGER NOT NULL,      -- Requests allowed
    window_sec INTEGER NOT NULL,       -- Time window in seconds
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

### Example Data
```sql
INSERT INTO clients VALUES
('client-a', 'Banking Service', 100, 60),
('client-b', 'Logistics Provider', 5000, 60),
('client-c', 'AI Model Service', 1000, 60);
```

---

## How It Works

### Step 1: Client Configuration Stored in PostgreSQL
```
clients table:
┌──────────┬──────────────────┬────────┬────────────┐
│ id       │ name             │ limit  │ window_sec │
├──────────┼──────────────────┼────────┼────────────┤
│ client-a │ Banking Service  │ 100    │ 60         │
│ client-b │ Logistics        │ 5000   │ 60         │
│ client-c │ AI Service       │ 1000   │ 60         │
└──────────┴──────────────────┴────────┴────────────┘
```

### Step 2: Request Arrives
```go
// Request body
{
  "client_id": "client-a",
  "resource": "/api/endpoint"
}
```

### Step 3: Lookup Client Config
**Location:** `internal/ratelimiter/limiter.go`

```go
func (rl *RateLimiter) CheckLimit(req models.RateLimitRequest) (*models.RateLimitResponse, error) {
    // Get client configuration from database
    client, err := rl.postgres.GetClient(req.ClientID)
    // Returns: {ID: "client-a", Limit: 100, WindowSec: 60}
    
    // Use client's specific limits
    allowed, remaining, resetAt, err := rl.redis.CheckRateLimit(
        req.ClientID,
        client.Limit,    // ← Client-specific: 100
        client.WindowSec, // ← Client-specific: 60
    )
    
    return &models.RateLimitResponse{
        Allowed:   allowed,
        Remaining: remaining,
        Limit:     client.Limit, // ← Returns client's limit
        ResetAt:   resetAt,
    }, nil
}
```

### Step 4: Apply to Redis Check
```
Client-A:
  Redis Key: "ratelimit:client-a"
  Limit: 100 requests per 60 seconds
  
Client-B:
  Redis Key: "ratelimit:client-b"
  Limit: 5000 requests per 60 seconds
  
Each client has independent tracking!
```

---

## Code Implementation

### PostgreSQL Client Lookup
**Location:** `internal/storage/postgres.go`

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
        &client.Limit,      // ← Client-specific limit
        &client.WindowSec,  // ← Client-specific window
        &client.CreatedAt,
        &client.UpdatedAt,
    )
    
    if err == sql.ErrNoRows {
        return nil, nil // Client not found
    }
    
    return &client, nil
}
```

### Default Fallback
```go
// If client not found in database, use defaults
if client == nil {
    client = &models.Client{
        ID:        req.ClientID,
        Limit:     rl.config.DefaultLimit,     // 100
        WindowSec: rl.config.DefaultWindowSec, // 60
    }
}
```

---

## Real-World Example

### Scenario: Three Different Clients

**Client A (Banking):**
```
Limit: 100 req/min
Time: 10:00:00 - 10:01:00

10:00:05 → Request 1   → Allowed (99 remaining)
10:00:10 → Request 2   → Allowed (98 remaining)
...
10:00:55 → Request 100 → Allowed (0 remaining)
10:00:56 → Request 101 → DENIED ❌
```

**Client B (Logistics) - Same time:**
```
Limit: 5000 req/min
Time: 10:00:00 - 10:01:00

10:00:05 → Request 1    → Allowed (4999 remaining)
10:00:10 → Request 2    → Allowed (4998 remaining)
...
10:00:55 → Request 4999 → Allowed (1 remaining)
10:00:56 → Request 5000 → Allowed (0 remaining)
10:00:57 → Request 5001 → DENIED ❌
```

**Key Point:** Client A and Client B are completely independent!

---

## Creating New Clients

### Via API
```bash
POST http://localhost:8080/api/v1/clients
Content-Type: application/json

{
  "id": "new-client",
  "name": "New Service",
  "limit": 2000,
  "window_sec": 60
}
```

### Via Flutter Dashboard
```
1. Click "New Client" button
2. Fill form:
   - Client ID: new-client
   - Name: New Service
   - Limit: 2000
   - Window: 60
3. Click "Create"
```

### Via SQL
```sql
INSERT INTO clients (id, name, rate_limit, window_sec, created_at, updated_at)
VALUES ('new-client', 'New Service', 2000, 60, NOW(), NOW());
```

---

## Updating Client Limits

### Via API
```bash
PUT http://localhost:8080/api/v1/clients/client-a
Content-Type: application/json

{
  "name": "Banking Service (Updated)",
  "limit": 200,    # Increased from 100
  "window_sec": 60
}
```

**Effect:** Immediate! Next request uses new limit.

### Via Flutter Dashboard
```
1. Click on client card
2. Click "Edit" button (if implemented)
3. Update fields
4. Save
```

---

## Window Configuration

### Different Time Windows

**Fast Window (10 seconds):**
```json
{
  "id": "fast-client",
  "limit": 10,
  "window_sec": 10  // 10 requests per 10 seconds
}
```

**Standard Window (1 minute):**
```json
{
  "id": "standard-client",
  "limit": 100,
  "window_sec": 60  // 100 requests per 60 seconds
}
```

**Long Window (1 hour):**
```json
{
  "id": "hourly-client",
  "limit": 10000,
  "window_sec": 3600  // 10,000 requests per hour
}
```

---

## Client Independence

### Separate Redis Keys
```
Client A: ratelimit:client-a
├─ Limit: 100
├─ Window: 60 seconds
└─ Current: 50 requests

Client B: ratelimit:client-b
├─ Limit: 5000
├─ Window: 60 seconds
└─ Current: 2000 requests

Client C: ratelimit:client-c
├─ Limit: 1000
├─ Window: 60 seconds
└─ Current: 500 requests
```

Each client has **completely independent tracking**. Client A's usage doesn't affect Client B or C.

---

## Performance Optimization: Caching

### Current Implementation (No Cache)
```
Every request → Query PostgreSQL for client config
Latency: ~2-5ms per request
```

### Potential Optimization (Future)
```go
// In-memory cache with TTL
type ClientCache struct {
    cache map[string]*models.Client
    mu    sync.RWMutex
    ttl   time.Duration
}

func (rl *RateLimiter) CheckLimit(req models.RateLimitRequest) {
    // Check cache first
    client := rl.cache.Get(req.ClientID)
    if client == nil {
        // Cache miss, query database
        client, _ = rl.postgres.GetClient(req.ClientID)
        rl.cache.Set(req.ClientID, client, 5*time.Minute)
    }
    
    // Use cached client config
    allowed, _, _, _ := rl.redis.CheckRateLimit(
        req.ClientID,
        client.Limit,
        client.WindowSec,
    )
}
```

**Trade-off:** 
- ✅ Faster (no DB query)
- ⚠️ Config changes take up to 5 min to propagate
- ⚠️ Memory usage increases

**Current choice:** No cache (always fresh config, acceptable 2-5ms overhead).

---

## Multi-Tenant Considerations

### Client Isolation
Each client's rate limit is **completely isolated**:

```
Client A uses 100/100 → Client A blocked
Client B uses 0/5000  → Client B still allowed

No "noisy neighbor" problem!
```

### Fair Resource Allocation
```
System capacity: 15,000 req/sec (3 instances × 5,000)

Client A: 100/min   = 1.67/sec   (0.01% of capacity)
Client B: 5000/min  = 83.33/sec  (0.56% of capacity)
Client C: 1000/min  = 16.67/sec  (0.11% of capacity)

Total: ~102/sec (0.68% of capacity)
Plenty of headroom for more clients!
```

---

## Dashboard Integration

### Client List View (Flutter)
```dart
// Fetches all clients with their limits
GET /api/v1/clients

Response:
{
  "clients": [
    {
      "id": "client-a",
      "name": "Banking Service",
      "limit": 100,
      "window_sec": 60
    },
    {
      "id": "client-b",
      "name": "Logistics",
      "limit": 5000,
      "window_sec": 60
    }
  ]
}
```

### Client Details View
```dart
// Shows current usage vs limit
GET /api/v1/stats/client-a

Response:
{
  "client_id": "client-a",
  "current_used": 50,
  "limit": 100,
  "remaining": 50,
  "window_sec": 60
}
```

---

## Default Configuration

**Location:** `internal/config/config.go`

```go
type Config struct {
    DefaultLimit     int `env:"DEFAULT_LIMIT" envDefault:"100"`
    DefaultWindowSec int `env:"DEFAULT_WINDOW_SEC" envDefault:"60"`
}
```

**Used when:** Client not found in database

**Example:** New client makes first request before being registered:
```go
// Client "unknown-client" not in database
client := &models.Client{
    ID:        "unknown-client",
    Limit:     100,  // ← Default
    WindowSec: 60,   // ← Default
}
```

---

## Migration Strategy

### Adding New Clients
```sql
-- No downtime required
INSERT INTO clients (id, name, rate_limit, window_sec, created_at, updated_at)
VALUES ('new-client', 'New Service', 1500, 60, NOW(), NOW());

-- Immediately effective on next request
```

### Updating Existing Clients
```sql
-- No downtime required
UPDATE clients
SET rate_limit = 200,  -- Change limit
    updated_at = NOW()
WHERE id = 'client-a';

-- Next request uses new limit
```

### Removing Clients
```sql
-- No downtime required
DELETE FROM clients WHERE id = 'old-client';

-- Future requests will use default limits
```

---

## Summary

**Per-client rate limits provide:**
- ✅ Flexibility (different clients, different needs)
- ✅ Fairness (no noisy neighbors)
- ✅ Scalability (thousands of clients, minimal overhead)
- ✅ Dynamic configuration (change limits without restart)
- ✅ Granular control (per-client monitoring and adjustment)

**Key components:**
- PostgreSQL stores client configuration
- Redis enforces client-specific limits
- Each client has independent tracking
- Limits can be changed on-the-fly
- Default fallback for unknown clients

This design is **flexible** and **production-ready** for multi-tenant rate limiting.
