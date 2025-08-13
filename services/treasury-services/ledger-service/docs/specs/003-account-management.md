# Account Management Specification

> **Status**: Draft  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-13  
> **Author(s)**: Engineering Team  
> **Reviewer(s)**: Platform Team, Treasury Team  
> **Confluence**: https://example.atlassian.net/wiki/spaces/LEDGER/pages/003/Account+Management  

## Executive Summary

The Ledger Service requires a comprehensive account management system to handle the creation, retrieval, and modification of financial accounts within the ledger. This specification defines the core account structure, API endpoints for CRUD operations, and data persistence patterns that will serve as the foundation for all accounting operations, enabling proper categorization and tracking of financial entities across the system. Currency validation will be handled through integration with the Treasury Service's currency management system to ensure consistency across services.

## Problem Statement

### Current State
The Ledger Service has established database connectivity and migration capabilities (specs 001-002) but lacks the fundamental account structure needed for ledger operations. Without a proper account management system, the service cannot track financial entities, categorize transactions, or provide the basic building blocks for double-entry bookkeeping.

### Desired State
The Ledger Service will implement a robust account management system with well-defined account types following standard accounting principles. The system will provide gRPC endpoints for creating, reading, and updating accounts with proper validation, support external system integration through external IDs, and maintain clean separation of concerns with account-specific logic isolated in its own module.

## Scope

### In Scope
- Account data model with standard accounting types
- Create account endpoint with validation
- Get account by ID endpoint
- Update account endpoint with field masking
- List accounts endpoint with filtering and pagination
- External ID and External Group ID for integration
- Currency code support per account
- Account-specific code organization in /account folder
- Database schema and migrations for accounts table

### Out of Scope
- Account balances (separate feature)
- Transactions and journal entries (separate feature)
- Account deletion (accounts should be deactivated, not deleted)
- Balance calculations and reporting
- Multi-currency conversion
- Account hierarchies or parent-child relationships
- Audit trail and change history (future enhancement)
- Account reconciliation features

## User Stories

### Story 1: Create Account
**As a** financial system integrator  
**I want to** create new accounts in the ledger  
**So that** I can track different financial entities and categories  

**Acceptance Criteria:**
- [ ] Account created with unique system-generated ID
- [ ] Name is required and validated (non-empty, max 255 chars)
- [ ] External ID is required and unique within the system
- [ ] External Group ID is optional
- [ ] Currency code validated against ISO 4217 standards
- [ ] Account type must be one of the valid enum values
- [ ] Duplicate external ID rejected with ALREADY_EXISTS error
- [ ] Invalid account type rejected with INVALID_ARGUMENT error
- [ ] Created timestamp automatically set
- [ ] Account returned in response with all fields

### Story 2: Retrieve Account
**As a** financial system user  
**I want to** retrieve account details by ID  
**So that** I can view account information and metadata  

**Acceptance Criteria:**
- [ ] Account retrieved by system ID
- [ ] All account fields returned in response
- [ ] NOT_FOUND error for non-existent account ID
- [ ] INVALID_ARGUMENT error for malformed ID
- [ ] Response time < 100ms for single account lookup

### Story 3: Update Account
**As a** account administrator  
**I want to** update account properties  
**So that** I can maintain accurate account information  

**Acceptance Criteria:**
- [ ] Update account name
- [ ] Update external group ID (including setting to null)
- [ ] Update account type (with validation)
- [ ] Field mask support for partial updates
- [ ] External ID cannot be changed (immutable)
- [ ] Currency code cannot be changed (immutable)
- [ ] Updated timestamp automatically set
- [ ] NOT_FOUND error for non-existent account
- [ ] INVALID_ARGUMENT for invalid field values
- [ ] Optimistic locking to prevent concurrent update conflicts

### Story 4: List Accounts
**As a** financial analyst  
**I want to** list and filter accounts  
**So that** I can find specific accounts or analyze account distribution  

**Acceptance Criteria:**
- [ ] List all accounts with pagination
- [ ] Filter by account type
- [ ] Filter by currency code
- [ ] Filter by external group ID
- [ ] Search by name (partial match)
- [ ] Sort by name or created date
- [ ] Page size configurable (default 50, max 200)
- [ ] Page token for cursor-based pagination
- [ ] Total count included in response

### Story 5: Retrieve Account by External ID
**As a** external system integrator  
**I want to** retrieve accounts using external system IDs  
**So that** I can map between systems without storing internal IDs  

**Acceptance Criteria:**
- [ ] Account retrieved by external ID
- [ ] Same response format as get by ID
- [ ] NOT_FOUND error for non-existent external ID
- [ ] Case-sensitive external ID matching
- [ ] Response time < 100ms with proper indexing

## Technical Design

### Architecture Overview
The account management system will be implemented as a dedicated module within the ledger service, with all account-related code organized in a `/account` subdirectory. This includes the gRPC server implementation, business logic manager, data access layer, and validation utilities. The system will use ImmuDB for persistence with proper indexing for performance.

### API Design

#### RPC Methods

```protobuf
// Account management service
// Spec: docs/specs/003-account-management.md
service AccountService {
    // Create a new account
    rpc CreateAccount (CreateAccountRequest) returns (CreateAccountResponse) {}
    
    // Get account by ID
    rpc GetAccount (GetAccountRequest) returns (GetAccountResponse) {}
    
    // Get account by external ID
    rpc GetAccountByExternalId (GetAccountByExternalIdRequest) returns (GetAccountByExternalIdResponse) {}
    
    // Update account fields
    rpc UpdateAccount (UpdateAccountRequest) returns (UpdateAccountResponse) {}
    
    // List accounts with filtering
    rpc ListAccounts (ListAccountsRequest) returns (ListAccountsResponse) {}
}
```

#### Data Models

```protobuf
// Account represents a financial account in the ledger
message Account {
    string id = 1;                          // System-generated UUID
    string name = 2;                        // Human-readable account name
    string external_id = 3;                 // External system identifier (unique)
    string external_group_id = 4;           // External group identifier (optional)
    string currency_code = 5;               // ISO 4217 currency code
    AccountType account_type = 6;           // Standard accounting type
    google.protobuf.Timestamp created_at = 7;  // Creation timestamp
    google.protobuf.Timestamp updated_at = 8;  // Last update timestamp
    int64 version = 9;                      // Version for optimistic locking
}

// Standard accounting types
enum AccountType {
    ACCOUNT_TYPE_UNSPECIFIED = 0;  // Unknown or unspecified
    ACCOUNT_TYPE_ASSET = 1;        // Asset accounts
    ACCOUNT_TYPE_LIABILITY = 2;    // Liability accounts
    ACCOUNT_TYPE_REVENUE = 3;      // Revenue accounts
    ACCOUNT_TYPE_EXPENSE = 4;      // Expense accounts
    ACCOUNT_TYPE_EQUITY = 5;       // Equity accounts
}

// Create account request
message CreateAccountRequest {
    string name = 1;                // Required: Account name
    string external_id = 2;         // Required: External identifier
    string external_group_id = 3;   // Optional: External group
    string currency_code = 4;       // Required: ISO 4217 code
    AccountType account_type = 5;   // Required: Account type
}

message CreateAccountResponse {
    Account account = 1;
}

// Get account request
message GetAccountRequest {
    string account_id = 1;  // System account ID
}

message GetAccountResponse {
    Account account = 1;
}

// Get by external ID request
message GetAccountByExternalIdRequest {
    string external_id = 1;
}

message GetAccountByExternalIdResponse {
    Account account = 1;
}

// Update account request
message UpdateAccountRequest {
    string account_id = 1;                     // Required: Account to update
    Account account = 2;                        // Updated account data
    google.protobuf.FieldMask update_mask = 3;  // Fields to update
}

message UpdateAccountResponse {
    Account account = 1;
}

// List accounts request
message ListAccountsRequest {
    int32 page_size = 1;            // Number of results (max 200)
    string page_token = 2;           // Pagination token
    AccountType account_type = 3;    // Filter by type
    string currency_code = 4;        // Filter by currency
    string external_group_id = 5;    // Filter by group
    string name_search = 6;          // Search in name (partial match)
}

message ListAccountsResponse {
    repeated Account accounts = 1;
    string next_page_token = 2;
    int32 total_count = 3;
}
```

### Database Schema

```sql
-- Migration: 003_create_accounts_table
-- Spec: docs/specs/003-account-management.md
CREATE TABLE IF NOT EXISTS accounts (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    external_group_id VARCHAR(255),
    currency_code VARCHAR(3) NOT NULL,
    account_type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    version INTEGER DEFAULT 1,
    
    -- Constraints
    UNIQUE(external_id),
    CHECK(account_type IN ('ASSET', 'LIABILITY', 'REVENUE', 'EXPENSE', 'EQUITY'))
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_accounts_external_id ON accounts(external_id);
CREATE INDEX IF NOT EXISTS idx_accounts_external_group_id ON accounts(external_group_id);
CREATE INDEX IF NOT EXISTS idx_accounts_type ON accounts(account_type);
CREATE INDEX IF NOT EXISTS idx_accounts_currency ON accounts(currency_code);
CREATE INDEX IF NOT EXISTS idx_accounts_name ON accounts(name);
```

### Code Organization

```
services/treasury-services/ledger-service/
├── account/                      # Account module directory
│   ├── server.go                # gRPC server implementation
│   ├── manager.go               # Business logic manager
│   ├── repository.go            # Data access layer
│   ├── validator.go             # Input validation
│   ├── mapper.go                # Proto <-> DB mapping
│   └── errors.go                # Account-specific errors
├── proto/
│   └── ledger_service.proto     # Updated with account messages
└── main.go                      # Wire up account server
```

### Implementation Details

#### Account Manager
```go
// AccountManager handles account business logic
// Spec: docs/specs/003-account-management.md
type AccountManager struct {
    repo   *AccountRepository
    logger *zap.Logger
}

// CreateAccount creates a new account
// Spec: docs/specs/003-account-management.md#story-1-create-account
func (m *AccountManager) CreateAccount(ctx context.Context, req *CreateAccountRequest) (*Account, error) {
    // 1. Validate input
    // 2. Check external ID uniqueness
    // 3. Generate UUID
    // 4. Create account record
    // 5. Return created account
}

// GetAccount retrieves account by ID
// Spec: docs/specs/003-account-management.md#story-2-retrieve-account
func (m *AccountManager) GetAccount(ctx context.Context, accountID string) (*Account, error) {
    // 1. Validate ID format
    // 2. Query database
    // 3. Handle not found
    // 4. Return account
}
```

#### Validation Rules
```go
// ValidateCreateAccount validates account creation request
func ValidateCreateAccount(req *CreateAccountRequest) error {
    // Name: required, 1-255 characters
    // External ID: required, 1-255 characters
    // Currency Code: required, validated via Treasury Service
    // Account Type: required, valid enum value
}

// CurrencyValidator validates currency codes via Treasury Service
// Spec: docs/specs/003-account-management.md
type CurrencyValidator struct {
    treasuryClient treasury.CurrencyServiceClient
    cache          map[string]bool  // Simple cache for valid codes
    cacheTTL       time.Duration
    mu             sync.RWMutex
}

// ValidateCurrency checks if currency code is valid
func (v *CurrencyValidator) ValidateCurrency(ctx context.Context, code string) error {
    // Check cache first
    if v.isCached(code) {
        return nil
    }
    
    // Query Treasury Service
    resp, err := v.treasuryClient.GetCurrency(ctx, &treasury.GetCurrencyRequest{
        Code: code,
    })
    if err != nil {
        if status.Code(err) == codes.NotFound {
            return fmt.Errorf("invalid currency code: %s", code)
        }
        return fmt.Errorf("failed to validate currency: %w", err)
    }
    
    // Check if currency is active
    if !resp.Currency.IsActive {
        return fmt.Errorf("currency %s is not active", code)
    }
    
    // Update cache
    v.updateCache(code)
    return nil
}
```

### Error Handling

| Error Scenario | gRPC Code | Error Message |
|---------------|-----------|---------------|
| Missing required field | INVALID_ARGUMENT | "field {name} is required" |
| Invalid account type | INVALID_ARGUMENT | "invalid account type: {type}" |
| Invalid currency code | INVALID_ARGUMENT | "invalid currency code: {code}" |
| Duplicate external ID | ALREADY_EXISTS | "account with external_id {id} already exists" |
| Account not found | NOT_FOUND | "account {id} not found" |
| Database error | INTERNAL | "internal database error" |
| Concurrent update conflict | ABORTED | "account was modified, retry update" |

### Performance Requirements

- Create account: < 50ms average, < 200ms P99
- Get account by ID: < 20ms average, < 100ms P99
- Get by external ID: < 20ms average, < 100ms P99
- Update account: < 50ms average, < 200ms P99
- List accounts (50 items): < 100ms average, < 500ms P99

## Implementation Plan

### Phase 1: Foundation (Day 1)
- [ ] Create /account directory structure
- [ ] Define protobuf messages and service
- [ ] Generate protobuf code
- [ ] Create database migration for accounts table
- [ ] Implement AccountRepository with basic CRUD

### Phase 2: Core Operations (Day 2)
- [ ] Implement AccountManager business logic
- [ ] Add input validation for all operations
- [ ] Implement Create and Get operations
- [ ] Add unique constraint handling
- [ ] Write unit tests for manager

### Phase 3: Advanced Operations (Day 3)
- [ ] Implement Update with field masks
- [ ] Add optimistic locking for updates
- [ ] Implement List with filtering
- [ ] Add pagination support
- [ ] Implement GetByExternalId

### Phase 4: Integration (Day 4)
- [ ] Wire up gRPC server
- [ ] Integrate with main service
- [ ] Add logging and metrics
- [ ] Integration testing
- [ ] Performance testing

## Dependencies

### Service Dependencies
- ImmuDB connection (spec 001-immudb-connection.md)
- Database migration system (spec 002-database-migrations.md)
- Treasury Service for currency validation (spec treasury-service/003-currency-management.md)
  - Note: Ledger health checks should verify Treasury liveness (not health) to avoid circular dependency
- Health check system for monitoring

### Go Dependencies
```go
github.com/google/uuid            // UUID generation
github.com/codenotary/immudb      // Database client
google.golang.org/protobuf        // Protobuf support
google.golang.org/grpc            // gRPC framework
go.uber.org/zap                   // Structured logging
example.com/go-mono-repo/proto/treasury  // Treasury service proto for currency validation
```

### External Dependencies
- Treasury Service for currency validation (via gRPC)
- ImmuDB instance with write permissions

## Security Considerations

### Input Validation
- Strict validation of all input fields
- SQL injection prevention through parameterized queries
- Length limits on string fields to prevent DoS
- Enum validation for account types

### Access Control
- Future: Add authorization checks per operation
- Future: Tenant isolation for multi-tenant deployments
- Audit logging of all account modifications

### Data Privacy
- No PII stored in account records
- External IDs should not contain sensitive data
- Account names should be business identifiers only

## Testing Strategy

### Unit Tests
- [ ] Account validation logic
- [ ] Manager business logic
- [ ] Repository CRUD operations
- [ ] Proto/DB mapping functions
- [ ] Error handling paths

### Integration Tests
- [ ] Full gRPC endpoint testing
- [ ] Database constraint validation
- [ ] Concurrent update handling
- [ ] Pagination correctness
- [ ] Filter combinations

### Performance Tests
- [ ] Load testing with 10K accounts
- [ ] Concurrent create/update operations
- [ ] Index effectiveness validation
- [ ] Query performance under load

## Monitoring & Observability

### Metrics
- account_operations_total{operation, status} - Counter of operations
- account_operation_duration_seconds{operation} - Histogram of latencies
- accounts_total{type, currency} - Gauge of total accounts
- account_validation_errors_total{field} - Counter of validation failures

### Logs
- Account creation with external_id
- Account updates with changed fields
- Validation failures with details
- Database errors with context

### Alerts
- Account creation rate > 1000/minute (potential abuse)
- Get account latency P99 > 200ms
- Validation error rate > 10%
- Database connection failures

## Documentation Updates

Upon implementation, update:
- [ ] Ledger service README with account endpoints
- [ ] CLAUDE.md with account management patterns
- [ ] Migration README with account table details
- [ ] API documentation with example requests

## Open Questions

1. Should we support soft deletion instead of permanent deletion in the future?
2. Do we need to support multiple external IDs per account for different systems?
3. Should account types be configurable or strictly limited to standard types?
4. Do we need account status (active/inactive) for this phase?
5. Should we add metadata/tags field for extensibility?
6. ~Should we cache currency validation results, and if so, for how long?~ Decided: 5-minute TTL
7. How should we handle treasury service unavailability during account creation?

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-13 | No delete operation | Accounts should never be deleted for audit trail | Team |
| 2025-01-13 | External ID immutable | Prevents breaking external system references | Team |
| 2025-01-13 | Currency immutable | Changing currency would invalidate historical data | Team |
| 2025-01-13 | Use /account folder | Better code organization and separation | Team |
| 2025-01-13 | Standard types only | Following GAAP accounting principles | Team |
| 2025-01-13 | Validate via Treasury Service | Single source of truth for currency data | Team |
| 2025-01-13 | Cache currency validation | Reduce Treasury Service load, 5-minute TTL | Team |
| 2025-01-13 | Check Treasury liveness only | Avoid circular dependency in health checks | Team |

## References

- [ImmuDB Connection Spec](./001-immudb-connection.md)
- [Database Migration Spec](./002-database-migrations.md)
- [Protobuf Patterns](../../../../docs/PROTOBUF_PATTERNS.md)
- [GAAP Account Types](https://www.investopedia.com/terms/g/gaap.asp)
- [ISO 4217 Currency Codes](https://www.iso.org/iso-4217-currency-codes.html)

## Appendix

### Example API Calls

#### Create Account
```json
// Request
{
    "name": "Cash on Hand",
    "external_id": "EXT-CASH-001",
    "external_group_id": "CURRENT-ASSETS",
    "currency_code": "USD",
    "account_type": "ACCOUNT_TYPE_ASSET"
}

// Response
{
    "account": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "Cash on Hand",
        "external_id": "EXT-CASH-001",
        "external_group_id": "CURRENT-ASSETS",
        "currency_code": "USD",
        "account_type": "ACCOUNT_TYPE_ASSET",
        "created_at": "2025-01-13T10:00:00Z",
        "updated_at": "2025-01-13T10:00:00Z",
        "version": 1
    }
}
```

#### List Accounts with Filters
```json
// Request
{
    "page_size": 50,
    "account_type": "ACCOUNT_TYPE_ASSET",
    "currency_code": "USD",
    "name_search": "Cash"
}

// Response
{
    "accounts": [
        {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "name": "Cash on Hand",
            "external_id": "EXT-CASH-001",
            "currency_code": "USD",
            "account_type": "ACCOUNT_TYPE_ASSET"
        },
        {
            "id": "660e8400-e29b-41d4-a716-446655440001",
            "name": "Cash in Bank",
            "external_id": "EXT-CASH-002",
            "currency_code": "USD",
            "account_type": "ACCOUNT_TYPE_ASSET"
        }
    ],
    "next_page_token": "eyJvZmZzZXQiOjUwfQ==",
    "total_count": 125
}
```