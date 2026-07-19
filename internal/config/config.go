package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	// Server settings
	ServerPort string
	
	// Redis settings
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	
	// PostgreSQL settings
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string
	
	// Rate limiter settings
	DefaultLimit      int  // Default requests per minute if client not found
	DefaultWindowSec  int  // Default time window in seconds
	FailSafeMode      bool // Allow requests when Redis is down
	
	// Logging settings
	LogLevel          string
	AsyncLogBatchSize int // Number of logs to batch before writing
	AsyncLogInterval  int // Seconds between log flushes
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore error in production)
	_ = godotenv.Load()
	
	config := &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     getEnv("REDIS_PASSWORD", ""),
		RedisDB:           getEnvAsInt("REDIS_DB", 0),
		PostgresHost:      getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:      getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:      getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword:  getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:        getEnv("POSTGRES_DB", "ratelimiter"),
		PostgresSSLMode:   getEnv("POSTGRES_SSLMODE", "disable"),
		DefaultLimit:      getEnvAsInt("DEFAULT_LIMIT", 100),
		DefaultWindowSec:  getEnvAsInt("DEFAULT_WINDOW_SEC", 60),
		FailSafeMode:      getEnvAsBool("FAILSAFE_MODE", true),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		AsyncLogBatchSize: getEnvAsInt("ASYNC_LOG_BATCH_SIZE", 100),
		AsyncLogInterval:  getEnvAsInt("ASYNC_LOG_INTERVAL", 5),
	}
	
	return config, nil
}

// PostgresDSN returns the PostgreSQL connection string
func (c *Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresDB,
		c.PostgresSSLMode,
	)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt gets an environment variable as an integer or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

// getEnvAsBool gets an environment variable as a boolean or returns a default value
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}
