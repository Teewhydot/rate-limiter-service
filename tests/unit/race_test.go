package unit

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tundesmac/rate-limiter-service/internal/config"
	"github.com/tundesmac/rate-limiter-service/internal/storage"
)

// TestConcurrentRequests tests race conditions with concurrent requests
func TestConcurrentRequests(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       1,
	}

	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()

	clientID := "test-client-concurrent"
	limit := 100
	windowSec := 5
	concurrentRequests := 200

	redis.ResetClient(clientID)

	var wg sync.WaitGroup
	allowedCount := int32(0)
	blockedCount := int32(0)
	var mu sync.Mutex

	// Launch concurrent requests
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			allowed, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
			assert.NoError(t, err)

			mu.Lock()
			if allowed {
				allowedCount++
			} else {
				blockedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Verify that exactly 'limit' requests were allowed
	assert.Equal(t, int32(limit), allowedCount, "Exactly %d requests should be allowed", limit)
	assert.Equal(t, int32(concurrentRequests-limit), blockedCount, "Remaining requests should be blocked")

	redis.ResetClient(clientID)
}

// TestConcurrentMultipleClients tests concurrent requests from multiple clients
func TestConcurrentMultipleClients(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       1,
	}

	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()

	numClients := 10
	limit := 50
	windowSec := 5
	requestsPerClient := 75

	// Track results per client
	type clientResult struct {
		allowed int32
		blocked int32
	}
	results := make(map[string]*clientResult)
	var resultsMu sync.Mutex

	var wg sync.WaitGroup

	// Launch concurrent requests for multiple clients
	for clientIdx := 0; clientIdx < numClients; clientIdx++ {
		clientID := "test-client-multi-" + string(rune('A'+clientIdx))
		redis.ResetClient(clientID)

		results[clientID] = &clientResult{}

		for reqIdx := 0; reqIdx < requestsPerClient; reqIdx++ {
			wg.Add(1)
			go func(cID string) {
				defer wg.Done()

				allowed, _, _, err := redis.CheckRateLimit(cID, limit, windowSec)
				assert.NoError(t, err)

				resultsMu.Lock()
				if allowed {
					results[cID].allowed++
				} else {
					results[cID].blocked++
				}
				resultsMu.Unlock()
			}(clientID)
		}
	}

	wg.Wait()

	// Verify each client independently has exactly 'limit' allowed requests
	for clientID, result := range results {
		assert.Equal(t, int32(limit), result.allowed, "Client %s should have exactly %d allowed requests", clientID, limit)
		assert.Equal(t, int32(requestsPerClient-limit), result.blocked, "Client %s should have correct blocked count", clientID)
		redis.ResetClient(clientID)
	}
}

// TestRaceConditionWithReset tests race conditions when resetting client data
func TestRaceConditionWithReset(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       1,
	}

	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()

	clientID := "test-client-reset-race"
	limit := 10
	windowSec := 2

	redis.ResetClient(clientID)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Goroutine making requests
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
			if err != nil {
				errors <- err
			}
		}()
	}

	// Goroutine resetting client concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			time.Sleep(10 * time.Millisecond)
			redis.ResetClient(clientID)
		}
	}()

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Logf("Error occurred: %v", err)
		errorCount++
	}

	// Some operations may fail due to reset, but should not panic
	assert.Less(t, errorCount, 50, "Not all operations should fail")

	redis.ResetClient(clientID)
}

// TestHighContentionScenario tests system behavior under extreme contention
func TestHighContentionScenario(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       1,
	}

	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()

	clientID := "test-client-high-contention"
	limit := 500
	windowSec := 10
	concurrentRequests := 1000

	redis.ResetClient(clientID)

	var wg sync.WaitGroup
	start := time.Now()
	allowedCount := int32(0)
	blockedCount := int32(0)
	var mu sync.Mutex

	// Launch high-contention concurrent requests
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			allowed, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
			if err != nil {
				t.Logf("Error during high contention: %v", err)
				return
			}

			mu.Lock()
			if allowed {
				allowedCount++
			} else {
				blockedCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("High contention test completed in %v", elapsed)
	t.Logf("Allowed: %d, Blocked: %d", allowedCount, blockedCount)

	// Verify accuracy under high contention
	assert.Equal(t, int32(limit), allowedCount, "Should maintain accuracy under high contention")
	assert.Equal(t, int32(concurrentRequests-limit), blockedCount)

	// Performance check: should complete reasonably fast
	assert.Less(t, elapsed, 30*time.Second, "Should handle high contention efficiently")

	redis.ResetClient(clientID)
}

// TestDataRaceWithGetCurrentCount tests concurrent reads and writes
func TestDataRaceWithGetCurrentCount(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       1,
	}

	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()

	clientID := "test-client-data-race"
	limit := 100
	windowSec := 5

	redis.ResetClient(clientID)

	var wg sync.WaitGroup
	stopReading := make(chan struct{})

	// Goroutines writing (making requests)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				redis.CheckRateLimit(clientID, limit, windowSec)
				time.Sleep(1 * time.Millisecond)
			}
		}()
	}

	// Goroutines reading (getting count)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopReading:
					return
				default:
					_, err := redis.GetCurrentCount(clientID, windowSec)
					if err != nil {
						t.Logf("Error reading count: %v", err)
					}
					time.Sleep(2 * time.Millisecond)
				}
			}
		}()
	}

	// Wait for writers to complete
	time.Sleep(2 * time.Second)
	close(stopReading)

	wg.Wait()

	// If we reach here without panic, the test passes
	t.Log("Concurrent read/write test completed successfully")

	redis.ResetClient(clientID)
}

// TestAtomicityOfOperations tests that rate limit operations are atomic
func TestAtomicityOfOperations(t *testing.T) {
	cfg := &config.Config{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       1,
	}

	redis, err := storage.NewRedisClient(cfg)
	if err != nil {
		t.Skipf("Redis not available: %v", err)
		return
	}
	defer redis.Close()

	clientID := "test-client-atomicity"
	limit := 1 // Very small limit to test atomicity
	windowSec := 2
	concurrentRequests := 100

	redis.ResetClient(clientID)

	var wg sync.WaitGroup
	allowedCount := int32(0)
	var mu sync.Mutex

	// Launch many concurrent requests with limit of 1
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			allowed, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
			assert.NoError(t, err)

			if allowed {
				mu.Lock()
				allowedCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// With atomic operations, exactly 1 request should be allowed
	assert.Equal(t, int32(1), allowedCount, "Atomic operations should allow exactly 1 request")

	redis.ResetClient(clientID)
}
