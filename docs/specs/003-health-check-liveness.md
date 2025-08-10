# Health Check and Liveness Service Specification

> **Status**: Draft  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-10  
> **Author(s)**: Engineering Team  
> **Reviewer(s)**: Platform Team, SRE Team  
> **Confluence**: https://example.atlassian.net/wiki/spaces/PLATFORM/pages/003/Health+Check+Liveness  

## Executive Summary

The Health Check and Liveness services provide standardized endpoints across all services for monitoring service health, readiness, and dependency status. These endpoints enable consistent health monitoring, load balancer integration, and dependency verification across the entire platform, ensuring robust service orchestration and operational visibility.

## Problem Statement

### Current State
Services lack standardized health monitoring endpoints, making it difficult to determine service readiness, dependency health, and overall system status. Load balancers and orchestration systems cannot reliably determine when services are ready to accept traffic or when they should be restarted.

### Desired State
All services implement standardized Liveness and Health Check endpoints that provide consistent, actionable information about service readiness and dependency health, enabling automated orchestration decisions and comprehensive health monitoring.

## Scope

### In Scope
- Liveness endpoint (service is alive and ready to accept traffic)
- Health Check endpoint (service and its dependencies are healthy)
- Dependency health verification
- Dependency configuration metadata
- Standardized health status enumeration
- Graceful degradation support

### Out of Scope
- Performance metrics (use Metrics endpoints)
- Detailed error logs (use Logging service)
- Service manifest information (use Manifest service)
- Resource utilization (CPU, memory - use Metrics)
- Business-level health metrics
- Authentication/authorization for health endpoints

## User Stories

### Story 1: Service Liveness Check
**As a** load balancer/orchestrator  
**I want to** know if a service is ready to accept traffic  
**So that** I can route requests only to ready instances  

**Acceptance Criteria:**
- [ ] Returns success when service is fully initialized
- [ ] Returns failure during startup/shutdown
- [ ] Checks critical initialization steps (config loaded, pools ready, cache warmed)
- [ ] Response time < 50ms
- [ ] No external dependency checks

### Story 2: Dependency Health Monitoring
**As a** platform operator  
**I want to** know the health status of all service dependencies  
**So that** I can understand cascading failures and system health  

**Acceptance Criteria:**
- [ ] Lists all service dependencies
- [ ] Shows health status for each dependency
- [ ] Includes dependency configuration details
- [ ] Provides last check timestamp
- [ ] Supports partial failure states

### Story 3: Automated Recovery
**As a** Kubernetes/orchestration system  
**I want to** detect unhealthy services  
**So that** I can automatically restart or replace them  

**Acceptance Criteria:**
- [ ] Clear pass/fail status for liveness
- [ ] Configurable liveness criteria
- [ ] Fast response (< 50ms)
- [ ] Doesn't cause cascading restarts

### Story 4: Dependency Configuration Visibility
**As a** developer/operator  
**I want to** see dependency configuration details  
**So that** I can verify correct service connections and troubleshoot issues  

**Acceptance Criteria:**
- [ ] Shows hostname/address for each dependency
- [ ] Shows port and protocol information
- [ ] Shows database/queue/topic names where applicable
- [ ] Excludes sensitive information (passwords, tokens)
- [ ] Includes connection pool status

### Story 5: Graceful Degradation Support
**As a** service owner  
**I want to** indicate partial service availability  
**So that** the service can operate in degraded mode when non-critical dependencies fail  

**Acceptance Criteria:**
- [ ] Distinguishes critical vs non-critical dependencies
- [ ] Returns degraded status when non-critical dependencies fail
- [ ] Returns unhealthy only when critical dependencies fail
- [ ] Provides clear degradation reason

## Technical Design

### Architecture Overview
The Health Check and Liveness services are lightweight, read-only gRPC services that perform real-time checks of service readiness and dependency health. Liveness checks are internal-only, while health checks verify external dependencies.

### API Design

#### RPC Methods

```protobuf
// Spec: docs/specs/003-health-check-liveness.md
service Health {
    // GetLiveness checks if the service is alive and ready to accept traffic
    // Should be used by load balancers and orchestrators
    rpc GetLiveness (LivenessRequest) returns (LivenessResponse) {}
    
    // GetHealth performs comprehensive health check including dependencies
    // Should be used for monitoring and debugging
    rpc GetHealth (HealthRequest) returns (HealthResponse) {}
}

message LivenessRequest {
    // No fields required - liveness is binary
}

message LivenessResponse {
    // Overall liveness status
    ServiceStatus status = 1;
    
    // Human-readable message
    string message = 2;
    
    // Component readiness checks
    repeated ComponentCheck checks = 3;
    
    // Timestamp of the check
    string checked_at = 4;  // RFC3339 format
}

message HealthRequest {
    // Include detailed dependency info
    bool include_details = 1;
    
    // Check specific dependencies only
    repeated string dependency_filter = 2;
}

message HealthResponse {
    // Overall health status
    ServiceStatus status = 1;
    
    // Human-readable message
    string message = 2;
    
    // Service liveness (internal readiness)
    LivenessInfo liveness = 3;
    
    // Dependency health checks
    repeated DependencyHealth dependencies = 4;
    
    // Timestamp of the check
    string checked_at = 5;  // RFC3339 format
    
    // Time taken to perform all checks (milliseconds)
    int64 check_duration_ms = 6;
}

enum ServiceStatus {
    // Service is fully operational
    HEALTHY = 0;
    
    // Service is operational but with non-critical issues
    DEGRADED = 1;
    
    // Service is not operational
    UNHEALTHY = 2;
}

message ComponentCheck {
    // Component name (e.g., "config", "database_pool", "cache")
    string name = 1;
    
    // Component status
    bool ready = 2;
    
    // Optional message
    string message = 3;
}

message LivenessInfo {
    // Is service ready to accept traffic
    bool is_alive = 1;
    
    // Service initialization status
    bool config_loaded = 2;
    bool pools_ready = 3;
    bool cache_warmed = 4;
    
    // Custom readiness checks
    repeated ComponentCheck components = 5;
}

message DependencyHealth {
    // Dependency name (e.g., "postgres", "redis", "user-service")
    string name = 1;
    
    // Dependency type
    DependencyType type = 2;
    
    // Health status
    ServiceStatus status = 3;
    
    // Is this dependency critical for service operation
    bool is_critical = 4;
    
    // Human-readable status message
    string message = 5;
    
    // Configuration details (non-sensitive)
    DependencyConfig config = 6;
    
    // Last successful check timestamp
    string last_success = 7;  // RFC3339 format
    
    // Last check timestamp
    string last_check = 8;  // RFC3339 format
    
    // Response time of health check (milliseconds)
    int64 response_time_ms = 9;
    
    // Error details if unhealthy
    string error = 10;
}

enum DependencyType {
    DATABASE = 0;
    CACHE = 1;
    MESSAGE_QUEUE = 2;
    GRPC_SERVICE = 3;
    HTTP_SERVICE = 4;
    STORAGE = 5;
    OTHER = 6;
}

message DependencyConfig {
    // Connection information (non-sensitive)
    string hostname = 1;
    int32 port = 2;
    string protocol = 3;  // e.g., "grpc", "http", "postgres"
    
    // Database/schema/topic information
    string database_name = 4;
    string schema_name = 5;
    string topic_name = 6;
    
    // Connection pool information
    ConnectionPoolInfo pool_info = 7;
    
    // Service version if applicable
    string version = 8;
    
    // Additional metadata
    map<string, string> metadata = 9;
}

message ConnectionPoolInfo {
    int32 max_connections = 1;
    int32 active_connections = 2;
    int32 idle_connections = 3;
    int32 wait_count = 4;
    int64 wait_duration_ms = 5;
}
```

### Implementation Requirements

#### Liveness Check Implementation
```go
// GetLiveness checks service readiness
// Spec: docs/specs/003-health-check-liveness.md#story-1-service-liveness-check
func (s *server) GetLiveness(ctx context.Context, req *pb.LivenessRequest) (*pb.LivenessResponse, error) {
    checks := []pb.ComponentCheck{
        {Name: "config", Ready: s.configLoaded, Message: "Configuration loaded"},
        {Name: "grpc_server", Ready: s.grpcReady, Message: "gRPC server ready"},
        {Name: "database_pool", Ready: s.dbPoolReady(), Message: "Database pool status"},
        {Name: "cache", Ready: s.cacheReady(), Message: "Cache connection status"},
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
```

#### Health Check Implementation
```go
// GetHealth performs comprehensive health check
// Spec: docs/specs/003-health-check-liveness.md#story-2-dependency-health-monitoring
func (s *server) GetHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
    startTime := time.Now()
    
    // Check liveness first
    livenessResp, _ := s.GetLiveness(ctx, &pb.LivenessRequest{})
    
    // Check dependencies
    dependencies := s.checkDependencies(ctx, req.DependencyFilter)
    
    // Determine overall status
    overallStatus := s.calculateOverallStatus(livenessResp, dependencies)
    
    return &pb.HealthResponse{
        Status:       overallStatus,
        Message:      s.getStatusMessage(overallStatus),
        Liveness:     s.convertLivenessInfo(livenessResp),
        Dependencies: dependencies,
        CheckedAt:    time.Now().Format(time.RFC3339),
        CheckDurationMs: time.Since(startTime).Milliseconds(),
    }, nil
}
```

### State Management
- Liveness state is tracked in-memory during service initialization
- Dependency health checks are performed in real-time with configurable caching
- Circuit breakers prevent excessive health check requests to failing dependencies

### Error Handling

| Error Code | Description | Response |
|------------|-------------|----------|
| OK | Service is healthy | 200 OK |
| UNAVAILABLE | Service is unhealthy | 503 Service Unavailable |
| DEADLINE_EXCEEDED | Health check timeout | 504 Gateway Timeout |

### Performance Requirements

#### Liveness Endpoint
- Response time: < 50ms (P99)
- No external calls
- Memory-only checks
- Cached component status

#### Health Check Endpoint
- Response time: < 500ms (P99)
- Parallel dependency checks
- Configurable timeouts per dependency
- Circuit breaker for failed dependencies

## Implementation Plan

### Phase 1: Liveness Endpoint
- [ ] Define protobuf messages in shared proto
- [ ] Implement basic liveness checks
- [ ] Add component readiness tracking
- [ ] Add to all existing services

### Phase 2: Basic Health Check
- [ ] Implement dependency health framework
- [ ] Add database health checks
- [ ] Add cache health checks
- [ ] Add gRPC service health checks

### Phase 3: Advanced Features
- [ ] Add dependency configuration details
- [ ] Implement graceful degradation
- [ ] Add circuit breakers
- [ ] Add health check caching

### Phase 4: Integration
- [ ] Configure Kubernetes probes
- [ ] Set up monitoring dashboards
- [ ] Configure alerts
- [ ] Document patterns

## Dependencies

### Service Dependencies
- None - Health service must be standalone

### External Dependencies
- google.golang.org/protobuf
- google.golang.org/grpc
- Database drivers (for health checks)
- Cache clients (for health checks)

### Data Dependencies
- Service configuration
- Connection pools
- Runtime state

## Security Considerations

### Information Disclosure
- No sensitive credentials in configuration details
- No internal IP addresses in production
- Rate limiting on health endpoints
- Consider separate internal/external health endpoints

### Access Control
- Health endpoints are public within service mesh
- External access controlled at gateway level
- No authentication on liveness (required for orchestrators)
- Optional authentication on detailed health checks

## Testing Strategy

### Unit Tests
- [ ] Liveness state tracking
- [ ] Dependency health check logic
- [ ] Status calculation
- [ ] Configuration masking

### Integration Tests
- [ ] Full health check flow
- [ ] Dependency failure scenarios
- [ ] Graceful degradation
- [ ] Circuit breaker behavior

### Acceptance Tests
- [ ] Kubernetes probe integration
- [ ] Load balancer integration
- [ ] Monitoring system integration
- [ ] Alert triggering

### Chaos Engineering
- [ ] Dependency failure injection
- [ ] Network partition simulation
- [ ] Cascading failure prevention
- [ ] Recovery time validation

## Monitoring & Observability

### Metrics
- Liveness check request rate
- Health check request rate
- Health check duration (per dependency)
- Dependency failure rate
- Circuit breaker trips
- Status transitions (healthy → degraded → unhealthy)

### Logs
- Health check failures (warning level)
- Dependency connection failures
- Circuit breaker state changes
- Liveness state transitions

### Alerts
- Service unhealthy for > 1 minute
- Multiple dependencies unhealthy
- Liveness failing (triggers restart)
- Degraded mode for > 10 minutes

### Dashboards
- Service health overview
- Dependency health matrix
- Health check latency
- Historical health trends

## Documentation Updates

Upon implementation, update:
- [ ] Add Health pattern to PROTOBUF_PATTERNS.md
- [ ] Update SERVICE_DEVELOPMENT.md with Health requirements
- [ ] Add health check examples to service README files
- [ ] Update CLAUDE.md with health check testing commands
- [ ] Create runbook for health check troubleshooting

## Open Questions

1. Should we implement HTTP endpoints in addition to gRPC for compatibility?
2. Should health checks support custom business logic checks?
3. What should be the default timeout for dependency health checks?

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-10 | Separate Liveness and Health endpoints | Different purposes and performance requirements | Team |
| 2025-01-10 | Include dependency config (non-sensitive) | Critical for debugging connection issues | Team |
| 2025-01-10 | Real-time dependency checks (with caching) | Accurate health status more important than performance | Team |
| 2025-01-10 | Support graceful degradation | Allow services to operate with non-critical dependency failures | Team |
| 2025-01-10 | No authentication on endpoints | Required for orchestrators, security at network level | Team |

## References

- [Kubernetes Liveness and Readiness Probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
- [Manifest Service Specification](./001-manifest.md)
- [Protobuf Patterns](../PROTOBUF_PATTERNS.md)
- [Service Development Guide](../SERVICE_DEVELOPMENT.md)

## Appendix

### Example Liveness Response
```json
{
  "status": "HEALTHY",
  "message": "Service is ready",
  "checks": [
    {"name": "config", "ready": true, "message": "Configuration loaded"},
    {"name": "grpc_server", "ready": true, "message": "gRPC server ready"},
    {"name": "database_pool", "ready": true, "message": "Pool initialized with 10 connections"},
    {"name": "cache", "ready": true, "message": "Redis connection established"}
  ],
  "checked_at": "2025-01-10T10:30:00Z"
}
```

### Example Health Response
```json
{
  "status": "DEGRADED",
  "message": "Service operational with degraded performance",
  "liveness": {
    "is_alive": true,
    "config_loaded": true,
    "pools_ready": true,
    "cache_warmed": true
  },
  "dependencies": [
    {
      "name": "postgres-primary",
      "type": "DATABASE",
      "status": "HEALTHY",
      "is_critical": true,
      "message": "Connection pool healthy",
      "config": {
        "hostname": "postgres.internal",
        "port": 5432,
        "protocol": "postgresql",
        "database_name": "ledger",
        "pool_info": {
          "max_connections": 20,
          "active_connections": 5,
          "idle_connections": 15
        }
      },
      "last_check": "2025-01-10T10:30:00Z",
      "response_time_ms": 15
    },
    {
      "name": "redis-cache",
      "type": "CACHE",
      "status": "UNHEALTHY",
      "is_critical": false,
      "message": "Connection refused",
      "config": {
        "hostname": "redis.internal",
        "port": 6379,
        "protocol": "redis"
      },
      "last_success": "2025-01-10T10:25:00Z",
      "last_check": "2025-01-10T10:30:00Z",
      "error": "dial tcp redis.internal:6379: connection refused"
    }
  ],
  "checked_at": "2025-01-10T10:30:00Z",
  "check_duration_ms": 125
}
```

### Kubernetes Configuration Example
```yaml
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: ledger-service
    livenessProbe:
      grpc:
        port: 50051
        service: Health.GetLiveness
      initialDelaySeconds: 10
      periodSeconds: 10
      timeoutSeconds: 1
      failureThreshold: 3
    readinessProbe:
      grpc:
        port: 50051
        service: Health.GetHealth
      initialDelaySeconds: 5
      periodSeconds: 5
      timeoutSeconds: 2
      failureThreshold: 2
```