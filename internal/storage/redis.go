package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/tundesmac/rate-limiter-service/internal/config"
)

// RedisClient wraps the Redis client with rate limiting operations
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	
	ctx := context.Background()
	
	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// CheckRateLimit implements sliding window counter algorithm
// Returns: allowed (bool), remaining (int), resetAt (unix timestamp), error
func (r *RedisClient) CheckRateLimit(clientID string, limit int, windowSec int) (bool, int, int64, error) {
	now := time.Now()
	windowStart := now.Add(-time.Duration(windowSec) * time.Second).UnixNano()
	nowNano := now.UnixNano()
	
	key := fmt.Sprintf("ratelimit:%s", clientID)
	
	// Use Redis pipeline for atomic operations
	pipe := r.client.Pipeline()
	
	// Remove old entries outside the window
	pipe.ZRemRangeByScore(r.ctx, key, "0", fmt.Sprintf("%d", windowStart))
	
	// Count current requests in the window
	countCmd := pipe.ZCount(r.ctx, key, fmt.Sprintf("%d", windowStart), "+inf")
	
	// Add current request with score as timestamp
	pipe.ZAdd(r.ctx, key, &redis.Z{
		Score:  float64(nowNano),
		Member: fmt.Sprintf("%d", nowNano),
	})
	
	// Set expiration on the key (cleanup)
	pipe.Expire(r.ctx, key, time.Duration(windowSec)*time.Second)
	
	// Execute pipeline
	_, err := pipe.Exec(r.ctx)
	if err != nil {
		return false, 0, 0, fmt.Errorf("redis pipeline error: %w", err)
	}
	
	// Get the count of requests before adding the current one
	currentCount := countCmd.Val()
	
	// Check if limit is exceeded
	allowed := currentCount < int64(limit)
	remaining := limit - int(currentCount) - 1
	if remaining < 0 {
		remaining = 0
	}
	
	// Calculate reset time (end of current window)
	resetAt := now.Add(time.Duration(windowSec) * time.Second).Unix()
	
	// If not allowed, remove the request we just added
	if !allowed {
		r.client.ZRem(r.ctx, key, fmt.Sprintf("%d", nowNano))
	}
	
	return allowed, remaining, resetAt, nil
}

// GetCurrentCount returns the current request count for a client
func (r *RedisClient) GetCurrentCount(clientID string, windowSec int) (int64, error) {
	now := time.Now()
	windowStart := now.Add(-time.Duration(windowSec) * time.Second).UnixNano()
	
	key := fmt.Sprintf("ratelimit:%s", clientID)
	
	count, err := r.client.ZCount(r.ctx, key, fmt.Sprintf("%d", windowStart), "+inf").Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %w", err)
	}
	
	return count, nil
}

// HealthCheck checks if Redis is available
func (r *RedisClient) HealthCheck() error {
	return r.client.Ping(r.ctx).Err()
}

// ResetClient removes all rate limit data for a specific client (useful for testing)
func (r *RedisClient) ResetClient(clientID string) error {
	key := fmt.Sprintf("ratelimit:%s", clientID)
	return r.client.Del(r.ctx, key).Err()
}
