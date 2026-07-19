package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/tundesmac/rate-limiter-service/internal/logger"
	"github.com/tundesmac/rate-limiter-service/internal/models"
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
