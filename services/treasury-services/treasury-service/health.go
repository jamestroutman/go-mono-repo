package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "example.com/go-mono-repo/proto/treasury"
	ledgerpb "example.com/go-mono-repo/proto/ledger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	
	// Dependencies
	dependencies []DependencyChecker
	
	// Ledger service configuration
	ledgerServiceHost string
	ledgerServicePort int32
}

// DependencyChecker interface for checking dependency health
type DependencyChecker interface {
	Check(ctx context.Context) *pb.DependencyHealth
}

// NewHealthServer creates a new health server instance
// Spec: docs/specs/003-health-check-liveness.md
func NewHealthServer(startTime time.Time) *HealthServer {
	server := &HealthServer{
		startTime:         startTime,
		configLoaded:      false,
		grpcReady:         false,
		ledgerServiceHost: "localhost",
		ledgerServicePort: 50051,
	}
	
	// Add ledger service dependency checker
	// Spec: docs/specs/003-health-check-liveness.md#story-4-dependency-configuration-visibility
	server.dependencies = []DependencyChecker{
		&LedgerServiceChecker{
			hostname: server.ledgerServiceHost,
			port:     server.ledgerServicePort,
		},
	}
	
	return server
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
	
	// Check all registered dependencies
	// Spec: docs/specs/003-health-check-liveness.md#story-4-dependency-configuration-visibility
	for _, checker := range s.dependencies {
		dep := checker.Check(ctx)
		if s.shouldCheckDependency(dep.Name, filter) {
			dependencies = append(dependencies, dep)
		}
	}
	
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
			if dep.IsCritical {
				// Critical dependency in degraded state affects overall health
				hasNonCriticalFailure = true
			} else {
				hasNonCriticalFailure = true
			}
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

// LedgerServiceChecker checks the health of the ledger service
// Spec: docs/specs/003-health-check-liveness.md#story-2-dependency-health-monitoring
type LedgerServiceChecker struct {
	hostname string
	port     int32
}

// Check implements DependencyChecker for ledger service
func (l *LedgerServiceChecker) Check(ctx context.Context) *pb.DependencyHealth {
	startTime := time.Now()
	
	// Create dependency health response
	dep := &pb.DependencyHealth{
		Name:       "ledger-service",
		Type:       pb.DependencyType_GRPC_SERVICE,
		IsCritical: true, // Treasury service depends on ledger service
		Config: &pb.DependencyConfig{
			Hostname: l.hostname,
			Port:     l.port,
			Protocol: "grpc",
			Metadata: map[string]string{
				"service": "ledger",
				"version": "v1",
			},
		},
		LastCheck: time.Now().Format(time.RFC3339),
	}
	
	// Create a context with timeout for the connection
	dialCtx, dialCancel := context.WithTimeout(ctx, 2*time.Second)
	defer dialCancel()
	
	// Try to connect and check health
	conn, err := grpc.DialContext(
		dialCtx,
		fmt.Sprintf("%s:%d", l.hostname, l.port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		dep.Status = pb.ServiceStatus_UNHEALTHY
		dep.Message = "Failed to connect to ledger service"
		dep.Error = err.Error()
		dep.ResponseTimeMs = time.Since(startTime).Milliseconds()
		return dep
	}
	defer conn.Close()
	
	// Create health client and check liveness
	healthClient := ledgerpb.NewHealthClient(conn)
	
	// Use a short timeout for health check
	checkCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	
	livenessResp, err := healthClient.GetLiveness(checkCtx, &ledgerpb.LivenessRequest{})
	if err != nil {
		dep.Status = pb.ServiceStatus_UNHEALTHY
		dep.Message = "Ledger service health check failed"
		dep.Error = err.Error()
		dep.ResponseTimeMs = time.Since(startTime).Milliseconds()
		return dep
	}
	
	// Map ledger service status to treasury's dependency status
	switch livenessResp.Status {
	case ledgerpb.ServiceStatus_HEALTHY:
		dep.Status = pb.ServiceStatus_HEALTHY
		dep.Message = "Ledger service is healthy"
		dep.LastSuccess = time.Now().Format(time.RFC3339)
	case ledgerpb.ServiceStatus_DEGRADED:
		dep.Status = pb.ServiceStatus_DEGRADED
		dep.Message = "Ledger service is degraded"
		dep.LastSuccess = time.Now().Format(time.RFC3339)
	case ledgerpb.ServiceStatus_UNHEALTHY:
		dep.Status = pb.ServiceStatus_UNHEALTHY
		dep.Message = "Ledger service is unhealthy"
	default:
		dep.Status = pb.ServiceStatus_UNHEALTHY
		dep.Message = "Unknown ledger service status"
	}
	
	dep.ResponseTimeMs = time.Since(startTime).Milliseconds()
	return dep
}