package main

import (
	"context"
	"sync"
	"time"

	pb "example.com/go-mono-repo/proto/ledger"
)

// HealthServer implements the Health service
// Spec: docs/specs/003-health-check-liveness.md
type HealthServer struct {
	pb.UnimplementedHealthServer
	
	// Service readiness tracking
	configLoaded bool
	grpcReady    bool
	startTime    time.Time
	
	// Mutex for thread-safe access
	mu sync.RWMutex
	
	// Dependencies (to be expanded as services are added)
	dependencies []DependencyChecker
}

// DependencyChecker interface for checking dependency health
type DependencyChecker interface {
	Check(ctx context.Context) *pb.DependencyHealth
}

// NewHealthServer creates a new health server instance
// Spec: docs/specs/003-health-check-liveness.md
func NewHealthServer(startTime time.Time) *HealthServer {
	return &HealthServer{
		startTime:    startTime,
		configLoaded: false,
		grpcReady:    false,
		dependencies: []DependencyChecker{},
	}
}

// SetConfigLoaded marks configuration as loaded
func (s *HealthServer) SetConfigLoaded(loaded bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configLoaded = loaded
}

// SetGRPCReady marks gRPC server as ready
func (s *HealthServer) SetGRPCReady(ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grpcReady = ready
}

// GetLiveness checks service readiness
// Spec: docs/specs/003-health-check-liveness.md#story-1-service-liveness-check
func (s *HealthServer) GetLiveness(ctx context.Context, req *pb.LivenessRequest) (*pb.LivenessResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Component checks
	checks := []*pb.ComponentCheck{
		{
			Name:    "config",
			Ready:   s.configLoaded,
			Message: s.getConfigMessage(),
		},
		{
			Name:    "grpc_server",
			Ready:   s.grpcReady,
			Message: s.getGRPCMessage(),
		},
		{
			Name:    "database_pool",
			Ready:   s.dbPoolReady(),
			Message: s.getDBPoolMessage(),
		},
		{
			Name:    "cache",
			Ready:   s.cacheReady(),
			Message: s.getCacheMessage(),
		},
	}
	
	// Determine overall status
	allReady := true
	for _, check := range checks {
		if !check.Ready {
			allReady = false
			break
		}
	}
	
	status := pb.ServiceStatus_HEALTHY
	message := "Service is ready"
	if !allReady {
		status = pb.ServiceStatus_UNHEALTHY
		message = "Service is not ready"
	}
	
	return &pb.LivenessResponse{
		Status:    status,
		Message:   message,
		Checks:    checks,
		CheckedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// GetHealth performs comprehensive health check
// Spec: docs/specs/003-health-check-liveness.md#story-2-dependency-health-monitoring
func (s *HealthServer) GetHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	startTime := time.Now()
	
	// Check liveness first
	livenessResp, _ := s.GetLiveness(ctx, &pb.LivenessRequest{})
	
	// Check dependencies
	dependencies := s.checkDependencies(ctx, req.DependencyFilter)
	
	// Convert liveness response to LivenessInfo
	livenessInfo := s.convertLivenessInfo(livenessResp)
	
	// Determine overall status
	overallStatus := s.calculateOverallStatus(livenessResp, dependencies)
	
	return &pb.HealthResponse{
		Status:          overallStatus,
		Message:         s.getStatusMessage(overallStatus),
		Liveness:        livenessInfo,
		Dependencies:    dependencies,
		CheckedAt:       time.Now().Format(time.RFC3339),
		CheckDurationMs: time.Since(startTime).Milliseconds(),
	}, nil
}

// Helper methods

func (s *HealthServer) getConfigMessage() string {
	if s.configLoaded {
		return "Configuration loaded successfully"
	}
	return "Configuration not loaded"
}

func (s *HealthServer) getGRPCMessage() string {
	if s.grpcReady {
		return "gRPC server ready"
	}
	return "gRPC server not ready"
}

func (s *HealthServer) dbPoolReady() bool {
	// For MVP, we're using in-memory storage, so always ready
	// Spec: docs/specs/003-health-check-liveness.md
	// This will be expanded when database is added
	return true
}

func (s *HealthServer) getDBPoolMessage() string {
	// For MVP, using in-memory storage
	return "In-memory storage ready"
}

func (s *HealthServer) cacheReady() bool {
	// For MVP, no cache implemented yet
	// This will be expanded when cache is added
	return true
}

func (s *HealthServer) getCacheMessage() string {
	// For MVP, no cache
	return "No cache configured (not required)"
}

func (s *HealthServer) checkDependencies(ctx context.Context, filter []string) []*pb.DependencyHealth {
	var dependencies []*pb.DependencyHealth
	
	// For MVP, the ledger service has no external dependencies
	// This method will be expanded as dependencies are added
	// Spec: docs/specs/003-health-check-liveness.md#story-4-dependency-configuration-visibility
	
	// Example: When a database is added, it would look like:
	// dependencies = append(dependencies, s.checkDatabase(ctx))
	
	// Example: When treasury service dependency is added:
	// if s.shouldCheckDependency("treasury-service", filter) {
	//     dependencies = append(dependencies, s.checkTreasuryService(ctx))
	// }
	
	return dependencies
}

func (s *HealthServer) convertLivenessInfo(resp *pb.LivenessResponse) *pb.LivenessInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	isAlive := resp.Status == pb.ServiceStatus_HEALTHY
	
	// Extract component statuses
	components := make([]*pb.ComponentCheck, 0, len(resp.Checks))
	for _, check := range resp.Checks {
		// Skip the standard checks that are explicitly in LivenessInfo
		if check.Name != "config" && check.Name != "grpc_server" && 
		   check.Name != "database_pool" && check.Name != "cache" {
			components = append(components, check)
		}
	}
	
	return &pb.LivenessInfo{
		IsAlive:      isAlive,
		ConfigLoaded: s.configLoaded,
		PoolsReady:   s.dbPoolReady(),
		CacheWarmed:  s.cacheReady(),
		Components:   components,
	}
}

func (s *HealthServer) calculateOverallStatus(liveness *pb.LivenessResponse, dependencies []*pb.DependencyHealth) pb.ServiceStatus {
	// If liveness is unhealthy, service is unhealthy
	if liveness.Status == pb.ServiceStatus_UNHEALTHY {
		return pb.ServiceStatus_UNHEALTHY
	}
	
	// Check critical dependencies
	hasCriticalFailure := false
	hasNonCriticalFailure := false
	
	for _, dep := range dependencies {
		if dep.Status == pb.ServiceStatus_UNHEALTHY {
			if dep.IsCritical {
				hasCriticalFailure = true
			} else {
				hasNonCriticalFailure = true
			}
		} else if dep.Status == pb.ServiceStatus_DEGRADED {
			hasNonCriticalFailure = true
		}
	}
	
	// Determine overall status based on spec
	// Spec: docs/specs/003-health-check-liveness.md#story-5-graceful-degradation-support
	if hasCriticalFailure {
		return pb.ServiceStatus_UNHEALTHY
	}
	if hasNonCriticalFailure {
		return pb.ServiceStatus_DEGRADED
	}
	
	return pb.ServiceStatus_HEALTHY
}

func (s *HealthServer) getStatusMessage(status pb.ServiceStatus) string {
	switch status {
	case pb.ServiceStatus_HEALTHY:
		return "Service is fully operational"
	case pb.ServiceStatus_DEGRADED:
		return "Service is operational with degraded performance"
	case pb.ServiceStatus_UNHEALTHY:
		return "Service is not operational"
	default:
		return "Unknown status"
	}
}

// shouldCheckDependency checks if a dependency should be checked based on filter
func (s *HealthServer) shouldCheckDependency(name string, filter []string) bool {
	if len(filter) == 0 {
		return true
	}
	for _, f := range filter {
		if f == name {
			return true
		}
	}
	return false
}

// Example dependency checker for future use
// This shows how to implement a dependency checker when needed

// DatabaseChecker checks database health
type DatabaseChecker struct {
	hostname string
	port     int32
	dbName   string
	// db connection would go here
}

// Check implements DependencyChecker
func (d *DatabaseChecker) Check(ctx context.Context) *pb.DependencyHealth {
	// This is an example implementation
	// Would perform actual database ping here
	return &pb.DependencyHealth{
		Name:       "postgres-primary",
		Type:       pb.DependencyType_DATABASE,
		Status:     pb.ServiceStatus_HEALTHY,
		IsCritical: true,
		Message:    "Database connection healthy",
		Config: &pb.DependencyConfig{
			Hostname:     d.hostname,
			Port:         d.port,
			Protocol:     "postgresql",
			DatabaseName: d.dbName,
			PoolInfo: &pb.ConnectionPoolInfo{
				MaxConnections:    20,
				ActiveConnections: 5,
				IdleConnections:   15,
				WaitCount:         0,
				WaitDurationMs:    0,
			},
		},
		LastCheck:      time.Now().Format(time.RFC3339),
		LastSuccess:    time.Now().Format(time.RFC3339),
		ResponseTimeMs: 15,
	}
}