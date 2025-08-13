# Ledger Service Database Migrations

This directory contains database migration files for the ledger service.

## Migration File Format

Migration files follow the naming convention:
```
NNN_description.sql
```

Where:
- `NNN` is a 3-digit sequential number (001, 002, 003, etc.)
- `description` is a brief description using underscores

Examples:
- `001_initial_schema.sql`
- `002_create_accounts_table.sql`
- `003_add_transaction_indexes.sql`

## Creating New Migrations

To create a new migration, use the make command from the repo root:

```bash
make migration-ledger-new NAME=your_migration_name
```

This will:
1. Create a new migration file with the next available number
2. Add a template with proper headers
3. Include the current date and spec reference

## Running Migrations

From the repo root:

```bash
# Run all pending migrations
make migrate-ledger

# Check migration status
make migrate-ledger-status

# Validate migration files
make migrate-ledger-validate

# Dry run (show what would be executed)
make migrate-ledger-dry-run
```

## Migration Best Practices

### DO's
- Keep migrations small and focused on a single change
- Use `IF NOT EXISTS` for idempotency
- Include descriptive comments in your migrations
- Test migrations locally before committing
- Use transactions where appropriate (ImmuDB supports them)

### DON'Ts
- Never modify existing migration files after they've been committed
- Don't skip migration numbers
- Don't use `DROP` statements (ImmuDB is append-only)
- Don't include sensitive data in migrations
- Don't use non-deterministic functions (like RANDOM without seed)

## ImmuDB-Specific Considerations

Since ImmuDB is an append-only, immutable database:

1. **No UPDATE or DELETE**: Data cannot be modified or deleted once written
2. **No ALTER TABLE**: To add columns, create a new table and migrate data
3. **Indexes are safe**: Can be added without affecting existing data
4. **CREATE TABLE IF NOT EXISTS**: Always use to ensure idempotency

## Example Migration

```sql
-- Migration: 001_initial_schema
-- Author: Platform Team
-- Date: 2025-01-11
-- Description: Creates initial ledger schema
-- Spec: docs/specs/002-database-migrations.md

CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(20) UNIQUE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE',
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100)
);

CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(type);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
```

## Migration Status

The migration system tracks applied migrations in the `ledger_schema_migrations` table. Each migration records:
- Version number
- Migration name
- Checksum (to detect modifications)
- Execution timestamp
- Execution duration
- Success/failure status

## Troubleshooting

### Migration Fails
1. Check the error message in the output
2. Review the migration SQL for syntax errors
3. Ensure ImmuDB is running and accessible
4. Check that you have proper permissions

### Checksum Mismatch
If you see a warning about checksum mismatch:
1. Someone has modified an already-applied migration
2. This is a warning only - the migration won't re-run
3. Review the changes and ensure they're intentional

### Connection Issues
Ensure ImmuDB is running:
```bash
make infrastructure-status
```

If not running:
```bash
make infrastructure-up
```

## References

- [Migration Specification](../docs/specs/002-database-migrations.md)
- [ImmuDB SQL Documentation](https://docs.immudb.io/master/develop/sql/)
- [Service Development Guide](../../../../docs/SERVICE_DEVELOPMENT.md)