# Database Connection and Management Specification

> **Status**: Draft  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-10  
> **Author(s)**: Engineering Team  
> **Reviewer(s)**: Platform Team, DBA Team  
> **Confluence**: https://example.atlassian.net/wiki/spaces/TREASURY/pages/001/Database+Connection  

## Executive Summary

The Treasury Service requires a persistent data store for managing financial transactions, account balances, and audit trails. This specification defines the implementation of PostgreSQL database connectivity, connection pooling, and health monitoring integration for the Treasury Service, ensuring reliable data persistence and proper dependency management.

## Problem Statement

### Current State
The Treasury Service currently lacks database connectivity, limiting it to in-memory operations. This prevents data persistence, makes the service stateless between restarts, and blocks implementation of critical financial features requiring ACID compliance and durability.

### Desired State
The Treasury Service will establish secure, pooled connections to a PostgreSQL database, with proper configuration management, health monitoring, graceful connection handling, and integration with the existing health check system per the Health Check specification.

## Scope

### In Scope
- PostgreSQL connection configuration and management
- Connection pooling with configurable parameters
- Database health check implementation
- Environment-based configuration (.env files)
- Integration with existing health check endpoints
- Graceful connection handling and retry logic
- Database migration framework setup
- Connection metrics and monitoring

### Out of Scope
- Database schema design (separate spec)
- Data access patterns and repositories (separate spec)
- Read/write splitting and replication
- Database backup and recovery procedures
- Query optimization and indexing strategies
- Multi-tenant database isolation

## User Stories

### Story 1: Database Configuration Management
**As a** DevOps engineer  
**I want to** configure database connections via environment variables  
**So that** I can manage different environments without code changes  

**Acceptance Criteria:**
- [ ] Database configuration read from .env file
- [ ] Support for all standard PostgreSQL connection parameters
- [ ] .env.example file with all required variables documented
- [ ] Configuration validation on startup
- [ ] Sensitive values never logged

### Story 2: Connection Pool Management
**As a** service operator  
**I want to** have a properly configured connection pool  
**So that** the service can handle concurrent requests efficiently  

**Acceptance Criteria:**
- [ ] Configurable max connections (default: 25)
- [ ] Configurable max idle connections (default: 5)
- [ ] Connection lifetime management (default: 1 hour)
- [ ] Connection pool metrics available
- [ ] Graceful pool shutdown on service stop

### Story 3: Database Health Monitoring
**As a** platform engineer  
**I want to** monitor database connectivity health  
**So that** I can detect and respond to database issues  

**Acceptance Criteria:**
- [ ] Database health integrated into Health.GetHealth endpoint
- [ ] Connection pool status included in health response
- [ ] Response time metrics for health checks
- [ ] Circuit breaker for failed connections
- [ ] Proper error reporting without exposing credentials

### Story 4: Graceful Degradation
**As a** service owner  
**I want to** handle database connectivity issues gracefully  
**So that** the service can provide partial functionality when possible  

**Acceptance Criteria:**
- [ ] Service starts even if database is initially unavailable
- [ ] Retry logic with exponential backoff
- [ ] Read-only operations when write connection fails
- [ ] Clear error messages for database issues
- [ ] Automatic recovery when database becomes available

### Story 5: Local Development Support
**As a** developer  
**I want to** easily connect to a local PostgreSQL instance  
**So that** I can develop and test features locally  

**Acceptance Criteria:**
- [ ] Docker Compose configuration for local PostgreSQL
- [ ] Default local development configuration in .env.example
- [ ] Database initialization scripts
- [ ] Make targets for database operations
- [ ] Documentation for local setup

## Technical Design

### Architecture Overview
The Treasury Service will use the standard `database/sql` package with the `pgx` driver for PostgreSQL connectivity. Connection configuration will be managed through environment variables, with the connection pool integrated into the service's health monitoring system.

### Configuration Structure

#### Environment Variables (.env)
```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=treasury_db
DB_USER=treasury_user
DB_PASSWORD=treasury_pass
DB_SCHEMA=public
DB_SSL_MODE=disable

# Connection Pool Configuration
DB_MAX_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5
DB_CONNECTION_MAX_LIFETIME=3600  # seconds
DB_CONNECTION_MAX_IDLE_TIME=900  # seconds

# Health Check Configuration
DB_HEALTH_CHECK_INTERVAL=30  # seconds
DB_PING_TIMEOUT=5  # seconds
```

### Implementation Components

#### 1. Configuration Loading (config.go)
```go
// DatabaseConfig holds database connection parameters
// Spec: docs/specs/001-database-connection.md
type DatabaseConfig struct {
    Host                  string
    Port                  int
    Database              string
    User                  string
    Password              string
    Schema                string
    SSLMode               string
    MaxConnections        int
    MaxIdleConnections    int
    ConnectionMaxLifetime time.Duration
    ConnectionMaxIdleTime time.Duration
    HealthCheckInterval   time.Duration
    PingTimeout           time.Duration
}

// LoadDatabaseConfig loads database configuration from environment
// Spec: docs/specs/001-database-connection.md#story-1-database-configuration-management
func LoadDatabaseConfig() (*DatabaseConfig, error) {
    // Implementation details...
}
```

#### 2. Database Connection Manager
```go
// DatabaseManager manages database connections and health
// Spec: docs/specs/001-database-connection.md
type DatabaseManager struct {
    db     *sql.DB
    config *DatabaseConfig
    mu     sync.RWMutex
    
    // Connection metrics
    connectTime    time.Time
    lastHealthCheck time.Time
    isHealthy      bool
    errorCount     int64
}

// Connect establishes database connection with retry logic
// Spec: docs/specs/001-database-connection.md#story-4-graceful-degradation
func (dm *DatabaseManager) Connect(ctx context.Context) error {
    // Implementation with exponential backoff
}

// GetConnectionPoolStats returns current pool statistics
// Spec: docs/specs/001-database-connection.md#story-2-connection-pool-management
func (dm *DatabaseManager) GetConnectionPoolStats() *ConnectionPoolInfo {
    // Return pool statistics
}
```

#### 3. Health Check Integration
```go
// PostgreSQLChecker implements DependencyChecker for database
// Spec: docs/specs/001-database-connection.md#story-3-database-health-monitoring
type PostgreSQLChecker struct {
    manager *DatabaseManager
}

// Check implements health check for PostgreSQL
func (p *PostgreSQLChecker) Check(ctx context.Context) *pb.DependencyHealth {
    startTime := time.Now()
    
    dep := &pb.DependencyHealth{
        Name:       "postgresql-primary",
        Type:       pb.DependencyType_DATABASE,
        IsCritical: true,
        Config: &pb.DependencyConfig{
            Hostname:     p.manager.config.Host,
            Port:         int32(p.manager.config.Port),
            Protocol:     "postgresql",
            DatabaseName: p.manager.config.Database,
            SchemaName:   p.manager.config.Schema,
        },
    }
    
    // Perform health check with timeout
    ctx, cancel := context.WithTimeout(ctx, p.manager.config.PingTimeout)
    defer cancel()
    
    if err := p.manager.db.PingContext(ctx); err != nil {
        dep.Status = pb.ServiceStatus_UNHEALTHY
        dep.Message = "Database connection failed"
        dep.Error = err.Error()
    } else {
        dep.Status = pb.ServiceStatus_HEALTHY
        dep.Message = "Database connection healthy"
        dep.Config.PoolInfo = p.manager.GetConnectionPoolStats()
    }
    
    dep.ResponseTimeMs = time.Since(startTime).Milliseconds()
    dep.LastCheck = time.Now().Format(time.RFC3339)
    
    return dep
}
```

### Connection String Format
```
postgresql://user:password@host:port/database?sslmode=disable&search_path=schema
```

### Error Handling

| Error Type | Handling Strategy | Recovery Action |
|------------|-------------------|-----------------|
| Connection Refused | Exponential backoff | Retry with increasing delays |
| Authentication Failed | Fail fast | Log error, check configuration |
| Connection Pool Exhausted | Queue or reject | Return 503, increase pool size |
| Query Timeout | Cancel and retry | Use context with timeout |
| Network Partition | Circuit breaker | Fail fast, periodic retry |

### Performance Requirements

- Connection establishment: < 5 seconds
- Health check ping: < 100ms (P99)
- Connection pool acquisition: < 10ms (P99)
- Graceful shutdown: < 30 seconds
- Recovery time after failure: < 60 seconds

## Implementation Plan

### Phase 1: Basic Connectivity (Week 1)
- [ ] Create .env and .env.example files
- [ ] Implement configuration loading in config.go
- [ ] Add pgx driver dependency
- [ ] Create DatabaseManager struct
- [ ] Implement basic connection logic

### Phase 2: Health Integration (Week 1)
- [ ] Implement PostgreSQLChecker
- [ ] Integrate with HealthServer
- [ ] Add connection pool metrics
- [ ] Update health check responses
- [ ] Test health check scenarios

### Phase 3: Resilience (Week 2)
- [ ] Add exponential backoff retry
- [ ] Implement circuit breaker
- [ ] Add connection pool monitoring
- [ ] Implement graceful shutdown
- [ ] Add recovery mechanisms

### Phase 4: Documentation & Testing (Week 2)
- [ ] Unit tests for all components
- [ ] Integration tests with test database
- [ ] Update service documentation
- [ ] Create runbook for database issues
- [ ] Performance testing

## Dependencies

### Service Dependencies
- Infrastructure: PostgreSQL instance from docker-compose.yml
- Health Service: Integration per spec 003-health-check-liveness.md

### Go Dependencies
```go
github.com/jackc/pgx/v5        // PostgreSQL driver
github.com/jackc/pgx/v5/stdlib // database/sql compatibility
github.com/joho/godotenv       // .env file loading
```

### Infrastructure Dependencies
- PostgreSQL 16 (via Docker Compose)
- Network connectivity to database
- Proper DNS resolution

## Security Considerations

### Credential Management
- Database passwords never logged
- Use environment variables for secrets
- Consider secret management system for production
- Rotate credentials regularly

### Connection Security
- Use SSL/TLS in production (sslmode=require)
- Implement connection encryption
- Use prepared statements to prevent SQL injection
- Audit database access

### Access Control
- Principle of least privilege for database user
- Separate read/write users if needed
- Row-level security where applicable
- Audit trail for all modifications

## Testing Strategy

### Unit Tests
- [ ] Configuration loading with validation
- [ ] Connection string building
- [ ] Health check logic
- [ ] Error handling scenarios
- [ ] Pool statistics calculation

### Integration Tests
- [ ] Actual database connection
- [ ] Connection pool behavior
- [ ] Health check with real database
- [ ] Recovery after database restart
- [ ] Concurrent connection handling

### Performance Tests
- [ ] Connection pool saturation
- [ ] Health check under load
- [ ] Recovery time measurement
- [ ] Graceful shutdown timing

### Chaos Engineering
- [ ] Database connection loss
- [ ] Slow database responses
- [ ] Connection pool exhaustion
- [ ] Network latency injection

## Monitoring & Observability

### Metrics
- Database connection status (up/down)
- Connection pool utilization
- Query execution time (P50, P95, P99)
- Connection establishment time
- Health check response time
- Error rates by type

### Logs
- Connection establishment/failure
- Configuration loading
- Health check results
- Pool exhaustion events
- Query errors (without sensitive data)

### Alerts
- Database unreachable for > 1 minute
- Connection pool > 80% utilized
- Health check failing repeatedly
- Authentication failures
- Unusual error rate increase

### Dashboards
- Connection pool visualization
- Database health status
- Query performance trends
- Error rate graphs
- Recovery time tracking

## Documentation Updates

Upon implementation, update:
- [ ] Update treasury-service README with database setup
- [ ] Add database configuration to CLAUDE.md
- [ ] Create runbook for database connectivity issues
- [ ] Update docker-compose.yml documentation
- [ ] Add troubleshooting guide

## Migration Strategy

### Initial Setup
1. No existing database - fresh installation
2. Create treasury_db database
3. Create treasury_user with appropriate permissions
4. Run initial schema migrations
5. Verify connectivity

### Future Migrations
- Use golang-migrate for schema versioning
- Migrations in services/treasury-services/treasury-service/migrations/
- Up and down migrations for all changes
- Automated migration on service start (configurable)

## Open Questions

1. Should we implement read/write connection splitting from the start?
2. Do we need connection pooling per operation type (read/write)?
3. Should health checks use a separate connection pool?
4. What should be the default connection pool size for production?

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-10 | Use pgx/v5 driver | Modern, performant, good PostgreSQL feature support | Team |
| 2025-01-10 | Start with single connection pool | Simplicity for MVP, can split later if needed | Team |
| 2025-01-10 | Health checks use main pool | Avoid connection overhead, monitor actual pool | Team |
| 2025-01-10 | Database as critical dependency | Treasury service requires persistence | Team |
| 2025-01-10 | Support graceful degradation | Service should start even if DB initially unavailable | Team |

## References

- [Health Check Specification](../../../docs/specs/003-health-check-liveness.md)
- [PostgreSQL Connection Documentation](https://www.postgresql.org/docs/current/libpq-connect.html)
- [pgx Driver Documentation](https://github.com/jackc/pgx)
- [Docker Compose Configuration](../../../../infrastructure/docker-compose.yml)
- [Service Development Guide](../../../../docs/SERVICE_DEVELOPMENT.md)

## Appendix

### Example Health Check Response with Database
```json
{
  "status": "HEALTHY",
  "message": "Service is fully operational",
  "dependencies": [
    {
      "name": "postgresql-primary",
      "type": "DATABASE",
      "status": "HEALTHY",
      "is_critical": true,
      "message": "Database connection healthy",
      "config": {
        "hostname": "localhost",
        "port": 5432,
        "protocol": "postgresql",
        "database_name": "treasury_db",
        "schema_name": "public",
        "pool_info": {
          "max_connections": 25,
          "active_connections": 3,
          "idle_connections": 2,
          "wait_count": 0
        }
      },
      "last_check": "2025-01-10T10:30:00Z",
      "response_time_ms": 8
    },
    {
      "name": "ledger-service",
      "type": "GRPC_SERVICE",
      "status": "HEALTHY",
      "is_critical": true,
      "message": "Ledger service is healthy",
      "config": {
        "hostname": "localhost",
        "port": 50051,
        "protocol": "grpc"
      },
      "last_check": "2025-01-10T10:30:00Z",
      "response_time_ms": 15
    }
  ],
  "checked_at": "2025-01-10T10:30:00Z",
  "check_duration_ms": 23
}
```

### Docker Compose Integration
```yaml
services:
  treasury-service:
    build: ./services/treasury-services/treasury-service
    environment:
      - DB_HOST=postgres
      - DB_PORT=5432
      - DB_NAME=treasury_db
      - DB_USER=treasury_user
      - DB_PASSWORD=${TREASURY_DB_PASSWORD}
    depends_on:
      postgres:
        condition: service_healthy
    networks:
      - monorepo-network
```