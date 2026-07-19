-- Create API keys table

CREATE TABLE IF NOT EXISTS api_keys (
    id VARCHAR(255) PRIMARY KEY NOT NULL,
    key_hash VARCHAR(255) UNIQUE NOT NULL,
    client_id VARCHAR(255),
    name VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL
);


-- Create index on client id
CREATE INDEX IF NOT EXISTS idx_api_keys_client_id ON api_keys(client_id);

-- Add foreign key constraint
ALTER TABLE api_keys 
ADD CONSTRAINT fk_api_keys_client 
FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE;