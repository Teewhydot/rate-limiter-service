#!/bin/bash
echo "=== Testing Rate Limiter Functionality ==="
echo ""

# Test 1: Create a new client with custom limits
echo "1. Creating test client with 10 req/min limit..."
CREATE_RESP=$(curl -s -X POST http://localhost:8080/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "test-client",
    "name": "Test Client", 
    "rate_limit": 10,
    "window_seconds": 60
  }')
echo "$CREATE_RESP" | jq .
echo ""

# Test 2: Use exactly the limit
echo "2. Sending exactly 10 requests (the limit)..."
ALLOWED=0
BLOCKED=0
for i in {1..10}; do
    RESP=$(curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
      -H "Content-Type: application/json" \
      -d '{"client_id":"test-client"}')
    if echo "$RESP" | grep -q '"allowed":true'; then
        ((ALLOWED++))
    else
        ((BLOCKED++))
    fi
done
echo "Results: Allowed=$ALLOWED, Blocked=$BLOCKED"
if [ "$ALLOWED" -eq 10 ]; then
    echo "✓ PASS: All 10 requests allowed"
else
    echo "✗ FAIL: Expected 10, got $ALLOWED"
fi
echo ""

# Test 3: Exceed the limit
echo "3. Sending 11th request (should be blocked)..."
RESP=$(curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"test-client"}')
echo "$RESP" | jq .
if echo "$RESP" | grep -q '"allowed":false'; then
    echo "✓ PASS: 11th request correctly blocked"
else
    echo "✗ FAIL: 11th request should have been blocked"
fi
echo ""

# Test 4: List all clients
echo "4. Listing all clients..."
curl -s http://localhost:8080/api/v1/clients | jq '.clients | length'
echo ""

# Test 5: Performance check
echo "5. Performance test (measure latency)..."
START=$(date +%s%N)
for i in {1..50}; do
    curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
      -H "Content-Type: application/json" \
      -d '{"client_id":"client-b"}' > /dev/null
done
END=$(date +%s%N)
DURATION=$((($END - $START) / 1000000))
AVG_LATENCY=$(($DURATION / 50))
echo "50 requests completed in ${DURATION}ms"
echo "Average latency: ${AVG_LATENCY}ms per request"
if [ "$AVG_LATENCY" -lt 100 ]; then
    echo "✓ PASS: Performance is good (< 100ms per request)"
else
    echo "⚠ WARNING: Latency is high (> 100ms per request)"
fi
echo ""

echo "=== All Tests Complete ==="
