package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/tundesmac/rate-limiter-service/internal/logger"
	"github.com/tundesmac/rate-limiter-service/internal/models"
	"net/http"
	
	"github.com/tundesmac/rate-limiter-service/internal/auth"
	"github.com/tundesmac/rate-limiter-service/internal/storage"

)

// LoggerMiddleware logs all incoming requests
func LoggerMiddleware(logger *zap.Logger, asyncLogger *logger.AsyncLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		
		// Process request
		c.Next()
		
		// Log after request
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		
		logger.Info("Request processed",
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", clientIP),
		)
		
		// Log rate limit requests to database (if this is a rate limit check)
		if path == "/api/v1/ratelimit/check" && method == "POST" {
			// Get the request data from context if it was set by the handler
			if clientID, exists := c.Get("client_id"); exists {
				resource, _ := c.Get("resource")
				allowed, _ := c.Get("allowed")
				
				asyncLogger.Log(models.RequestLog{
					ClientID:     clientID.(string),
					Resource:     resource.(string),
					Allowed:      allowed.(bool),
					ResponseTime: latency.Milliseconds(),
					Timestamp:    start,
				})
			}
		}
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	}
}


// APIKeyAuthMiddleware validates API key and extracts client_id
func APIKeyAuthMiddleware(postgres *storage.PostgresClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract API key from header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Try Authorization header: "Bearer <key>"
			apiKey = c.GetHeader("Authorization")
			if len(apiKey) > 7 && apiKey[:7] == "Bearer " {
				apiKey = apiKey[7:] // Remove "Bearer " prefix
			}
		}
		
		// 2. Validate key exists
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API key required",
				"code":  "missing_api_key",
			})
			c.Abort()
			return
		}
		
		// 3. Hash the incoming API key
		keyHash := auth.HashAPIKey(apiKey)
		
		// 4. Look up client_id in database using the hash
		clientID, err := postgres.GetClientIDByAPIKey(keyHash)
		if err != nil || clientID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or revoked API key",
				"code":  "invalid_api_key",
			})
			c.Abort()
			return
		}
		
		// 5. Store client_id in context for handlers to use
		c.Set("client_id", clientID)
		
		// 6. Continue to next handler
		c.Next()
	}
}
