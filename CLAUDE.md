# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Documentation

For detailed information about the monorepo architecture and patterns, refer to:

### Core Documentation (Root /docs)
- **[Architecture Overview](./docs/ARCHITECTURE.md)** - Core design principles, module strategy, and workspace configuration
- **[Development Container](./docs/DEVCONTAINER.md)** - VS Code devcontainer setup, configuration, and usage
- **[Infrastructure](./docs/INFRASTRUCTURE.md)** - Database services, ImmuDB, and container orchestration
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

## Development Environment

### Prerequisites
- **VS Code** with Dev Containers extension
- **Docker Desktop** (Windows/Mac) or Docker Engine (Linux)
- **Git** for version control

### Getting Started
1. Open the repository in VS Code
2. When prompted, click "Reopen in Container" (or use Command Palette: "Dev Containers: Reopen in Container")
3. Wait for the container to build (first time takes a few minutes)
4. All tools and services will be automatically available

For detailed setup instructions, see [DEVCONTAINER.md](./docs/DEVCONTAINER.md)

## Build and Run Commands

**IMPORTANT**: All commands below should be run from within the devcontainer terminal.

### Primary Development Commands
```bash
# Main development command - runs all migrations and starts all services
make dev

# Service Commands
make run-ledger          # Start ledger service (includes migrations + proto generation)
make run-treasury        # Start treasury service (includes migrations + proto generation)
make run-payroll         # Start payroll service (proto generation)
make payroll-service     # Alias for run-payroll
make run-all             # Start all services (no migrations)

# Migration Commands
make migrate             # Run all service migrations
make migrate-ledger      # Run ledger service migrations only
make migrate-treasury    # Run treasury service migrations only
make migrate-status      # Check all migration status
make migrate-new-ledger NAME=description    # Create new ledger migration
make migrate-new-treasury                   # Create new treasury migration (interactive)

# Health Check Commands (Spec: docs/specs/003-health-check-liveness.md)
make health              # Check all services health
make health-ledger       # Check ledger service health
make health-treasury     # Check treasury service health
make health-payroll      # Check payroll service health
make liveness            # Check all services liveness
make liveness-ledger     # Check ledger service liveness
make liveness-treasury   # Check treasury service liveness
make liveness-payroll    # Check payroll service liveness

# Infrastructure services (postgres, immudb, redis) are automatically started with devcontainer

# Sync workspace modules
go work sync

# Test gRPC endpoints directly (from devcontainer)
grpcurl -plaintext localhost:50051 ledger.Manifest/GetManifest
grpcurl -plaintext localhost:50052 treasury.Manifest/GetManifest
grpcurl -plaintext localhost:50053 payroll.Manifest/GetManifest
grpcurl -plaintext localhost:50051 ledger.Health/GetLiveness
grpcurl -plaintext localhost:50051 ledger.Health/GetHealth
grpcurl -plaintext localhost:50053 payroll.Health/GetHealth
grpcurl -plaintext localhost:50053 payroll.Health/GetLiveness
grpcurl -plaintext -d '{"name": "Developer"}' localhost:50053 payroll.PayrollService/HelloWorld
```

### Working with Individual Services
```bash
# Run a specific service directly (from devcontainer)
go run ./services/treasury-services/ledger-service/main.go
go run ./services/payroll-services/payroll-service/main.go

# Generate protobuf code manually (from devcontainer)
protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
       --go-grpc_out=. --go-grpc_opt=module=example.com/go-mono-repo \
       services/treasury-services/ledger-service/proto/ledger_service.proto

protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
       --go-grpc_out=. --go-grpc_opt=module=example.com/go-mono-repo \
       services/payroll-services/payroll-service/proto/payroll_service.proto
```

### Infrastructure Services

The devcontainer automatically starts these services:

#### PostgreSQL
- **Host**: postgres (from devcontainer) or localhost:5432 (from host)
- **Database**: monorepo_dev
- **Credentials**: postgres/postgres

#### ImmuDB
- **gRPC**: immudb:3322 (from devcontainer) or localhost:3322 (from host)
- **PostgreSQL Wire**: immudb:5433 (from devcontainer) or localhost:5433 (from host)
- **Web Console**: http://localhost:8080 (credentials: immudb/immudb)

### Container Management
```bash
# View running services (from host)
docker compose -f .devcontainer/docker-compose.yml ps

# View logs (from host)
docker compose -f .devcontainer/docker-compose.yml logs -f [service]

# Restart a service (from host)
docker compose -f .devcontainer/docker-compose.yml restart [service]
```

## Architecture Overview

This is a Go monorepo using **Go Workspaces** (Go 1.24+) for managing multiple microservices with shared protocol buffer definitions. Development is done within a VS Code devcontainer that provides a consistent, isolated environment with all required tools and services.

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