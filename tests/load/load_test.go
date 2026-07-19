package load

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tundesmac/rate-limiter-service/internal/config"
	"github.com/tundesmac/rate-limiter-service/internal/storage"
)

// TestLoadPerformance measures performance under load
func TestLoadPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

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

	clientID := "load-test-client"
	limit := 10000
	windowSec := 60
	totalRequests := 50000
	concurrency := 100

	redis.ResetClient(clientID)

	var wg sync.WaitGroup
	requestsPerGoroutine := totalRequests / concurrency

	start := time.Now()
	var successCount, failureCount int64
	var totalLatency int64

	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				reqStart := time.Now()
				_, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
				reqLatency := time.Since(reqStart).Microseconds()

				atomic.AddInt64(&totalLatency, reqLatency)

				if err != nil {
					atomic.AddInt64(&failureCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	// Calculate metrics
	throughput := float64(totalRequests) / elapsed.Seconds()
	avgLatency := float64(totalLatency) / float64(successCount) / 1000.0 // Convert to milliseconds

	t.Logf("Load Test Results:")
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Failed: %d", failureCount)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f req/sec", throughput)
	t.Logf("  Avg Latency: %.2f ms", avgLatency)

	// Assertions
	assert.Greater(t, successCount, int64(0), "Should have successful requests")
	assert.Less(t, avgLatency, 10.0, "Average latency should be less than 10ms")
	assert.Greater(t, throughput, 1000.0, "Should handle >1000 req/sec")

	redis.ResetClient(clientID)
}

// TestSustainedLoad tests system behavior under sustained load
func TestSustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping sustained load test in short mode")
	}

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

	clientID := "sustained-load-client"
	limit := 1000
	windowSec := 60
	duration := 30 * time.Second
	concurrency := 50

	redis.ResetClient(clientID)

	var wg sync.WaitGroup
	stopTime := time.Now().Add(duration)
	var requestCount, allowedCount, blockedCount int64
	latencies := make([]int64, 0, 10000)
	var latencyMu sync.Mutex

	t.Logf("Starting sustained load test for %v...", duration)

	// Launch workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for time.Now().Before(stopTime) {
				reqStart := time.Now()
				allowed, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
				reqLatency := time.Since(reqStart).Microseconds()

				atomic.AddInt64(&requestCount, 1)

				if err == nil {
					if allowed {
						atomic.AddInt64(&allowedCount, 1)
					} else {
						atomic.AddInt64(&blockedCount, 1)
					}

					// Sample latencies (every 10th request)
					if requestCount%10 == 0 {
						latencyMu.Lock()
						latencies = append(latencies, reqLatency)
						latencyMu.Unlock()
					}
				}

				// Small delay to simulate realistic load
				time.Sleep(100 * time.Microsecond)
			}
		}()
	}

	wg.Wait()

	// Calculate statistics
	var totalLatency int64
	for _, lat := range latencies {
		totalLatency += lat
	}
	avgLatency := float64(totalLatency) / float64(len(latencies)) / 1000.0

	t.Logf("Sustained Load Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", requestCount)
	t.Logf("  Allowed: %d", allowedCount)
	t.Logf("  Blocked: %d", blockedCount)
	t.Logf("  Throughput: %.2f req/sec", float64(requestCount)/duration.Seconds())
	t.Logf("  Avg Latency: %.2f ms", avgLatency)

	// System should remain stable under sustained load
	assert.Greater(t, requestCount, int64(10000), "Should process significant number of requests")
	assert.Less(t, avgLatency, 20.0, "Latency should remain reasonable under sustained load")

	redis.ResetClient(clientID)
}

// TestLatencyPercentiles measures latency distribution
func TestLatencyPercentiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping latency test in short mode")
	}

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

	clientID := "latency-test-client"
	limit := 5000
	windowSec := 60
	numRequests := 10000

	redis.ResetClient(clientID)

	latencies := make([]int64, numRequests)
	var wg sync.WaitGroup

	// Collect latency samples
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		idx := i
		go func() {
			defer wg.Done()

			start := time.Now()
			redis.CheckRateLimit(clientID, limit, windowSec)
			latencies[idx] = time.Since(start).Microseconds()
		}()
	}

	wg.Wait()

	// Sort latencies for percentile calculation
	sortedLatencies := make([]int64, len(latencies))
	copy(sortedLatencies, latencies)
	// Simple bubble sort for test purposes
	for i := 0; i < len(sortedLatencies); i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	p50 := sortedLatencies[len(sortedLatencies)*50/100] / 1000.0
	p95 := sortedLatencies[len(sortedLatencies)*95/100] / 1000.0
	p99 := sortedLatencies[len(sortedLatencies)*99/100] / 1000.0

	t.Logf("Latency Percentiles (ms):")
	t.Logf("  P50: %.2f", float64(p50))
	t.Logf("  P95: %.2f", float64(p95))
	t.Logf("  P99: %.2f", float64(p99))

	// Assert latency requirements (few milliseconds)
	assert.Less(t, float64(p50), 5.0, "P50 latency should be <5ms")
	assert.Less(t, float64(p95), 10.0, "P95 latency should be <10ms")
	assert.Less(t, float64(p99), 20.0, "P99 latency should be <20ms")

	redis.ResetClient(clientID)
}

// TestMultiInstanceSimulation simulates multiple service instances
func TestMultiInstanceSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping multi-instance test in short mode")
	}

	cfg := &config.Config{
		RedisAddr:     "localhost:6379",
		RedisPassword: "",
		RedisDB:       1,
	}

	// Simulate 3 service instances
	instances := make([]*storage.RedisClient, 3)
	for i := 0; i < 3; i++ {
		redis, err := storage.NewRedisClient(cfg)
		if err != nil {
			t.Skipf("Redis not available: %v", err)
			return
		}
		instances[i] = redis
		defer redis.Close()
	}

	clientID := "multi-instance-client"
	limit := 300
	windowSec := 10
	requestsPerInstance := 150

	instances[0].ResetClient(clientID)

	var wg sync.WaitGroup
	var totalAllowed, totalBlocked int64

	// Each instance processes requests concurrently
	for instanceIdx, instance := range instances {
		wg.Add(1)
		go func(idx int, redis *storage.RedisClient) {
			defer wg.Done()

			var allowed, blocked int64
			for j := 0; j < requestsPerInstance; j++ {
				isAllowed, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
				if err != nil {
					t.Logf("Instance %d error: %v", idx, err)
					continue
				}

				if isAllowed {
					allowed++
				} else {
					blocked++
				}
			}

			atomic.AddInt64(&totalAllowed, allowed)
			atomic.AddInt64(&totalBlocked, blocked)

			t.Logf("Instance %d: Allowed=%d, Blocked=%d", idx, allowed, blocked)
		}(instanceIdx, instance)
	}

	wg.Wait()

	totalRequests := int64(requestsPerInstance * 3)
	t.Logf("Multi-Instance Test Results:")
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Total Allowed: %d", totalAllowed)
	t.Logf("  Total Blocked: %d", totalBlocked)
	t.Logf("  Expected Limit: %d", limit)

	// Verify distributed rate limiting accuracy
	// Should allow exactly 'limit' requests across all instances
	assert.Equal(t, int64(limit), totalAllowed, "Distributed rate limiting should be accurate across instances")
	assert.Equal(t, totalRequests-int64(limit), totalBlocked)

	instances[0].ResetClient(clientID)
}

// TestStressTest applies extreme stress to the system
func TestStressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

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

	clientID := "stress-test-client"
	limit := 50000
	windowSec := 120
	totalRequests := 100000
	concurrency := 500

	redis.ResetClient(clientID)

	t.Logf("Starting stress test: %d requests with %d concurrent workers...", totalRequests, concurrency)

	var wg sync.WaitGroup
	requestsPerWorker := totalRequests / concurrency
	start := time.Now()
	var successCount, errorCount int64

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				_, _, _, err := redis.CheckRateLimit(clientID, limit, windowSec)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	throughput := float64(totalRequests) / elapsed.Seconds()

	t.Logf("Stress Test Results:")
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Successful: %d", successCount)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Duration: %v", elapsed)
	t.Logf("  Throughput: %.2f req/sec", throughput)

	// System should handle extreme load without too many errors
	errorRate := float64(errorCount) / float64(totalRequests) * 100
	assert.Less(t, errorRate, 1.0, "Error rate should be <1%% under stress")
	assert.Greater(t, throughput, 5000.0, "Should maintain >5000 req/sec under stress")

	redis.ResetClient(clientID)
}
