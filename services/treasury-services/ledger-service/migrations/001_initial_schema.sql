-- Migration: 001_initial_schema
-- Author: Platform Team
-- Date: 2025-01-11
-- Description: Creates initial schema for ledger service
-- Spec: docs/specs/002-database-migrations.md

-- Create accounts table for managing ledger accounts
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,  -- ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20),
    metadata VARCHAR,
    created_at TIMESTAMP,
    created_by VARCHAR(100),
    updated_at TIMESTAMP
);

-- Indexes can only be created on empty tables in ImmuDB
-- Since this migration might run on existing data, skip indexes here

-- Create transactions table for recording ledger entries
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(36) PRIMARY KEY,
    account_id VARCHAR(36) NOT NULL,
    amount DECIMAL(20,4) NOT NULL,
    balance DECIMAL(20,4) NOT NULL,  -- Running balance after this transaction
    type VARCHAR(20) NOT NULL,  -- DEBIT, CREDIT
    reference VARCHAR(100),
    description VARCHAR,
    metadata VARCHAR,
    created_at TIMESTAMP,
    created_by VARCHAR(100)
);

-- Indexes can only be created on empty tables in ImmuDB

-- Create journal entries table for double-entry bookkeeping
CREATE TABLE IF NOT EXISTS journal_entries (
    id VARCHAR(36) PRIMARY KEY,
    entry_date TIMESTAMP NOT NULL,
    description VARCHAR,
    reference VARCHAR(100),
    status VARCHAR(20),  -- PENDING, POSTED, CANCELLED
    metadata VARCHAR,
    created_at TIMESTAMP,
    created_by VARCHAR(100)
);

-- Create journal entry lines for debit/credit entries
CREATE TABLE IF NOT EXISTS journal_entry_lines (
    id VARCHAR(36) PRIMARY KEY,
    journal_entry_id VARCHAR(36) NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    debit_amount DECIMAL(20,4),
    credit_amount DECIMAL(20,4),
    description VARCHAR,
    created_at TIMESTAMP
);

-- Indexes can only be created on empty tables in ImmuDB

-- Create audit log table for tracking all changes
CREATE TABLE IF NOT EXISTS audit_log (
    id VARCHAR(36) PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,  -- accounts, transactions, journal_entries
    entity_id VARCHAR(36) NOT NULL,
    action VARCHAR(20) NOT NULL,  -- CREATE, UPDATE, DELETE
    old_values VARCHAR,
    new_values VARCHAR,
    user_id VARCHAR(100),
    timestamp TIMESTAMP,
    metadata VARCHAR
);

-- Indexes can only be created on empty tables in ImmuDB