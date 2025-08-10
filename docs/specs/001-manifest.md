# Manifest Service Specification

> **Status**: Review  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-10  
> **Author(s)**: Engineering Team  
> **Reviewer(s)**: Platform Team  
> **Confluence**: https://example.atlassian.net/wiki/spaces/PLATFORM/pages/001/Manifest+Service  

## Executive Summary

The Manifest service provides a standardized endpoint across all services for service discovery, identification, and operational metadata. This enables consistent service introspection, monitoring, and debugging capabilities across the entire platform.

## Problem Statement

### Current State
Services implement basic manifest endpoints inconsistently with minimal information (just name and version). There's no standardized way to discover service capabilities, runtime information, or operational metadata.

### Desired State
A standardized Manifest endpoint implemented by all services that provides comprehensive service metadata for operations, monitoring, and service discovery purposes.

## Scope

### In Scope
- Service identification (name, version, API version)
- Build information (commit, build time)
- Runtime information (start time, environment)
- Service metadata (description, owner, documentation)
- Basic capabilities discovery

### Out of Scope
- Health status (use Health service)
- Liveness/readiness checks (use Health service)
- Metrics data (use Metrics endpoints)
- Dynamic configuration values
- Sensitive information (secrets, credentials)

## User Stories

### Story 1: Service Identification
**As a** platform operator  
**I want to** query any service for its identity  
**So that** I can verify which service and version is running  

**Acceptance Criteria:**
- [ ] Returns service name
- [ ] Returns semantic version
- [ ] Returns API version
- [ ] Response is cacheable

### Story 2: Build Traceability
**As a** developer  
**I want to** know the exact build of a running service  
**So that** I can trace issues back to specific code commits  

**Acceptance Criteria:**
- [ ] Returns git commit hash
- [ ] Returns build timestamp
- [ ] Returns builder information
- [ ] Links to source repository

### Story 3: Operational Context
**As a** SRE  
**I want to** understand the service's operational context  
**So that** I can properly route, monitor, and debug issues  

**Acceptance Criteria:**
- [ ] Returns environment (dev/staging/prod)
- [ ] Returns deployment region/zone
- [ ] Returns instance identifier
- [ ] Returns service start time

### Story 4: Service Discovery
**As a** client service  
**I want to** discover service capabilities  
**So that** I can properly interact with the service  

**Acceptance Criteria:**
- [ ] Returns supported API versions
- [ ] Returns enabled features
- [ ] Returns service dependencies
- [ ] Response format is consistent

## Technical Design

### Architecture Overview
The Manifest service is a lightweight, read-only gRPC service that returns static and semi-static metadata about the running service instance.

### API Design

#### RPC Methods

```protobuf
// Spec: docs/specs/001-manifest.md
service Manifest {
    // GetManifest returns comprehensive service metadata
    rpc GetManifest (ManifestRequest) returns (ManifestResponse) {}
}

message ManifestRequest {
    // No fields required - manifest is always the same for an instance
}

message ManifestResponse {
    // Service identity
    ServiceIdentity identity = 1;
    
    // Build information
    BuildInfo build_info = 2;
    
    // Runtime information
    RuntimeInfo runtime_info = 3;
    
    // Service metadata
    ServiceMetadata metadata = 4;
    
    // Service capabilities
    ServiceCapabilities capabilities = 5;
}

message ServiceIdentity {
    string name = 1;              // Service name (e.g., "ledger-service")
    string version = 2;           // Semantic version (e.g., "1.2.3")
    string api_version = 3;       // API version (e.g., "v1")
    string description = 4;       // Brief service description
}

message BuildInfo {
    string commit_hash = 1;       // Git commit SHA
    string branch = 2;            // Git branch name
    string build_time = 3;        // RFC3339 timestamp
    string builder = 4;           // CI system or user
    bool is_dirty = 5;           // Whether build had uncommitted changes
}

message RuntimeInfo {
    string instance_id = 1;       // Unique instance identifier
    string hostname = 2;          // Host machine name
    string started_at = 3;        // RFC3339 timestamp
    string environment = 4;       // Environment (dev/staging/prod)
    string region = 5;            // Deployment region/zone
    int64 uptime_seconds = 6;    // Seconds since start
}

message ServiceMetadata {
    string owner = 1;             // Team or owner email
    string repository_url = 2;    // Source code repository
    string documentation_url = 3; // Service documentation
    string support_contact = 4;   // Support contact info
    map<string, string> labels = 5; // Additional labels/tags
}

message ServiceCapabilities {
    repeated string api_versions = 1;     // Supported API versions
    repeated string protocols = 2;        // Supported protocols (grpc, http)
    repeated string features = 3;         // Enabled feature flags
    repeated ServiceDependency dependencies = 4; // Required services
}

message ServiceDependency {
    string name = 1;              // Dependency service name
    string version = 2;           // Required version (semver range)
    bool is_optional = 3;         // Whether dependency is optional
}
```

### State Management
The Manifest data is computed once at service startup and cached in memory. Only runtime_info.uptime_seconds is calculated dynamically on each request.

### Error Handling

| Error Code | Description | Response |
|------------|-------------|----------|
| INTERNAL | Failed to gather manifest data | 500 Internal Error |

## Code References

### Implementation Files
```go
// GetManifest returns service metadata
// Spec: docs/specs/001-manifest.md
func (s *server) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
    return s.manifestCache, nil
}

// computeManifest builds the manifest at startup
// Spec: docs/specs/001-manifest.md#runtime-info
func computeManifest() *pb.ManifestResponse {
    // Implementation...
}
```

### Configuration
```go
// Manifest configuration from environment
// Spec: docs/specs/001-manifest.md
type ManifestConfig struct {
    ServiceName    string `env:"SERVICE_NAME"`
    ServiceVersion string `env:"SERVICE_VERSION"`
    Environment    string `env:"ENVIRONMENT" default:"dev"`
    Region         string `env:"REGION" default:"local"`
    Owner          string `env:"SERVICE_OWNER"`
    RepoURL        string `env:"REPO_URL"`
    DocsURL        string `env:"DOCS_URL"`
}
```

### Test Files
```go
// TestGetManifest verifies manifest response
// Spec: docs/specs/001-manifest.md
func TestGetManifest(t *testing.T) {
    // Test implementation...
}
```

## Implementation Plan

### Phase 1: Core Fields
- [ ] Define protobuf messages in shared proto
- [ ] Implement basic identity and version fields
- [ ] Add to all existing services

### Phase 2: Build Information
- [ ] Add build-time code generation
- [ ] Inject git commit and build metadata
- [ ] Add build info to response

### Phase 3: Runtime & Metadata
- [ ] Add runtime information gathering
- [ ] Add configuration for metadata fields
- [ ] Implement caching mechanism

### Phase 4: Capabilities Discovery
- [ ] Define capability detection
- [ ] Add feature flag support
- [ ] Document dependency management

## Dependencies

### Service Dependencies
None - Manifest must be standalone

### External Dependencies
- google.golang.org/protobuf
- google.golang.org/grpc

### Data Dependencies
- Environment variables for configuration
- Build-time variables injection

## Security Considerations

### Information Disclosure
- No sensitive data in manifest (no secrets, credentials, or PII)
- Consider limiting detailed build info in production
- Instance IDs should be non-guessable

### Access Control
- Manifest endpoint is public within service mesh
- No authentication required (service mesh handles network security)
- Rate limiting may be applied at gateway level

## Testing Strategy

### Unit Tests
- [ ] Manifest data gathering
- [ ] Uptime calculation
- [ ] Configuration loading

### Integration Tests
- [ ] Full manifest response
- [ ] Caching behavior
- [ ] Error conditions

### Acceptance Tests
- [ ] All fields populated correctly
- [ ] Response time < 10ms
- [ ] Consistent across service restarts

## Monitoring & Observability

### Metrics
- Manifest request rate
- Response time (should be < 10ms)
- Cache hit rate

### Logs
- Manifest endpoint access (debug level)
- Configuration loading errors
- Build info at startup

### Alerts
None required - Manifest is not critical path

## Documentation Updates

Upon implementation, update:
- [ ] Add Manifest pattern to PROTOBUF_PATTERNS.md
- [ ] Update SERVICE_DEVELOPMENT.md with Manifest requirements
- [ ] Add example Manifest responses to service README files
- [ ] Update CLAUDE.md with Manifest testing commands

## Open Questions

None - all questions have been resolved.

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-10 | Exclude health from manifest | Separation of concerns, health is dynamic | Team |
| 2025-01-10 | Include build info | Critical for debugging production issues | Team |
| 2025-01-10 | Cache manifest data | Performance, data is mostly static | Team |
| 2025-01-10 | Exclude resource usage (CPU/memory) | Keep manifest lightweight, metrics belong elsewhere | Team |
| 2025-01-10 | Exclude rate limit information | Will be handled in separate rate limiting spec | Team |
| 2025-01-10 | Expose full build info in production | Services are behind gateway, security handled there | Team |
| 2025-01-10 | No partial manifest queries | Not natively supported in gRPC, keep it simple | Team |

## References

- [Protobuf Patterns](../PROTOBUF_PATTERNS.md)
- [Service Development Guide](../SERVICE_DEVELOPMENT.md)
- [OpenTelemetry Resource Semantic Conventions](https://opentelemetry.io/docs/reference/specification/resource/semantic_conventions/)