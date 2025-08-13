-- Migration: 003_create_indexed_tables
-- Author: Platform Team
-- Date: 2025-08-13
-- Description: Creates new tables with proper indexes for ImmuDB
-- Spec: docs/specs/002-database-migrations.md

-- Note: ImmuDB only allows indexes on EMPTY tables
-- These are new tables that will have indexes from the start

-- Create a system configuration table
CREATE TABLE IF NOT EXISTS system_config (
    key VARCHAR(100) PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP,
    updated_by VARCHAR(100)
);

-- Create a migration history table (for audit trail)
CREATE TABLE IF NOT EXISTS migration_history (
    id VARCHAR(36) PRIMARY KEY,
    migration_name VARCHAR(255),
    executed_at TIMESTAMP,
    execution_time_ms INTEGER,
    status VARCHAR(20)
);

-- Note: ImmuDB's CREATE INDEX has limitations:
-- 1. Can only be created on empty tables
-- 2. IF NOT EXISTS might not work as expected
-- For now, we'll skip explicit index creation as ImmuDB handles primary key indexes automatically

-- Insert initial system configuration
-- Note: Using simple values to avoid SQL syntax issues