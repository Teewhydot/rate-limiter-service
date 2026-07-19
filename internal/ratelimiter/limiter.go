package ratelimiter

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"github.com/tundesmac/rate-limiter-service/internal/config"
	"github.com/tundesmac/rate-limiter-service/internal/logger"
	"github.com/tundesmac/rate-limiter-service/internal/models"
	"github.com/tundesmac/rate-limiter-service/internal/storage"
)

// RateLimiter handles rate limiting logic
type RateLimiter struct {
	redis       *storage.RedisClient
	postgres    *storage.PostgresClient
	asyncLogger *logger.AsyncLogger
	config      *config.Config
	logger      *zap.Logger
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(
	redis *storage.RedisClient,
	postgres *storage.PostgresClient,
	asyncLogger *logger.AsyncLogger,
	cfg *config.Config,
	log *zap.Logger,
) *RateLimiter {
	return &RateLimiter{
		redis:       redis,
		postgres:    postgres,
		asyncLogger: asyncLogger,
		config:      cfg,
		logger:      log,
	}
}

// CheckLimit checks if a request should be allowed based on rate limits
func (rl *RateLimiter) CheckLimit(req models.RateLimitRequest) (*models.RateLimitResponse, error) {
	// Get client configuration from database
	client, err := rl.postgres.GetClient(req.ClientID)
	if err != nil {
		rl.logger.Error("Failed to get client from database",
			zap.String("client_id", req.ClientID),
			zap.Error(err),
		)
		// Use default limits if database fails
		client = &models.Client{
			ID:        req.ClientID,
			Limit:     rl.config.DefaultLimit,
			WindowSec: rl.config.DefaultWindowSec,
		}
	}
	
	// If client not found, use defaults
	if client == nil {
		client = &models.Client{
			ID:        req.ClientID,
			Limit:     rl.config.DefaultLimit,
			WindowSec: rl.config.DefaultWindowSec,
		}
	}
	
	// Check rate limit in Redis
	allowed, remaining, resetAt, err := rl.redis.CheckRateLimit(
		req.ClientID,
		client.Limit,
		client.WindowSec,
	)
	
	// Handle Redis failure with fail-safe strategy
	if err != nil {
		rl.logger.Error("Redis rate limit check failed",
			zap.String("client_id", req.ClientID),
			zap.Error(err),
		)
		
		if rl.config.FailSafeMode {
			// Allow request when Redis is down (fail-open)
			rl.logger.Warn("Fail-safe mode: allowing request due to Redis error",
				zap.String("client_id", req.ClientID),
			)
			allowed = true
			remaining = client.Limit
			resetAt = time.Now().Add(time.Duration(client.WindowSec) * time.Second).Unix()
		} else {
			// Fail-closed: deny request when Redis is down
			allowed = false
			remaining = 0
			resetAt = time.Now().Add(time.Duration(client.WindowSec) * time.Second).Unix()
		}
	}
	
	// Build response
	response := &models.RateLimitResponse{
		Allowed:   allowed,
		Remaining: remaining,
		Limit:     client.Limit,
		ResetAt:   resetAt,
	}
	
	// Add retry-after if not allowed
	if !allowed {
		retryAfter := int(time.Until(time.Unix(resetAt, 0)).Seconds())
		if retryAfter < 0 {
			retryAfter = 0
		}
		response.RetryAfter = retryAfter
	}
	
	return response, nil
}

// GetClientStats retrieves current client statistics
func (rl *RateLimiter) GetClientStats(clientID string) (map[string]interface{}, error) {
	client, err := rl.postgres.GetClient(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	
	if client == nil {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}
	
	// Get current count from Redis
	currentCount, err := rl.redis.GetCurrentCount(clientID, client.WindowSec)
	if err != nil {
		rl.logger.Error("Failed to get current count from Redis",
			zap.String("client_id", clientID),
			zap.Error(err),
		)
		currentCount = 0
	}
	
	remaining := client.Limit - int(currentCount)
	if remaining < 0 {
		remaining = 0
	}
	
	return map[string]interface{}{
		"client_id":    clientID,
		"limit":        client.Limit,
		"window_sec":   client.WindowSec,
		"current_used": currentCount,
		"remaining":    remaining,
	}, nil
}

// Close closes all connections
func (rl *RateLimiter) Close() {
	if rl.asyncLogger != nil {
		rl.asyncLogger.Close()
	}
}
