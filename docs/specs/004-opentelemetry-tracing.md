# OpenTelemetry Distributed Tracing with Sentry.io Integration Specification

> **Status**: Partially Implemented  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-14  
> **Author(s)**: Platform Team  
> **Reviewer(s)**: Engineering Team, DevOps Team  
> **Confluence**: https://example.atlassian.net/wiki/spaces/PLATFORM/pages/004/OpenTelemetry+Distributed+Tracing  

## Executive Summary

This specification defines the implementation of OpenTelemetry distributed tracing across all microservices in the monorepo, with integrated telemetry export to Sentry.io for centralized trace visualization and performance monitoring. This enables end-to-end observability of request flows across service boundaries and provides actionable insights into system performance.

## Problem Statement

### Current State
Services in the monorepo operate independently without unified tracing capabilities, making it difficult to:
- Track requests across multiple service boundaries
- Identify performance bottlenecks in distributed workflows  
- Debug issues that span multiple services
- Measure end-to-end request latency
- Correlate errors with their root causes in upstream services

### Desired State
A comprehensive distributed tracing solution that:
- Captures and correlates traces across all services in the monorepo
- Provides end-to-end visibility of request flows
- Integrates seamlessly with existing service architecture
- Exports telemetry data to Sentry.io for centralized analysis
- Maintains high performance with minimal overhead
- Supports sampling strategies to manage trace volume

## Scope

### In Scope
- OpenTelemetry SDK integration across all services
- Distributed trace propagation via gRPC headers
- Sentry.io integration for trace export and visualization
- Configuration management for tracing parameters
- Automatic instrumentation for gRPC servers and clients
- Trace sampling strategies and configuration
- Environment-based configuration (including Sentry DSN)
- Service-to-service trace correlation
- Integration with existing health check and configuration patterns

### Out of Scope  
- Application logging integration (separate concern)
- Custom business events and metrics (separate observability concern)
- Non-gRPC protocol instrumentation (HTTP, database - future enhancement)
- Real-time alerting based on traces (handled by Sentry.io)
- Trace storage beyond Sentry.io integration
- Performance metrics collection (separate metrics system)

## User Stories

### Story 1: End-to-End Request Tracing
**As a** developer  
**I want to** trace a request from entry point through all service calls  
**So that** I can understand the complete request flow and identify performance bottlenecks  

**Acceptance Criteria:**
- [ ] Traces are generated for all incoming gRPC requests
- [ ] Trace context is propagated to all downstream service calls
- [ ] All services contribute spans to the distributed trace
- [ ] Complete trace appears in Sentry.io with all service spans
- [ ] Trace timing accurately reflects actual service call durations

### Story 2: Service Performance Analysis
**As a** platform engineer  
**I want to** analyze service performance across the entire stack  
**So that** I can optimize system performance and identify slow services  

**Acceptance Criteria:**
- [ ] Each service's contribution to overall request latency is visible
- [ ] Service call dependencies are clearly mapped
- [ ] Performance anomalies are identifiable across services
- [ ] Historical performance trends are available in Sentry.io
- [ ] Sampling ensures representative performance data without overwhelming the system

### Story 3: Cross-Service Error Correlation
**As a** developer troubleshooting an issue  
**I want to** see the complete context when an error occurs  
**So that** I can identify the root cause even if it originated in an upstream service  

**Acceptance Criteria:**
- [ ] Error traces include all upstream service calls leading to the failure
- [ ] Error context is preserved across service boundaries
- [ ] Failed traces are automatically captured regardless of sampling rate
- [ ] Error traces include relevant service metadata and request details
- [ ] Root cause service is identifiable from error traces

### Story 4: Configurable Tracing Deployment
**As a** platform operator  
**I want to** configure tracing behavior per environment  
**So that** I can optimize for development feedback vs production performance  

**Acceptance Criteria:**
- [ ] Sentry DSN is configurable via environment variables
- [ ] Trace sampling rate is configurable per environment
- [ ] Tracing can be disabled entirely without code changes
- [ ] Configuration follows existing service configuration patterns
- [ ] Invalid configuration fails fast with clear error messages

### Story 5: Development Workflow Integration
**As a** developer in local development  
**I want to** see traces for my local service interactions  
**So that** I can understand and debug service interactions during development  

**Acceptance Criteria:**
- [ ] Local development environment supports tracing to development Sentry project
- [ ] Trace data helps with local debugging workflows
- [ ] Tracing configuration works seamlessly with existing devcontainer setup
- [ ] Performance overhead is minimal in development environment
- [ ] Trace data is clearly identified as development environment

## Technical Design

### Architecture Overview
OpenTelemetry tracing is implemented as a cross-cutting concern using the interceptor pattern for gRPC services. Each service initializes an OpenTelemetry tracer provider configured with Sentry.io export capabilities. Trace context propagation happens automatically through gRPC metadata, ensuring seamless distributed trace correlation.

### Integration with Existing Architecture
The tracing system integrates with existing monorepo patterns:
- Uses existing configuration management for trace settings
- Leverages gRPC interceptors for automatic instrumentation  
- Follows service initialization patterns established in other specs
- Maintains compatibility with health check and manifest endpoints

### Configuration Integration

#### Extension to Configuration Management Spec
Building on [Configuration Management Spec](./002-configuration-management.md), add tracing configuration to service Config structs:

```go
// Tracing Configuration
type TracingConfig struct {
    Enabled        bool    `envconfig:"TRACING_ENABLED" default:"true"`
    SentryDSN      string  `envconfig:"SENTRY_DSN" default:""`
    SampleRate     float64 `envconfig:"TRACE_SAMPLE_RATE" default:"0.01"`  // 1% default for production safety
    Environment    string  `envconfig:"TRACE_ENVIRONMENT" default:""`  // Defaults to main Environment field
    ServiceName    string  `envconfig:"TRACE_SERVICE_NAME" default:""`  // Defaults to main ServiceName field
    ServiceVersion string  `envconfig:"TRACE_SERVICE_VERSION" default:""`  // Defaults to main ServiceVersion field
}

// GetEnvironment returns the tracing environment or falls back to provided default
func (c *TracingConfig) GetEnvironment(fallback string) string {
    if c.Environment != "" {
        return c.Environment
    }
    return fallback
}

// GetServiceName returns the tracing service name or falls back to provided default
func (c *TracingConfig) GetServiceName(fallback string) string {
    if c.ServiceName != "" {
        return c.ServiceName
    }
    return fallback
}

// GetServiceVersion returns the tracing service version or falls back to provided default
func (c *TracingConfig) GetServiceVersion(fallback string) string {
    if c.ServiceVersion != "" {
        return c.ServiceVersion
    }
    return fallback
}

// Extended Config struct for services
type Config struct {
    // ... existing configuration fields ...
    
    // Tracing Configuration
    Tracing TracingConfig `envconfig:""`
}
```

#### Environment File (.env) Extensions
```bash
# Tracing Configuration
TRACING_ENABLED=true
SENTRY_DSN=https://your-dsn@sentry.io/project-id
TRACE_SAMPLE_RATE=0.01  # 1% for production, increase for dev/staging
TRACE_ENVIRONMENT=dev
# TRACE_SERVICE_NAME and TRACE_SERVICE_VERSION default to SERVICE_NAME and SERVICE_VERSION
```

### Core Implementation Components

#### 1. Tracing Initialization Module

The tracing module is located in the `common/tracing` directory to indicate it's shared infrastructure available to all services.

```go
// common/tracing/provider.go
// Spec: docs/specs/004-opentelemetry-tracing.md

package tracing

import (
    "context"
    "fmt"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"  // Note: Version compatibility with OTel SDK
    
    "github.com/getsentry/sentry-go"
    sentryotel "github.com/getsentry/sentry-go/otel"
)

// InitializeTracing sets up OpenTelemetry tracing with Sentry.io integration
func InitializeTracing(cfg TracingConfig) (func(), error) {
    if !cfg.Enabled {
        // Set up no-op tracer provider
        otel.SetTracerProvider(sdktrace.NewTracerProvider())
        return func() {}, nil
    }
    
    // Initialize Sentry with tracing enabled
    err := sentry.Init(sentry.ClientOptions{
        Dsn:              cfg.SentryDSN,
        Environment:      cfg.GetEnvironment("development"),
        Release:          cfg.GetServiceVersion("v1.0.0"),
        EnableTracing:    true,
        TracesSampleRate: cfg.SampleRate,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to initialize Sentry: %w", err)
    }
    
    // Create resource with service identification
    res := resource.NewWithAttributes(
        semconv.SchemaURL,
        semconv.ServiceName(cfg.GetServiceName("unknown-service")),
        semconv.ServiceVersion(cfg.GetServiceVersion("v1.0.0")),
        semconv.DeploymentEnvironment(cfg.GetEnvironment("development")),
    )
    
    // Create tracer provider with Sentry span processor
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithResource(res),
        sdktrace.WithSpanProcessor(sentryotel.NewSentrySpanProcessor()),
        sdktrace.WithSampler(sdktrace.TraceIDRatioBased(cfg.SampleRate)),
    )
    
    // Set global tracer provider and propagator
    otel.SetTracerProvider(tp)
    // Use composite propagator for better compatibility
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        sentryotel.NewSentryPropagator(),
        propagation.TraceContext{},
        propagation.Baggage{},
    ))
    
    // Return cleanup function
    return func() {
        _ = tp.Shutdown(context.Background())
        sentry.Flush(2 * time.Second)
    }, nil
}
```

#### 2. gRPC Interceptors

```go
// common/tracing/interceptors.go  
// Spec: docs/specs/004-opentelemetry-tracing.md

package tracing

import (
    "context"
    "google.golang.org/grpc"
    "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// NewServerInterceptors returns gRPC interceptors with tracing enabled
func NewServerInterceptors() (grpc.UnaryServerInterceptor, grpc.StreamServerInterceptor) {
    return otelgrpc.UnaryServerInterceptor(), otelgrpc.StreamServerInterceptor()
}

// NewClientInterceptors returns gRPC client interceptors with tracing enabled  
func NewClientInterceptors() (grpc.UnaryClientInterceptor, grpc.StreamClientInterceptor) {
    return otelgrpc.UnaryClientInterceptor(), otelgrpc.StreamClientInterceptor()
}
```

#### 3. Service Integration Pattern

```go
// Example service integration in main.go
// Spec: docs/specs/004-opentelemetry-tracing.md

func main() {
    // Load configuration (following existing pattern from spec 002)
    cfg, err := LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }
    
    // Validate configuration  
    if err := cfg.Validate(); err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }
    
    // Initialize tracing - convert service config to tracing.TracingConfig
    tracingCfg := tracing.TracingConfig{
        Enabled:        cfg.Tracing.Enabled,
        SentryDSN:      cfg.Tracing.SentryDSN,
        SampleRate:     cfg.Tracing.SampleRate,
        Environment:    cfg.Tracing.Environment,
        ServiceName:    cfg.Tracing.ServiceName,
        ServiceVersion: cfg.Tracing.ServiceVersion,
    }
    
    cleanup, err := tracing.InitializeTracing(tracingCfg)
    if err != nil {
        log.Fatalf("Failed to initialize tracing: %v", err)
    }
    defer cleanup()
    
    // Create gRPC server with tracing interceptors
    unaryInterceptor, streamInterceptor := tracing.NewServerInterceptors()
    server := grpc.NewServer(
        grpc.UnaryInterceptor(unaryInterceptor),
        grpc.StreamInterceptor(streamInterceptor),
    )
    
    // Register services
    pb.RegisterLedgerServiceServer(server, &ledgerServer{})
    
    // Start server...
}
```

### Trace Context Propagation

Trace context propagation happens automatically through gRPC metadata using OpenTelemetry's standard propagation mechanisms:

1. **Incoming Requests**: Server interceptor extracts trace context from gRPC metadata
2. **Outgoing Calls**: Client interceptor injects trace context into gRPC metadata
3. **Cross-Service Correlation**: Trace and span IDs are maintained across service boundaries
4. **Baggage Support**: Additional context can be propagated using OpenTelemetry baggage

### Performance Considerations

#### Sampling Strategy
- **Default Rate**: 1% sampling in production environments (configurable up to 100%)
- **Development**: Higher sampling rates (50-100%) for debugging
- **Error Traces**: Always sampled regardless of sampling rate
- **Head-based Sampling**: Decision made at trace root to ensure complete traces

#### Resource Management
- **Batch Export**: Traces exported in batches to minimize network overhead
- **Async Processing**: Span processing happens asynchronously to avoid blocking service calls
- **Resource Limits**: Configurable limits on span attributes and events
- **Graceful Shutdown**: Ensures all pending traces are exported on service termination

## Dependencies

### Service Dependencies
- All services in the monorepo will depend on the tracing module
- No additional service-to-service dependencies introduced

### External Dependencies  
#### Required Go Modules
```go
require (
    go.opentelemetry.io/otel v1.37.0
    go.opentelemetry.io/otel/sdk v1.36.0
    go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.49.0
    github.com/getsentry/sentry-go v0.35.1
    github.com/getsentry/sentry-go/otel v0.35.1
)
```

**Note**: Version compatibility between OpenTelemetry components is important. The semconv package version may differ from the core SDK version.

### Infrastructure Dependencies
- **Sentry.io Account**: Required for trace export and visualization
- **Network Connectivity**: Services need outbound HTTPS access to Sentry.io
- **DNS Resolution**: Reliable DNS resolution for Sentry endpoints

## Security Considerations

### Data Privacy
- **PII Exclusion**: Trace data must not include personally identifiable information
- **Sensitive Data**: Request/response bodies are not captured by default
- **Sanitization**: Automatic sanitization of common sensitive fields in trace attributes
- **Data Retention**: Traces are retained according to Sentry.io's retention policies

### Access Control  
- **Sentry DSN Security**: DSN is considered sensitive and must be stored securely
- **Environment Separation**: Separate Sentry projects for different environments
- **Team Access**: Sentry project access controlled through Sentry.io RBAC

### Network Security
- **TLS Encryption**: All telemetry data transmitted over HTTPS
- **Certificate Validation**: Proper TLS certificate validation for Sentry endpoints
- **Firewall Rules**: Outbound HTTPS access required for telemetry export

## Testing Strategy

### Unit Tests
- [ ] Tracing initialization and configuration
- [ ] Interceptor functionality and context propagation
- [ ] Sampling logic and trace decision making
- [ ] Error handling and graceful degradation
- [ ] Configuration validation and defaults

### Integration Tests  
- [ ] End-to-end trace propagation across services
- [ ] Sentry.io export functionality
- [ ] Performance impact measurement
- [ ] Configuration loading and validation
- [ ] Service startup and shutdown with tracing enabled

### Acceptance Tests
- [ ] Complete user story workflows with trace validation
- [ ] Cross-service error correlation verification
- [ ] Performance benchmarking with tracing enabled
- [ ] Development environment workflow validation
- [ ] Production-like environment testing

### Performance Testing
- [ ] Latency impact measurement (target: <5% overhead)
- [ ] Memory usage impact assessment
- [ ] Throughput impact under load
- [ ] Trace export performance under high volume

## Implementation Plan

### Phase 1: Foundation (Completed)
- [x] Create shared tracing module in `common/tracing` directory
- [x] Implement basic OpenTelemetry + Sentry.io integration
- [x] Add tracing configuration to Config management pattern
- [x] Create gRPC interceptors for automatic instrumentation
- [ ] Write unit tests for core tracing functionality

### Phase 2: Initial Service Integration (Completed)  
- [x] Integrate tracing into ledger-service as pilot
- [x] Validate service starts with tracing enabled
- [x] Test with and without Sentry DSN (graceful degradation)
- [x] Integrate tracing into treasury-service
- [ ] Validate cross-service trace propagation (when services communicate)
- [ ] Performance baseline measurement

### Phase 3: Full Service Rollout (Weeks 5-6)
- [x] Integrate tracing into treasury-service (completed)
- [ ] Integrate tracing into payroll-service
- [ ] Update service initialization patterns
- [ ] Complete integration testing across all services

### Phase 4: Configuration and Testing (Week 7)
- [ ] Complete environment variable configuration
- [ ] Implement development environment support
- [ ] Add comprehensive integration tests
- [ ] Performance testing and optimization
- [ ] Documentation and runbook creation

### Phase 5: Production Readiness (Week 8)
- [ ] Production Sentry.io project setup
- [ ] Security review and sensitive data handling
- [ ] Monitoring and alerting configuration
- [ ] Deployment and rollback procedures
- [ ] Team training and knowledge transfer

## Monitoring & Observability

### Metrics
- Tracing system performance metrics:
  - Trace sampling rate (actual vs configured)
  - Span processing latency
  - Export batch success rate
  - Memory usage by tracing components
  - Network errors during trace export

### Logs
- Tracing initialization and configuration
- Export failures and retry attempts  
- Sampling decisions (debug level)
- Configuration validation errors
- Sentry.io connection issues

### Alerts
- Trace export failure rate > 5%
- Tracing causing service latency increase > 10%
- Sentry.io connectivity issues
- Memory usage increase > 50MB due to tracing
- Configuration errors preventing tracing initialization

## Documentation Updates

Upon implementation, update:
- [ ] Add tracing patterns to SERVICE_DEVELOPMENT.md
- [ ] Update CLAUDE.md with tracing testing commands
- [ ] Create tracing troubleshooting runbook
- [ ] Update service README files with tracing information
- [ ] Document Sentry.io project setup and configuration

## Implementation Notes

### Completed
- **Module Location**: Tracing module placed in `common/tracing` for better organization
- **Ledger Service**: Successfully integrated with graceful degradation when Sentry unavailable
- **Treasury Service**: Successfully integrated with full tracing support
- **Configuration**: Environment-based configuration working with sensible defaults

### Lessons Learned
1. **Version Compatibility**: OpenTelemetry components may have version mismatches; semconv package versioning differs from core SDK
2. **Import Paths**: Service code imports from `example.com/go-mono-repo/common/tracing`
3. **Config Types**: Separate TracingConfig in both service and common to avoid circular dependencies
4. **Propagator Setup**: Composite propagator provides better compatibility across different systems
5. **Go Workspace**: Use `go work sync` after adding tracing dependencies to ensure workspace consistency
6. **Build Flags**: May need `-buildvcs=false` when building outside git repository context

## Open Questions

1. **Trace Retention**: What is the appropriate trace retention period for different environments?
2. **Custom Spans**: Should services be allowed to create custom spans for business logic?
3. **Database Tracing**: Should we include database query tracing in the scope?
4. **Alert Integration**: Should trace-based alerts be configured in Sentry.io or external monitoring?
5. **Multi-tenant Support**: How should tracing handle multi-tenant service architectures in the future?

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-14 | Use OpenTelemetry + Sentry.io | Industry standard with existing team knowledge | Platform Team |
| 2025-01-14 | 1% default sampling rate | Conservative default for production safety, configurable per environment | Platform Team |
| 2025-01-14 | gRPC interceptor approach | Automatic instrumentation with minimal code changes | Platform Team |
| 2025-01-14 | Environment-based configuration | Consistent with existing configuration patterns | Platform Team |
| 2025-01-14 | Exclude logging integration | Separate concern, avoid scope creep | Platform Team |
| 2025-08-14 | Place tracing in `common/` directory | Better organization for shared infrastructure modules | Implementation Team |
| 2025-08-14 | Use composite propagator | Better compatibility with multiple tracing systems | Implementation Team |
| 2025-08-14 | Separate TracingConfig types | Avoid circular dependencies between service and common code | Implementation Team |

## References

- [OpenTelemetry Go SDK Documentation](https://opentelemetry.io/docs/instrumentation/go/)
- [Sentry OpenTelemetry Integration](https://docs.sentry.io/platforms/go/tracing/instrumentation/opentelemetry/)
- [gRPC OpenTelemetry Instrumentation](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc)
- [Configuration Management Specification](./002-configuration-management.md)
- [Health Check Specification](./003-health-check-liveness.md)
- [Architecture Overview](../ARCHITECTURE.md)

## Appendix

### Service Integration Checklist

When integrating tracing into a new service, follow these steps:

1. **Update config.go**:
   - Add `Tracing TracingConfig` field to main Config struct
   - Add TracingConfig type definition with helper methods
   - Process tracing config in LoadConfig function

2. **Update main.go**:
   - Import `example.com/go-mono-repo/common/tracing`
   - Initialize tracing after config validation
   - Convert service config to tracing.TracingConfig
   - Call `tracing.InitializeTracing()` and defer cleanup
   - Add interceptors to gRPC server creation

3. **Update .env file**:
   - Add TRACING_ENABLED=true
   - Add SENTRY_DSN (leave empty for local dev)
   - Set TRACE_SAMPLE_RATE appropriately

4. **Sync dependencies**:
   - Run `go work sync` from repository root
   - Build with `-buildvcs=false` if needed

5. **Test**:
   - Service starts with tracing enabled
   - Service starts without Sentry DSN (graceful degradation)
   - gRPC endpoints still function correctly

### Example Trace Flow
```
1. HTTP Request → API Gateway
2. API Gateway → ledger-service.CreateAccount()
   - Span: ledger-service.CreateAccount
   - Trace ID: abc123, Span ID: def456
3. ledger-service → treasury-service.ValidateAccount()
   - Span: treasury-service.ValidateAccount  
   - Trace ID: abc123, Span ID: ghi789, Parent: def456
4. treasury-service → postgres (future enhancement)
   - Span: postgres.query
   - Trace ID: abc123, Span ID: jkl012, Parent: ghi789
```

### Sentry.io Project Configuration
```yaml
# Recommended Sentry project settings
sampling:
  error_sampling: 1.0
  transaction_sampling: 0.1
retention:
  errors: 90d  
  transactions: 30d
alerts:
  - error_rate > 1%
  - transaction_failure_rate > 5%
  - p95_response_time > 1s
```

### Development Environment Setup
```bash
# Add to .env for local development
TRACING_ENABLED=true
# Leave SENTRY_DSN empty for local development (no Sentry export)
SENTRY_DSN=
TRACE_SAMPLE_RATE=1.0  # 100% for development
# TRACE_ENVIRONMENT defaults to ENVIRONMENT
# TRACE_SERVICE_NAME defaults to SERVICE_NAME
# TRACE_SERVICE_VERSION defaults to SERVICE_VERSION
```

### Troubleshooting

#### Common Issues and Solutions

1. **Import Error: cannot find module**
   - Solution: Run `go work sync` from repository root
   - Ensure service is in go.work file

2. **Build Error: error obtaining VCS status**
   - Solution: Use `go build -buildvcs=false`
   - Occurs when building outside git context

3. **Tracing not working but service runs**
   - Check TRACING_ENABLED=true in .env
   - Verify Sentry DSN format if provided
   - Check logs for "Failed to initialize tracing"

4. **Service fails to start after adding tracing**
   - Ensure all imports are correct
   - Verify TracingConfig type is defined
   - Check interceptor integration in gRPC server

5. **Traces not appearing in Sentry**
   - Verify SENTRY_DSN is correct
   - Check TRACE_SAMPLE_RATE > 0
   - Ensure network connectivity to Sentry
   - Wait 1-2 minutes for traces to appear