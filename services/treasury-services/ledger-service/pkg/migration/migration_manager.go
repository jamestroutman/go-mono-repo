package migration

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/codenotary/immudb/pkg/client"
)

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
	Version  int
	Name     string
	Filename string
	Content  string
	Checksum string
}

// MigrationConfig configures migration behavior
type MigrationConfig struct {
	MigrationsPath string        // Path to migrations directory
	RunOnBoot      bool          // Execute on service startup
	DryRun         bool          // Show what would be executed
	Timeout        time.Duration // Max time per migration
	TableName      string        // Migration tracking table
	ServiceName    string        // Service name (ledger, treasury, etc.)
}

// MigrationStatus represents the current migration state
type MigrationStatus struct {
	Applied []AppliedMigration
	Pending []Migration
	Total   int
	LastRun *time.Time
}

// AppliedMigration represents a migration that has been applied
type AppliedMigration struct {
	Version       int
	Name          string
	Checksum      string
	ExecutedAt    time.Time
	ExecutionTime int64 // milliseconds
	AppliedBy     string
	Success       bool
	ErrorMessage  string
}

// NewMigrationManager creates a new migration manager
// Spec: docs/specs/002-database-migrations.md
func NewMigrationManager(client client.ImmuClient, config *MigrationConfig) *MigrationManager {
	if config.TableName == "" {
		config.TableName = "ledger_schema_migrations"
	}
	if config.ServiceName == "" {
		config.ServiceName = "ledger"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	return &MigrationManager{
		client: client,
		config: config,
	}
}

// GetConfig returns the migration configuration
func (m *MigrationManager) GetConfig() *MigrationConfig {
	return m.config
}

// Run executes pending migrations
// Spec: docs/specs/002-database-migrations.md#story-2-pre-boot-migration-execution
func (m *MigrationManager) Run(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 1. Ensure migration tracking table exists
	if err := m.ensureMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}
	
	// 2. Load migration files from disk
	if err := m.loadMigrations(); err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}
	
	// 3. Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	// 4. Identify pending migrations
	pending := m.getPendingMigrations(applied)
	
	if len(pending) == 0 {
		log.Println("No pending migrations")
		return nil
	}
	
	log.Printf("Found %d pending migration(s)", len(pending))
	
	// 5. Execute pending migrations
	for _, migration := range pending {
		if m.config.DryRun {
			log.Printf("[DRY RUN] Would execute migration %03d_%s", migration.Version, migration.Name)
			continue
		}
		
		log.Printf("Executing migration %03d_%s...", migration.Version, migration.Name)
		start := time.Now()
		
		// Execute with timeout
		migCtx, cancel := context.WithTimeout(ctx, m.config.Timeout)
		err := m.executeMigration(migCtx, migration)
		cancel()
		
		executionTime := time.Since(start).Milliseconds()
		
		// Record the migration
		recordErr := m.recordMigration(ctx, migration, executionTime, err)
		if recordErr != nil {
			log.Printf("Failed to record migration: %v", recordErr)
		}
		
		if err != nil {
			return fmt.Errorf("migration %03d_%s failed: %w", migration.Version, migration.Name, err)
		}
		
		log.Printf("Migration %03d_%s completed in %dms", migration.Version, migration.Name, executionTime)
	}
	
	return nil
}

// Status returns migration status
// Spec: docs/specs/002-database-migrations.md#story-4-migration-tracking  
func (m *MigrationManager) Status(ctx context.Context) (*MigrationStatus, error) {
	// Ensure table exists
	if err := m.ensureMigrationTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to create migration table: %w", err)
	}
	
	// Load migrations
	if err := m.loadMigrations(); err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}
	
	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	
	// Get pending migrations
	pending := m.getPendingMigrations(applied)
	
	// Find last run time
	var lastRun *time.Time
	if len(applied) > 0 {
		lastRun = &applied[len(applied)-1].ExecutedAt
	}
	
	return &MigrationStatus{
		Applied: applied,
		Pending: pending,
		Total:   len(m.migrations),
		LastRun: lastRun,
	}, nil
}

// Validate checks migration files for errors
func (m *MigrationManager) Validate() error {
	if err := m.loadMigrations(); err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}
	
	// Check for gaps in numbering
	for i, migration := range m.migrations {
		expectedVersion := i + 1
		if migration.Version != expectedVersion {
			return fmt.Errorf("migration numbering gap: expected %03d, got %03d in %s", 
				expectedVersion, migration.Version, migration.Filename)
		}
	}
	
	// Check for duplicate versions
	versions := make(map[int]string)
	for _, migration := range m.migrations {
		if existing, ok := versions[migration.Version]; ok {
			return fmt.Errorf("duplicate migration version %03d in files: %s and %s",
				migration.Version, existing, migration.Filename)
		}
		versions[migration.Version] = migration.Filename
	}
	
	log.Printf("Validated %d migration file(s)", len(m.migrations))
	return nil
}

// ensureMigrationTable creates the migration tracking table if it doesn't exist
func (m *MigrationManager) ensureMigrationTable(ctx context.Context) error {
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version INTEGER,
			name VARCHAR[255],
			service VARCHAR[100],
			checksum VARCHAR[64],
			executed_at TIMESTAMP,
			execution_time_ms INTEGER,
			applied_by VARCHAR[100],
			success BOOLEAN,
			error_message VARCHAR,
			PRIMARY KEY (version)
		)
	`, m.config.TableName)
	
	_, err := m.client.SQLExec(ctx, createTableSQL, nil)
	if err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}
	
	// Create indexes using ImmuDB syntax (no index names)
	indexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS ON %s(executed_at)
	`, m.config.TableName)
	
	_, err = m.client.SQLExec(ctx, indexSQL, nil)
	if err != nil {
		// Index creation failure is not critical
		log.Printf("Warning: failed to create index: %v", err)
	}
	
	return nil
}

// loadMigrations loads migration files from disk
func (m *MigrationManager) loadMigrations() error {
	pattern := filepath.Join(m.config.MigrationsPath, "*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list migration files: %w", err)
	}
	
	m.migrations = []Migration{}
	
	// Regular expression to parse migration filenames
	re := regexp.MustCompile(`^(\d{3})_(.+)\.sql$`)
	
	for _, file := range files {
		filename := filepath.Base(file)
		matches := re.FindStringSubmatch(filename)
		if len(matches) != 3 {
			log.Printf("Skipping invalid migration filename: %s", filename)
			continue
		}
		
		version, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Printf("Skipping migration with invalid version: %s", filename)
			continue
		}
		
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filename, err)
		}
		
		checksum := calculateChecksum(string(content))
		
		m.migrations = append(m.migrations, Migration{
			Version:  version,
			Name:     matches[2],
			Filename: filename,
			Content:  string(content),
			Checksum: checksum,
		})
	}
	
	// Sort migrations by version
	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})
	
	return nil
}

// getAppliedMigrations retrieves migrations that have been applied
func (m *MigrationManager) getAppliedMigrations(ctx context.Context) ([]AppliedMigration, error) {
	query := fmt.Sprintf(`
		SELECT version, name, checksum, executed_at, execution_time_ms, 
		       applied_by, success, error_message
		FROM %s
		WHERE service = @service AND success = true
		ORDER BY version
	`, m.config.TableName)
	
	params := map[string]interface{}{
		"service": m.config.ServiceName,
	}
	
	result, err := m.client.SQLQuery(ctx, query, params, true)
	if err != nil {
		// Table might not exist yet
		if strings.Contains(err.Error(), "does not exist") {
			return []AppliedMigration{}, nil
		}
		return nil, fmt.Errorf("failed to query applied migrations: %w", err)
	}
	
	var applied []AppliedMigration
	for _, row := range result.Rows {
		migration := AppliedMigration{
			Version:       int(row.Values[0].GetN()),
			Name:          string(row.Values[1].GetS()),
			Checksum:      string(row.Values[2].GetS()),
			ExecutedAt:    time.UnixMicro(row.Values[3].GetTs()),
			ExecutionTime: row.Values[4].GetN(),
			AppliedBy:     string(row.Values[5].GetS()),
			Success:       row.Values[6].GetB(),
		}
		if len(row.Values) > 7 && row.Values[7] != nil {
			migration.ErrorMessage = string(row.Values[7].GetS())
		}
		applied = append(applied, migration)
	}
	
	return applied, nil
}

// getPendingMigrations identifies migrations that haven't been applied
func (m *MigrationManager) getPendingMigrations(applied []AppliedMigration) []Migration {
	appliedMap := make(map[int]string)
	for _, a := range applied {
		appliedMap[a.Version] = a.Checksum
	}
	
	var pending []Migration
	for _, migration := range m.migrations {
		if checksum, ok := appliedMap[migration.Version]; ok {
			// Check if checksum matches
			if checksum != migration.Checksum {
				log.Printf("WARNING: Migration %03d_%s has been modified since it was applied", 
					migration.Version, migration.Name)
			}
			continue
		}
		pending = append(pending, migration)
	}
	
	return pending
}

// executeMigration runs a single migration
func (m *MigrationManager) executeMigration(ctx context.Context, migration Migration) error {
	// Split the migration content into individual statements
	statements := splitSQLStatements(migration.Content)
	
	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}
		
		_, err := m.client.SQLExec(ctx, stmt, nil)
		if err != nil {
			return fmt.Errorf("failed to execute statement %d: %w", i+1, err)
		}
	}
	
	return nil
}

// recordMigration records a migration execution in the tracking table
func (m *MigrationManager) recordMigration(ctx context.Context, migration Migration, executionTime int64, migrationErr error) error {
	success := migrationErr == nil
	errorMsg := ""
	if migrationErr != nil {
		errorMsg = migrationErr.Error()
	}
	
	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (version, name, service, checksum, executed_at, 
		                execution_time_ms, applied_by, success, error_message)
		VALUES (@version, @name, @service, @checksum, NOW(), 
		        @execution_time, @applied_by, @success, @error_message)
	`, m.config.TableName)
	
	params := map[string]interface{}{
		"version":        migration.Version,
		"name":           migration.Name,
		"service":        m.config.ServiceName,
		"checksum":       migration.Checksum,
		"execution_time": executionTime,
		"applied_by":     "ledger-service",
		"success":        success,
		"error_message":  errorMsg,
	}
	
	_, err := m.client.SQLExec(ctx, insertSQL, params)
	return err
}

// calculateChecksum calculates SHA256 checksum of content
func calculateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// splitSQLStatements splits SQL content into individual statements
func splitSQLStatements(content string) []string {
	// Simple statement splitter - splits on semicolons not within quotes
	var statements []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)
	
	for _, r := range content {
		if !inQuote {
			if r == '\'' || r == '"' {
				inQuote = true
				quoteChar = r
			} else if r == ';' {
				statements = append(statements, current.String())
				current.Reset()
				continue
			}
		} else if r == quoteChar {
			inQuote = false
		}
		current.WriteRune(r)
	}
	
	if current.Len() > 0 {
		statements = append(statements, current.String())
	}
	
	return statements
}

// CreateMigration creates a new migration file with the next available number
// Spec: docs/specs/002-database-migrations.md#story-5-migration-development-workflow
func (m *MigrationManager) CreateMigration(name string) error {
	// Load existing migrations to find next number
	if err := m.loadMigrations(); err != nil {
		return fmt.Errorf("failed to load existing migrations: %w", err)
	}
	
	// Find next version number
	nextVersion := 1
	if len(m.migrations) > 0 {
		nextVersion = m.migrations[len(m.migrations)-1].Version + 1
	}
	
	// Create filename
	filename := fmt.Sprintf("%03d_%s.sql", nextVersion, name)
	filepath := filepath.Join(m.config.MigrationsPath, filename)
	
	// Create template content
	template := fmt.Sprintf(`-- Migration: %03d_%s
-- Author: [Author Name]
-- Date: %s
-- Description: [Description]
-- Spec: docs/specs/002-database-migrations.md

-- Add your migration SQL here
-- Remember: ImmuDB is append-only, no UPDATE or DELETE operations

`, nextVersion, name, time.Now().Format("2006-01-02"))
	
	// Write file
	if err := ioutil.WriteFile(filepath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}
	
	log.Printf("Created migration file: %s", filename)
	return nil
}