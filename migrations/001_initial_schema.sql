-- Create clients table
CREATE TABLE IF NOT EXISTS clients (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    rate_limit INTEGER NOT NULL CHECK (rate_limit > 0),
    window_sec INTEGER NOT NULL CHECK (window_sec > 0),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create request_logs table for analytics
CREATE TABLE IF NOT EXISTS request_logs (
    id BIGSERIAL PRIMARY KEY,
    client_id VARCHAR(255) NOT NULL,
    resource VARCHAR(500),
    allowed BOOLEAN NOT NULL,
    response_time_ms BIGINT NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_request_logs_client_id ON request_logs(client_id);
CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_request_logs_client_timestamp ON request_logs(client_id, timestamp);

-- Create index for analytics queries
CREATE INDEX IF NOT EXISTS idx_request_logs_analytics ON request_logs(client_id, timestamp, allowed);

-- Insert sample clients for testing
INSERT INTO clients (id, name, rate_limit, window_sec, created_at, updated_at)
VALUES 
    ('client-a', 'Client A - Banking Service', 100, 60, NOW(), NOW()),
    ('client-b', 'Client B - Logistics Provider', 5000, 60, NOW(), NOW()),
    ('client-c', 'Client C - AI Model Service', 1000, 60, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;
