# Monorepo Architecture

## Overview

This monorepo uses Go Workspaces to manage multiple microservices with shared protocol buffer definitions. The architecture is designed to provide clean separation between services while maintaining efficient code sharing and dependency management.

The development environment is containerized using VS Code Dev Containers, providing a consistent development experience with all required tools and services pre-configured. See [DEVCONTAINER.md](./DEVCONTAINER.md) for detailed setup instructions.

## Core Design Principles

### 1. Module Independence
Each service maintains its own `go.mod` file, allowing for:
- Independent versioning of service dependencies
- Service-specific dependency updates without affecting other services
- Clear ownership boundaries

### 2. Shared Protocol Definitions
All generated protobuf code lives in the root module, enabling:
- Single source of truth for service contracts
- Consistent type definitions across all services
- Easy cross-service communication

### 3. No Replace Directives
Go Workspaces eliminate the need for `replace` directives by:
- Automatically resolving local module dependencies
- Maintaining clean `go.mod` files
- Simplifying CI/CD pipelines

## Directory Structure

```
go-mono-repo/
├── go.work                    # Workspace configuration
├── go.mod                     # Root module (shared code)
├── .devcontainer/             # Dev container configuration
│   ├── devcontainer.json     # VS Code container settings
│   ├── docker-compose.yml    # Multi-service orchestration
│   └── Dockerfile            # Development image definition
├── proto/                     # Generated protobuf code
│   ├── ledger/               # Ledger service types
│   ├── treasury/             # Treasury service types
│   └── payroll/              # Payroll service types
├── services/                  # Microservices
│   ├── treasury-services/    # Treasury domain
│   │   ├── ledger-service/   # Ledger management
│   │   └── treasury-service/ # Treasury operations
│   └── payroll-services/     # Payroll domain
│       └── payroll-service/  # Payroll processing
└── docs/                      # Documentation
    ├── ARCHITECTURE.md       # This file
    ├── DEVCONTAINER.md      # Container environment docs
    ├── INFRASTRUCTURE.md    # Infrastructure details
    └── SERVICE_DEVELOPMENT.md # Service creation guide
```

## Module Strategy

### Root Module (`example.com/go-mono-repo`)
- **Purpose**: Houses generated protobuf code and shared utilities
- **Dependencies**: Common dependencies used across services
- **Imports**: None (this is the base module)

### Service Modules
- **Purpose**: Individual service implementation
- **Dependencies**: Service-specific requirements
- **Imports**: Generated types from root module

Example service module structure:
```go
module github.com/jamestroutman/ledger-service

require (
    example.com/go-mono-repo v0.0.0  // Workspace resolves this
    google.golang.org/grpc v1.74.2
)
```

## Workspace Configuration

The `go.work` file defines the workspace:
```
go 1.24

use (
    .                                          # Root module
    ./services/payroll-services/payroll-service
    ./services/treasury-services/ledger-service
    ./services/treasury-services/treasury-service
)
```

### Workspace Benefits
1. **Local Development**: Changes to shared code immediately available to services
2. **Dependency Resolution**: Automatic resolution of local module references
3. **Build Optimization**: Single module cache for all workspace modules
4. **IDE Support**: Full IntelliSense across module boundaries

## Service Communication

### gRPC Pattern
All services expose gRPC endpoints with:
- Protocol buffer definitions in `{service}/proto/`
- Generated code in root `proto/{package}/`
- Reflection enabled for debugging
- Unique port assignments

### Service Discovery
Services implement a standard `Manifest` endpoint:
```protobuf
service Manifest {
    rpc GetManifest (ManifestRequest) returns (ManifestResponse) {}
}
```

This provides:
- Service identification
- Health checking capability
- Version information (when extended)

## Build Pipeline

### Proto Generation Flow
1. Source: `services/{domain}/{service}/proto/*.proto`
2. Generation: `protoc` with Go plugins
3. Output: `proto/{package}/*.pb.go`
4. Import: Services import from root module

### Makefile Targets
- `make install-reqs`: Verify development environment
- `make run-ledger`: Start ledger service with migrations
- `make run-treasury`: Start treasury service with migrations
- `make run-payroll`: Start payroll service
- `make run-all`: Run all services concurrently
- `make dev`: Run migrations and start all services

## Infrastructure Layer

The monorepo includes infrastructure services that support the application layer:

### Local Development
- **Docker Compose**: Orchestrates infrastructure services
- **PostgreSQL**: Primary data store (port 5432)
- **Network Isolation**: Dedicated Docker network for services
- **Data Persistence**: Docker volumes for stateful services

### Production Environment
- **Amazon RDS**: Managed PostgreSQL
- **AWS Services**: Leveraging managed services for reliability
- **Infrastructure as Code**: Terraform for production provisioning (planned)

For detailed infrastructure documentation, see [INFRASTRUCTURE.md](./INFRASTRUCTURE.md).

## Deployment Considerations

### Containerization
Each service can be containerized independently:
- Build from workspace root for shared code access
- Copy only necessary service directories
- Multi-stage builds for minimal images

### Scaling
Services scale independently:
- Horizontal scaling per service
- Service-specific resource allocation
- Independent deployment cycles

### Versioning
- Root module: Semantic versioning for breaking changes
- Service modules: Independent versioning
- API contracts: Backward compatibility via protobuf

## Best Practices

1. **Service Boundaries**: Keep services focused on single domains
2. **Shared Code**: Only share types and utilities, not business logic
3. **Proto Organization**: One proto package per service
4. **Port Management**: Document and reserve ports in configuration
5. **Testing**: Unit tests per service, integration tests at root
6. **Documentation**: Keep service-specific docs with the service

## Future Considerations

### Potential Enhancements
- Service mesh integration (Istio/Linkerd)
- Distributed tracing (OpenTelemetry)
- Circuit breakers and retry logic
- API gateway for external access
- Centralized configuration management

### Scaling the Monorepo
As the monorepo grows:
- Consider domain-driven design for service organization
- Implement code ownership via CODEOWNERS
- Add pre-commit hooks for proto validation
- Consider build caching strategies
- Evaluate partial checkout strategies for large repos