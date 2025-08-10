# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Documentation

For detailed information about the monorepo architecture and patterns, refer to:

### Core Documentation (Root /docs)
- **[Architecture Overview](./docs/ARCHITECTURE.md)** - Core design principles, module strategy, and workspace configuration
- **[Protobuf Patterns](./docs/PROTOBUF_PATTERNS.md)** - Protocol buffer conventions, message design patterns, and code generation
- **[Service Development Guide](./docs/SERVICE_DEVELOPMENT.md)** - Step-by-step guide for creating new services, implementation patterns, and best practices
- **[Spec-Driven Development](./docs/SPEC_DRIVEN_DEVELOPMENT.md)** - How to write and use specifications for features
- **[Spec Template](./docs/SPEC_TEMPLATE.md)** - Template for creating new feature specifications

### Service Documentation Pattern
Each service maintains its own documentation in `services/{domain}/{service-name}/docs/`:
- **specs/** - Feature specifications (numbered sequentially: 001-feature.md, 002-feature.md)
- **adrs/** - Architecture Decision Records for service-specific decisions
- **runbooks/** - Operational guides for deployment and troubleshooting

Example: `services/treasury-services/ledger-service/docs/specs/001-account-management.md`

## Build and Run Commands

### Primary Development Commands
```bash
# Start the ledger service (includes proto generation)
make ledger-service

# Start the treasury service (includes proto generation)
make treasury-service

# Run all services
make all-services

# Development alias (same as ledger-service)
make dev

# Install prerequisites (Go, protoc, plugins)
make install-reqs

# Sync workspace modules
go work sync

# Test gRPC endpoints
grpcurl -plaintext localhost:50051 ledger.Manifest/GetManifest
grpcurl -plaintext localhost:50052 treasury.Manifest/GetManifest

# Health Check endpoints (Spec: docs/specs/003-health-check-liveness.md)
make health-check-ledger      # Check ledger service health
make liveness-check-ledger    # Check ledger service liveness
make health-check-treasury    # Check treasury service health
make liveness-check-treasury  # Check treasury service liveness
make health-check-all         # Check all services health
make liveness-check-all       # Check all services liveness

# Or directly with grpcurl
grpcurl -plaintext localhost:50051 ledger.Health/GetLiveness
grpcurl -plaintext localhost:50051 ledger.Health/GetHealth
```

### Working with Individual Services
```bash
# Run a specific service directly
go run ./services/treasury-services/ledger-service/main.go

# Generate protobuf code manually
protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
       --go-grpc_out=. --go-grpc_opt=module=example.com/go-mono-repo \
       services/treasury-services/ledger-service/proto/ledger_service.proto
```

## Architecture Overview

This is a Go monorepo using **Go Workspaces** (Go 1.24+) for managing multiple microservices with shared protocol buffer definitions.

### Key Design Decisions

1. **Go Workspace Structure**: Uses `go.work` to manage multiple modules without replace directives
   - Root module: `example.com/go-mono-repo` - contains shared proto definitions
   - Service modules: Independent `go.mod` files per service

2. **Protobuf Code Generation Pattern**:
   - Proto files live in: `services/{domain}/{service}/proto/*.proto`
   - Generated code goes to: `proto/{package}/*.pb.go` at the root
   - All services import from: `example.com/go-mono-repo/proto/{package}`

3. **Service Structure Pattern**:
   - Each service implements a gRPC server with reflection enabled
   - Services run on unique ports (ledger: 50051, treasury: 50052)
   - Each service has a Manifest endpoint for identification

### Module Dependencies

- Root module handles shared dependencies and generated proto code
- Service modules import generated code from root module
- No replace directives needed due to workspace configuration

### Adding New Services

1. Create directory: `services/{domain}/{service-name}/`
2. Initialize module: `go mod init {org}/{service-name}`
3. Add to workspace: `go work use ./services/{domain}/{service-name}`
4. Create proto files with `go_package = "example.com/go-mono-repo/proto/{package}"`
5. Add Makefile target for proto generation and service startup
6. Import generated types from root module in service implementation

### Spec-Driven Development Process

When implementing new features:
1. **Write Spec First**: Create a specification in `services/{domain}/{service-name}/docs/specs/` using the [Spec Template](./docs/SPEC_TEMPLATE.md)
2. **Get Approval**: Have the spec reviewed and approved before implementation
3. **Implement**: Follow the approved specification exactly
4. **Link to Spec**: Reference the spec in code comments, tests, and documentation
5. **Update Status**: Mark the spec as "Implemented" when complete