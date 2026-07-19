package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tundesmac/rate-limiter-service/internal/models"
	"github.com/tundesmac/rate-limiter-service/internal/ratelimiter"
	"github.com/tundesmac/rate-limiter-service/internal/storage"
	"github.com/tundesmac/rate-limiter-service/internal/auth"
	"go.uber.org/zap"
)

// Handler holds all HTTP handlers
type Handler struct {
	rateLimiter *ratelimiter.RateLimiter
	postgres    *storage.PostgresClient
	redis       *storage.RedisClient
	logger      *zap.Logger
}

// NewHandler creates a new handler instance
func NewHandler(
	rl *ratelimiter.RateLimiter,
	postgres *storage.PostgresClient,
	redis *storage.RedisClient,
	logger *zap.Logger,
) *Handler {
	return &Handler{
		rateLimiter: rl,
		postgres:    postgres,
		redis:       redis,
		logger:      logger,
	}
}

// CheckRateLimit handles rate limit check requests
// POST /api/v1/ratelimit/check
func (h *Handler) CheckRateLimit(c *gin.Context) {
	var req models.RateLimitRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	response, err := h.rateLimiter.CheckLimit(req)
	if err != nil {
		h.logger.Error("Rate limit check failed",
			zap.String("client_id", req.ClientID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
		return
	}

	// Set context values for middleware logging
	c.Set("client_id", req.ClientID)
	c.Set("resource", req.Resource)
	c.Set("allowed", response.Allowed)

	// Set standard rate limit headers
	c.Header("X-RateLimit-Limit", strconv.Itoa(response.Limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(response.Remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(response.ResetAt, 10))

	if !response.Allowed {
		c.Header("Retry-After", strconv.Itoa(response.RetryAfter))
		c.JSON(http.StatusTooManyRequests, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// CreateClient handles client creation
// POST /api/v1/clients
func (h *Handler) CreateClient(c *gin.Context) {
	var client models.Client

	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if client.ID == "" || client.Name == "" || client.Limit <= 0 || client.WindowSec <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing or invalid required fields: id, name, limit, window_sec",
		})
		return
	}

	// Create client and get API key
	apikeyString, err := h.postgres.CreateClient(&client)
	if err != nil {
		h.logger.Error("Failed to create client",
			zap.String("client_id", client.ID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create client",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         client.ID,
		"name":       client.Name,
		"rate_limit": client.Limit,
		"window_sec": client.WindowSec,
		"created_at": client.CreatedAt,
		"api_key":    apikeyString,
		"message":    "Save this API key securely, This key wont be shown again!",
	})
}

// GetClient handles retrieving a single client
// GET /api/v1/clients/:id
func (h *Handler) GetClient(c *gin.Context) {
	clientID := c.Param("id")

	if clientID == "me" {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.GetHeader("Authorization")
			if len(apiKey) > 7 && apiKey[:7] == "Bearer " {
				apiKey = apiKey[7:]
			}
		}
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "API key required for /me"})
			return
		}
		keyHash := auth.HashAPIKey(apiKey)
		authClientID, err := h.postgres.GetClientIDByAPIKey(keyHash)
		if err != nil || authClientID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}
		clientID = authClientID
	}

	client, err := h.postgres.GetClient(clientID)
	if err != nil {
		h.logger.Error("Failed to get client",
			zap.String("client_id", clientID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get client",
		})
		return
	}

	if client == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Client not found",
		})
		return
	}

	c.JSON(http.StatusOK, client)
}

// ListClients handles retrieving all clients
// GET /api/v1/clients
func (h *Handler) ListClients(c *gin.Context) {
	clients, err := h.postgres.ListClients()
	if err != nil {
		h.logger.Error("Failed to list clients", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list clients",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"clients": clients,
		"count":   len(clients),
	})
}

// UpdateClient handles updating a client
// PUT /api/v1/clients/:id
func (h *Handler) UpdateClient(c *gin.Context) {
	clientID := c.Param("id")

	var client models.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	client.ID = clientID

	if err := h.postgres.UpdateClient(&client); err != nil {
		h.logger.Error("Failed to update client",
			zap.String("client_id", clientID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update client",
		})
		return
	}

	c.JSON(http.StatusOK, client)
}

// Revoke API Key
func (h *Handler) RevokeAPIKey(c *gin.Context) {
	clientId := c.Param("id")

	var revoke_request models.RevokeAPIKeyRequest
	if err := c.ShouldBindJSON(&revoke_request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}
	if err := h.postgres.RevokeAPIKey(clientId); err != nil {
		h.logger.Error("Failed to update client",
			zap.String("client_id", clientId),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update client",
		})
		return
	}
}

// GetUsageStats handles retrieving usage statistics for dashboard
// GET /api/v1/dashboard/usage/:client_id
func (h *Handler) GetUsageStats(c *gin.Context) {
	clientID := c.Param("client_id")

	// Parse query parameters for date range
	days := c.DefaultQuery("days", "30")
	daysInt, err := strconv.Atoi(days)
	if err != nil || daysInt <= 0 {
		daysInt = 30
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -daysInt)

	// Allow custom date range if provided
	if start := c.Query("start_date"); start != "" {
		if t, err := time.Parse(time.RFC3339, start); err == nil {
			startDate = t
		}
	}
	if end := c.Query("end_date"); end != "" {
		if t, err := time.Parse(time.RFC3339, end); err == nil {
			endDate = t
		}
	}

	stats, err := h.postgres.GetUsageStats(clientID, startDate, endDate)
	if err != nil {
		h.logger.Error("Failed to get usage stats",
			zap.String("client_id", clientID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get usage stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetTrendData handles retrieving trend data for graphs
// GET /api/v1/dashboard/trends/:client_id
func (h *Handler) GetTrendData(c *gin.Context) {
	clientID := c.Param("client_id")

	// Parse query parameters
	days := c.DefaultQuery("days", "30")
	daysInt, err := strconv.Atoi(days)
	if err != nil || daysInt <= 0 {
		daysInt = 30
	}

	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -daysInt)

	// Determine interval based on days
	intervalHours := 1
	if daysInt > 7 {
		intervalHours = 4
	}
	if daysInt > 15 {
		intervalHours = 12
	}
	if daysInt > 30 {
		intervalHours = 24
	}

	trends, err := h.postgres.GetTrendData(clientID, startDate, endDate, intervalHours)
	if err != nil {
		h.logger.Error("Failed to get trend data",
			zap.String("client_id", clientID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get trend data",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"client_id": clientID,
		"period": gin.H{
			"start": startDate,
			"end":   endDate,
			"days":  daysInt,
		},
		"interval_hours": intervalHours,
		"data":           trends,
	})
}

// GetCurrentStats handles retrieving real-time client statistics
// GET /api/v1/stats/:client_id
func (h *Handler) GetCurrentStats(c *gin.Context) {
	clientID := c.Param("client_id")

	stats, err := h.rateLimiter.GetClientStats(clientID)
	if err != nil {
		h.logger.Error("Failed to get current stats",
			zap.String("client_id", clientID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get current stats",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// HealthCheck handles health check requests
// GET /health
func (h *Handler) HealthCheck(c *gin.Context) {
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
	}

	// Check Redis
	if err := h.redis.HealthCheck(); err != nil {
		health["redis"] = "unhealthy"
		health["redis_error"] = err.Error()
		health["status"] = "degraded"
	} else {
		health["redis"] = "healthy"
	}

	// Check PostgreSQL
	if err := h.postgres.HealthCheck(); err != nil {
		health["postgres"] = "unhealthy"
		health["postgres_error"] = err.Error()
		health["status"] = "degraded"
	} else {
		health["postgres"] = "healthy"
	}

	statusCode := http.StatusOK
	if health["status"] == "degraded" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}
