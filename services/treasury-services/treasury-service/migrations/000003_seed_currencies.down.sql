-- Remove seeded currency data
-- Spec: docs/specs/003-currency-management.md

BEGIN;

-- Delete all currencies added by system
DELETE FROM treasury.currencies WHERE created_by = 'system';

COMMIT;