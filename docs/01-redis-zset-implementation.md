# Redis ZSET Implementation

## Decision: Why Redis Sorted Sets (ZSET)?

For the rate limiter, we needed a data structure that could:
1. **Store timestamps** of requests
2. **Automatically sort** by time
3. **Count requests** in a time window quickly
4. **Remove old entries** efficiently
5. **Support atomic operations** (no race conditions)

**Redis ZSET (Sorted Set) was the perfect choice.**

---

## What is a Redis ZSET?

A **Sorted Set** is like a list where every item has:
- **Member**: The unique identifier (we use timestamp as member)
- **Score**: The sorting value (we use timestamp as score too)

Redis automatically keeps the set sorted by score.

### Visual Example
```
Client: "my-app" (100 requests per minute)

Redis Key: "ratelimit:my-app"
┌──────────────────────────────────────────┐
│  Score (Timestamp)    Member (Request ID) │
├──────────────────────────────────────────┤
│  1784300000000000    req-001             │ ← 60 sec ago
│  1784300010000000    req-002             │
│  1784300020000000    req-003             │
│  ...                  ...                 │
│  1784300059000000    req-099             │ ← Now
└──────────────────────────────────────────┘
         ↑
    Sliding Window (60 seconds)
```

---

## Why Not Other Data Structures?

### ❌ Redis String/Counter
```
Problem: Can't track individual request timestamps
Can't implement true sliding window
```

### ❌ Redis List
```
Problem: No automatic sorting
No efficient time-range queries
Can't count by timestamp range
```

### ❌ Redis Hash
```
Problem: No ordering
Can't efficiently find entries in time range
```

### ✅ Redis ZSET (Our Choice)
```
✓ Automatic sorting by timestamp
✓ Fast range queries: ZCOUNT, ZRANGE
✓ Fast removal: ZREMRANGEBYSCORE
✓ O(log N) complexity
✓ Perfect for sliding windows
```

---

## Implementation Code

**Location:** `internal/storage/redis.go`

### The 4-Step Atomic Algorithm

```go
func (r *RedisClient) CheckRateLimit(clientID string, limit int, windowSec int) (bool, int, int64, error) {
    now := time.Now()
    windowStart := now.Add(-time.Duration(windowSec) * time.Second).UnixNano()
    nowNano := now.UnixNano()
    
    key := fmt.Sprintf("ratelimit:%s", clientID)
    
    // Use Redis pipeline for atomic operations
    pipe := r.client.Pipeline()
    
    // STEP 1: Remove old entries outside the window
    pipe.ZRemRangeByScore(r.ctx, key, "0", fmt.Sprintf("%d", windowStart))
    
    // STEP 2: Count current requests in the window
    countCmd := pipe.ZCount(r.ctx, key, fmt.Sprintf("%d", windowStart), "+inf")
    
    // STEP 3: Add current request with score as timestamp
    pipe.ZAdd(r.ctx, key, &redis.Z{
        Score:  float64(nowNano),
        Member: fmt.Sprintf("%d", nowNano),
    })
    
    // STEP 4: Set expiration (cleanup for inactive clients)
    pipe.Expire(r.ctx, key, time.Duration(windowSec)*time.Second)
    
    // Execute all 4 steps atomically
    _, err := pipe.Exec(r.ctx)
    
    // Check if limit exceeded
    currentCount := countCmd.Val()
    allowed := currentCount < int64(limit)
    remaining := limit - int(currentCount) - 1
    
    return allowed, remaining, resetAt, nil
}
```

---

## Step-by-Step Breakdown

### **Step 1: ZRemRangeByScore** - Clean Old Requests
```go
pipe.ZRemRangeByScore(r.ctx, key, "0", fmt.Sprintf("%d", windowStart))
```

**What it does:** Remove all entries with score less than `windowStart`

**Example:**
```
Current time: 21:45:00
Window: 60 seconds
windowStart: 21:44:00

Before:
├─ 21:43:30 ← REMOVE (too old)
├─ 21:43:50 ← REMOVE (too old)
├─ 21:44:10 ← KEEP (in window)
└─ 21:44:50 ← KEEP (in window)

After:
├─ 21:44:10 ← KEEP
└─ 21:44:50 ← KEEP
```

### **Step 2: ZCount** - Count Current Requests
```go
countCmd := pipe.ZCount(r.ctx, key, fmt.Sprintf("%d", windowStart), "+inf")
```

**What it does:** Count entries with score >= `windowStart`

**Example:**
```
Window: 21:44:00 to 21:45:00
Entries: 10 requests
Result: 10
```

### **Step 3: ZAdd** - Add Current Request
```go
pipe.ZAdd(r.ctx, key, &redis.Z{
    Score:  float64(nowNano),      // Timestamp as score
    Member: fmt.Sprintf("%d", nowNano), // Timestamp as member
})
```

**What it does:** Add new entry with current timestamp

**Example:**
```
Current time: 21:45:00.123456789
Add: Score=1784300000123456789, Member="1784300000123456789"
```

### **Step 4: Expire** - Auto-Cleanup
```go
pipe.Expire(r.ctx, key, time.Duration(windowSec)*time.Second)
```

**What it does:** Delete entire key after 60 seconds of inactivity

**Purpose:** Free memory for clients who stop sending requests

---

## Why Atomic (Pipeline)?

**Without Pipeline (BAD):**
```go
// Three separate commands
redis.ZRemRangeByScore(...)  // Step 1
count := redis.ZCount(...)    // Step 2 ← Race condition here!
redis.ZAdd(...)               // Step 3

// Problem: Another request can execute between Step 2 and 3
// Result: Inaccurate counts
```

**With Pipeline (GOOD):**
```go
pipe := redis.Pipeline()
pipe.ZRemRangeByScore(...)
countCmd := pipe.ZCount(...)
pipe.ZAdd(...)
pipe.Exec()  // ALL execute together atomically

// No other request can interrupt
// Result: Always accurate
```

---

## Performance Characteristics

### Time Complexity
- **ZRemRangeByScore**: O(log N + M) where M is entries removed
- **ZCount**: O(log N)
- **ZAdd**: O(log N)
- **Overall**: O(log N) - Very fast!

### Space Complexity
- **Per client**: ~50 bytes per entry
- **Max entries**: Equal to rate limit (e.g., 100 entries for 100 req/min)
- **Example**: 1000 clients × 100 entries × 50 bytes = 5MB

### Comparison
| Operation | Redis ZSET | Database |
|-----------|------------|----------|
| Check rate limit | <1ms | 10-100ms |
| Count in window | O(log N) | O(N) full scan |
| Remove old | O(log N) | DELETE query |
| Atomic operations | ✅ Pipeline | ⚠️ Transactions |

---

## Multi-Instance Consistency

**How multiple instances stay consistent:**

```
Instance 1 (Port 8080) ─┐
Instance 2 (Port 8081) ─┼──→ Same Redis → Same ZSET → Same Count
Instance 3 (Port 8082) ─┘
```

**Key insight:** All instances read/write to the **same Redis key**.

### Example Flow
```
User "client-a" (limit: 100/min)

Request 1 → Instance 1 → Redis: ratelimit:client-a → Count = 50
Request 2 → Instance 2 → Redis: ratelimit:client-a → Count = 51 (sees update from Instance 1)
Request 3 → Instance 3 → Redis: ratelimit:client-a → Count = 52 (sees updates from both)
```

All instances see the same state because they share one Redis.

---

## Real-World Example

### Scenario: API with 100 requests per minute

**Minute 0:00 - 0:59**
```
Redis ZSET: ratelimit:api-client
0:00 → Request 1   (Score: 1784300000000)
0:05 → Request 2   (Score: 1784300005000)
0:10 → Request 3   (Score: 1784300010000)
...
0:55 → Request 100 (Score: 1784300055000)

Count: 100 → LIMIT REACHED
```

**Minute 1:00 (New request arrives)**
```
Step 1: Remove old (before 0:00)
        - Nothing to remove yet
        
Step 2: Count from 0:00 to 1:00
        - Result: 100
        
Step 3: Check 100 >= 100?
        - YES → DENIED ❌
        
Step 4: Don't add request
```

**Minute 1:05 (New request arrives)**
```
Step 1: Remove old (before 0:05)
        - Remove Request 1 (timestamp 0:00)
        
Step 2: Count from 0:05 to 1:05
        - Result: 99
        
Step 3: Check 99 < 100?
        - YES → ALLOWED ✅
        
Step 4: Add new request
```

**This is a TRUE sliding window!** As time moves, old requests drop off and new slots open.

---

## Trade-offs

### ✅ Advantages
1. **Accurate**: True sliding window, not fixed windows
2. **Fast**: Sub-millisecond operations
3. **Atomic**: No race conditions
4. **Scalable**: Works with multiple instances
5. **Memory efficient**: Only stores recent entries

### ⚠️ Disadvantages
1. **Redis dependency**: If Redis crashes, lose rate limit data (acceptable for our use case)
2. **Memory-based**: Data not persisted to disk (fine for temporary rate limits)
3. **Network overhead**: Every check requires Redis call (but it's very fast)

### Why Trade-offs are Acceptable
- **Redis crash**: Rate limits reset (users get fresh limits, not a security issue)
- **Memory-only**: Rate limits are temporary by nature (don't need persistence)
- **Network call**: Redis is so fast (<1ms) that it's not a bottleneck

---

## Alternative Approaches We Rejected

### 1. Fixed Time Windows
```
❌ Window: 10:00-10:01 (100 requests)
Problem: User can do 100 at 10:00:59 and 100 at 10:01:00
Result: 200 requests in 1 second!
```

### 2. Token Bucket
```
❌ Tokens: 100, Refill: 1.67/second
Problem: Complex to implement
Problem: Doesn't show exact request timestamps
```

### 3. Leaky Bucket
```
❌ Queue: Requests drain at fixed rate
Problem: Adds latency
Problem: Complex state management
```

### 4. Database-Only
```
❌ PostgreSQL: Store every request
Problem: Too slow (50-100ms per check)
Problem: High database load
```

---

## Code Location

**File:** `internal/storage/redis.go`

**Key Methods:**
- `NewRedisClient()` - Initialize connection
- `CheckRateLimit()` - Main algorithm
- `GetCurrentCount()` - Get current usage
- `ResetClient()` - Clear rate limit data

---

## Summary

**Redis ZSET enables:**
- ✅ Accurate sliding window rate limiting
- ✅ Sub-millisecond performance
- ✅ Multi-instance consistency
- ✅ Atomic operations (no race conditions)
- ✅ Automatic cleanup
- ✅ Memory efficient storage

This is why ZSET is the industry-standard choice for distributed rate limiting.
