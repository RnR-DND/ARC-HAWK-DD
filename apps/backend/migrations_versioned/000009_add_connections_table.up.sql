-- Migration: Add connections table for database-backed connection management
-- Replaces YAML file storage with encrypted PostgreSQL storage

CREATE TABLE connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type VARCHAR(50) NOT NULL,
    profile_name VARCHAR(255) NOT NULL,
    config_encrypted BYTEA NOT NULL,
    validation_status VARCHAR(50) DEFAULT 'pending',
    last_validated_at TIMESTAMPTZ,
    validation_error TEXT,
    created_by VARCHAR(255) NOT NULL DEFAULT 'system',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_source_profile UNIQUE(source_type, profile_name)
);

-- Indexes for efficient querying
CREATE INDEX idx_connections_source_type ON connections(source_type);
CREATE INDEX idx_connections_validation_status ON connections(validation_status);
CREATE INDEX idx_connections_created_at ON connections(created_at DESC);

-- Trigger for automatic updated_at timestamp
CREATE OR REPLACE FUNCTION update_connections_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_connections_updated_at
    BEFORE UPDATE ON connections
    FOR EACH ROW
    EXECUTE FUNCTION update_connections_updated_at();
