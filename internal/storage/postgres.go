package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/tundesmac/rate-limiter-service/internal/config"
	"github.com/tundesmac/rate-limiter-service/internal/models"
)

// PostgresClient wraps the PostgreSQL database connection
type PostgresClient struct {
	db *sql.DB
}

// NewPostgresClient creates a new PostgreSQL client
func NewPostgresClient(cfg *config.Config) (*PostgresClient, error) {
	db, err := sql.Open("postgres", cfg.PostgresDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	
	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	
	return &PostgresClient{db: db}, nil
}

// Close closes the database connection
func (p *PostgresClient) Close() error {
	return p.db.Close()
}

// GetClient retrieves a client by ID
func (p *PostgresClient) GetClient(clientID string) (*models.Client, error) {
	query := `
		SELECT id, name, rate_limit, window_sec, created_at, updated_at
		FROM clients
		WHERE id = $1
	`
	
	var client models.Client
	err := p.db.QueryRow(query, clientID).Scan(
		&client.ID,
		&client.Name,
		&client.Limit,
		&client.WindowSec,
		&client.CreatedAt,
		&client.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // Client not found
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	
	return &client, nil
}

// CreateClient creates a new client
func (p *PostgresClient) CreateClient(client *models.Client) error {
	query := `
		INSERT INTO clients (id, name, rate_limit, window_sec, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	
	now := time.Now()
	_, err := p.db.Exec(query,
		client.ID,
		client.Name,
		client.Limit,
		client.WindowSec,
		now,
		now,
	)
	
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	
	return nil
}

// UpdateClient updates an existing client
func (p *PostgresClient) UpdateClient(client *models.Client) error {
	query := `
		UPDATE clients
		SET name = $2, rate_limit = $3, window_sec = $4, updated_at = $5
		WHERE id = $1
	`
	
	_, err := p.db.Exec(query,
		client.ID,
		client.Name,
		client.Limit,
		client.WindowSec,
		time.Now(),
	)
	
	if err != nil {
		return fmt.Errorf("failed to update client: %w", err)
	}
	
	return nil
}

// ListClients retrieves all clients
func (p *PostgresClient) ListClients() ([]models.Client, error) {
	query := `
		SELECT id, name, rate_limit, window_sec, created_at, updated_at
		FROM clients
		ORDER BY created_at DESC
	`
	
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list clients: %w", err)
	}
	defer rows.Close()
	
	var clients []models.Client
	for rows.Next() {
		var client models.Client
		err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.Limit,
			&client.WindowSec,
			&client.CreatedAt,
			&client.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}
		clients = append(clients, client)
	}
	
	return clients, nil
}

// LogRequest logs a request asynchronously (called by async logger)
func (p *PostgresClient) LogRequest(log *models.RequestLog) error {
	query := `
		INSERT INTO request_logs (client_id, resource, allowed, response_time_ms, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	_, err := p.db.Exec(query,
		log.ClientID,
		log.Resource,
		log.Allowed,
		log.ResponseTime,
		log.Timestamp,
	)
	
	if err != nil {
		return fmt.Errorf("failed to log request: %w", err)
	}
	
	return nil
}

// LogRequestBatch logs multiple requests in a single transaction (for performance)
func (p *PostgresClient) LogRequestBatch(logs []models.RequestLog) error {
	if len(logs) == 0 {
		return nil
	}
	
	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	stmt, err := tx.Prepare(`
		INSERT INTO request_logs (client_id, resource, allowed, response_time_ms, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	for _, log := range logs {
		_, err := stmt.Exec(
			log.ClientID,
			log.Resource,
			log.Allowed,
			log.ResponseTime,
			log.Timestamp,
		)
		if err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	return nil
}

// GetUsageStats retrieves usage statistics for a client within a time range
func (p *PostgresClient) GetUsageStats(clientID string, startDate, endDate time.Time) (*models.UsageStats, error) {
	query := `
		SELECT
			client_id,
			COUNT(*) as total_requests,
			SUM(CASE WHEN allowed = true THEN 1 ELSE 0 END) as allowed_requests,
			SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) as blocked_requests,
			AVG(response_time_ms) as avg_response_time
		FROM request_logs
		WHERE client_id = $1 AND timestamp BETWEEN $2 AND $3
		GROUP BY client_id
	`
	
	var stats models.UsageStats
	err := p.db.QueryRow(query, clientID, startDate, endDate).Scan(
		&stats.ClientID,
		&stats.TotalRequests,
		&stats.AllowedRequests,
		&stats.BlockedRequests,
		&stats.AvgResponseTime,
	)
	
	if err == sql.ErrNoRows {
		// No data for this period, return zero stats
		return &models.UsageStats{
			ClientID:    clientID,
			PeriodStart: startDate,
			PeriodEnd:   endDate,
		}, nil
	}
	
	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}
	
	stats.PeriodStart = startDate
	stats.PeriodEnd = endDate
	
	return &stats, nil
}

// GetTrendData retrieves time-series data for trend graphs
func (p *PostgresClient) GetTrendData(clientID string, startDate, endDate time.Time, intervalHours int) ([]models.TrendData, error) {
	query := `
		SELECT
			DATE_TRUNC('hour', timestamp) + 
			INTERVAL '1 hour' * (EXTRACT(hour FROM timestamp)::int / $4) * $4 as time_bucket,
			COUNT(*) as request_count,
			SUM(CASE WHEN allowed = false THEN 1 ELSE 0 END) as blocked_count,
			AVG(response_time_ms) as avg_response_time
		FROM request_logs
		WHERE client_id = $1 AND timestamp BETWEEN $2 AND $3
		GROUP BY time_bucket
		ORDER BY time_bucket
	`
	
	rows, err := p.db.Query(query, clientID, startDate, endDate, intervalHours)
	if err != nil {
		return nil, fmt.Errorf("failed to get trend data: %w", err)
	}
	defer rows.Close()
	
	var trends []models.TrendData
	for rows.Next() {
		var trend models.TrendData
		err := rows.Scan(
			&trend.Timestamp,
			&trend.RequestCount,
			&trend.BlockedCount,
			&trend.AvgResponseTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trend data: %w", err)
		}
		trends = append(trends, trend)
	}
	
	return trends, nil
}

// HealthCheck checks if PostgreSQL is available
func (p *PostgresClient) HealthCheck() error {
	return p.db.Ping()
}
