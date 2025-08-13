-- Migration: 000004_create_financial_institutions_table.down.sql
-- Spec: docs/specs/004-financial-institutions.md

BEGIN;

-- Drop reference tracking table
DROP TABLE IF EXISTS treasury.institution_references;

-- Drop routing numbers table
DROP TABLE IF EXISTS treasury.institution_routing_numbers;

-- Drop triggers
DROP TRIGGER IF EXISTS update_institutions_updated_at ON treasury.financial_institutions;

-- Drop indexes
DROP INDEX IF EXISTS treasury.idx_institutions_code;
DROP INDEX IF EXISTS treasury.idx_institutions_swift;
DROP INDEX IF EXISTS treasury.idx_institutions_country;
DROP INDEX IF EXISTS treasury.idx_institutions_type;
DROP INDEX IF EXISTS treasury.idx_institutions_status;
DROP INDEX IF EXISTS treasury.idx_institutions_is_active;

-- Drop financial institutions table
DROP TABLE IF EXISTS treasury.financial_institutions;

COMMIT;