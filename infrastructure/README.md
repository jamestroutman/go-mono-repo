# Infrastructure Services

This directory contains the configuration and setup for infrastructure services used in local development. These services run alongside our application services but are not part of our custom codebase.

## Services

### PostgreSQL
- **Local Development**: Docker container running PostgreSQL 16
- **Production**: Amazon RDS
- **Default Connection**: `postgresql://postgres:postgres@localhost:5432/monorepo_dev`
- **Container Name**: `monorepo-postgres`

### ImmuDB
- **Local Development**: Docker container running ImmuDB latest
- **Production**: Managed ImmuDB instance or self-hosted
- **gRPC Port**: 3322 (native ImmuDB gRPC API)
- **PostgreSQL Wire Protocol**: 5433 (PostgreSQL-compatible interface)
- **Web Console UI**: http://localhost:8080
- **Metrics Port**: 9497
- **Container Name**: `monorepo-immudb`
- **Default Credentials**: immudb/immudb (username/password)

## Quick Start

Start all infrastructure services:
```bash
make infrastructure-up
```

Stop all infrastructure services:
```bash
make infrastructure-down
```

View infrastructure status:
```bash
make infrastructure-status
```

Clean infrastructure (removes volumes):
```bash
make infrastructure-clean
```

## Docker Compose Configuration

The `docker-compose.yml` file defines all infrastructure services needed for local development:

- **PostgreSQL**: Main database server
  - Port: 5432
  - Username: postgres
  - Password: postgres
  - Default Database: monorepo_dev
  - Data persisted in Docker volume

- **ImmuDB**: Immutable database server
  - gRPC Port: 3322 (native ImmuDB API)
  - PostgreSQL Wire Protocol Port: 5433
  - Web Console UI Port: 8080
  - Metrics Port: 9497
  - Username: immudb
  - Password: immudb
  - Data persisted in Docker volume

## Database Initialization

Place any SQL initialization scripts in `infrastructure/init-scripts/` and they will be automatically executed when the PostgreSQL container is first created. Scripts are executed in alphabetical order.

## Network Configuration

All infrastructure services run on the `monorepo-network` Docker network, allowing services to communicate with each other using container names.

## Environment Variables

Infrastructure services use the following default environment variables for local development:

| Service | Variable | Default Value |
|---------|----------|---------------|
| PostgreSQL | POSTGRES_USER | postgres |
| PostgreSQL | POSTGRES_PASSWORD | postgres |
| PostgreSQL | POSTGRES_DB | monorepo_dev |
| ImmuDB | IMMUDB_ADDRESS | 0.0.0.0 |
| ImmuDB | IMMUDB_PORT | 3322 |
| ImmuDB | IMMUDB_WEBCONSOLE | true |
| ImmuDB | IMMUDB_WEBCONSOLE_PORT | 8080 |

## Production Considerations

In production environments:
- PostgreSQL is replaced with Amazon RDS
- ImmuDB runs as a managed service or self-hosted cluster with proper authentication
- Connection strings and credentials are managed through environment variables
- Infrastructure is provisioned through Infrastructure as Code (e.g., Terraform)

## Troubleshooting

### PostgreSQL Won't Start
- Check if port 5432 is already in use: `lsof -i :5432`
- View container logs: `docker logs monorepo-postgres`
- Ensure Docker is running: `docker ps`

### Reset Database
To completely reset the database:
```bash
make infrastructure-clean
make infrastructure-up
```

### Connection Issues
- Verify the container is running: `docker ps | grep monorepo-postgres`
- Test connection: `psql -h localhost -U postgres -d monorepo_dev`
- Check container health: `docker inspect monorepo-postgres --format='{{.State.Health.Status}}'`

### ImmuDB Won't Start
- Check if ports are already in use: `lsof -i :3322` or `lsof -i :8080`
- View container logs: `docker logs monorepo-immudb`
- Ensure Docker has enough resources allocated

### ImmuDB Web Console Access
- Access the web console at: http://localhost:8080
- Default credentials: immudb/immudb
- If the console doesn't load, check that port 8080 is not blocked by firewall
- Verify the container is healthy: `docker inspect monorepo-immudb --format='{{.State.Health.Status}}'`