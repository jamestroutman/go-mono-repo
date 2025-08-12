-- Rollback currencies table creation
-- Spec: docs/specs/003-currency-management.md

BEGIN;

-- Drop trigger
DROP TRIGGER IF EXISTS update_currencies_updated_at ON treasury.currencies;

-- Drop indexes
DROP INDEX IF EXISTS treasury.idx_currencies_code;
DROP INDEX IF EXISTS treasury.idx_currencies_numeric_code;
DROP INDEX IF EXISTS treasury.idx_currencies_status;
DROP INDEX IF EXISTS treasury.idx_currencies_country_codes;
DROP INDEX IF EXISTS treasury.idx_currencies_is_active;

-- Drop table
DROP TABLE IF EXISTS treasury.currencies;

COMMIT;