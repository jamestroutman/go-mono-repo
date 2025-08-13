package migration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/codenotary/immudb/pkg/client"
	pb "example.com/go-mono-repo/proto/ledger"
)

// MigrationChecker checks migration status for health reporting
// Spec: docs/specs/002-database-migrations.md
type MigrationChecker struct {
	client  client.ImmuClient
	config  *MigrationConfig
	manager *MigrationManager
}

// NewMigrationChecker creates a new migration health checker
func NewMigrationChecker(client client.ImmuClient, config *MigrationConfig) *MigrationChecker {
	return &MigrationChecker{
		client:  client,
		config:  config,
		manager: NewMigrationManager(client, config),
	}
}

// Check implements DependencyChecker for migration status
// Spec: docs/specs/002-database-migrations.md#story-4-migration-tracking
func (m *MigrationChecker) Check(ctx context.Context) *pb.DependencyHealth {
	startTime := time.Now()
	
	dep := &pb.DependencyHealth{
		Name:       "database-migrations",
		Type:       pb.DependencyType_DATABASE,
		IsCritical: false, // Migrations are not critical for service operation
		Config: &pb.DependencyConfig{
			Hostname:     m.config.TableName,
			Protocol:     "immudb-sql",
			DatabaseName: "migrations",
		},
	}
	
	// Get migration status
	status, err := m.manager.Status(ctx)
	if err != nil {
		dep.Status = pb.ServiceStatus_DEGRADED
		dep.Message = fmt.Sprintf("Failed to check migration status: %v", err)
		dep.Error = err.Error()
	} else {
		// Build detailed message with migration info
		var details []string
		
		if status.LastRun != nil {
			dep.LastSuccess = status.LastRun.Format(time.RFC3339)
		}
		
		// List pending migrations (first 3)
		if len(status.Pending) > 0 {
			pendingList := ""
			for i, migration := range status.Pending {
				if i >= 3 {
					pendingList += fmt.Sprintf(", ... and %d more", len(status.Pending)-3)
					break
				}
				if i > 0 {
					pendingList += ", "
				}
				pendingList += fmt.Sprintf("%03d_%s", migration.Version, migration.Name)
			}
			details = append(details, fmt.Sprintf("Pending: %s", pendingList))
		}
		
		// List recent applied migrations (last 2)
		if len(status.Applied) > 0 {
			appliedList := ""
			start := len(status.Applied) - 2
			if start < 0 {
				start = 0
			}
			for i := start; i < len(status.Applied); i++ {
				if i > start {
					appliedList += ", "
				}
				migration := status.Applied[i]
				appliedList += fmt.Sprintf("%03d_%s", migration.Version, migration.Name)
			}
			details = append(details, fmt.Sprintf("Recent: %s", appliedList))
		}
		
		// Determine health status based on migrations
		baseMessage := fmt.Sprintf("Applied: %d, Pending: %d, Total: %d", 
			len(status.Applied), len(status.Pending), status.Total)
		
		if len(status.Pending) == 0 {
			dep.Status = pb.ServiceStatus_HEALTHY
			dep.Message = fmt.Sprintf("All migrations applied. %s", baseMessage)
		} else if m.config.RunOnBoot {
			// If migrations run on boot, pending migrations are OK
			dep.Status = pb.ServiceStatus_HEALTHY
			dep.Message = fmt.Sprintf("Auto-migration enabled. %s", baseMessage)
		} else {
			// Pending migrations indicate a degraded state
			dep.Status = pb.ServiceStatus_DEGRADED
			dep.Message = fmt.Sprintf("Manual migration required. %s", baseMessage)
		}
		
		// Add details if available
		if len(details) > 0 {
			dep.Message = fmt.Sprintf("%s | %s", dep.Message, strings.Join(details, " | "))
		}
	}
	
	dep.ResponseTimeMs = time.Since(startTime).Milliseconds()
	dep.LastCheck = time.Now().Format(time.RFC3339)
	
	return dep
}

// RunPendingMigrations runs any pending migrations
// This is called on boot if configured
// Spec: docs/specs/002-database-migrations.md#story-3-on-boot-migration-execution
func (m *MigrationChecker) RunPendingMigrations(ctx context.Context) error {
	if !m.config.RunOnBoot {
		return nil
	}
	
	return m.manager.Run(ctx)
}

// GetMigrationSummary returns a simple summary for logging
func (m *MigrationChecker) GetMigrationSummary(ctx context.Context) string {
	status, err := m.manager.Status(ctx)
	if err != nil {
		return fmt.Sprintf("Migration status unknown: %v", err)
	}
	
	if len(status.Pending) == 0 {
		return fmt.Sprintf("All %d migrations applied", len(status.Applied))
	}
	
	return fmt.Sprintf("%d migrations applied, %d pending", 
		len(status.Applied), len(status.Pending))
}