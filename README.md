# Go Monorepo

A Go monorepo using **Go Workspaces** for managing multiple microservices with shared protocol buffer definitions.

## Architecture

### Monorepo Structure
```
.
├── Makefile                 # Build automation and service management
├── go.work                  # Go workspace configuration
├── go.mod                   # Root module for shared dependencies
├── infrastructure/          # Infrastructure services (Docker Compose)
│   ├── docker-compose.yml  # PostgreSQL and other infrastructure
│   ├── init-scripts/        # Database initialization scripts
│   └── README.md           # Infrastructure documentation
├── proto/                   # Generated protobuf Go files
│   ├── ledger/
│   │   ├── *.pb.go         # Generated protocol buffer types
│   │   └── *_grpc.pb.go    # Generated gRPC service stubs
│   └── treasury/
│       ├── *.pb.go         # Generated protocol buffer types
│       └── *_grpc.pb.go    # Generated gRPC service stubs
└── services/                # Microservices
    └── treasury-services/
        ├── ledger-service/
        │   ├── go.mod       # Service-specific module
        │   ├── main.go      # Service implementation
        │   └── proto/       # Service protocol definitions
        │       └── *.proto
        └── treasury-service/
            ├── go.mod       # Service-specific module
            ├── main.go      # Service implementation
            └── proto/       # Service protocol definitions
                └── *.proto
```

### Go Workspace Design

This monorepo uses **Go Workspaces** (Go 1.18+) for dependency management:

- **Root Module** (`example.com/go-mono-repo`): Contains shared protobuf definitions and common dependencies
- **Service Modules**: Each service has its own `go.mod` with independent versioning
- **No Replace Directives**: Workspace automatically resolves local module dependencies
- **Shared Proto Generation**: All `.proto` files generate Go code to the root `proto/` directory

## Protobuf Design

### Service Definition Pattern
Each service defines its own `.proto` files in `services/{domain}/{service}/proto/`:

```protobuf
syntax = "proto3";

package ledger;
option go_package = "example.com/go-mono-repo/proto/ledger";

service Manifest {
  rpc GetManifest (ManifestRequest) returns (ManifestResponse) {}
}
```

### Code Generation
- **Input**: Service-specific `.proto` files
- **Output**: Generated Go code in `proto/{package}/`
- **Module**: All generated code uses the root module path
- **Imports**: Services import from `example.com/go-mono-repo/proto/{package}`

## Getting Started

### Prerequisites
The Makefile will automatically install these if missing:
- [Homebrew](https://brew.sh) (macOS)
- Go 1.24+
- Protocol Buffers compiler (`protoc`)
- Go protobuf plugins
- Docker and Docker Compose (for infrastructure services)

### Quick Start
```bash
# Start everything (infrastructure + services)
make dev

# Or start components individually:
make infrastructure-up    # Start PostgreSQL and ImmuDB
make ledger-service      # Start ledger service
make treasury-service    # Start treasury service
```

### Available Commands

#### Development
| Command | Description |
|---------|-------------|
| `make dev` | Start infrastructure and all services |
| `make install-reqs` | Install prerequisites (Go, protoc, plugins) |
| `make all-services` | Start all application services |

#### Infrastructure
| Command | Description |
|---------|-------------|
| `make infrastructure-up` | Start PostgreSQL, ImmuDB and other infrastructure |
| `make infrastructure-down` | Stop infrastructure services |
| `make infrastructure-status` | View infrastructure status |
| `make infrastructure-clean` | Remove infrastructure and volumes |

#### Services
| Command | Description |
|---------|-------------|
| `make ledger-service` | Generate protos and start ledger service |
| `make treasury-service` | Generate protos and start treasury service |

## Services

### Ledger Service
- **Port**: `:50051`
- **Protocol**: gRPC
- **Endpoint**: `ledger.Manifest/GetManifest`
- **Response**: Returns `{"message": "ledger-service"}`

#### Testing the Service
```bash
# Start the service
make ledger-service

# Test with grpcurl (in another terminal)
grpcurl -plaintext localhost:50051 ledger.Manifest/GetManifest
```

### Treasury Service
- **Port**: `:50052`
- **Protocol**: gRPC
- **Endpoint**: `treasury.Manifest/GetManifest`
- **Response**: Returns `{"message": "treasury-service"}`

#### Testing the Service
```bash
# Start the service
make treasury-service

# Test with grpcurl (in another terminal)
grpcurl -plaintext localhost:50052 treasury.Manifest/GetManifest
```

## Infrastructure

### PostgreSQL Database
- **Local**: Docker container with PostgreSQL 16
- **Port**: 5432
- **Credentials**: postgres/postgres
- **Database**: monorepo_dev
- **Production**: Amazon RDS

### ImmuDB Database
- **Local**: Docker container with ImmuDB (immutable database)
- **gRPC Port**: 3322 (native ImmuDB API)
- **Web Console UI**: http://localhost:8080
- **PostgreSQL Wire Port**: 5433
- **Credentials**: immudb/immudb
- **Production**: Managed ImmuDB or self-hosted cluster

### Managing Infrastructure
```bash
# Start infrastructure before development
make infrastructure-up

# Check status
make infrastructure-status

# Clean everything (including data)
make infrastructure-clean
```

For detailed infrastructure documentation, see [docs/INFRASTRUCTURE.md](docs/INFRASTRUCTURE.md).

## Development Workflow

### Adding a New Service
1. Create service directory: `services/{domain}/{service-name}/`
2. Add `go.mod`: `go mod init {org}/{service-name}`
3. Create `.proto` files in `{service}/proto/`
4. Add service target to Makefile
5. Update `go.work` to include the new service
6. Implement service in `main.go`

### Adding New Proto Definitions
1. Define `.proto` files in service directory
2. Update `go_package` option to use root module path
3. Add protoc generation to Makefile
4. Import generated types from `example.com/go-mono-repo/proto/{package}`

### Working with Workspaces
```bash
# Sync all workspace modules
go work sync

# Add a new module to workspace
go work use ./services/new-service

# View workspace status
go work edit -print
```

## Project Standards

### Module Naming
- **Root Module**: `example.com/go-mono-repo`
- **Service Modules**: `{org}/{domain}/{service-name}`
- **Proto Packages**: Use root module path for `go_package`

### Service Patterns
- Each service runs on a unique port
- gRPC with reflection enabled for debugging
- Manifest endpoint for service identification
- Clean shutdown handling recommended

### Dependency Management
- Shared dependencies in root `go.mod`
- Service-specific dependencies in service `go.mod`
- No replace directives (handled by workspace)
- Use `go work sync` to keep modules aligned