-- Rollback initial schema setup
-- Spec: docs/specs/002-database-migrations.md

BEGIN;

SET search_path TO treasury, public;

-- Drop functions
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- Drop metadata table
DROP TABLE IF EXISTS migration_metadata CASCADE;

-- Drop schema if empty
DROP SCHEMA IF EXISTS treasury CASCADE;

COMMIT;