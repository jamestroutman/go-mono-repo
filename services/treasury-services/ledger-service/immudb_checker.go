package main

import (
	"context"

	pb "example.com/go-mono-repo/proto/ledger"
)

// ImmuDBChecker implements DependencyChecker for ImmuDB
// Spec: docs/specs/001-immudb-connection.md#story-3-immudb-health-monitoring
type ImmuDBChecker struct {
	manager *ImmuDBManager
}

// NewImmuDBChecker creates a new ImmuDB health checker
func NewImmuDBChecker(manager *ImmuDBManager) *ImmuDBChecker {
	return &ImmuDBChecker{
		manager: manager,
	}
}

// Check implements health check for ImmuDB
// Spec: docs/specs/001-immudb-connection.md#story-3-immudb-health-monitoring
func (i *ImmuDBChecker) Check(ctx context.Context) *pb.DependencyHealth {
	if i.manager == nil {
		return &pb.DependencyHealth{
			Name:       "immudb-primary",
			Type:       pb.DependencyType_DATABASE,
			Status:     pb.ServiceStatus_UNHEALTHY,
			IsCritical: true,
			Message:    "ImmuDB manager not initialized",
			Error:      "Manager is nil",
		}
	}

	// Delegate to the manager's CheckHealth method
	dep, _ := i.manager.CheckHealth(ctx)
	return dep
}

// Name returns the name of this dependency checker
func (i *ImmuDBChecker) Name() string {
	return "immudb-primary"
}