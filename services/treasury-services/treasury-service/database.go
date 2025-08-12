package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	pb "example.com/go-mono-repo/proto/treasury"
)

// DatabaseManager manages database connections and health
// Spec: docs/specs/001-database-connection.md
type DatabaseManager struct {
	db               *sql.DB
	config           *DatabaseConfig
	migrationManager *MigrationManager
	mu               sync.RWMutex

	// Connection metrics
	connectTime     time.Time
	lastHealthCheck time.Time
	isHealthy       bool
	errorCount      int64
}

// NewDatabaseManager creates a new database manager
// Spec: docs/specs/001-database-connection.md
func NewDatabaseManager(config *DatabaseConfig) *DatabaseManager {
	return &DatabaseManager{
		config: config,
	}
}

// Connect establishes database connection with retry logic
// Spec: docs/specs/001-database-connection.md#story-4-graceful-degradation
func (dm *DatabaseManager) Connect(ctx context.Context) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Close existing connection if any
	if dm.db != nil {
		dm.db.Close()
	}

	// Open database connection
	db, err := sql.Open("pgx", dm.config.GetConnectionString())
	if err != nil {
		dm.isHealthy = false
		dm.errorCount++
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	// Spec: docs/specs/001-database-connection.md#story-2-connection-pool-management
	db.SetMaxOpenConns(dm.config.MaxConnections)
	db.SetMaxIdleConns(dm.config.MaxIdleConnections)
	db.SetConnMaxLifetime(dm.config.ConnectionMaxLifetime)
	db.SetConnMaxIdleTime(dm.config.ConnectionMaxIdleTime)

	// Test the connection
	ctx, cancel := context.WithTimeout(ctx, dm.config.PingTimeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		dm.isHealthy = false
		dm.errorCount++
		return fmt.Errorf("failed to ping database: %w", err)
	}

	dm.db = db
	dm.connectTime = time.Now()
	dm.isHealthy = true
	dm.lastHealthCheck = time.Now()
	dm.errorCount = 0

	log.Printf("Successfully connected to database %s:%d/%s", 
		dm.config.Host, dm.config.Port, dm.config.Database)

	return nil
}

// ConnectWithRetry establishes database connection with exponential backoff
// Spec: docs/specs/001-database-connection.md#story-4-graceful-degradation
func (dm *DatabaseManager) ConnectWithRetry(ctx context.Context, maxRetries int) error {
	var lastErr error
	backoff := time.Second

	for i := 0; i < maxRetries; i++ {
		if err := dm.Connect(ctx); err == nil {
			return nil
		} else {
			lastErr = err
			log.Printf("Database connection attempt %d/%d failed: %v", i+1, maxRetries, err)
		}

		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				// Exponential backoff with jitter
				backoff = backoff * 2
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}
			}
		}
	}

	return fmt.Errorf("failed to connect to database after %d attempts: %w", maxRetries, lastErr)
}

// GetDB returns the database connection
func (dm *DatabaseManager) GetDB() *sql.DB {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.db
}

// IsHealthy returns the health status of the database connection
func (dm *DatabaseManager) IsHealthy() bool {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.isHealthy
}

// SetMigrationManager stores the migration manager for health checks
// Spec: docs/specs/002-database-migrations.md
func (dm *DatabaseManager) SetMigrationManager(mm *MigrationManager) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	dm.migrationManager = mm
}

// GetMigrationManager returns the migration manager
// Spec: docs/specs/002-database-migrations.md
func (dm *DatabaseManager) GetMigrationManager() *MigrationManager {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	return dm.migrationManager
}

// Close closes the database connection
func (dm *DatabaseManager) Close() error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Close migration manager if exists
	if dm.migrationManager != nil {
		if err := dm.migrationManager.Close(); err != nil {
			log.Printf("Warning: Failed to close migration manager: %v", err)
		}
		dm.migrationManager = nil
	}

	if dm.db != nil {
		err := dm.db.Close()
		dm.db = nil
		dm.isHealthy = false
		return err
	}
	return nil
}

// GetConnectionPoolStats returns current pool statistics
// Spec: docs/specs/001-database-connection.md#story-2-connection-pool-management
func (dm *DatabaseManager) GetConnectionPoolStats() *pb.ConnectionPoolInfo {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	if dm.db == nil {
		return &pb.ConnectionPoolInfo{}
	}

	stats := dm.db.Stats()
	return &pb.ConnectionPoolInfo{
		MaxConnections:    int32(stats.MaxOpenConnections),
		ActiveConnections: int32(stats.InUse),
		IdleConnections:   int32(stats.Idle),
		WaitCount:         int32(stats.WaitCount),
		WaitDurationMs:    stats.WaitDuration.Milliseconds(),
	}
}

// PostgreSQLChecker implements DependencyChecker for database
// Spec: docs/specs/001-database-connection.md#story-3-database-health-monitoring
type PostgreSQLChecker struct {
	manager *DatabaseManager
}

// NewPostgreSQLChecker creates a new PostgreSQL health checker
func NewPostgreSQLChecker(manager *DatabaseManager) *PostgreSQLChecker {
	return &PostgreSQLChecker{
		manager: manager,
	}
}

// Check implements health check for PostgreSQL
// Spec: docs/specs/001-database-connection.md#story-3-database-health-monitoring
func (p *PostgreSQLChecker) Check(ctx context.Context) *pb.DependencyHealth {
	startTime := time.Now()

	dep := &pb.DependencyHealth{
		Name:       "postgresql-primary",
		Type:       pb.DependencyType_DATABASE,
		IsCritical: true,
		Config: &pb.DependencyConfig{
			Hostname:     p.manager.config.Host,
			Port:         int32(p.manager.config.Port),
			Protocol:     "postgresql",
			DatabaseName: p.manager.config.Database,
			SchemaName:   p.manager.config.Schema,
		},
		LastCheck: time.Now().Format(time.RFC3339),
	}

	// Check if database manager is initialized
	if p.manager == nil || p.manager.GetDB() == nil {
		dep.Status = pb.ServiceStatus_UNHEALTHY
		dep.Message = "Database connection not initialized"
		dep.Error = "Database manager is nil or connection is closed"
		dep.ResponseTimeMs = time.Since(startTime).Milliseconds()
		return dep
	}

	// Perform health check with timeout
	checkCtx, cancel := context.WithTimeout(ctx, p.manager.config.PingTimeout)
	defer cancel()

	if err := p.manager.GetDB().PingContext(checkCtx); err != nil {
		p.manager.mu.Lock()
		p.manager.isHealthy = false
		p.manager.errorCount++
		p.manager.mu.Unlock()

		dep.Status = pb.ServiceStatus_UNHEALTHY
		dep.Message = "Database connection failed"
		dep.Error = err.Error()
	} else {
		p.manager.mu.Lock()
		p.manager.isHealthy = true
		p.manager.lastHealthCheck = time.Now()
		p.manager.mu.Unlock()

		dep.Status = pb.ServiceStatus_HEALTHY
		dep.Message = "Database connection healthy"
		dep.LastSuccess = time.Now().Format(time.RFC3339)
		
		// Add connection pool statistics
		dep.Config.PoolInfo = p.manager.GetConnectionPoolStats()
		
		// Add metadata
		dep.Config.Metadata = map[string]string{
			"connect_time": p.manager.connectTime.Format(time.RFC3339),
			"error_count":  fmt.Sprintf("%d", p.manager.errorCount),
		}
	}

	dep.ResponseTimeMs = time.Since(startTime).Milliseconds()
	return dep
}