#!/bin/bash
echo "=== Comprehensive System Test ==="
echo ""

# Test 1: Create a new client with 10 req/min limit
echo "1. Creating custom client (10 req/min)..."
curl -s -X POST http://localhost:8080/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{
    "id": "custom-client",
    "name": "Custom Test Client", 
    "limit": 10,
    "window_sec": 60
  }' | jq .
echo ""

# Test 2: Verify exactly 10 requests are allowed
echo "2. Testing exact limit (10 requests)..."
ALLOWED=0
BLOCKED=0
for i in {1..15}; do
    RESP=$(curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
      -H "Content-Type: application/json" \
      -d '{"client_id":"custom-client"}')
    if echo "$RESP" | grep -q '"allowed":true'; then
        ((ALLOWED++))
    else
        ((BLOCKED++))
    fi
done
echo "Sent 15 requests: Allowed=$ALLOWED, Blocked=$BLOCKED"
if [ "$ALLOWED" -eq 10 ] && [ "$BLOCKED" -eq 5 ]; then
    echo "✓ PASS: Rate limiting working perfectly!"
else
    echo "✗ FAIL: Expected 10 allowed / 5 blocked"
fi
echo ""

# Test 3: Check dashboard stats
echo "3. Dashboard stats for custom-client..."
curl -s "http://localhost:8080/api/v1/dashboard/usage/custom-client?days=1" | jq '{client_id, total_requests, allowed_requests, blocked_requests}'
echo ""

# Test 4: Real-time stats
echo "4. Real-time stats..."
curl -s "http://localhost:8080/api/v1/stats/custom-client" | jq .
echo ""

echo "=== Test Complete ==="
