# ImmuDB Connection and Management Specification

> **Status**: Ready for Review  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-11  
> **Author(s)**: Engineering Team  
> **Reviewer(s)**: Platform Team, Security Team  
> **Confluence**: https://example.atlassian.net/wiki/spaces/LEDGER/pages/001/ImmuDB+Connection  

## Executive Summary

The Ledger Service requires an immutable, cryptographically verifiable data store for managing financial transactions, account balances, and audit trails with tamper-proof guarantees. This specification defines the implementation of ImmuDB database connectivity, connection pooling, and health monitoring integration for the Ledger Service, ensuring data immutability, cryptographic verification, and proper dependency management.

## Problem Statement

### Current State
The Ledger Service currently lacks database connectivity, limiting it to in-memory operations. This prevents data persistence, makes the service stateless between restarts, and critically lacks the immutability and cryptographic verification required for financial ledger operations where audit trails and tamper-proof records are essential.

### Desired State
The Ledger Service will establish secure, pooled connections to an ImmuDB database, with proper configuration management, health monitoring, graceful connection handling, cryptographic verification capabilities, and integration with the existing health check system per the Health Check specification. The solution will leverage ImmuDB's unique features including immutability, cryptographic proof, and time-travel queries.

## Scope

### In Scope
- ImmuDB gRPC connection configuration and management
- Connection pooling with configurable parameters for gRPC
- Database health check implementation
- Environment-based configuration (.env files)
- Integration with existing health check endpoints
- Graceful connection handling and retry logic
- ImmuDB-specific features (verification, immutability checks)
- Connection metrics and monitoring
- ImmuDB client SDK integration

### Out of Scope
- Database schema design (separate spec)
- Data access patterns and repositories (separate spec)
- Read/write splitting and replication
- Database backup and recovery procedures (handled by ImmuDB)
- Query optimization and indexing strategies
- Multi-tenant database isolation
- PostgreSQL wire protocol usage (gRPC preferred for native features)

## User Stories

### Story 1: ImmuDB Configuration Management
**As a** DevOps engineer  
**I want to** configure ImmuDB connections via environment variables  
**So that** I can manage different environments without code changes  

**Acceptance Criteria:**
- [ ] ImmuDB configuration read from .env file
- [ ] Support for gRPC connection parameters
- [ ] Support for ImmuDB authentication settings
- [ ] .env.example file with all required variables documented
- [ ] Configuration validation on startup
- [ ] Sensitive values never logged

### Story 2: Connection Pool Management
**As a** service operator  
**I want to** have a properly configured connection pool for gRPC  
**So that** the service can handle concurrent requests efficiently  

**Acceptance Criteria:**
- [ ] Configurable max connections (default: 25)
- [ ] Configurable max idle connections (default: 5)
- [ ] Connection lifetime management (default: 1 hour)
- [ ] Connection pool metrics available
- [ ] Graceful pool shutdown on service stop
- [ ] gRPC-specific optimizations enabled
- [ ] Follow ImmuDB community best practices for pooling

### Story 3: ImmuDB Health Monitoring
**As a** platform engineer  
**I want to** monitor ImmuDB connectivity and database health  
**So that** I can detect and respond to database issues  

**Acceptance Criteria:**
- [ ] ImmuDB health integrated into Health.GetHealth endpoint
- [ ] Connection pool status included in health response
- [ ] Response time metrics for health checks
- [ ] ImmuDB-specific metrics (verified transactions, current root)
- [ ] Circuit breaker for failed connections
- [ ] Proper error reporting without exposing credentials

### Story 4: Cryptographic Verification
**As a** security engineer  
**I want to** verify the cryptographic integrity of stored data  
**So that** I can ensure data has not been tampered with  

**Acceptance Criteria:**
- [ ] Automatic verification of critical transactions
- [ ] On-demand verification capability
- [ ] Verification metrics in health checks
- [ ] Configurable verification levels
- [ ] Clear logging of verification failures

### Story 5: Graceful Degradation
**As a** service owner  
**I want to** handle ImmuDB connectivity issues gracefully  
**So that** the service can provide partial functionality when possible  

**Acceptance Criteria:**
- [ ] Service starts even if ImmuDB is initially unavailable
- [ ] Retry logic with exponential backoff
- [ ] Read-only operations when write connection fails
- [ ] Clear error messages for database issues
- [ ] Automatic recovery when ImmuDB becomes available

### Story 6: Local Development Support
**As a** developer  
**I want to** easily connect to a local ImmuDB instance  
**So that** I can develop and test features locally  

**Acceptance Criteria:**
- [ ] Docker Compose configuration for local ImmuDB
- [ ] Default local development configuration in .env.example
- [ ] Make targets for database operations
- [ ] Documentation for local setup
- [ ] Access to ImmuDB Web Console UI

## Technical Design

### Architecture Overview
The Ledger Service will use the official ImmuDB Go SDK (`github.com/codenotary/immudb/pkg/client`) for database connectivity via gRPC. Connection configuration will be managed through environment variables, with the connection pool integrated into the service's health monitoring system.

### Configuration Structure

#### Environment Variables (.env)
```bash
# ImmuDB Configuration
IMMUDB_HOST=localhost
IMMUDB_PORT=3322
IMMUDB_DATABASE=ledgerdb
IMMUDB_USERNAME=ledger_user
IMMUDB_PASSWORD=ledger_pass

# Connection Configuration
IMMUDB_MAX_CONNECTIONS=25
IMMUDB_MAX_IDLE_CONNECTIONS=5
IMMUDB_CONNECTION_MAX_LIFETIME=3600  # seconds
IMMUDB_CONNECTION_MAX_IDLE_TIME=900  # seconds

# Security Configuration  
IMMUDB_VERIFY_TRANSACTIONS=true  # Always true per spec decision
IMMUDB_SERVER_SIGNING_PUB_KEY=""  # Optional: for server signature verification
IMMUDB_CLIENT_KEY_PATH=""         # Optional: for mTLS
IMMUDB_CLIENT_CERT_PATH=""        # Optional: for mTLS

# Health Check Configuration
IMMUDB_HEALTH_CHECK_INTERVAL=30  # seconds
IMMUDB_PING_TIMEOUT=5  # seconds

# Performance Configuration
IMMUDB_CHUNK_SIZE=64              # Size of chunks for batch operations
IMMUDB_MAX_RECV_MSG_SIZE=4194304  # 4MB default
```

### Implementation Components

#### 1. Configuration Loading (config.go)
```go
// ImmuDBConfig holds ImmuDB connection parameters
// Spec: docs/specs/001-immudb-connection.md
type ImmuDBConfig struct {
    Host                  string
    Port                  int
    Database              string
    Username              string
    Password              string
    MaxConnections        int
    MaxIdleConnections    int
    ConnectionMaxLifetime time.Duration
    ConnectionMaxIdleTime time.Duration
    VerifyTransactions    bool
    ServerSigningPubKey   string
    ClientKeyPath         string
    ClientCertPath        string
    HealthCheckInterval   time.Duration
    PingTimeout           time.Duration
    ChunkSize             int
    MaxRecvMsgSize        int
}

// Add to existing Config struct
type Config struct {
    // ... existing fields ...
    
    // ImmuDB Configuration
    ImmuDB *ImmuDBConfig `envconfig:"-"`
}

// LoadImmuDBConfig loads ImmuDB configuration from environment
// Spec: docs/specs/001-immudb-connection.md#story-1-immudb-configuration-management
func LoadImmuDBConfig() (*ImmuDBConfig, error) {
    // Implementation details...
}
```

#### 2. ImmuDB Connection Manager
```go
// ImmuDBManager manages ImmuDB connections and health
// Spec: docs/specs/001-immudb-connection.md
type ImmuDBManager struct {
    client  client.ImmuClient
    config  *ImmuDBConfig
    mu      sync.RWMutex
    
    // Connection metrics
    connectTime     time.Time
    lastHealthCheck time.Time
    isHealthy       bool
    errorCount      int64
    verifiedTxCount int64
    lastRootHash    []byte
}

// Connect establishes ImmuDB connection with retry logic
// Spec: docs/specs/001-immudb-connection.md#story-5-graceful-degradation
func (im *ImmuDBManager) Connect(ctx context.Context) error {
    opts := client.DefaultOptions().
        WithAddress(im.config.Host).
        WithPort(im.config.Port).
        WithDatabase(im.config.Database).
        WithUsername(im.config.Username).
        WithPassword(im.config.Password).
        WithMaxRecvMsgSize(im.config.MaxRecvMsgSize)
    
    // Configure verification if enabled
    if im.config.VerifyTransactions {
        opts = opts.WithServerSigningPubKey(im.config.ServerSigningPubKey)
    }
    
    // Implementation with exponential backoff
}

// VerifyTransaction verifies a transaction's cryptographic proof
// Spec: docs/specs/001-immudb-connection.md#story-4-cryptographic-verification
func (im *ImmuDBManager) VerifyTransaction(ctx context.Context, txID uint64) error {
    // Verification implementation
}

// GetConnectionStats returns current connection statistics
// Spec: docs/specs/001-immudb-connection.md#story-2-connection-pool-management
func (im *ImmuDBManager) GetConnectionStats() *ConnectionStats {
    // Return connection statistics
}
```

#### 3. Health Check Integration
```go
// ImmuDBChecker implements DependencyChecker for ImmuDB
// Spec: docs/specs/001-immudb-connection.md#story-3-immudb-health-monitoring
type ImmuDBChecker struct {
    manager *ImmuDBManager
}

// Check implements health check for ImmuDB
func (i *ImmuDBChecker) Check(ctx context.Context) *pb.DependencyHealth {
    startTime := time.Now()
    
    dep := &pb.DependencyHealth{
        Name:       "immudb-primary",
        Type:       pb.DependencyType_DATABASE,
        IsCritical: true,
        Config: &pb.DependencyConfig{
            Hostname:     i.manager.config.Host,
            Port:         int32(i.manager.config.Port),
            Protocol:     "grpc",
            DatabaseName: i.manager.config.Database,
        },
    }
    
    // Perform health check with timeout
    ctx, cancel := context.WithTimeout(ctx, i.manager.config.PingTimeout)
    defer cancel()
    
    // Check connection and get database state
    if health, err := i.manager.client.Health(ctx); err != nil {
        dep.Status = pb.ServiceStatus_UNHEALTHY
        dep.Message = "ImmuDB connection failed"
        dep.Error = err.Error()
    } else {
        dep.Status = pb.ServiceStatus_HEALTHY
        dep.Message = fmt.Sprintf("ImmuDB healthy, verified txs: %d", 
                                 i.manager.verifiedTxCount)
        
        // Add ImmuDB-specific metrics
        dep.Metadata = map[string]string{
            "pending_count": fmt.Sprintf("%d", health.PendingCount),
            "last_committed_tx": fmt.Sprintf("%d", health.LastCommittedTxID),
            "verified_transactions": fmt.Sprintf("%d", i.manager.verifiedTxCount),
        }
        
        dep.Config.PoolInfo = &pb.ConnectionPoolInfo{
            MaxConnections:    int32(i.manager.config.MaxConnections),
            ActiveConnections: int32(i.manager.getActiveConnections()),
            IdleConnections:   int32(i.manager.getIdleConnections()),
        }
    }
    
    dep.ResponseTimeMs = time.Since(startTime).Milliseconds()
    dep.LastCheck = time.Now().Format(time.RFC3339)
    
    return dep
}
```

### Connection String Format
ImmuDB uses gRPC connections, not traditional connection strings:
```
Host: localhost
Port: 3322
Database: ledgerdb
```

### Error Handling

| Error Type | Handling Strategy | Recovery Action |
|------------|-------------------|-----------------| 
| Connection Refused | Exponential backoff | Retry with increasing delays |
| Authentication Failed | Fail fast | Log error, check configuration |
| Connection Pool Exhausted | Queue or reject | Return 503, increase pool size |
| Query Timeout | Cancel and retry | Use context with timeout |
| Network Partition | Circuit breaker | Fail fast, periodic retry |
| Verification Failed | Alert and log | Investigate potential tampering |

### Performance Requirements

- Connection establishment: < 5 seconds
- Health check ping: < 100ms (P99)
- Transaction verification: < 200ms (P99)
- Graceful shutdown: < 30 seconds
- Recovery time after failure: < 60 seconds

### ImmuDB-Specific Features

#### Immutability Guarantees
- All data written is immutable
- Each transaction gets a unique ID
- Cryptographic linking between transactions
- Merkle tree structure for verification

#### Verification Capabilities
- Client-side verification of all transactions
- Server signature verification (optional)
- Inclusion proof verification
- Consistency proof verification

#### Time Travel Queries
- Query data at specific transaction IDs
- Historical data access
- Audit trail capabilities

## Implementation Plan

### Phase 1: Basic Connectivity (Week 1)
- [ ] Create .env and .env.example files with ImmuDB settings
- [ ] Implement ImmuDB configuration loading in config.go
- [ ] Add ImmuDB Go SDK dependency
- [ ] Create ImmuDBManager struct
- [ ] Implement basic connection logic

### Phase 2: Health Integration (Week 1)
- [ ] Implement ImmuDBChecker
- [ ] Integrate with HealthServer
- [ ] Add connection metrics
- [ ] Update health check responses
- [ ] Test health check scenarios

### Phase 3: Verification & Security (Week 2)
- [ ] Implement transaction verification
- [ ] Add verification metrics
- [ ] Configure mTLS support (optional)
- [ ] Implement audit logging
- [ ] Test cryptographic features

### Phase 4: Resilience & Testing (Week 2)
- [ ] Add exponential backoff retry
- [ ] Implement circuit breaker
- [ ] Add connection pool monitoring
- [ ] Implement graceful shutdown
- [ ] Performance testing

## Dependencies

### Service Dependencies
- Infrastructure: ImmuDB instance from docker-compose.yml
- Health Service: Integration per spec 003-health-check-liveness.md

### Go Dependencies
```go
github.com/codenotary/immudb/pkg/client  // ImmuDB Go SDK
github.com/joho/godotenv                 // .env file loading
google.golang.org/grpc                   // gRPC support
```

### Infrastructure Dependencies
- ImmuDB latest (via Docker Compose)
- Network connectivity to ImmuDB
- Proper DNS resolution

## Security Considerations

### Credential Management
- Database passwords never logged
- Use environment variables for secrets
- Consider secret management system for production
- Rotate credentials regularly

### Connection Security
- gRPC with TLS by default
- Optional mTLS for enhanced security
- Server signature verification available
- Encrypted data in transit

### Data Integrity
- Cryptographic verification of all data
- Tamper-proof audit logs
- Immutable transaction history
- Merkle tree proofs

### Access Control
- Principle of least privilege for database user
- Separate read/write users if needed
- Database-level permissions
- Audit trail for all operations

## Testing Strategy

### Unit Tests
- [ ] Configuration loading with validation
- [ ] Connection establishment
- [ ] Health check logic
- [ ] Error handling scenarios
- [ ] Verification logic

### Integration Tests
- [ ] Actual ImmuDB connection
- [ ] Transaction verification
- [ ] Health check with real database
- [ ] Recovery after database restart
- [ ] Concurrent connection handling

### Performance Tests
- [ ] Connection pool saturation
- [ ] Health check under load
- [ ] Verification performance
- [ ] Recovery time measurement
- [ ] Graceful shutdown timing

### Security Tests
- [ ] Verification of tampered data detection
- [ ] mTLS connection testing
- [ ] Authentication failure handling
- [ ] Audit log completeness

## Monitoring & Observability

### Metrics
- ImmuDB connection status (up/down)
- Connection pool utilization
- Transaction execution time (P50, P95, P99)
- Verification success/failure rate
- Connection establishment time
- Health check response time
- Error rates by type
- Verified transaction count

### Logs
- Connection establishment/failure
- Configuration loading
- Health check results
- Pool exhaustion events
- Verification failures
- Query errors (without sensitive data)

### Alerts
- ImmuDB unreachable for > 1 minute
- Connection pool > 80% utilized
- Health check failing repeatedly
- Authentication failures
- Verification failures detected
- Unusual error rate increase

### Dashboards
- Connection pool visualization
- ImmuDB health status
- Transaction performance trends
- Verification metrics
- Error rate graphs
- Recovery time tracking

## Documentation Updates

Upon implementation, update:
- [ ] Update ledger-service README with ImmuDB setup
- [ ] Add ImmuDB configuration to CLAUDE.md
- [ ] Create runbook for ImmuDB connectivity issues
- [ ] Update docker-compose.yml documentation
- [ ] Add troubleshooting guide

## Migration Strategy

### Initial Setup
1. No existing database - fresh installation
2. ImmuDB automatically creates database on first connection
3. Create ledger_user with appropriate permissions
4. Configure verification settings
5. Verify connectivity

### Future Migrations
- ImmuDB handles schema evolution internally
- No traditional migrations needed
- Data is immutable - no updates or deletes
- New indexes can be added without affecting existing data

## Open Questions

~~1. Should we use PostgreSQL wire protocol as fallback for compatibility?~~ **Resolved: No, gRPC only**  
~~2. Do we need separate pools for read vs write operations?~~ **Resolved: Follow ImmuDB best practices**  
~~3. What verification level should be default (all transactions vs critical only)?~~ **Resolved: Verify all transactions**  
~~4. Should we implement client-side caching of verified transactions?~~ **Resolved: Yes, future spec**  
~~5. How should we handle ImmuDB Web Console access in production?~~ **Resolved: Configure when reaching production**  

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-11 | Use gRPC connection over PostgreSQL wire | Native features, better performance, full API access | Team |
| 2025-01-11 | Enable transaction verification by default | Critical for ledger integrity | Team |
| 2025-01-11 | Start with single connection pool | Simplicity for MVP, can split later if needed | Team |
| 2025-01-11 | ImmuDB as critical dependency | Ledger service requires immutable persistence | Team |
| 2025-01-11 | Support graceful degradation | Service should start even if ImmuDB initially unavailable | Team |
| 2025-01-11 | No PostgreSQL wire protocol fallback | Keep it simple - all up or all down | Product |
| 2025-01-11 | Follow ImmuDB best practices for pooling | Use community recommended patterns for read/write operations | Team |
| 2025-01-11 | Verify all transactions by default | Start with maximum security, scale back if needed | Product |
| 2025-01-11 | Client-side caching deferred | Future spec will address verified transaction caching | Product |
| 2025-01-11 | Production console config deferred | Will configure when reaching production deployment | Product |

## References

- [Health Check Specification](../../../docs/specs/003-health-check-liveness.md)
- [ImmuDB Documentation](https://docs.immudb.io)
- [ImmuDB Go SDK](https://github.com/codenotary/immudb/tree/master/pkg/client)
- [Docker Compose Configuration](../../../../.devcontainer/docker-compose.yml)
- [Service Development Guide](../../../../docs/SERVICE_DEVELOPMENT.md)

## Appendix

### Example Health Check Response with ImmuDB
```json
{
  "status": "HEALTHY",
  "message": "Service is fully operational",
  "dependencies": [
    {
      "name": "immudb-primary",
      "type": "DATABASE",
      "status": "HEALTHY",
      "is_critical": true,
      "message": "ImmuDB healthy, verified txs: 1523",
      "config": {
        "hostname": "localhost",
        "port": 3322,
        "protocol": "grpc",
        "database_name": "ledgerdb",
        "pool_info": {
          "max_connections": 25,
          "active_connections": 3,
          "idle_connections": 2,
          "wait_count": 0
        }
      },
      "metadata": {
        "pending_count": "0",
        "last_committed_tx": "1523",
        "verified_transactions": "1523"
      },
      "last_check": "2025-01-11T10:30:00Z",
      "response_time_ms": 12
    },
    {
      "name": "treasury-service",
      "type": "GRPC_SERVICE",
      "status": "HEALTHY",
      "is_critical": false,
      "message": "Treasury service is healthy",
      "config": {
        "hostname": "localhost",
        "port": 50052,
        "protocol": "grpc"
      },
      "last_check": "2025-01-11T10:30:00Z",
      "response_time_ms": 15
    }
  ],
  "checked_at": "2025-01-11T10:30:00Z",
  "check_duration_ms": 27
}
```

### Docker Compose Integration
```yaml
services:
  ledger-service:
    build: ./services/treasury-services/ledger-service
    environment:
      - IMMUDB_HOST=immudb
      - IMMUDB_PORT=3322
      - IMMUDB_DATABASE=ledgerdb
      - IMMUDB_USERNAME=ledger_user
      - IMMUDB_PASSWORD=${LEDGER_IMMUDB_PASSWORD}
      - IMMUDB_VERIFY_TRANSACTIONS=true
    depends_on:
      immudb:
        condition: service_healthy
    networks:
      - monorepo-network
```

### ImmuDB Web Console Access
For local development, the ImmuDB Web Console is available at:
- URL: http://localhost:8080
- Default credentials: immudb/immudb
- Features:
  - Database exploration
  - Transaction history
  - Verification tools
  - Performance metrics
  - SQL query interface