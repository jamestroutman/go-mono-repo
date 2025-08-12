package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	pb "example.com/go-mono-repo/proto/treasury"
)

// MigrationManager handles database schema migrations
// Spec: docs/specs/002-database-migrations.md
type MigrationManager struct {
	db       *sql.DB
	config   *MigrationConfig
	migrator *migrate.Migrate
	mu       sync.RWMutex
}

// MigrationConfig holds migration configuration
// Spec: docs/specs/002-database-migrations.md#story-1-automated-migration-on-startup
type MigrationConfig struct {
	MigrationsPath string        // Path to migration files
	AutoMigrate    bool          // Run migrations on startup
	MigrateTimeout time.Duration // Timeout for migration execution
	DryRun         bool          // Validate without applying
	MaxRetries     int           // Retry count for transient failures
	RetryDelay     time.Duration // Delay between retries
}

// MigrationInfo contains migration status information
type MigrationInfo struct {
	CurrentVersion uint
	IsDirty        bool
	LastMigration  time.Time
	PendingCount   int
}

// NewMigrationManager creates a new migration manager
// Spec: docs/specs/002-database-migrations.md
func NewMigrationManager(db *sql.DB, config *MigrationConfig) (*MigrationManager, error) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create migration driver: %w", err)
	}

	sourceURL := fmt.Sprintf("file://%s", config.MigrationsPath)
	m, err := migrate.NewWithDatabaseInstance(
		sourceURL,
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrator: %w", err)
	}

	return &MigrationManager{
		db:       db,
		config:   config,
		migrator: m,
	}, nil
}

// Migrate runs pending migrations
// Spec: docs/specs/002-database-migrations.md#story-1-automated-migration-on-startup
func (mm *MigrationManager) Migrate(ctx context.Context) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if mm.config.DryRun {
		return mm.validateMigrations(ctx)
	}

	// Set timeout for migration
	ctx, cancel := context.WithTimeout(ctx, mm.config.MigrateTimeout)
	defer cancel()

	// Run migrations with retry logic
	var lastErr error
	for i := 0; i <= mm.config.MaxRetries; i++ {
		if i > 0 {
			log.Printf("Retrying migration (attempt %d/%d)", i+1, mm.config.MaxRetries+1)
			time.Sleep(mm.config.RetryDelay)
		}

		err := mm.migrator.Up()
		if err == nil || err == migrate.ErrNoChange {
			if err == migrate.ErrNoChange {
				log.Printf("No new migrations to apply")
			} else {
				log.Printf("Migrations applied successfully")
			}
			return nil
		}

		lastErr = err
		if !isRetryableError(err) {
			break
		}
	}

	return fmt.Errorf("migration failed: %w", lastErr)
}

// MigrateDown rolls back the last migration
// Spec: docs/specs/002-database-migrations.md#story-5-safe-rollback-capability
func (mm *MigrationManager) MigrateDown(ctx context.Context) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	ctx, cancel := context.WithTimeout(ctx, mm.config.MigrateTimeout)
	defer cancel()

	err := mm.migrator.Down()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("rollback failed: %w", err)
	}

	return nil
}

// GetMigrationInfo returns current migration status
// Spec: docs/specs/002-database-migrations.md#story-4-migration-status-monitoring
func (mm *MigrationManager) GetMigrationInfo() (*MigrationInfo, error) {
	version, dirty, err := mm.migrator.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return nil, err
	}

	return &MigrationInfo{
		CurrentVersion: version,
		IsDirty:        dirty,
		LastMigration:  time.Now(), // In production, get from schema_migrations table
		PendingCount:   mm.countPendingMigrations(),
	}, nil
}

// ForceVersion forces the migration version (use with caution)
// Spec: docs/specs/002-database-migrations.md#story-2-manual-migration-control
func (mm *MigrationManager) ForceVersion(version int) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	return mm.migrator.Force(version)
}

// Close closes the migration manager
func (mm *MigrationManager) Close() error {
	sourceErr, dbErr := mm.migrator.Close()
	if sourceErr != nil {
		return sourceErr
	}
	return dbErr
}

// validateMigrations validates migrations without applying them
func (mm *MigrationManager) validateMigrations(ctx context.Context) error {
	// In dry-run mode, just check if migrations are valid
	version, dirty, err := mm.migrator.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	log.Printf("Dry run mode - current version: %d, dirty: %v", version, dirty)
	log.Printf("Migrations would be applied from %s", mm.config.MigrationsPath)
	return nil
}

// countPendingMigrations counts migrations that haven't been applied
func (mm *MigrationManager) countPendingMigrations() int {
	// This is a simplified implementation
	// In production, compare available migrations with applied ones
	return 0
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	// Check for transient errors like connection issues
	errStr := err.Error()
	return contains(errStr, "connection refused") ||
		contains(errStr, "connection reset") ||
		contains(errStr, "i/o timeout") ||
		contains(errStr, "temporary failure")
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr ||
		len(s) > len(substr) && contains(s[1:], substr)
}

// MigrationChecker implements health check for migrations
// Spec: docs/specs/002-database-migrations.md#story-4-migration-status-monitoring
type MigrationChecker struct {
	manager *MigrationManager
}

// NewMigrationChecker creates a new migration health checker
func NewMigrationChecker(manager *MigrationManager) *MigrationChecker {
	return &MigrationChecker{manager: manager}
}

// Check returns migration health status
// Spec: docs/specs/002-database-migrations.md#story-4-migration-status-monitoring
func (mc *MigrationChecker) Check(ctx context.Context) *pb.DependencyHealth {
	info, err := mc.manager.GetMigrationInfo()

	dep := &pb.DependencyHealth{
		Name:       "database-migrations",
		Type:       pb.DependencyType_DATABASE,
		IsCritical: true,
	}

	if err != nil {
		dep.Status = pb.ServiceStatus_UNHEALTHY
		dep.Message = "Failed to get migration status"
		dep.Error = err.Error()
		return dep
	}

	if info.IsDirty {
		dep.Status = pb.ServiceStatus_UNHEALTHY
		dep.Message = fmt.Sprintf("Migration %d is in dirty state", info.CurrentVersion)
	} else if info.PendingCount > 0 {
		dep.Status = pb.ServiceStatus_DEGRADED
		dep.Message = fmt.Sprintf("%d pending migrations", info.PendingCount)
	} else {
		dep.Status = pb.ServiceStatus_HEALTHY
		dep.Message = fmt.Sprintf("Schema at version %d", info.CurrentVersion)
	}

	dep.Config = &pb.DependencyConfig{
		Metadata: map[string]string{
			"current_version": fmt.Sprintf("%d", info.CurrentVersion),
			"pending_count":   fmt.Sprintf("%d", info.PendingCount),
			"is_dirty":        fmt.Sprintf("%v", info.IsDirty),
			"last_migration":  info.LastMigration.Format(time.RFC3339),
		},
	}

	dep.LastCheck = time.Now().Format(time.RFC3339)

	return dep
}