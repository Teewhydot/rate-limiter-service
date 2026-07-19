package models

import "time"

// Client represents a client with rate limit configuration
type Client struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Limit       int       `json:"limit" db:"rate_limit"`      // Requests per window
	WindowSec   int       `json:"window_sec" db:"window_sec"` // Time window in seconds
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// RateLimitRequest represents an incoming rate limit check request
type RateLimitRequest struct {
	ClientID  string `json:"client_id" binding:"required"`
	Resource  string `json:"resource"`  // Optional: specific API endpoint
}

type RevokeAPIKeyRequest struct {
	ClientID  string `json:"client_id" binding:"required"`
}

// RateLimitResponse represents the response to a rate limit check
type RateLimitResponse struct {
	Allowed       bool   `json:"allowed"`
	Remaining     int    `json:"remaining"`
	Limit         int    `json:"limit"`
	ResetAt       int64  `json:"reset_at"`        // Unix timestamp
	RetryAfter    int    `json:"retry_after,omitempty"` // Seconds to wait if not allowed
}

// RequestLog represents a logged request for analytics
type RequestLog struct {
	ID            int64     `json:"id" db:"id"`
	ClientID      string    `json:"client_id" db:"client_id"`
	Resource      string    `json:"resource" db:"resource"`
	Allowed       bool      `json:"allowed" db:"allowed"`
	ResponseTime  int64     `json:"response_time_ms" db:"response_time_ms"` // Milliseconds
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
}

// UsageStats represents client usage statistics for dashboard
type UsageStats struct {
	ClientID          string    `json:"client_id"`
	TotalRequests     int64     `json:"total_requests"`
	AllowedRequests   int64     `json:"allowed_requests"`
	BlockedRequests   int64     `json:"blocked_requests"`
	AvgResponseTime   float64   `json:"avg_response_time_ms"`
	PeriodStart       time.Time `json:"period_start"`
	PeriodEnd         time.Time `json:"period_end"`
}

// TrendData represents time-series data for trend graphs
type TrendData struct {
	Timestamp       time.Time `json:"timestamp"`
	RequestCount    int64     `json:"request_count"`
	BlockedCount    int64     `json:"blocked_count"`
	AvgResponseTime float64   `json:"avg_response_time_ms"`
}

// DashboardFilter represents filter parameters for dashboard queries
type DashboardFilter struct {
	ClientID    string    `json:"client_id" form:"client_id"`
	StartDate   time.Time `json:"start_date" form:"start_date"`
	EndDate     time.Time `json:"end_date" form:"end_date"`
	Days        int       `json:"days" form:"days"` // Shortcut: 10, 15, or 30 days
	Resource    string    `json:"resource" form:"resource"`
}
