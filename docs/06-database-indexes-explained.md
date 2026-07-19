# Database Indexes Explained

## What is a Database Index?

Think of a database index like the **index at the back of a textbook**:

**Without an index (textbook):**
- Want to find "PostgreSQL"?
- Read every page from start to finish
- Very slow! 📚😫

**With an index (textbook):**
- Look up "PostgreSQL" in index
- Index says: "Page 42, 87, 201"
- Jump directly to those pages
- Very fast! 📖✨

---

## Real-World Analogy

### Phone Book Without Index
```
Search for "John Smith"
↓
Read every single entry:
- Aaron Anderson
- Betty Brown
- Charlie Clark
- David Davis
... 10,000 names later ...
- John Smith ← Found it!

Time: 10 minutes
```

### Phone Book With Index (Alphabetical)
```
Search for "John Smith"
↓
Jump to "J" section
↓
Jump to "Sm" section
↓
John Smith ← Found it!

Time: 5 seconds
```

**Database index = Alphabetical organization**

---

## Your Migration File Explained

Let's break down each index in your schema:

### The Table

```sql
CREATE TABLE request_logs (
    id BIGSERIAL PRIMARY KEY,           -- Auto-incrementing ID
    client_id VARCHAR(255) NOT NULL,    -- Which client made request
    resource VARCHAR(500),               -- Which endpoint
    allowed BOOLEAN NOT NULL,            -- Was it allowed?
    response_time_ms BIGINT NOT NULL,    -- How fast?
    timestamp TIMESTAMP NOT NULL         -- When?
);
```

Without indexes, every query scans **ALL ROWS** (slow!).

---

### Index 1: `idx_request_logs_client_id`

```sql
CREATE INDEX idx_request_logs_client_id ON request_logs(client_id);
```

**What it does:**
- Creates a sorted "phone book" for the `client_id` column
- Makes lookups by client_id super fast

**Used by this query:**
```sql
-- Get all logs for specific client
SELECT * FROM request_logs WHERE client_id = 'client-a';
```

**Without index:**
```
Scan all 1,000,000 rows → 500ms
```

**With index:**
```
Look up 'client-a' in index → Jump to rows → 5ms
```

**Name explanation:**
- `idx_` = prefix meaning "this is an index"
- `request_logs` = table name
- `client_id` = column being indexed

So: `idx_request_logs_client_id` = **"index on request_logs table for client_id column"**

---

### Index 2: `idx_request_logs_timestamp`

```sql
CREATE INDEX idx_request_logs_timestamp ON request_logs(timestamp);
```

**What it does:**
- Creates a sorted "phone book" for the `timestamp` column
- Makes date range queries fast

**Used by this query:**
```sql
-- Get logs from last 7 days
SELECT * FROM request_logs 
WHERE timestamp >= NOW() - INTERVAL '7 days';
```

**Why it's fast:**
- Timestamps are sorted in the index
- Database can quickly find "start of 7 days ago"
- Then scan forward until "now"
- No need to check every row

**Without index:** Check every row's timestamp = SLOW
**With index:** Jump to start date, scan forward = FAST

---

### Index 3: `idx_request_logs_client_timestamp` (Composite)

```sql
CREATE INDEX idx_request_logs_client_timestamp 
ON request_logs(client_id, timestamp);
```

**What it does:**
- Creates a "phone book" sorted by BOTH columns
- First by client_id, then by timestamp within each client

**Visual representation:**
```
Index organized like this:

client-a
  ├─ 2026-07-01 10:00:00
  ├─ 2026-07-01 10:01:00
  ├─ 2026-07-01 10:02:00
  └─ ... more timestamps

client-b
  ├─ 2026-07-01 09:00:00
  ├─ 2026-07-01 09:01:00
  └─ ... more timestamps

client-c
  └─ ... timestamps
```

**Used by this query:**
```sql
-- Get logs for specific client in date range
SELECT * FROM request_logs 
WHERE client_id = 'client-a' 
  AND timestamp BETWEEN '2026-07-01' AND '2026-07-31';
```

**Why it's better than using two separate indexes:**
- Single index lookup instead of two
- More efficient memory usage
- Faster filtering

**This is your MOST USED INDEX** because dashboard queries filter by both client and date!

---

### Index 4: `idx_request_logs_analytics` (Composite with 3 columns)

```sql
CREATE INDEX idx_request_logs_analytics 
ON request_logs(client_id, timestamp, allowed);
```

**What it does:**
- "Phone book" sorted by client_id, then timestamp, then allowed
- Optimized for analytics queries

**Used by this query:**
```sql
-- Count blocked requests for client in date range
SELECT 
    client_id,
    COUNT(*) as total,
    SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) as blocked
FROM request_logs
WHERE client_id = 'client-a'
  AND timestamp BETWEEN '2026-07-01' AND '2026-07-31'
GROUP BY client_id;
```

**Why include `allowed` in index:**
- Dashboard shows "blocked vs allowed" stats
- Including `allowed` in index = "covering index"
- Database can answer query ENTIRELY from index
- Never needs to touch actual table rows = SUPER FAST

**Covering Index:** Index contains ALL columns needed by query.

---

## Index Naming Convention

**Pattern:** `idx_<table>_<column1>_<column2>_...`

Examples from your project:
- `idx_request_logs_client_id` = Index on `request_logs` table, `client_id` column
- `idx_request_logs_timestamp` = Index on `request_logs` table, `timestamp` column
- `idx_request_logs_client_timestamp` = Index on `request_logs` table, `client_id` and `timestamp` columns

**You didn't make these names up!** This is standard PostgreSQL naming convention.

---

## Why Multiple Indexes?

**Question:** Why not just one index?

**Answer:** Different queries need different indexes.

### Example Queries:

**Query A:** Filter by client only
```sql
WHERE client_id = 'client-a'
→ Uses: idx_request_logs_client_id
```

**Query B:** Filter by date only
```sql
WHERE timestamp > '2026-07-01'
→ Uses: idx_request_logs_timestamp
```

**Query C:** Filter by client AND date
```sql
WHERE client_id = 'client-a' AND timestamp > '2026-07-01'
→ Uses: idx_request_logs_client_timestamp (best choice)
```

**Query D:** Analytics (client + date + allowed)
```sql
WHERE client_id = 'client-a' 
  AND timestamp > '2026-07-01'
  AND allowed = false
→ Uses: idx_request_logs_analytics (covering index!)
```

PostgreSQL automatically chooses the best index for each query!

---

## Performance Comparison

Let's say you have **1 million request logs**:

### Without Indexes:
```sql
SELECT * FROM request_logs WHERE client_id = 'client-a';

Process:
1. Read row 1: Check if client_id = 'client-a'? No.
2. Read row 2: Check if client_id = 'client-a'? No.
3. Read row 3: Check if client_id = 'client-a'? Yes! Add to results.
... repeat for ALL 1,000,000 rows ...

Time: 500ms - 2 seconds
```

### With Index:
```sql
SELECT * FROM request_logs WHERE client_id = 'client-a';

Process:
1. Look up 'client-a' in index (binary search)
2. Index says: "Rows 12450-15789 match"
3. Jump directly to those rows
4. Read only matching rows

Time: 2-10ms
```

**50-100x faster!** 🚀

---

## Trade-offs

### Advantages ✅
- Queries are MUCH faster
- Essential for large tables
- Automatic query optimization

### Disadvantages ⚠️
- Takes disk space (each index = copy of columns)
- Slows down INSERT/UPDATE/DELETE slightly (index must be updated)
- More indexes = more maintenance

**Rule of thumb:**
- Index columns used in WHERE clauses
- Index columns used in JOIN conditions
- Index columns used in ORDER BY
- Don't index everything (only what's queried frequently)

---

## How PostgreSQL Uses Indexes

### Query Planner

PostgreSQL has a "query planner" that automatically chooses the best index:

```sql
-- You write this
SELECT * FROM request_logs 
WHERE client_id = 'client-a' 
  AND timestamp > '2026-07-01';

-- PostgreSQL thinks:
-- Option 1: Use idx_request_logs_client_id
-- Option 2: Use idx_request_logs_timestamp
-- Option 3: Use idx_request_logs_client_timestamp ← BEST!
-- Option 4: No index (full table scan)

-- Chooses Option 3 automatically!
```

You can see the plan with `EXPLAIN`:

```sql
EXPLAIN SELECT * FROM request_logs 
WHERE client_id = 'client-a' 
  AND timestamp > '2026-07-01';

Result:
Index Scan using idx_request_logs_client_timestamp
  Index Cond: ((client_id = 'client-a') AND (timestamp > '2026-07-01'))
  Rows: 1000  Cost: 5.23
```

---

## When Are Indexes NOT Needed?

### Small Tables
```sql
-- Table has 100 rows
-- Index overhead > benefit
-- Full table scan is fine
```

### Write-Heavy Tables
```sql
-- Table gets 10,000 inserts/second
-- Every insert updates indexes
-- Too much overhead
```

### Columns With Few Unique Values
```sql
-- Column 'allowed' has only 2 values: true, false
-- Index not very helpful (50% of rows match either value)
-- Full scan might be faster
```

**Your case:** `request_logs` will have millions of rows → Indexes are ESSENTIAL!

---

## Creating Indexes - Best Practices

### ✅ Good Index
```sql
-- Single column, frequently queried
CREATE INDEX idx_orders_customer_id ON orders(customer_id);

-- Composite, matches query pattern
CREATE INDEX idx_orders_customer_date ON orders(customer_id, order_date);
```

### ❌ Bad Index
```sql
-- Too many columns (rarely needed)
CREATE INDEX idx_orders_everything ON orders(customer_id, order_date, total, status, created_at);

-- Random column order (doesn't match queries)
CREATE INDEX idx_orders_date_customer ON orders(order_date, customer_id);
-- But queries filter by customer_id first!
```

### Order Matters in Composite Indexes!

```sql
-- Index: (client_id, timestamp)
-- GOOD: WHERE client_id = 'x' AND timestamp > 'y'  ✅
-- BAD:  WHERE timestamp > 'y' AND client_id = 'x'  ⚠️ (less efficient)
-- BAD:  WHERE timestamp > 'y'                      ❌ (can't use index)

-- Index: (timestamp, client_id)
-- GOOD: WHERE timestamp > 'y' AND client_id = 'x'  ✅
-- GOOD: WHERE timestamp > 'y'                      ✅
-- BAD:  WHERE client_id = 'x'                      ❌ (can't use index)
```

**Rule:** Most selective column first (the one that filters most rows).

---

## Checking Index Usage

### See All Indexes
```sql
\di request_logs*
```

### See Index Usage Stats
```sql
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan as times_used,
    idx_tup_read as rows_read
FROM pg_stat_user_indexes
WHERE tablename = 'request_logs'
ORDER BY idx_scan DESC;
```

### Find Unused Indexes
```sql
-- Indexes that are never used (can be dropped)
SELECT * FROM pg_stat_user_indexes 
WHERE idx_scan = 0 
  AND indexname NOT LIKE '%_pkey';
```

---

## Summary

### What You Have:

1. **`idx_request_logs_client_id`**
   - For: Queries filtering by client only
   - Example: Get all logs for a client

2. **`idx_request_logs_timestamp`**
   - For: Queries filtering by date only
   - Example: Get all logs from last week

3. **`idx_request_logs_client_timestamp`**
   - For: Queries filtering by client AND date (most common!)
   - Example: Dashboard showing client activity over time

4. **`idx_request_logs_analytics`**
   - For: Analytics queries needing client, date, and allowed status
   - Example: Count blocked requests per client

### Where Did These Names Come From?

**You created them!** When you wrote:
```sql
CREATE INDEX idx_request_logs_client_id ON request_logs(client_id);
```

- `idx_request_logs_client_id` = The index name YOU chose
- Following standard naming convention: `idx_<table>_<columns>`

### Key Takeaway

**Indexes = Speed up queries by organizing data like a phone book** 📞

Without them, your dashboard queries would take seconds instead of milliseconds!

---

**Questions?**

- Confused about a specific index?
- Want to know if you need more/fewer indexes?
- Wondering why a query is slow?

Let me know! 🎯
