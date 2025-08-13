-- Migration: 004_recreate_accounts_table
-- Spec: docs/specs/003-account-management.md
-- Description: Recreate accounts table for new account management system

-- Drop the old accounts table created in migration 001
DROP TABLE IF EXISTS accounts;

-- Create new accounts table with updated schema per spec
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(36),
    name VARCHAR(255),
    external_id VARCHAR(255),
    external_group_id VARCHAR(255),
    currency_code VARCHAR(3),
    account_type VARCHAR(20),
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    version INTEGER,
    PRIMARY KEY (id)
);

-- Note: ImmuDB limitations:
-- 1. DEFAULT values not supported - will handle in application
-- 2. NOT NULL constraints beyond PRIMARY KEY may not be enforced
-- 3. Can only create indexes on empty tables
-- 4. CREATE UNIQUE INDEX not supported
-- 
-- For external_id uniqueness, we'll enforce this in application code
-- as ImmuDB doesn't support UNIQUE constraints beyond PRIMARY KEY.
-- 
-- ImmuDB will automatically handle indexing for the PRIMARY KEY (id).
-- Additional performance optimizations can be done through ImmuDB's 
-- internal indexing mechanisms if needed.