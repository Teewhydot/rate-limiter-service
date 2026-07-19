# API Key Authentication Implementation Guide

## Overview

This guide provides pointers for implementing API key authentication so that `client_id` is automatically extracted from the API key instead of being manually provided in requests.

---

## Implementation Checklist

### 1. Database Schema (`migrations/002_api_keys.sql`)

Create a new migration file to store API keys:

**Table: `api_keys`**
- [x ] `id` - BIGSERIAL PRIMARY KEY
- [ x] `key_hash` - VARCHAR(255) UNIQUE NOT NULL (hashed version of key)
- [x ] `client_id` - VARCHAR(255) NOT NULL (foreign key to clients table)
- [x ] `name` - VARCHAR(255) NOT NULL (descriptive name like "Production Key")
- [x ] `created_at` - TIMESTAMP NOT NULL DEFAULT NOW()
- [ x] `last_used_at` - TIMESTAMP (track usage)
- [x ] `is_active` - BOOLEAN NOT NULL DEFAULT true (for revocation)
- [ x] `expires_at` - TIMESTAMP (optional: for key expiration)

**Indexes:**
- [ x] Primary key on `id`
- [ x] Unique index on `key_hash`
- [x ] Index on `client_id` for fast lookups
- [ x] Foreign key constraint: `client_id` REFERENCES `clients(id)` ON DELETE CASCADE

---

### 2. API Key Generation (`internal/api/handlers.go`)

**Function: `GenerateAPIKey(clientID string) (string, error)`**

Steps:
- [ ] Generate 32 random bytes using `crypto/rand.Read()`
- [ ] Encode to base64 or hex string
- [ ] Add prefix: `sk_live_` for production or `sk_test_` for testing
- [ ] Return format: `sk_live_a1b2c3d4e5f6...` (total ~50-60 chars)

Example format:
```
Prefix: sk_live_ or sk_test_
Random part: 32 bytes encoded as hex
Full example: sk_live_XXXXXXXX_RANDOM_CHARACTERS_HERE_XXXXXXXX
```

**Go packages needed:**
```go
import (
    "crypto/rand"
    "encoding/base64" // or "encoding/hex"
    "fmt"
)
```

---

### 3. API Key Hashing (`internal/auth/` or in handlers)

**Function: `HashAPIKey(apiKey string) string`**

Why hash?
- Store hash in database, NOT the raw key
- If database is compromised, actual keys aren't exposed
- Similar to password hashing

Steps:
- [ ] Use `crypto/sha256` to hash the API key
- [ ] Convert hash to hex string using `hex.EncodeToString()`
- [ ] Return the hash (this goes in the database)

**Go packages needed:**
```go
import (
    "crypto/sha256"
    "encoding/hex"
)
```

**Security note:**
- Raw key is shown to user ONCE when generated
- Only the hash is stored
- To verify later: hash incoming key and compare with stored hash

---

### 4. PostgreSQL Methods (`internal/storage/postgres.go`)

**Method 1: `StoreAPIKey(keyHash, clientID, name string) error`**
- [ ] INSERT into `api_keys` table
- [ ] Parameters: key_hash, client_id, name, created_at, is_active
- [ ] Use parameterized query to prevent SQL injection
- [ ] Return error if duplicate key_hash

**Method 2: `GetClientIDByAPIKey(keyHash string) (string, error)`**
- [ ] SELECT client_id from `api_keys` WHERE key_hash = $1 AND is_active = true
- [ ] Return client_id if found
- [ ] Return error if not found or inactive
- [ ] Optional: UPDATE last_used_at (can be done async for performance)

**Method 3: `RevokeAPIKey(keyHash string) error`**
- [ ] UPDATE api_keys SET is_active = false WHERE key_hash = $1
- [ ] Used to disable compromised keys without deleting

**Method 4: `ListAPIKeysForClient(clientID string) ([]APIKey, error)`**
- [ ] SELECT * from api_keys WHERE client_id = $1
- [ ] Return list of keys (without revealing actual keys)
- [ ] Show: name, created_at, last_used_at, is_active

---

### 5. Authentication Middleware (`internal/api/middleware.go`)

**Function: `APIKeyAuthMiddleware(postgres *storage.PostgresClient) gin.HandlerFunc`**

Steps in the middleware:
- [ ] Extract API key from request header
  - Check `X-API-Key` header, OR
  - Check `Authorization: Bearer <key>` header
- [ ] Validate key is not empty
- [ ] Hash the incoming API key using `HashAPIKey()`
- [ ] Call `postgres.GetClientIDByAPIKey(keyHash)`
- [ ] If valid:
  - Store client_id in context: `c.Set("client_id", clientID)`
  - Call `c.Next()` to continue to handler
- [ ] If invalid:
  - Return 401 Unauthorized
  - Call `c.Abort()`
  - Don't continue to handler

**Error responses:**
- Missing key: `{"error": "API key required", "code": "missing_api_key"}`
- Invalid key: `{"error": "Invalid API key", "code": "invalid_api_key"}`
- Revoked key: `{"error": "API key has been revoked", "code": "revoked_api_key"}`

---

### 6. Handler Updates

**New Handler: `CreateAPIKey(c *gin.Context)`**

Location: `internal/api/handlers.go`

Request:
```
POST /admin/clients/:client_id/api-keys
{
  "name": "Production Key"
}
```

Steps:
- [ ] Get client_id from URL params: `c.Param("client_id")`
- [ ] Verify client exists in database
- [ ] Parse request body to get `name`
- [ ] Generate raw API key using `GenerateAPIKey()`
- [ ] Hash the key using `HashAPIKey()`
- [ ] Store hash in database: `postgres.StoreAPIKey(hash, clientID, name)`
- [ ] Return raw key to user (ONLY TIME they see it!)

Response:
```json
{
  "api_key": "sk_live_EXAMPLE_KEY_NOT_REAL_123456",
  "name": "Production Key",
  "created_at": "2026-07-19T10:00:00Z",
  "message": "Save this key securely. You won't see it again!"
}
```

**Updated Handler: `CheckRateLimit(c *gin.Context)`**

Changes:
- [ ] Remove `client_id` from request body
- [ ] Get from context instead: `clientID, exists := c.Get("client_id")`
- [ ] If doesn't exist, return 401 (shouldn't happen if middleware works)
- [ ] Get resource from request URL: `c.Request.URL.Path`
- [ ] Create RateLimitRequest with extracted client_id

Old request body:
```json
{
  "client_id": "client-a",  ← Remove this
  "resource": "/api/endpoint"
}
```

New request (no body needed):
```
GET /api/endpoint
X-API-Key: sk_live_EXAMPLE_KEY_NOT_REAL
```

---

### 7. Routes Updates (`internal/api/routes.go`)

**Create Protected Route Group:**

```go
// Public endpoints (no auth required)
public := r.Group("/")
{
    public.GET("/health", handler.HealthCheck)
}

// Protected endpoints (API key required)
protected := r.Group("/api/v1")
protected.Use(APIKeyAuthMiddleware(postgres))  // ← Add middleware
{
    // Your actual API endpoints
    protected.GET("/resource", handler.SomeEndpoint)
    protected.POST("/data", handler.AnotherEndpoint)
    
    // Rate limit check is automatic via middleware
}

// Admin endpoints (different auth - JWT, Basic Auth, etc.)
admin := r.Group("/admin")
admin.Use(AdminAuthMiddleware())  // Different auth for admin
{
    admin.POST("/clients", handler.CreateClient)
    admin.POST("/clients/:client_id/api-keys", handler.CreateAPIKey)
    admin.GET("/clients/:client_id/api-keys", handler.ListAPIKeys)
    admin.DELETE("/api-keys/:key_id", handler.RevokeAPIKey)
}
```

**Key points:**
- Public routes: no auth (health check)
- Protected routes: API key auth (your APIs)
- Admin routes: different auth (manage clients/keys)

---

### 8. Security Best Practices

**DO:**
- [ ] Always hash keys before storing in database
- [ ] Use HTTPS in production (keys in headers)
- [ ] Show raw key only ONCE at creation time
- [ ] Use constant-time comparison for hashes: `hmac.Equal(hash1, hash2)`
- [ ] Add rate limiting on key generation endpoint
- [ ] Log key usage (last_used_at)
- [ ] Implement key rotation (expires_at)
- [ ] Prefix keys with environment: `sk_live_` or `sk_test_`

**DON'T:**
- [ ] Never log raw API keys
- [ ] Never store raw keys in database
- [ ] Never return raw keys in list endpoints
- [ ] Never use simple hashing (MD5, SHA1) - use SHA256
- [ ] Never allow same key for multiple clients
- [ ] Don't expose detailed error messages in production

---

### 9. Data Models

**Create: `internal/models/api_key.go`**

```go
type APIKey struct {
    ID         int64     `json:"id"`
    KeyHash    string    `json:"-"` // Never expose in JSON
    ClientID   string    `json:"client_id"`
    Name       string    `json:"name"`
    CreatedAt  time.Time `json:"created_at"`
    LastUsedAt *time.Time `json:"last_used_at,omitempty"`
    IsActive   bool      `json:"is_active"`
    ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type CreateAPIKeyRequest struct {
    Name string `json:"name" binding:"required"`
}

type CreateAPIKeyResponse struct {
    APIKey    string    `json:"api_key"` // Raw key - shown once
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    Message   string    `json:"message"`
}
```

---

### 10. Testing Checklist

**Unit Tests:**
- [ ] Test key generation produces unique keys
- [ ] Test key hashing is consistent
- [ ] Test hash comparison works correctly
- [ ] Test different hashes for different keys

**Integration Tests:**
- [ ] Test valid API key authenticates successfully
- [ ] Test invalid API key returns 401
- [ ] Test missing API key returns 401
- [ ] Test revoked key returns 401
- [ ] Test expired key returns 401 (if implemented)
- [ ] Test multiple keys per client work independently
- [ ] Test rate limiting works with API key auth

**Manual Testing:**
```bash
# 1. Create client
POST /admin/clients
{
  "id": "test-client",
  "name": "Test Client",
  "limit": 100,
  "window_sec": 60
}

# 2. Generate API key
POST /admin/clients/test-client/api-keys
{
  "name": "Test Key"
}

Response:
{
  "api_key": "sk_live_EXAMPLE_KEY_NOT_REAL_123456",
  "message": "Save this key securely!"
}

# 3. Use API key
GET /api/v1/some-endpoint
X-API-Key: sk_live_EXAMPLE_KEY_NOT_REAL_123456

# Should work! Rate limiting happens automatically.
```

---

### 11. Documentation Updates

**Files to update:**

- [ ] `README.md` - Add authentication section
- [ ] `api-examples.md` - Show API key usage examples
- [ ] `QUICKSTART.md` - Add API key generation step
- [ ] Create `docs/06-api-key-management.md` - Detailed guide

**Example API documentation:**

```markdown
## Authentication

All API requests require authentication using an API key.

### Obtaining an API Key

Contact your administrator to generate an API key for your client.

### Using Your API Key

Include your API key in the `X-API-Key` header:

```bash
curl https://api.example.com/v1/endpoint \
  -H "X-API-Key: sk_live_your_api_key_here"
```

### Key Security

- Keep your API key secret
- Don't commit keys to version control
- Rotate keys regularly
- Revoke compromised keys immediately
```

---

## Implementation Order

Follow this order for smooth implementation:

1. **Database** - Create migration file and run it
2. **Models** - Define APIKey struct
3. **Hashing** - Implement hash function
4. **Generation** - Implement key generation
5. **PostgreSQL** - Add CRUD methods for api_keys
6. **Middleware** - Create auth middleware
7. **Handlers** - Add CreateAPIKey handler
8. **Routes** - Apply middleware to protected routes
9. **Testing** - Test the full flow
10. **Documentation** - Update all docs

---

## Common Pitfalls to Avoid

❌ **Storing raw keys in database**
✅ Store hashed keys only

❌ **Logging API keys**
✅ Log key ID or masked version: `sk_live_***...***xyz`

❌ **Returning keys in list endpoints**
✅ Only return key metadata (name, created_at, last_used)

❌ **Using weak random number generation**
✅ Use `crypto/rand`, not `math/rand`

❌ **Not handling middleware errors**
✅ Return proper 401 responses with clear messages

❌ **Forgetting to add indexes**
✅ Index on key_hash and client_id for fast lookups

---

## Production Considerations

**Monitoring:**
- Track API key usage (last_used_at)
- Alert on unused keys (potential compromise)
- Monitor failed authentication attempts
- Track keys nearing expiration

**Key Rotation:**
- Implement expiration dates
- Send alerts before expiration
- Allow multiple active keys per client (for rotation)
- Provide grace period for old keys

**Compliance:**
- Log all key generation/revocation events
- Implement key lifecycle management
- Consider GDPR implications (key access = client data access)

---

## Success Criteria

You'll know implementation is complete when:

✅ Clients can generate API keys via admin endpoint
✅ API keys authenticate requests automatically
✅ client_id is extracted from API key, not request body
✅ Invalid keys return 401 errors
✅ Rate limiting works seamlessly with API keys
✅ Keys can be revoked and become immediately invalid
✅ All tests pass
✅ Documentation is updated

---

**Good luck with implementation!** 🚀

Take your time, test thoroughly, and feel free to ask if you get stuck.
