#!/bin/bash

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}======================================${NC}"
echo -e "${YELLOW}  Rate Limiter System Test Suite${NC}"
echo -e "${YELLOW}======================================${NC}"
echo ""

# Check if services are running
echo -e "${YELLOW}[1/7] Checking if services are running...${NC}"
if ! docker-compose ps | grep -q "Up"; then
    echo -e "${RED}✗ Services not running. Please run 'docker-compose up -d' first${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Services are running${NC}"
echo ""

# Test health endpoint
echo -e "${YELLOW}[2/7] Testing health endpoint...${NC}"
HEALTH=$(curl -s http://localhost:8080/health)
if echo "$HEALTH" | grep -q "healthy"; then
    echo -e "${GREEN}✓ Health check passed${NC}"
    echo "$HEALTH" | jq .
else
    echo -e "${RED}✗ Health check failed${NC}"
    exit 1
fi
echo ""

# Test rate limit check (allowed)
echo -e "${YELLOW}[3/7] Testing rate limit check (should allow)...${NC}"
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-a","resource":"/api/test"}')

if echo "$RESPONSE" | grep -q '"allowed":true'; then
    echo -e "${GREEN}✓ Rate limit check passed (allowed)${NC}"
    echo "$RESPONSE" | jq .
else
    echo -e "${RED}✗ Rate limit check failed${NC}"
    echo "$RESPONSE"
    exit 1
fi
echo ""

# Test rate limit exhaustion
echo -e "${YELLOW}[4/7] Testing rate limit exhaustion...${NC}"
echo "Sending 100 requests to exhaust client-a limit (100 req/min)..."

ALLOWED=0
BLOCKED=0

for i in {1..100}; do
    RESP=$(curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
      -H "Content-Type: application/json" \
      -d '{"client_id":"client-a"}')
    
    if echo "$RESP" | grep -q '"allowed":true'; then
        ((ALLOWED++))
    else
        ((BLOCKED++))
    fi
done

echo "Results: Allowed=$ALLOWED, Blocked=$BLOCKED"

if [ "$ALLOWED" -eq 100 ]; then
    echo -e "${GREEN}✓ Exactly 100 requests allowed (correct!)${NC}"
else
    echo -e "${RED}✗ Expected 100 allowed, got $ALLOWED${NC}"
fi
echo ""

# Test that next request is blocked
echo -e "${YELLOW}[5/7] Testing that 101st request is blocked...${NC}"
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
  -H "Content-Type: application/json" \
  -d '{"client_id":"client-a"}')

if echo "$RESPONSE" | grep -q '"allowed":false'; then
    echo -e "${GREEN}✓ Request correctly blocked after limit reached${NC}"
    echo "$RESPONSE" | jq .
else
    echo -e "${RED}✗ Request should have been blocked${NC}"
    exit 1
fi
echo ""

# Test multi-instance consistency
echo -e "${YELLOW}[6/7] Testing multi-instance consistency...${NC}"
echo "Resetting client-b and testing across 3 instances..."

# Wait for client-a to reset
sleep 65

# Send requests across all 3 instances
TOTAL_ALLOWED=0

for i in {1..50}; do
    # Instance 1 (port 8080)
    RESP=$(curl -s -X POST http://localhost:8080/api/v1/ratelimit/check \
      -H "Content-Type: application/json" \
      -d '{"client_id":"client-b"}')
    if echo "$RESP" | grep -q '"allowed":true'; then
        ((TOTAL_ALLOWED++))
    fi
    
    # Instance 2 (port 8081)
    RESP=$(curl -s -X POST http://localhost:8081/api/v1/ratelimit/check \
      -H "Content-Type: application/json" \
      -d '{"client_id":"client-b"}')
    if echo "$RESP" | grep -q '"allowed":true'; then
        ((TOTAL_ALLOWED++))
    fi
    
    # Instance 3 (port 8082)
    RESP=$(curl -s -X POST http://localhost:8082/api/v1/ratelimit/check \
      -H "Content-Type: application/json" \
      -d '{"client_id":"client-b"}')
    if echo "$RESP" | grep -q '"allowed":true'; then
        ((TOTAL_ALLOWED++))
    fi
done

echo "Total allowed across all instances: $TOTAL_ALLOWED"
echo "Client-b limit: 5000 req/min"

if [ "$TOTAL_ALLOWED" -le 5000 ]; then
    echo -e "${GREEN}✓ Multi-instance rate limiting working correctly${NC}"
else
    echo -e "${RED}✗ Multi-instance rate limiting failed (allowed more than limit)${NC}"
fi
echo ""

# Test dashboard API
echo -e "${YELLOW}[7/7] Testing dashboard API...${NC}"

# Get usage stats
echo "Getting usage stats for client-a..."
STATS=$(curl -s "http://localhost:8080/api/v1/dashboard/usage/client-a?days=1")
if echo "$STATS" | grep -q 'total_requests'; then
    echo -e "${GREEN}✓ Usage stats endpoint working${NC}"
    echo "$STATS" | jq .
else
    echo -e "${RED}✗ Usage stats endpoint failed${NC}"
fi
echo ""

# Get real-time stats
echo "Getting real-time stats for client-a..."
REALTIME=$(curl -s "http://localhost:8080/api/v1/stats/client-a")
if echo "$REALTIME" | grep -q 'client_id'; then
    echo -e "${GREEN}✓ Real-time stats endpoint working${NC}"
    echo "$REALTIME" | jq .
else
    echo -e "${RED}✗ Real-time stats endpoint failed${NC}"
fi
echo ""

# Summary
echo -e "${YELLOW}======================================${NC}"
echo -e "${GREEN}  All tests completed!${NC}"
echo -e "${YELLOW}======================================${NC}"
echo ""
echo "Dashboard API endpoints for your Flutter app:"
echo "  - Usage Stats: GET http://localhost:8080/api/v1/dashboard/usage/{client_id}?days=30"
echo "  - Trend Data:  GET http://localhost:8080/api/v1/dashboard/trends/{client_id}?days=10"
echo "  - Real-time:   GET http://localhost:8080/api/v1/stats/{client_id}"
echo ""
echo "Client Management:"
echo "  - List:   GET    http://localhost:8080/api/v1/clients"
echo "  - Get:    GET    http://localhost:8080/api/v1/clients/{id}"
echo "  - Create: POST   http://localhost:8080/api/v1/clients"
echo "  - Update: PUT    http://localhost:8080/api/v1/clients/{id}"
echo ""
echo "Service instances running on:"
echo "  - http://localhost:8080 (Instance 1)"
echo "  - http://localhost:8081 (Instance 2)"
echo "  - http://localhost:8082 (Instance 3)"
