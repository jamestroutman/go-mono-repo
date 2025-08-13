-- Migration: 002_add_account_constraints
-- Author: Platform Team
-- Date: 2025-01-11
-- Description: Adds account validation tables and constraints
-- Spec: docs/specs/002-database-migrations.md

-- Create account types reference table
CREATE TABLE IF NOT EXISTS account_types (
    code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    normal_balance VARCHAR(10) NOT NULL,  -- DEBIT or CREDIT
    description TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert standard account types
INSERT INTO account_types (code, name, normal_balance, description) VALUES
    ('ASSET', 'Asset', 'DEBIT', 'Resources owned by the entity'),
    ('LIABILITY', 'Liability', 'CREDIT', 'Obligations owed by the entity'),
    ('EQUITY', 'Equity', 'CREDIT', 'Owner''s interest in the entity'),
    ('REVENUE', 'Revenue', 'CREDIT', 'Income earned by the entity'),
    ('EXPENSE', 'Expense', 'DEBIT', 'Costs incurred by the entity');

-- Create currency reference table
CREATE TABLE IF NOT EXISTS currencies (
    code VARCHAR(3) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    symbol VARCHAR(10),
    decimal_places INTEGER DEFAULT 2,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert common currencies
INSERT INTO currencies (code, name, symbol, decimal_places) VALUES
    ('USD', 'US Dollar', '$', 2),
    ('EUR', 'Euro', '€', 2),
    ('GBP', 'British Pound', '£', 2),
    ('JPY', 'Japanese Yen', '¥', 0),
    ('CHF', 'Swiss Franc', 'CHF', 2),
    ('CAD', 'Canadian Dollar', 'C$', 2),
    ('AUD', 'Australian Dollar', 'A$', 2),
    ('CNY', 'Chinese Yuan', '¥', 2);

-- Create account status reference table
CREATE TABLE IF NOT EXISTS account_statuses (
    code VARCHAR(20) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    can_transact BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert standard account statuses
INSERT INTO account_statuses (code, name, description, can_transact) VALUES
    ('ACTIVE', 'Active', 'Account is active and can receive transactions', TRUE),
    ('INACTIVE', 'Inactive', 'Account is temporarily inactive', FALSE),
    ('CLOSED', 'Closed', 'Account is permanently closed', FALSE),
    ('FROZEN', 'Frozen', 'Account is frozen for compliance reasons', FALSE),
    ('PENDING', 'Pending', 'Account is pending activation', FALSE);

-- Create balance snapshot table for period-end balances
CREATE TABLE IF NOT EXISTS balance_snapshots (
    id VARCHAR(36) PRIMARY KEY,
    account_id VARCHAR(36) NOT NULL,
    snapshot_date TIMESTAMP NOT NULL,
    balance DECIMAL(20,4) NOT NULL,
    transaction_count INTEGER,
    debit_total DECIMAL(20,4),
    credit_total DECIMAL(20,4),
    created_at TIMESTAMP
);

-- Indexes can only be created on empty tables in ImmuDB