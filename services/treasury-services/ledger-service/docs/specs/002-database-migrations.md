# Database Migration System Specification

> **Status**: Draft  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-11  
> **Author(s)**: Engineering Team  
> **Reviewer(s)**: Platform Team, Database Team  
> **Confluence**: https://example.atlassian.net/wiki/spaces/LEDGER/pages/002/Database+Migrations  

## Executive Summary

The Ledger Service requires a robust, versioned migration system to manage ImmuDB schema evolution, initial data seeding, and database structure changes in a controlled and auditable manner. This specification defines a file-based migration system using numbered SQL-like scripts that can be executed either via Makefile targets before service startup or automatically during service initialization, ensuring database schema consistency across environments while maintaining low operational overhead.

## Problem Statement

### Current State
The Ledger Service has established ImmuDB connectivity (per spec 001) but lacks a systematic approach to manage database schema changes, initial data loading, and structural evolution. Without a migration system, database changes are ad-hoc, non-repeatable across environments, and lack version control, making deployments risky and rollbacks impossible.

### Desired State
The Ledger Service will implement a lightweight, file-based migration system following the industry-standard numbered migration pattern (001_description.sql, 002_description.sql). The system will track applied migrations, ensure idempotent execution, support both pre-boot (via Makefile) and on-boot execution modes, and provide clear visibility into migration status while maintaining minimal complexity and overhead.

## Scope

### In Scope
- File-based migration system with numbered sequential files
- Migration tracking table in ImmuDB
- Makefile targets for migration execution
- Optional on-boot migration execution
- Migration validation and checksums
- Up migrations only (ImmuDB is append-only)
- Migration status reporting
- Dry-run capability for testing
- Transaction support for atomic migrations

### Out of Scope
- Down/rollback migrations (ImmuDB is immutable)
- Complex migration DSL or ORM integration
- Cross-database migration support
- Data transformation migrations (separate spec)
- Migration UI or web interface
- Distributed migration coordination
- Schema versioning beyond sequential numbering

## User Stories

### Story 1: Migration File Management
**As a** developer  
**I want to** create numbered migration files  
**So that** database changes are versioned and ordered  

**Acceptance Criteria:**
- [ ] Migration files follow pattern: `NNN_description.sql`
- [ ] Files are stored in `migrations/` directory
- [ ] Numbering is sequential (001, 002, 003)
- [ ] Each file contains ImmuDB-compatible SQL
- [ ] Files include metadata comments (author, date, description)
- [ ] README documents migration conventions

### Story 2: Pre-boot Migration Execution
**As a** DevOps engineer  
**I want to** run migrations via Makefile before service start  
**So that** the database is ready before the service boots  

**Acceptance Criteria:**
- [ ] `make migrate-ledger` runs pending migrations for ledger service
- [ ] `make migrate-ledger-status` shows migration state
- [ ] `make migrate-ledger-validate` checks migration files
- [ ] `make migrate-ledger-dry-run` shows what would execute
- [ ] `make migrate-all` runs migrations for all services
- [ ] Makefile target in main service startup flow
- [ ] Clear output showing migration progress
- [ ] Exit codes indicate success/failure

### Story 3: On-boot Migration Execution
**As a** service operator  
**I want to** optionally run migrations on service startup  
**So that** deployments are simplified in containerized environments  

**Acceptance Criteria:**
- [ ] Environment variable controls on-boot execution
- [ ] Migrations run before gRPC server starts
- [ ] Failed migrations prevent service startup
- [ ] Migration time tracked and logged
- [ ] Can be disabled for faster development
- [ ] Concurrent service starts handled safely

### Story 4: Migration Tracking
**As a** platform engineer  
**I want to** track which migrations have been applied  
**So that** I can ensure database consistency across environments  

**Acceptance Criteria:**
- [ ] Migration tracking table in ImmuDB
- [ ] Records migration number, name, checksum
- [ ] Tracks execution timestamp and duration
- [ ] Prevents duplicate migration execution
- [ ] Checksum validation detects changes
- [ ] Query interface for migration history

### Story 5: Migration Development Workflow
**As a** developer  
**I want to** easily create and test new migrations  
**So that** I can iterate quickly on database changes  

**Acceptance Criteria:**
- [ ] `make migration-ledger-new NAME=description` creates template for ledger
- [ ] `make migration-treasury-new NAME=description` creates template for treasury
- [ ] Template includes standard headers and structure
- [ ] Local testing with dry-run mode
- [ ] Validation catches common errors
- [ ] Documentation for migration best practices
- [ ] Examples of common migration patterns

## Technical Design

### Architecture Overview
The migration system uses a simple file-based approach with numbered SQL files executed in sequence. A migration tracking table in ImmuDB records applied migrations, preventing duplicates and ensuring consistency. The system supports both Makefile-driven execution (preferred for production) and on-boot execution (useful for development/containers).

### Migration File Structure

#### File Naming Convention
```
services/treasury-services/ledger-service/
└── migrations/
    ├── 001_initial_schema.sql
    ├── 002_add_indexes.sql
    ├── 003_create_accounts_table.sql
    ├── 004_add_audit_fields.sql
    └── README.md

services/treasury-services/treasury-service/
└── migrations/
    ├── 001_initial_schema.sql
    ├── 002_create_treasury_tables.sql
    └── README.md
```

#### Migration File Template
```sql
-- Migration: 001_initial_schema
-- Author: [Author Name]
-- Date: 2025-01-11
-- Description: Creates initial schema for ledger service
-- Spec: docs/specs/002-database-migrations.md

-- Check if migration should run (handled by migration system)
-- This is for documentation only

-- Create initial tables
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(type);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);

-- Insert initial data if needed
-- INSERT INTO system_config (key, value) VALUES ('version', '1.0.0');
```

### Migration Tracking Schema

```sql
-- Migration tracking table (created automatically per service)
-- Each service has its own migration tracking table
CREATE TABLE IF NOT EXISTS ledger_schema_migrations (
    version INTEGER PRIMARY KEY,           -- Migration number (1, 2, 3...)
    name VARCHAR(255) NOT NULL,           -- Migration name from filename
    service VARCHAR(100) NOT NULL,        -- Service name (ledger, treasury, etc.)
    checksum VARCHAR(64) NOT NULL,        -- SHA256 of migration content
    executed_at TIMESTAMP DEFAULT NOW(),  -- When migration was applied
    execution_time_ms INTEGER,            -- How long migration took
    applied_by VARCHAR(100),              -- User/service that ran migration
    success BOOLEAN DEFAULT TRUE,         -- Migration success status
    error_message TEXT                    -- Error details if failed
);

-- Index for quick lookup
CREATE INDEX IF NOT EXISTS idx_ledger_migrations_executed ON ledger_schema_migrations(executed_at);
CREATE INDEX IF NOT EXISTS idx_ledger_migrations_service ON ledger_schema_migrations(service);
```

### Migration Manager Implementation

```go
// MigrationManager handles database migrations
// Spec: docs/specs/002-database-migrations.md
type MigrationManager struct {
    client     client.ImmuClient
    config     *MigrationConfig
    migrations []Migration
    mu         sync.Mutex
}

// Migration represents a single migration file
type Migration struct {
    Version     int
    Name        string
    Filename    string
    Content     string
    Checksum    string
}

// MigrationConfig configures migration behavior
type MigrationConfig struct {
    MigrationsPath string        // Path to migrations directory
    RunOnBoot      bool          // Execute on service startup
    DryRun         bool          // Show what would be executed
    Timeout        time.Duration // Max time per migration
    TableName      string        // Migration tracking table
}

// Run executes pending migrations
// Spec: docs/specs/002-database-migrations.md#story-2-pre-boot-migration-execution
func (m *MigrationManager) Run(ctx context.Context) error {
    // 1. Load migration files from disk
    // 2. Check migration tracking table
    // 3. Identify pending migrations
    // 4. Execute in order with transactions
    // 5. Update tracking table
    // 6. Report results
}

// Status returns migration status
// Spec: docs/specs/002-database-migrations.md#story-4-migration-tracking
func (m *MigrationManager) Status(ctx context.Context) (*MigrationStatus, error) {
    // Return applied and pending migrations
}

// Validate checks migration files for errors
func (m *MigrationManager) Validate() error {
    // Check numbering sequence
    // Verify SQL syntax
    // Check for dangerous operations
}
```

### Makefile Integration

```makefile
# Migration commands for monorepo
# Spec: services/treasury-services/ledger-service/docs/specs/002-database-migrations.md

# Ledger Service Migrations
migrate-ledger:
	@echo "Running ledger service database migrations..."
	@go run ./services/treasury-services/ledger-service/cmd/migrate up

migrate-ledger-status:
	@echo "Checking ledger service migration status..."
	@go run ./services/treasury-services/ledger-service/cmd/migrate status

migrate-ledger-validate:
	@echo "Validating ledger service migration files..."
	@go run ./services/treasury-services/ledger-service/cmd/migrate validate

migrate-ledger-dry-run:
	@echo "Ledger service migration dry run..."
	@go run ./services/treasury-services/ledger-service/cmd/migrate up --dry-run

migration-ledger-new:
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migration-ledger-new NAME=description"; \
		exit 1; \
	fi
	@go run ./services/treasury-services/ledger-service/cmd/migrate create $(NAME)

# Treasury Service Migrations (future)
migrate-treasury:
	@echo "Running treasury service database migrations..."
	@go run ./services/treasury-services/treasury-service/cmd/migrate up

migrate-treasury-status:
	@echo "Checking treasury service migration status..."
	@go run ./services/treasury-services/treasury-service/cmd/migrate status

# Aggregate migration commands
migrate-all:
	@echo "Running all service migrations..."
	@make migrate-ledger
	@make migrate-treasury || true  # Continue even if treasury doesn't have migrations yet

migrate-all-status:
	@echo "===================================="
	@echo "   ALL SERVICE MIGRATION STATUS    "
	@echo "===================================="
	@make migrate-ledger-status || true
	@echo ""
	@make migrate-treasury-status || true

# Updated service targets with migrations
ledger-service: migrate-ledger
	@echo "Starting ledger service..."
	# ... existing commands ...

# Optional: separate target without migrations for development
ledger-service-fast:
	@echo "Starting ledger service (no migrations)..."
	# ... existing commands without migrate-ledger dependency ...

treasury-service: migrate-treasury
	@echo "Starting treasury service..."
	# ... existing commands ...

# Development target runs all migrations
dev: infrastructure-up migrate-all all-services
```

### On-Boot Integration

```go
// In main.go
func main() {
    // Load configuration
    config := LoadConfig()
    
    // Initialize ImmuDB connection
    immudbManager := NewImmuDBManager(config.ImmuDB)
    if err := immudbManager.Connect(ctx); err != nil {
        log.Fatal("Failed to connect to ImmuDB:", err)
    }
    
    // Run migrations if configured
    // Spec: docs/specs/002-database-migrations.md#story-3-on-boot-migration-execution
    if config.Migration.RunOnBoot {
        migrationManager := NewMigrationManager(immudbManager.Client(), config.Migration)
        
        log.Info("Running database migrations...")
        start := time.Now()
        
        if err := migrationManager.Run(ctx); err != nil {
            log.Fatal("Migration failed:", err)
        }
        
        log.Info("Migrations completed in", time.Since(start))
    }
    
    // Start gRPC server
    // ... existing server code ...
}
```

### Environment Configuration

```bash
# Ledger Service Migration Configuration
LEDGER_MIGRATION_RUN_ON_BOOT=false           # Run migrations on service start
LEDGER_MIGRATION_PATH=./migrations           # Path to migration files (relative to service)
LEDGER_MIGRATION_TIMEOUT=30                  # Timeout per migration (seconds)
LEDGER_MIGRATION_TABLE=ledger_schema_migrations  # Migration tracking table name
LEDGER_MIGRATION_AUTO_CREATE_TABLE=true      # Auto-create tracking table
LEDGER_MIGRATION_FAIL_ON_ERROR=true          # Stop on first error
LEDGER_MIGRATION_LOG_LEVEL=info              # Migration logging level

# Treasury Service Migration Configuration (when needed)
TREASURY_MIGRATION_RUN_ON_BOOT=false
TREASURY_MIGRATION_PATH=./migrations
TREASURY_MIGRATION_TABLE=treasury_schema_migrations
```

### Migration Patterns for ImmuDB

Since ImmuDB is append-only and immutable, migrations follow specific patterns:

#### Creating Tables
```sql
-- Safe to run multiple times
CREATE TABLE IF NOT EXISTS ledger_entries (
    id VARCHAR(36) PRIMARY KEY,
    account_id VARCHAR(36) NOT NULL,
    amount DECIMAL(20,4) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

#### Adding Indexes
```sql
-- Indexes can be added to existing tables
CREATE INDEX IF NOT EXISTS idx_entries_account 
ON ledger_entries(account_id);
```

#### Adding Columns (Not Supported)
```sql
-- ImmuDB doesn't support ALTER TABLE
-- Must create new table and migrate data
CREATE TABLE ledger_entries_v2 AS 
SELECT *, NULL as new_column FROM ledger_entries;
```

#### Data Migrations
```sql
-- Insert reference data
INSERT INTO account_types (code, name) 
VALUES ('ASSET', 'Asset Account'),
       ('LIABILITY', 'Liability Account');
```

### Error Handling

| Error Type | Handling Strategy | Recovery Action |
|------------|------------------|-----------------|
| File Not Found | Log and skip | Continue with next migration |
| SQL Syntax Error | Fail fast | Stop migration, log details |
| Checksum Mismatch | Fail fast | Prevent execution, alert |
| Connection Lost | Retry with backoff | Reconnect and resume |
| Timeout | Mark as failed | Manual intervention required |
| Duplicate Execution | Skip silently | Already applied, continue |

### Performance Requirements

- Migration file loading: < 100ms
- Status check: < 50ms
- Single migration execution: < 30s (configurable)
- Checksum calculation: < 10ms per file
- Total startup delay (10 migrations): < 5s

## Implementation Plan

### Phase 1: Core Infrastructure (Day 1-2)
- [ ] Create migrations directory structure
- [ ] Implement Migration and MigrationManager types
- [ ] Add migration tracking table creation
- [ ] Implement file loading and parsing
- [ ] Add checksum calculation

### Phase 2: Execution Engine (Day 2-3)
- [ ] Implement migration execution logic
- [ ] Add transaction support
- [ ] Implement tracking table updates
- [ ] Add dry-run capability
- [ ] Handle concurrent execution

### Phase 3: Makefile Integration (Day 3-4)
- [ ] Create migrate command-line tool
- [ ] Add Makefile targets
- [ ] Implement status reporting
- [ ] Add validation command
- [ ] Create migration template generator

### Phase 4: Service Integration (Day 4-5)
- [ ] Add on-boot execution option
- [ ] Integrate with configuration system
- [ ] Add health check integration
- [ ] Implement logging and metrics
- [ ] Test with real migrations

## Dependencies

### Service Dependencies
- ImmuDB connection (spec 001-immudb-connection.md)
- Configuration management system
- Health check system for status reporting

### Go Dependencies
```go
github.com/codenotary/immudb/pkg/client  // ImmuDB client
github.com/golang-migrate/migrate/v4     // Optional: migration library
crypto/sha256                            // Checksum calculation
```

### Infrastructure Dependencies
- ImmuDB instance must be running
- Migrations directory must be accessible
- Write permissions for tracking table

## Security Considerations

### Migration File Security
- Migration files should be version controlled
- Review all migrations before deployment
- No sensitive data in migration files
- Use environment variables for configuration

### Execution Security
- Migrations run with service credentials
- Limit permissions to necessary operations
- Audit log all migration executions
- Validate SQL to prevent injection

### Access Control
- Migration files read-only in production
- Only CI/CD can modify migration files
- Manual migration requires approval
- Track who executes migrations

## Testing Strategy

### Unit Tests
- [ ] Migration file parsing
- [ ] Checksum calculation
- [ ] Version ordering logic
- [ ] Tracking table operations
- [ ] Error handling paths

### Integration Tests
- [ ] Full migration execution
- [ ] Concurrent migration handling
- [ ] Recovery from partial failure
- [ ] Dry-run accuracy
- [ ] ImmuDB-specific operations

### End-to-End Tests
- [ ] Fresh database migration
- [ ] Incremental migrations
- [ ] Migration with service startup
- [ ] Performance under load
- [ ] Recovery scenarios

## Monitoring & Observability

### Metrics
- Migration execution count
- Migration execution time (P50, P95, P99)
- Failed migration count
- Pending migration count
- Last successful migration timestamp
- Checksum validation failures

### Logs
- Migration start/complete
- Individual migration timing
- Validation errors
- Execution errors
- Checksum mismatches
- Concurrent execution attempts

### Alerts
- Migration failure
- Migration taking > 60 seconds
- Checksum mismatch detected
- Pending migrations > threshold
- Tracking table write failure

## Documentation Updates

Upon implementation, update:
- [ ] CLAUDE.md with migration commands
- [ ] Ledger service README with migration setup
- [ ] Create migrations/README.md with patterns
- [ ] Add troubleshooting guide
- [ ] Document migration best practices

## Migration Best Practices

### Do's
- Keep migrations small and focused
- Use IF NOT EXISTS for idempotency
- Include descriptive comments
- Test migrations locally first
- Version control all migrations
- Use transactions where possible

### Don'ts
- Don't modify existing migration files
- Don't skip migration numbers
- Don't use DROP statements (ImmuDB is immutable)
- Don't put sensitive data in migrations
- Don't rely on external data sources
- Don't use non-deterministic functions

## Open Questions

1. Should we support migration bundling for faster execution?
2. Do we need migration groups or tags for feature sets?
3. Should failed migrations block service startup in production?
4. How should we handle migration timeouts in production?
5. Do we need a migration rollback strategy despite ImmuDB immutability?

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-11 | Use numbered files vs timestamps | Simpler, clearer ordering | Team |
| 2025-01-11 | No down migrations | ImmuDB is append-only | Team |
| 2025-01-11 | Makefile execution preferred | Explicit, visible, controlled | Team |
| 2025-01-11 | Support on-boot as option | Useful for containers | Team |
| 2025-01-11 | Single migration table | Simplicity over complexity | Team |

## References

- [ImmuDB Connection Spec](./001-immudb-connection.md)
- [ImmuDB SQL Documentation](https://docs.immudb.io/master/develop/sql/)
- [Migration Best Practices](https://www.liquibase.org/get-started/best-practices)
- [Go-Migrate Library](https://github.com/golang-migrate/migrate)

## Appendix

### Example Migration Files

#### 001_initial_schema.sql
```sql
-- Migration: 001_initial_schema
-- Author: Platform Team
-- Date: 2025-01-11
-- Description: Initial ledger service schema

CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(36) PRIMARY KEY,
    code VARCHAR(20) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) DEFAULT 'ACTIVE',
    metadata JSON,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100)
);

CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(type);
CREATE INDEX IF NOT EXISTS idx_accounts_status ON accounts(status);
CREATE INDEX IF NOT EXISTS idx_accounts_currency ON accounts(currency);
```

#### 002_create_transactions.sql
```sql
-- Migration: 002_create_transactions  
-- Author: Platform Team
-- Date: 2025-01-11
-- Description: Create transactions table

CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(36) PRIMARY KEY,
    account_id VARCHAR(36) NOT NULL,
    amount DECIMAL(20,4) NOT NULL,
    balance DECIMAL(20,4) NOT NULL,
    type VARCHAR(20) NOT NULL,
    reference VARCHAR(100),
    description TEXT,
    metadata JSON,
    created_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100)
);

CREATE INDEX IF NOT EXISTS idx_transactions_account ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_transactions_created ON transactions(created_at);
```

### Migration Status Output Example
```
$ make migrate-ledger-status

Migration Status for ledger-service
====================================
Database: ledgerdb @ localhost:3322
Tracking Table: ledger_schema_migrations
Migration Path: services/treasury-services/ledger-service/migrations

Applied Migrations (3):
  ✓ 001_initial_schema.sql       (2025-01-11 10:00:00, 120ms)
  ✓ 002_create_transactions.sql  (2025-01-11 10:00:01, 89ms)
  ✓ 003_add_audit_fields.sql     (2025-01-11 10:00:02, 45ms)

Pending Migrations (2):
  • 004_create_journal_entries.sql
  • 005_add_account_constraints.sql

Summary:
  Service:  ledger-service
  Applied:  3
  Pending:  2
  Total:    5
  Last Run: 2025-01-11 10:00:02
```

```
$ make migrate-all-status

====================================
   ALL SERVICE MIGRATION STATUS    
====================================

LEDGER SERVICE:
---------------
Database: ledgerdb @ localhost:3322
  Applied: 3
  Pending: 2
  Last Run: 2025-01-11 10:00:02

TREASURY SERVICE:
-----------------
Database: treasurydb @ localhost:3322
  Applied: 2
  Pending: 0
  Last Run: 2025-01-11 09:45:00
```

### CLI Tool Help Output
```
$ go run ./services/treasury-services/ledger-service/cmd/migrate --help

ledger-service database migration tool

Usage:
  migrate [command]

Available Commands:
  up          Run pending migrations
  status      Show migration status
  validate    Validate migration files
  create      Create new migration file
  version     Show migration tool version

Flags:
  --config string     Config file path (default "./config.yaml")
  --dry-run          Show what would be executed
  --migrations string Migration files path (default "./migrations")
  --service string    Service name (default "ledger")
  --timeout duration  Migration timeout (default 30s)
  --verbose          Enable verbose logging

Use "migrate [command] --help" for more information about a command.

Examples:
  # From repo root
  go run ./services/treasury-services/ledger-service/cmd/migrate up
  go run ./services/treasury-services/ledger-service/cmd/migrate status
  
  # Using make commands (preferred)
  make migrate-ledger
  make migrate-ledger-status
  make migration-ledger-new NAME=add_indexes
```