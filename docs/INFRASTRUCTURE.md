# Infrastructure Architecture

This document describes the infrastructure services and patterns used in the monorepo for both local development and production deployments.

## Overview

Infrastructure services are external dependencies that our application services rely on but are not part of our custom codebase. These include databases, message queues, caching layers, and other foundational services.

## Local Development Infrastructure

Local development uses Docker Compose to provide a consistent, reproducible environment that mirrors production capabilities.

### Directory Structure
```
infrastructure/
├── docker-compose.yml      # Service definitions
├── init-scripts/          # Database initialization scripts
└── README.md             # Quick reference guide
```

### Services

#### PostgreSQL
- **Purpose**: Primary relational database for transactional data
- **Version**: PostgreSQL 16 (Alpine Linux)
- **Container**: `monorepo-postgres`
- **Port**: 5432
- **Default Database**: `monorepo_dev`
- **Credentials**: postgres/postgres (development only)
- **Data Persistence**: Docker volume `postgres_data`
- **Health Checks**: Built-in with 10-second intervals

### Network Architecture

All infrastructure services run on a dedicated Docker bridge network (`monorepo-network`), enabling:
- Service discovery by container name
- Isolation from host network
- Secure inter-service communication
- No port conflicts with host services

### Database Initialization

The PostgreSQL container automatically executes SQL scripts placed in `infrastructure/init-scripts/` on first startup:
- Scripts execute in alphabetical order
- Useful for creating schemas, tables, and seed data
- Scripts only run on initial container creation

Example initialization script structure:
```sql
-- infrastructure/init-scripts/001-create-schemas.sql
CREATE SCHEMA IF NOT EXISTS ledger;
CREATE SCHEMA IF NOT EXISTS treasury;

-- infrastructure/init-scripts/002-create-tables.sql
CREATE TABLE IF NOT EXISTS ledger.accounts (...);

-- infrastructure/init-scripts/003-seed-data.sql
INSERT INTO ledger.accounts (...) VALUES (...);
```

## Production Infrastructure

Production environments use managed AWS services for reliability, scalability, and reduced operational overhead.

### Service Mappings

| Service | Local (Docker) | Production (AWS) |
|---------|---------------|------------------|
| PostgreSQL | postgres:16-alpine | Amazon RDS (PostgreSQL) |
| Redis* | redis:7-alpine | Amazon ElastiCache |
| Message Queue* | rabbitmq:3 | Amazon SQS/SNS |
| Object Storage* | minio | Amazon S3 |

*Future services to be added as needed

### Amazon RDS Configuration

Production PostgreSQL runs on Amazon RDS with:
- **Multi-AZ Deployment**: Automatic failover for high availability
- **Automated Backups**: Daily snapshots with point-in-time recovery
- **Performance Insights**: Query performance monitoring
- **Security**: VPC isolation, encryption at rest and in transit
- **Connection Pooling**: PgBouncer or RDS Proxy for connection management

### Environment-Specific Configuration

Services detect their environment through environment variables:

```go
// Example configuration detection
dbHost := os.Getenv("DB_HOST")
if dbHost == "" {
    dbHost = "localhost" // Local development default
}

dbSSLMode := "disable"
if os.Getenv("ENVIRONMENT") == "production" {
    dbSSLMode = "require"
}
```

## Infrastructure as Code

### Local Development
- **Tool**: Docker Compose
- **Configuration**: `infrastructure/docker-compose.yml`
- **Philosophy**: Simple, fast iteration with minimal setup

### Production Deployment
- **Tool**: Terraform (planned)
- **Configuration**: `infrastructure/terraform/` (future)
- **Philosophy**: Declarative, version-controlled, auditable

## Makefile Integration

The root Makefile provides convenient commands for infrastructure management:

```makefile
# Start infrastructure and services
make dev

# Infrastructure-specific commands
make infrastructure-up      # Start all infrastructure
make infrastructure-down    # Stop all infrastructure
make infrastructure-status  # View running containers
make infrastructure-clean   # Remove containers and data
```

## Connection Strings

### Local Development
```
PostgreSQL: postgresql://postgres:postgres@localhost:5432/monorepo_dev
```

### Production (via Environment Variables)
```
PostgreSQL: postgresql://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=require
```

## Monitoring and Observability

### Local Development
- Docker logs: `docker logs monorepo-postgres`
- Container stats: `docker stats`
- Database logs: Available in container stdout

### Production
- CloudWatch Logs for RDS
- CloudWatch Metrics for performance monitoring
- AWS RDS Performance Insights for query analysis
- Custom application metrics via OpenTelemetry (planned)

## Backup and Recovery

### Local Development
```bash
# Backup
docker exec monorepo-postgres pg_dump -U postgres monorepo_dev > backup.sql

# Restore
docker exec -i monorepo-postgres psql -U postgres monorepo_dev < backup.sql
```

### Production
- Automated daily backups via RDS
- Point-in-time recovery up to 35 days
- Cross-region backup replication for disaster recovery

## Security Considerations

### Local Development
- Default credentials for convenience
- No SSL/TLS enforcement
- Open network access
- Suitable only for development

### Production
- Strong, rotated credentials via AWS Secrets Manager
- SSL/TLS required for all connections
- VPC isolation with security groups
- Database encryption at rest
- Audit logging enabled

## Scaling Strategies

### Vertical Scaling
- **Local**: Adjust Docker resource limits in docker-compose.yml
- **Production**: Modify RDS instance class

### Horizontal Scaling
- **Read Replicas**: For read-heavy workloads
- **Connection Pooling**: PgBouncer or RDS Proxy
- **Caching Layer**: Redis/ElastiCache for frequent queries
- **Database Sharding**: For extreme scale (future consideration)

## Troubleshooting

### Common Issues and Solutions

#### PostgreSQL Won't Start (Local)
```bash
# Check if port is in use
lsof -i :5432

# View container logs
docker logs monorepo-postgres

# Reset completely
make infrastructure-clean
make infrastructure-up
```

#### Connection Refused
```bash
# Verify container is running
docker ps | grep monorepo-postgres

# Test connection directly
docker exec monorepo-postgres pg_isready -U postgres

# Check network
docker network ls
docker network inspect infrastructure_monorepo-network
```

#### Slow Queries (Production)
1. Enable RDS Performance Insights
2. Identify slow queries
3. Add appropriate indexes
4. Consider read replicas for read-heavy operations

## Future Infrastructure Services

Planned additions to the infrastructure stack:

1. **Redis Cache**: Session storage and caching
2. **Message Queue**: Asynchronous processing (RabbitMQ/SQS)
3. **Object Storage**: File uploads and static assets (MinIO/S3)
4. **Search Engine**: Full-text search capabilities (Elasticsearch)
5. **Metrics Storage**: Time-series data (Prometheus/CloudWatch)

Each new service will follow the same pattern:
- Docker Compose for local development
- Managed AWS service for production
- Environment-specific configuration
- Makefile integration for ease of use