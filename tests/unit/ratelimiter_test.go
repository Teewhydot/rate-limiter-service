package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tundesmac/rate-limiter-service/internal/config"
	"github.com/tundesmac/rate-limiter-service/internal/storage"
)

// TestSlidingWindowRateLimit tests basic rate limiting functionality
func TestSlidingWindowRateLimit(t *testing.T) {
	// Setup
	cfg := &config.Config{
		RedisAddr:    "localhost:6379",
		RedisPassword: "",
		RedisDB:      1, // Use different DB for testing
	}
	
	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()
	
	clientID := "test-client-basic"
	limit := 5
	windowSec := 2
	
	// Clean up before test
	redis.ResetClient(clientID)
	
	// Test: First 5 requests should be allowed
	for i := 0; i < limit; i++ {
		allowed, remaining, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
		assert.Equal(t, limit-i-1, remaining, "Remaining count incorrect")
	}
	
	// Test: 6th request should be blocked
	allowed, remaining, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
	assert.NoError(t, err)
	assert.False(t, allowed, "Request beyond limit should be blocked")
	assert.Equal(t, 0, remaining)
	
	// Test: After window expires, requests should be allowed again
	time.Sleep(time.Duration(windowSec+1) * time.Second)
	allowed, remaining, _, err = redis.CheckRateLimit(clientID, limit, windowSec)
	assert.NoError(t, err)
	assert.True(t, allowed, "Request should be allowed after window expiry")
	assert.Equal(t, limit-1, remaining)
	
	// Cleanup
	redis.ResetClient(clientID)
}

// TestSlidingWindowAccuracy tests the sliding window algorithm accuracy
func TestSlidingWindowAccuracy(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:    "localhost:6379",
		RedisPassword: "",
		RedisDB:      1,
	}
	
	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()
	
	clientID := "test-client-sliding"
	limit := 10
	windowSec := 3
	
	redis.ResetClient(clientID)
	
	// Send 5 requests
	for i := 0; i < 5; i++ {
		allowed, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}
	
	// Wait 2 seconds (still within window)
	time.Sleep(2 * time.Second)
	
	// Send 5 more requests (total 10 in window)
	for i := 0; i < 5; i++ {
		allowed, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+6)
	}
	
	// Next request should be blocked
	allowed, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
	assert.NoError(t, err)
	assert.False(t, allowed, "Request should be blocked")
	
	// Wait for first batch to expire (1 more second)
	time.Sleep(1500 * time.Millisecond)
	
	// Now first 5 requests should have expired, allowing 5 new ones
	for i := 0; i < 5; i++ {
		allowed, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
		assert.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed after partial window expiry", i+1)
	}
	
	redis.ResetClient(clientID)
}

// TestMultipleClients tests that different clients are isolated
func TestMultipleClients(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:    "localhost:6379",
		RedisPassword: "",
		RedisDB:      1,
	}
	
	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()
	
	client1 := "test-client-1"
	client2 := "test-client-2"
	limit := 3
	windowSec := 2
	
	redis.ResetClient(client1)
	redis.ResetClient(client2)
	
	// Exhaust client1's limit
	for i := 0; i < limit; i++ {
		allowed, _, _, err := redis.CheckRateLimit(client1, limit, windowSec)
		assert.NoError(t, err)
		assert.True(t, allowed)
	}
	
	// Client1 should be blocked
	allowed, _, _, err := redis.CheckRateLimit(client1, limit, windowSec)
	assert.NoError(t, err)
	assert.False(t, allowed)
	
	// Client2 should still be allowed (isolated)
	for i := 0; i < limit; i++ {
		allowed, _, _, err := redis.CheckRateLimit(client2, limit, windowSec)
		assert.NoError(t, err)
		assert.True(t, allowed, "Client2 should be independent of Client1")
	}
	
	redis.ResetClient(client1)
	redis.ResetClient(client2)
}

// TestGetCurrentCount tests the count retrieval functionality
func TestGetCurrentCount(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:    "localhost:6379",
		RedisPassword: "",
		RedisDB:      1,
	}
	
	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()
	
	clientID := "test-client-count"
	limit := 10
	windowSec := 2
	
	redis.ResetClient(clientID)
	
	// Send 5 requests
	for i := 0; i < 5; i++ {
		redis.CheckRateLimit(clientID, limit, windowSec)
	}
	
	// Check count
	count, err := redis.GetCurrentCount(clientID, windowSec)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	
	// Send 3 more requests
	for i := 0; i < 3; i++ {
		redis.CheckRateLimit(clientID, limit, windowSec)
	}
	
	count, err = redis.GetCurrentCount(clientID, windowSec)
	assert.NoError(t, err)
	assert.Equal(t, int64(8), count)
	
	redis.ResetClient(clientID)
}

// TestZeroLimit tests handling of edge case with zero limit
func TestZeroLimit(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:    "localhost:6379",
		RedisPassword: "",
		RedisDB:      1,
	}
	
	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()
	
	clientID := "test-client-zero"
	limit := 0
	windowSec := 1
	
	redis.ResetClient(clientID)
	
	// All requests should be blocked with zero limit
	allowed, remaining, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
	assert.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, 0, remaining)
	
	redis.ResetClient(clientID)
}

// TestResetAt tests that reset timestamp is correctly calculated
func TestResetAt(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:    "localhost:6379",
		RedisPassword: "",
		RedisDB:      1,
	}
	
	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()
	
	clientID := "test-client-reset"
	limit := 5
	windowSec := 10
	
	redis.ResetClient(clientID)
	
	before := time.Now()
	_, _, resetAt, err := redis.CheckRateLimit(clientID, limit, windowSec)
	after := time.Now()
	
	assert.NoError(t, err)
	
	expectedMin := before.Add(time.Duration(windowSec) * time.Second).Unix()
	expectedMax := after.Add(time.Duration(windowSec) * time.Second).Unix()
	
	assert.GreaterOrEqual(t, resetAt, expectedMin)
	assert.LessOrEqual(t, resetAt, expectedMax)
	
	redis.ResetClient(clientID)
}
