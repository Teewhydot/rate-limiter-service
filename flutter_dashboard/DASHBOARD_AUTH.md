# Flutter Dashboard - API Key Authentication

## ✅ What Was Implemented:

### Quick Login System
- Login form on dashboard (no separate screen)
- Enter API key + client ID
- View only your data
- Logout button in AppBar

## 📝 How to Use:

### 1. Start Backend
```bash
cd /Users/tundesmac/Projects/rate-limiter-service
docker-compose up
```

### 2. Create a Client (Get API Key)
```bash
curl -X POST http://localhost:8080/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{
    "id": "test-client",
    "name": "Test Client",
    "limit": 100,
    "window_sec": 60
  }'
```

**Response:**
```json
{
  "id": "test-client",
  "api_key": "sk_live_abc123...",  ← Copy this!
  "message": "Save this API key securely!"
}
```

### 3. Run Flutter Dashboard
```bash
cd flutter_dashboard
flutter run -d chrome --web-port 3000
```

### 4. Login
1. Dashboard shows login form
2. Enter API Key: `sk_live_abc123...`
3. Enter Client ID: `test-client`
4. Click "View Dashboard"
5. See your stats!

## 🔑 Features:

- ✅ API key stored in memory (cleared on logout)
- ✅ API key sent in `X-API-Key` header
- ✅ Protected `/dashboard/*` endpoints require auth
- ✅ Client sees only their own data
- ✅ Logout clears everything

## 🎯 What's Protected:

```
Protected (requires API key):
- GET /api/v1/dashboard/usage/:client_id
- GET /api/v1/dashboard/trends/:client_id

Unprotected:
- POST /api/v1/ratelimit/check
- GET /api/v1/clients
- GET /api/v1/stats/:client_id
```

## 🚀 Quick Test:

```bash
# 1. Create client and get API key
API_KEY=$(curl -s -X POST http://localhost:8080/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{"id":"demo-client","name":"Demo","limit":100,"window_sec":60}' | jq -r '.api_key')

echo "Your API Key: $API_KEY"

# 2. Test protected endpoint
curl http://localhost:8080/api/v1/dashboard/usage/demo-client?days=7 \
  -H "X-API-Key: $API_KEY"

# Should return usage data ✅
```

## 📊 Dashboard Flow:

```
User opens dashboard
      ↓
Shows login form
      ↓
User enters API key + Client ID
      ↓
Stores in memory
      ↓
Loads client's data
      ↓
Shows dashboard with:
  - Client info
  - Usage stats (protected)
  - Trend graphs (protected)
      ↓
User clicks Logout
      ↓
Clears API key
      ↓
Shows login form again
```

## ⏱️ Implementation Time: 10 minutes!

Simple, fast, works! ✅
