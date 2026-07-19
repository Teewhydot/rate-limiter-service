package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"github.com/tundesmac/rate-limiter-service/internal/api"
	"github.com/tundesmac/rate-limiter-service/internal/config"
	"github.com/tundesmac/rate-limiter-service/internal/logger"
	"github.com/tundesmac/rate-limiter-service/internal/ratelimiter"
	"github.com/tundesmac/rate-limiter-service/internal/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	
	// Initialize logger
	var zapLogger *zap.Logger
	if cfg.LogLevel == "debug" {
		zapLogger, _ = zap.NewDevelopment()
	} else {
		zapLogger, _ = zap.NewProduction()
	}
	defer zapLogger.Sync()
	
	zapLogger.Info("Starting Rate Limiter Service",
		zap.String("version", "1.0.0"),
		zap.String("port", cfg.ServerPort),
	)
	
	// Initialize Redis client
	zapLogger.Info("Connecting to Redis", zap.String("addr", cfg.RedisAddr))
	redisClient, err := storage.NewRedisClient(cfg)
	if err != nil {
		zapLogger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()
	zapLogger.Info("Redis connected successfully")
	
	// Initialize PostgreSQL client
	zapLogger.Info("Connecting to PostgreSQL", zap.String("host", cfg.PostgresHost))
	postgresClient, err := storage.NewPostgresClient(cfg)
	if err != nil {
		zapLogger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer postgresClient.Close()
	zapLogger.Info("PostgreSQL connected successfully")
	
	// Initialize async logger
	asyncLogger := logger.NewAsyncLogger(
		postgresClient,
		cfg.AsyncLogBatchSize,
		cfg.AsyncLogInterval,
		zapLogger,
	)
	defer asyncLogger.Close()
	zapLogger.Info("Async logger initialized",
		zap.Int("batch_size", cfg.AsyncLogBatchSize),
		zap.Int("flush_interval_sec", cfg.AsyncLogInterval),
	)
	
	// Initialize rate limiter
	rateLimiter := ratelimiter.NewRateLimiter(
		redisClient,
		postgresClient,
		asyncLogger,
		cfg,
		zapLogger,
	)
	defer rateLimiter.Close()
	
	// Initialize API handlers
	handler := api.NewHandler(rateLimiter, postgresClient, redisClient, zapLogger)
	
	// Setup routes
	router := api.SetupRoutes(handler, zapLogger, asyncLogger)
	
	// Start server in a goroutine
	go func() {
		addr := fmt.Sprintf(":%s", cfg.ServerPort)
		zapLogger.Info("Server starting", zap.String("address", addr))
		if err := router.Run(addr); err != nil {
			zapLogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()
	
	// Wait for interrupt signal to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	zapLogger.Info("Shutting down server...")
	zapLogger.Info("Server stopped")
}
