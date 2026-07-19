# API Examples for Testing

## Rate Limit Check

### Check Rate Limit (Allowed)
```bash
curl -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-a",
    "resource": "/api/payment"
  }'
```

**Expected Response (200 OK):**
```json
{
  "allowed": true,
  "remaining": 99,
  "limit": 100,
  "reset_at": 1700000000
}
```

### Check Rate Limit (Blocked)
```bash
# After exhausting limit
curl -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "client-a",
    "resource": "/api/payment"
  }'
```

**Expected Response (429 Too Many Requests):**
```json
{
  "allowed": false,
  "remaining": 0,
  "limit": 100,
  "reset_at": 1700000000,
  "retry_after": 45
}
```

## Client Management

### Create New Client
```bash
curl -X POST http://localhost:8080/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{
    "id": "client-d",
    "name": "Payment Gateway Service",
    "limit": 2000,
    "window_sec": 60
  }'
```

**Response (201 Created):**
```json
{
  "id": "client-d",
  "name": "Payment Gateway Service",
  "limit": 2000,
  "window_sec": 60,
  "created_at": "2024-01-31T10:30:00Z",
  "updated_at": "2024-01-31T10:30:00Z"
}
```

### Get Single Client
```bash
curl http://localhost:8080/api/v1/clients/client-a
```

**Response (200 OK):**
```json
{
  "id": "client-a",
  "name": "Client A - Banking Service",
  "limit": 100,
  "window_sec": 60,
  "created_at": "2024-01-31T10:00:00Z",
  "updated_at": "2024-01-31T10:00:00Z"
}
```

### List All Clients
```bash
curl http://localhost:8080/api/v1/clients
```

**Response (200 OK):**
```json
{
  "clients": [
    {
      "id": "client-a",
      "name": "Client A - Banking Service",
      "limit": 100,
      "window_sec": 60,
      "created_at": "2024-01-31T10:00:00Z",
      "updated_at": "2024-01-31T10:00:00Z"
    },
    {
      "id": "client-b",
      "name": "Client B - Logistics Provider",
      "limit": 5000,
      "window_sec": 60,
      "created_at": "2024-01-31T10:00:00Z",
      "updated_at": "2024-01-31T10:00:00Z"
    }
  ],
  "count": 2
}
```

### Update Client
```bash
curl -X PUT http://localhost:8080/api/v1/clients/client-a \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Client A - Updated Banking Service",
    "limit": 200,
    "window_sec": 60
  }'
```

**Response (200 OK):**
```json
{
  "id": "client-a",
  "name": "Client A - Updated Banking Service",
  "limit": 200,
  "window_sec": 60,
  "created_at": "2024-01-31T10:00:00Z",
  "updated_at": "2024-01-31T11:30:00Z"
}
```

## Dashboard API (For Flutter App)

### Get Usage Statistics
```bash
# Last 30 days (default)
curl "http://localhost:8080/api/v1/dashboard/usage/client-a?days=30"

# Last 10 days
curl "http://localhost:8080/api/v1/dashboard/usage/client-a?days=10"

# Custom date range
curl "http://localhost:8080/api/v1/dashboard/usage/client-a?start_date=2024-01-01T00:00:00Z&end_date=2024-01-31T23:59:59Z"
```

**Response (200 OK):**
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

### Get Trend Data (For Charts/Graphs)
```bash
# Last 10 days
curl "http://localhost:8080/api/v1/dashboard/trends/client-a?days=10"

# Last 15 days
curl "http://localhost:8080/api/v1/dashboard/trends/client-a?days=15"

# Last 30 days
curl "http://localhost:8080/api/v1/dashboard/trends/client-a?days=30"
```

**Response (200 OK):**
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
    },
    {
      "timestamp": "2024-01-21T04:00:00Z",
      "request_count": 1180,
      "blocked_count": 32,
      "avg_response_time_ms": 2.1
    },
    {
      "timestamp": "2024-01-21T08:00:00Z",
      "request_count": 1420,
      "blocked_count": 67,
      "avg_response_time_ms": 2.4
    }
  ]
}
```

### Get Real-Time Statistics
```bash
curl http://localhost:8080/api/v1/stats/client-a
```

**Response (200 OK):**
```json
{
  "client_id": "client-a",
  "limit": 100,
  "window_sec": 60,
  "current_used": 45,
  "remaining": 55
}
```

## Health Check

### Check Service Health
```bash
curl http://localhost:8080/health
```

**Response (200 OK - Healthy):**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-31T10:30:00Z",
  "redis": "healthy",
  "postgres": "healthy"
}
```

**Response (503 Service Unavailable - Degraded):**
```json
{
  "status": "degraded",
  "timestamp": "2024-01-31T10:30:00Z",
  "redis": "unhealthy",
  "redis_error": "connection refused",
  "postgres": "healthy"
}
```

## Load Testing Examples

### Using Apache Bench (ab)

```bash
# Create payload file
echo '{"client_id":"client-b"}' > /tmp/payload.json

# Send 10,000 requests with 100 concurrent connections
ab -n 10000 -c 100 \
  -p /tmp/payload.json \
  -T application/json \
  http://localhost:8080/api/v1/ratelimit/check
```

### Using curl in a loop

```bash
# Test rate limit accuracy
for i in {1..110}; do
  curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"client-a"}' | jq '.allowed'
done | grep -c "true"
# Should output exactly 100
```

### Multi-Instance Test

```bash
# Send requests to different instances
for i in {1..50}; do
  curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"test-client"}' &
  
  curl -s -X POST http://localhost:8081/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"test-client"}' &
  
  curl -s -X POST http://localhost:8082/api/v1/ratelimit/check \
    -H "Content-Type: application/json" \
    -d '{"client_id":"test-client"}' &
done
wait
```

## Error Responses

### Invalid Request Format
```bash
curl -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"invalid": "data"}'
```

**Response (400 Bad Request):**
```json
{
  "error": "Invalid request format",
  "details": "Key: 'RateLimitRequest.ClientID' Error:Field validation for 'ClientID' failed on the 'required' tag"
}
```

### Client Not Found
```bash
curl http://localhost:8080/api/v1/clients/non-existent-client
```

**Response (404 Not Found):**
```json
{
  "error": "Client not found"
}
```

### Internal Server Error
**Response (500 Internal Server Error):**
```json
{
  "error": "Internal server error"
}
```

## Response Headers

All rate limit responses include standard rate limit headers:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1700000000
Retry-After: 45  (only when blocked)
```

## Flutter Integration Example

```dart
// Example Flutter service class
class RateLimiterService {
  final String baseUrl = 'http://localhost:8080/api/v1';
  
  Future<UsageStats> getUsageStats(String clientId, int days) async {
    final response = await http.get(
      Uri.parse('$baseUrl/dashboard/usage/$clientId?days=$days'),
    );
    
    if (response.statusCode == 200) {
      return UsageStats.fromJson(json.decode(response.body));
    } else {
      throw Exception('Failed to load usage stats');
    }
  }
  
  Future<TrendData> getTrendData(String clientId, int days) async {
    final response = await http.get(
      Uri.parse('$baseUrl/dashboard/trends/$clientId?days=$days'),
    );
    
    if (response.statusCode == 200) {
      return TrendData.fromJson(json.decode(response.body));
    } else {
      throw Exception('Failed to load trend data');
    }
  }
  
  Future<RealtimeStats> getRealtimeStats(String clientId) async {
    final response = await http.get(
      Uri.parse('$baseUrl/stats/$clientId'),
    );
    
    if (response.statusCode == 200) {
      return RealtimeStats.fromJson(json.decode(response.body));
    } else {
      throw Exception('Failed to load realtime stats');
    }
  }
}
```

## Testing Checklist

- [x] Rate limit check returns correct allow/deny
- [x] Remaining count decrements correctly
- [x] Reset timestamp is accurate
- [x] Retry-After header present when blocked
- [x] Multiple instances share same limit
- [x] Client isolation (different clients don't interfere)
- [x] Sliding window accuracy
- [x] Dashboard APIs return valid data
- [x] Health check reflects system status
- [x] Fail-safe mode works when Redis is down
- [x] Async logging doesn't block requests
- [x] Performance meets <5ms latency target
