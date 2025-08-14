# Go Monorepo

A Go monorepo using **Go Workspaces** for managing multiple microservices with shared protocol buffer definitions. Development is containerized using VS Code Dev Containers for a consistent, isolated environment.

## Architecture

### Monorepo Structure
```
.
├── .devcontainer/           # VS Code dev container configuration
│   ├── devcontainer.json   # Container settings and extensions
│   ├── docker-compose.yml  # Multi-service orchestration
│   └── Dockerfile         # Development environment image
├── Makefile                 # Build automation and service management
├── go.work                  # Go workspace configuration
├── go.mod                   # Root module for shared dependencies
├── infrastructure/          # Legacy infrastructure location
│   └── init-scripts/        # Database initialization scripts
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
- **VS Code** with [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
- **Docker Desktop** (Windows/Mac) or Docker Engine (Linux)
- **Git** for version control

### Quick Start

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd go-mono-repo
   ```

2. **Open in VS Code**:
   ```bash
   code .
   ```

3. **Reopen in Container**:
   - VS Code will prompt "Reopen in Container" - click it
   - Or use Command Palette: `Dev Containers: Reopen in Container`
   - Wait for container build (first time takes a few minutes)

4. **Start services** (from devcontainer terminal):
   ```bash
   # Start ledger service
   make ledger-service
   
   # Or start treasury service
   make treasury-service
   
   # Or start all services
   make all-services
   ```

All tools (Go, protoc, etc.) and infrastructure services (PostgreSQL, ImmuDB) are pre-configured in the container.

For detailed setup instructions, see [docs/DEVCONTAINER.md](docs/DEVCONTAINER.md).

### Available Commands

**Note**: All `make` commands should be run from within the devcontainer terminal.

#### Development
| Command | Description |
|---------|-------------|
| `make dev` | Start ledger service (default) |
| `make all-services` | Start all application services |
| `go work sync` | Sync workspace modules |

#### Container Management (from host)
| Command | Description |
|---------|-------------|
| `docker compose -f .devcontainer/docker-compose.yml ps` | View container status |
| `docker compose -f .devcontainer/docker-compose.yml logs [service]` | View service logs |
| `docker compose -f .devcontainer/docker-compose.yml restart [service]` | Restart a service |
| `docker compose -f .devcontainer/docker-compose.yml down -v` | Remove all containers and data |

#### Services
| Command | Description |
|---------|-------------|
| `make ledger-service` | Generate protos and start ledger service |
| `make treasury-service` | Generate protos and start treasury service |
| `make payroll-service` | Generate protos and start payroll service |

## Services

### Ledger Service
- **Port**: `:50051`
- **Protocol**: gRPC
- **Endpoint**: `ledger.Manifest/GetManifest`
- **Response**: Returns `{"message": "ledger-service"}`

#### Testing the Service
```bash
# From devcontainer terminal
# Start the service
make ledger-service

# Test with grpcurl (in another devcontainer terminal)
grpcurl -plaintext localhost:50051 ledger.Manifest/GetManifest
grpcurl -plaintext localhost:50051 ledger.Health/GetHealth
```

### Treasury Service
- **Port**: `:50052`
- **Protocol**: gRPC
- **Endpoint**: `treasury.Manifest/GetManifest`
- **Response**: Returns `{"message": "treasury-service"}`

#### Testing the Service
```bash
# From devcontainer terminal
# Start the service
make treasury-service

# Test with grpcurl (in another devcontainer terminal)
grpcurl -plaintext localhost:50052 treasury.Manifest/GetManifest
grpcurl -plaintext localhost:50052 treasury.Health/GetHealth
```

### Payroll Service
- **Port**: `:50053`
- **Protocol**: gRPC
- **Endpoints**: 
  - `payroll.Manifest/GetManifest` - Service identification
  - `payroll.Health/GetHealth` - Health check
  - `payroll.Health/GetLiveness` - Liveness check
  - `payroll.PayrollService/HelloWorld` - Hello world endpoint
- **Response**: Returns `{"message": "payroll-service"}` for Manifest

#### Testing the Service
```bash
# From devcontainer terminal
# Start the service
make payroll-service

# Test with grpcurl (in another devcontainer terminal)
grpcurl -plaintext localhost:50053 payroll.Manifest/GetManifest
grpcurl -plaintext localhost:50053 payroll.Health/GetHealth
grpcurl -plaintext localhost:50053 payroll.Health/GetLiveness
grpcurl -plaintext -d '{"name": "Developer"}' localhost:50053 payroll.PayrollService/HelloWorld
```

## Infrastructure Services

All infrastructure services are automatically started with the devcontainer.

### PostgreSQL Database
- **Connection from devcontainer**: `postgres:5432`
- **Connection from host**: `localhost:5432`
- **Database**: `monorepo_dev`
- **Credentials**: `postgres/postgres`
- **Admin UI**: http://localhost:5050 (pgAdmin - admin@example.com/admin)
- **Data persistence**: Docker volume `postgres_data`

### ImmuDB Database
- **gRPC from devcontainer**: `immudb:3322`
- **gRPC from host**: `localhost:3322`
- **PostgreSQL wire**: Port 5433
- **Web Console**: http://localhost:8080 (immudb/immudb)
- **Data persistence**: Docker volume `immudb_data`

### Redis Cache
- **Connection from devcontainer**: `redis:6379`
- **Connection from host**: `localhost:6379`
- **Admin UI**: http://localhost:8081 (Redis Commander - admin/admin)
- **Data persistence**: Docker volume `redis_data` with AOF

### Web Admin Interfaces
| Service | URL | Credentials |
|---------|-----|-------------|
| pgAdmin | http://localhost:5050 | admin@example.com / admin |
| ImmuDB Console | http://localhost:8080 | immudb / immudb |
| Redis Commander | http://localhost:8081 | admin / admin |

### Connection Examples
```bash
# PostgreSQL (from devcontainer)
psql -h postgres -U postgres -d monorepo_dev

# ImmuDB (from devcontainer)
grpcurl -plaintext immudb:3322 immudb.schema.ImmuService/Health

# Redis (from devcontainer)
redis-cli -h redis ping
redis-cli -h redis set key "value"
redis-cli -h redis get key
```

For detailed infrastructure documentation, see [docs/INFRASTRUCTURE.md](docs/INFRASTRUCTURE.md).

## Development Workflow

All development should be done within the VS Code devcontainer.

### Adding a New Service
1. Open terminal in devcontainer
2. Create service directory: `services/{domain}/{service-name}/`
3. Add `go.mod`: `go mod init {org}/{service-name}`
4. Create `.proto` files in `{service}/proto/`
5. Add service target to Makefile
6. Update `go.work` to include the new service
7. Implement service in `main.go`

For detailed instructions, see [docs/SERVICE_DEVELOPMENT.md](docs/SERVICE_DEVELOPMENT.md).

### Adding New Proto Definitions
1. Define `.proto` files in service directory
2. Update `go_package` option to use root module path
3. Add protoc generation to Makefile
4. Import generated types from `example.com/go-mono-repo/proto/{package}`

### Working with Workspaces
```bash
# From devcontainer terminal
# Sync all workspace modules
go work sync

# Add a new module to workspace
go work use ./services/new-service

# View workspace status
go work edit -print
```

## Documentation

- [Architecture Overview](docs/ARCHITECTURE.md) - System design and patterns
- [Development Container](docs/DEVCONTAINER.md) - Container setup and usage
- [Infrastructure](docs/INFRASTRUCTURE.md) - Database and service configuration
- [Service Development](docs/SERVICE_DEVELOPMENT.md) - Creating new services
- [Spec-Driven Development](docs/SPEC_DRIVEN_DEVELOPMENT.md) - Specification process

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