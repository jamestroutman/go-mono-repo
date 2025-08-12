# Development Container Documentation

This document describes the devcontainer-based development environment for the Go Mono Repo.

## Overview

The monorepo uses VS Code Dev Containers to provide a consistent, containerized development environment with all required tools and services pre-configured. This eliminates "works on my machine" issues and ensures all developers have the same setup.

## Architecture

The devcontainer setup uses Docker Compose to orchestrate multiple services:

```
┌─────────────────────────────────────────────────────────┐
│                    Host Machine                          │
│                                                          │
│  ┌──────────────────────────────────────────────────┐  │
│  │              VS Code + Dev Container              │  │
│  └──────────────────────────────────────────────────┘  │
│                           │                              │
│                           ▼                              │
│  ┌──────────────────────────────────────────────────┐  │
│  │              Docker Compose Network               │  │
│  │                                                   │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐ │  │
│  │  │    Dev     │  │  PostgreSQL │  │   ImmuDB   │ │  │
│  │  │ Container  │  │   Service   │  │  Service   │ │  │
│  │  │            │  │             │  │            │ │  │
│  │  │ - Go 1.24  │  │ Port: 5432  │  │ Port: 3322 │ │  │
│  │  │ - Protoc   │  │             │  │      5433  │ │  │
│  │  │ - Tools    │  │             │  │      8080  │ │  │
│  │  └────────────┘  └────────────┘  └────────────┘ │  │
│  │                                                   │  │
│  │              monorepo-network (bridge)            │  │
│  └──────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## Services

### Dev Container
- **Purpose**: Primary development environment
- **Base Image**: Custom Dockerfile with Go and development tools
- **Features**:
  - Go 1.24+ with workspace support
  - Protocol Buffer compiler (protoc) and plugins
  - Homebrew for package management
  - GitLab CLI for repository operations
  - VS Code extensions pre-configured
- **Workspace**: Mounts project root at `/workspace`

### PostgreSQL Service
- **Image**: `postgres:16-alpine`
- **Port**: 5432
- **Database**: `monorepo_dev`
- **Credentials**: postgres/postgres
- **Health Check**: pg_isready
- **Persistent Volume**: `postgres_data`

### ImmuDB Service
- **Image**: `codenotary/immudb:latest`
- **Ports**:
  - 3322: gRPC API
  - 5433: PostgreSQL wire protocol
  - 8080: Web Console UI
  - 9497: Metrics endpoint
- **Health Check**: immuadmin status
- **Persistent Volume**: `immudb_data`
- **Features**:
  - Immutable database for audit logs
  - PostgreSQL compatibility mode
  - Web-based admin console

### Redis Service
- **Image**: `redis:7-alpine`
- **Port**: 6379
- **Persistent Volume**: `redis_data`
- **Health Check**: redis-cli ping
- **Features**:
  - In-memory data store
  - Cache and message broker
  - Append-only file persistence

### Redis Commander
- **Image**: `rediscommander/redis-commander:latest`
- **Port**: 8081
- **Web UI**: http://localhost:8081
- **Credentials**: admin/admin
- **Features**:
  - Web-based Redis management
  - Key browsing and editing
  - Real-time monitoring

### pgAdmin
- **Image**: `dpage/pgadmin4:latest`
- **Port**: 5050 (maps to internal port 80)
- **Web UI**: http://localhost:5050
- **Credentials**: admin@example.com/admin
- **Persistent Volume**: `pgadmin_data`
- **Features**:
  - PostgreSQL database management
  - Query tool and data viewer
  - Server monitoring

## Getting Started

### Prerequisites
1. **Docker Desktop** (Windows/Mac) or Docker Engine (Linux)
2. **VS Code** with Dev Containers extension
3. **Git** for version control

### Initial Setup

1. **Open in Dev Container**:
   ```bash
   # Clone the repository
   git clone <repository-url>
   cd go-mono-repo
   
   # Open VS Code
   code .
   ```

2. **VS Code will prompt**: "Reopen in Container"
   - Click "Reopen in Container"
   - Or use Command Palette: `Dev Containers: Reopen in Container`

3. **Wait for container build**:
   - First run will build the Docker image
   - Subsequent runs use cached image

### Verifying Setup

Once the container is running, verify the environment:

```bash
# Check Go version
go version
# Expected: go version go1.24.x linux/amd64

# Check protoc installation
protoc --version
# Expected: libprotoc 3.x.x

# Test database connections
# PostgreSQL
psql -h postgres -U postgres -d monorepo_dev -c "SELECT version();"

# ImmuDB gRPC
grpcurl -plaintext immudb:3322 immudb.schema.ImmuService/Health

# Check workspace
pwd
# Expected: /workspace
```

## Development Workflow

### Starting Services

The devcontainer automatically starts all services. Individual services can be managed:

```bash
# View running services
docker compose ps

# Restart a specific service
docker compose restart postgres

# View service logs
docker compose logs -f immudb

# Stop all services (preserves data)
docker compose stop

# Start services again
docker compose start
```

### Running Application Services

From within the devcontainer:

```bash
# Start ledger service
make ledger-service

# Start treasury service  
make treasury-service

# Run all services
make all-services

# Health checks
make health-check-all
make liveness-check-all
```

### Database Access

#### PostgreSQL
```bash
# Connect via psql
psql -h postgres -U postgres -d monorepo_dev

# Connection string for applications
postgresql://postgres:postgres@postgres:5432/monorepo_dev

# Web Admin Interface
# Open browser: http://localhost:5050
# Login: admin@example.com/admin
# Add server: Host=postgres, Port=5432, Username=postgres, Password=postgres
```

#### ImmuDB
```bash
# Web Console
# Open browser: http://localhost:8080
# Default credentials: immudb/immudb

# gRPC connection from Go code
conn, err := grpc.Dial("immudb:3322", grpc.WithInsecure())

# PostgreSQL wire protocol
psql -h immudb -p 5433 -U immudb
```

#### Redis
```bash
# Connect via redis-cli
redis-cli -h redis

# Connection string for applications
redis://redis:6379

# Web Admin Interface (Redis Commander)
# Open browser: http://localhost:8081
# Login: admin/admin

# Common commands
redis-cli -h redis ping
redis-cli -h redis info
redis-cli -h redis keys '*'
```

## Configuration Files

### .devcontainer/devcontainer.json
Main configuration file defining:
- Container service to use (`dev`)
- Docker Compose file reference
- VS Code extensions and settings
- User account and mounts
- Environment variables

### .devcontainer/docker-compose.yml
Orchestration file defining:
- Service containers and their configuration
- Network topology
- Volume mounts
- Port mappings
- Health checks

### .devcontainer/Dockerfile
Custom image definition:
- Base image selection
- Development tool installation
- User configuration
- Shell setup

## Troubleshooting

### Container Won't Start

1. **Check Docker status**:
   ```bash
   docker version
   docker compose version
   ```

2. **Clean rebuild**:
   ```bash
   # From host machine
   docker compose -f .devcontainer/docker-compose.yml down -v
   # Then reopen in container
   ```

3. **Check logs**:
   ```bash
   docker compose -f .devcontainer/docker-compose.yml logs
   ```

### Network Issues

1. **Services can't connect**:
   - Use service names (postgres, immudb) not localhost
   - Verify network: `docker network ls`

2. **Port conflicts**:
   - Check for services using ports 5432, 5433, 3322, 8080
   - Modify port mappings in docker-compose.yml if needed

### Performance Issues

1. **Slow file operations** (Windows/Mac):
   - Volume mount uses `:cached` for performance
   - Consider WSL2 on Windows for better performance

2. **Memory limits**:
   - Adjust Docker Desktop memory allocation
   - Check container limits: `docker stats`

## VS Code Extensions

The following extensions are automatically installed:

- **golang.go**: Go language support
- **esbenp.prettier-vscode**: Code formatting
- **eamodio.gitlens**: Git visualization
- **dbaeumer.vscode-eslint**: Linting
- **bierner.markdown-mermaid**: Diagram support
- **ms-vscode.makefile-tools**: Makefile support

## Environment Variables

Set in the devcontainer:

- `NODE_OPTIONS`: --max-old-space-size=4096
- `CLAUDE_CONFIG_DIR`: /home/node/.claude
- `POWERLEVEL9K_DISABLE_GITSTATUS`: true (performance optimization)

## Best Practices

1. **Always develop inside the container** to ensure consistency
2. **Commit devcontainer changes** when adding new tools or dependencies
3. **Use service names** for inter-service communication
4. **Preserve volumes** for database persistence across rebuilds
5. **Document new services** added to docker-compose.yml
6. **Test health checks** when adding new services

## Adding New Services

To add a new service to the devcontainer:

1. Edit `.devcontainer/docker-compose.yml`:
   ```yaml
   services:
     new-service:
       image: service-image:tag
       ports:
         - "host-port:container-port"
       networks:
         - monorepo-network
       environment:
         KEY: value
   ```

2. Update documentation in this file

3. Test the service integration:
   ```bash
   docker compose -f .devcontainer/docker-compose.yml up new-service
   ```

4. Add health checks if applicable

## Security Considerations

- **Never commit secrets** to devcontainer files
- Use environment variables or mounted secret files
- Database passwords are development-only defaults
- Production deployments must use proper secrets management
- Container runs as non-root user (`node`) for security