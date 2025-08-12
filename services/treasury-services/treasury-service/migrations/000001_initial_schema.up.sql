-- Initial schema setup for Treasury Service
-- Spec: docs/specs/002-database-migrations.md

BEGIN;

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS treasury;

-- Set search path
SET search_path TO treasury, public;

-- Create migration metadata table
CREATE TABLE IF NOT EXISTS migration_metadata (
    id SERIAL PRIMARY KEY,
    version INTEGER NOT NULL,
    description VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    execution_time_ms INTEGER,
    checksum VARCHAR(64)
);

-- Create base audit fields function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add migration record
INSERT INTO migration_metadata (version, description, checksum)
VALUES (1, 'initial_schema', 'sha256:pending');

COMMIT;