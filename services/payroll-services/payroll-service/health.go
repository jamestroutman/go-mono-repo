package main

import (
	"context"
	"sync"
	"time"

	pb "example.com/go-mono-repo/proto/payroll"
)

// HealthServer implements the Health service
// Spec: docs/specs/003-health-check-liveness.md
type HealthServer struct {
	pb.UnimplementedHealthServer
	startTime    time.Time
	config       *Config
	configLoaded bool
	grpcReady    bool
	mu           sync.RWMutex
}

// NewHealthServer creates a new health server instance
func NewHealthServer(cfg *Config, startTime time.Time) *HealthServer {
	return &HealthServer{
		startTime:    startTime,
		config:       cfg,
		configLoaded: true,
		grpcReady:    true,
	}
}

// GetLiveness checks if the service is alive and ready to accept traffic
// Spec: docs/specs/003-health-check-liveness.md#story-1-service-liveness-check
func (s *HealthServer) GetLiveness(ctx context.Context, req *pb.LivenessRequest) (*pb.LivenessResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	checks := []*pb.ComponentCheck{
		{
			Name:    "config",
			Ready:   s.configLoaded,
			Message: "Configuration loaded",
		},
		{
			Name:    "grpc_server",
			Ready:   s.grpcReady,
			Message: "gRPC server ready",
		},
	}

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

// GetHealth performs comprehensive health check including dependencies
// Spec: docs/specs/003-health-check-liveness.md#story-2-dependency-health-monitoring
func (s *HealthServer) GetHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	startTime := time.Now()

	// Check liveness first
	livenessResp, _ := s.GetLiveness(ctx, &pb.LivenessRequest{})

	// Check dependencies
	dependencies := s.checkDependencies(ctx, req.DependencyFilter)

	// Convert liveness response to LivenessInfo
	livenessInfo := s.convertLivenessInfo(livenessResp)

	// Calculate overall status
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

// checkDependencies checks the health of service dependencies
func (s *HealthServer) checkDependencies(ctx context.Context, filter []string) []*pb.DependencyHealth {
	dependencies := []*pb.DependencyHealth{}

	// For payroll service, we don't have any external dependencies yet
	// This is where you would add checks for databases, caches, other services, etc.

	// Example of what a database dependency check would look like:
	// if s.shouldCheckDependency("postgres", filter) {
	//     dependencies = append(dependencies, s.checkPostgresHealth(ctx))
	// }

	return dependencies
}

// shouldCheckDependency determines if a dependency should be checked based on filter
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

// convertLivenessInfo converts LivenessResponse to LivenessInfo
func (s *HealthServer) convertLivenessInfo(resp *pb.LivenessResponse) *pb.LivenessInfo {
	isAlive := resp.Status == pb.ServiceStatus_HEALTHY

	// Extract component status
	configLoaded := false
	poolsReady := true // Default to true since payroll doesn't have pools yet
	cacheWarmed := true // Default to true since payroll doesn't have cache yet

	for _, check := range resp.Checks {
		switch check.Name {
		case "config":
			configLoaded = check.Ready
		case "database_pool":
			poolsReady = check.Ready
		case "cache":
			cacheWarmed = check.Ready
		}
	}

	return &pb.LivenessInfo{
		IsAlive:      isAlive,
		ConfigLoaded: configLoaded,
		PoolsReady:   poolsReady,
		CacheWarmed:  cacheWarmed,
		Components:   resp.Checks,
	}
}

// calculateOverallStatus calculates the overall health status
// Spec: docs/specs/003-health-check-liveness.md#story-5-graceful-degradation-support
func (s *HealthServer) calculateOverallStatus(liveness *pb.LivenessResponse, dependencies []*pb.DependencyHealth) pb.ServiceStatus {
	// If liveness is unhealthy, the service is unhealthy
	if liveness.Status == pb.ServiceStatus_UNHEALTHY {
		return pb.ServiceStatus_UNHEALTHY
	}

	// Check for critical dependency failures
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

	if hasCriticalFailure {
		return pb.ServiceStatus_UNHEALTHY
	}
	if hasNonCriticalFailure {
		return pb.ServiceStatus_DEGRADED
	}

	return pb.ServiceStatus_HEALTHY
}

// getStatusMessage returns a human-readable message for the status
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

// SetConfigLoaded updates the config loaded status
func (s *HealthServer) SetConfigLoaded(loaded bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configLoaded = loaded
}

// SetGRPCReady updates the gRPC ready status
func (s *HealthServer) SetGRPCReady(ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.grpcReady = ready
}