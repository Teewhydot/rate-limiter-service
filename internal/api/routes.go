package api

import (
	"github.com/gin-gonic/gin"
	"github.com/tundesmac/rate-limiter-service/internal/logger"

	"go.uber.org/zap"
)

// SetupRoutes configures all API routes
func SetupRoutes(handler *Handler, zapLogger *zap.Logger, asyncLogger *logger.AsyncLogger) *gin.Engine {
	// Set Gin mode based on environment
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(LoggerMiddleware(zapLogger, asyncLogger))
	router.Use(CORSMiddleware())

	// Health check (no prefix)
	router.GET("/health", handler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Rate limit checking
		v1.POST("/ratelimit/check", handler.CheckRateLimit)

		// Client management
		clients := v1.Group("/clients")
		{
			clients.POST("", handler.CreateClient)
			clients.GET("", handler.ListClients)
			clients.GET("/:id", handler.GetClient)
			clients.PUT("/:id", handler.UpdateClient)
			clients.POST("/:id/apikey/revoke", handler.RevokeAPIKey)
		}

		// Real-time stats
		v1.GET("/stats/:client_id", handler.GetCurrentStats)

		// Dashboard endpoints (for Flutter app)
		router.Use(APIKeyAuthMiddleware(handler.postgres))
		dashboard := v1.Group("/dashboard")
		{
			dashboard.GET("/usage/:client_id", handler.GetUsageStats)
			dashboard.GET("/trends/:client_id", handler.GetTrendData)
		}
	}

	return router
}
