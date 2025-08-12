# Currency Management Specification

> **Status**: Draft  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-12  
> **Author(s)**: Engineering Team  
> **Reviewer(s)**: Treasury Team, Platform Team  
> **Confluence**: https://example.atlassian.net/wiki/spaces/TREASURY/pages/003/Currency+Management  

## Executive Summary

The Treasury Service requires a comprehensive currency management system to serve as the foundation for all treasury assets and financial operations. This specification defines a unified currency database with full CRUD operations, ISO 4217 compliance, and referential integrity constraints to ensure data consistency across the system.

## Problem Statement

### Current State
The Treasury Service lacks a centralized currency management system, making it impossible to properly track and manage multi-currency treasury operations. Without a unified currency database following international standards, the service cannot accurately represent financial assets, perform currency conversions, or maintain consistency across treasury operations.

### Desired State
The Treasury Service will implement a comprehensive currency management system with ISO 4217 compliant currency definitions, full CRUD operations with appropriate authorization, referential integrity enforcement to prevent orphaned references, complete audit trail for all currency modifications, and support for both active and historical currencies.

## Scope

### In Scope
- Currency table schema with ISO 4217 compliance
- Full CRUD operations (Create, Read, Update, Delete) via gRPC
- Referential integrity checks for safe deletion
- ISO 4217 metadata fields (numeric code, minor units, country codes)
- Currency status management (active/inactive/deprecated)
- Audit fields for tracking changes
- Support for cryptocurrency extensions
- Bulk operations for initial data loading
- Currency validation rules and constraints
- Migration scripts for currency table creation

### Out of Scope
- Exchange rate management (separate specification)
- Currency conversion calculations (separate specification)
- Multi-tenancy currency configurations
- Currency symbol rendering and localization
- Historical exchange rate tracking
- Currency basket or composite currency support
- Real-time currency feed integration

## User Stories

### Story 1: Create New Currency
**As a** Treasury administrator  
**I want to** add new currencies to the system  
**So that** I can support new markets and asset types  

**Acceptance Criteria:**
- [ ] Can create currency with ISO code, name, and metadata
- [ ] Validation ensures ISO code format (3 uppercase letters)
- [ ] Numeric code must be unique if provided
- [ ] Minor units default to 2 if not specified
- [ ] Audit fields automatically populated
- [ ] Duplicate currency codes rejected with ALREADY_EXISTS error

### Story 2: Query Currency Information
**As a** Treasury system user  
**I want to** retrieve currency information by various criteria  
**So that** I can properly process financial transactions  

**Acceptance Criteria:**
- [ ] Can get currency by ISO code (primary lookup)
- [ ] Can get currency by numeric code
- [ ] Can list all active currencies
- [ ] Can list currencies by country code
- [ ] Can filter currencies by status
- [ ] Response includes all metadata fields
- [ ] Pagination support for list operations

### Story 3: Update Currency Metadata
**As a** Treasury administrator  
**I want to** update currency information  
**So that** I can maintain accurate currency data  

**Acceptance Criteria:**
- [ ] Can update currency name and description
- [ ] Can update minor units for precision changes
- [ ] Can update status (active/inactive/deprecated)
- [ ] Cannot modify ISO code (immutable)
- [ ] Cannot modify numeric code once set
- [ ] Audit trail tracks all changes
- [ ] Version field prevents concurrent modifications

### Story 4: Deactivate Currency
**As a** Treasury administrator  
**I want to** deactivate currencies no longer in active use  
**So that** I can maintain accurate currency status  

**Acceptance Criteria:**
- [ ] Can update currency status to inactive/deprecated
- [ ] Foreign key constraints prevent hard deletion (ON DELETE RESTRICT)
- [ ] Soft delete option available (status = deleted)
- [ ] Deactivated currencies excluded from active lists
- [ ] Audit trail records status changes with user and timestamp
- [ ] Historical data with deactivated currencies remains intact

### Story 5: Bulk Currency Operations
**As a** System administrator  
**I want to** load multiple currencies at once  
**So that** I can efficiently initialize the system  

**Acceptance Criteria:**
- [ ] Can import currencies from ISO 4217 dataset
- [ ] Bulk create validates all entries before applying
- [ ] Transaction ensures all-or-nothing operation
- [ ] Progress reporting for large imports
- [ ] Duplicate handling options (skip/update/error)
- [ ] Import summary with success/failure counts

## Technical Design

### Architecture Overview
The currency management system will be implemented as part of the Treasury Service, using PostgreSQL for storage with strong consistency guarantees, gRPC APIs following established patterns, and integration with the existing migration and health check systems.

### Database Schema

```sql
-- Currency table with ISO 4217 compliance
-- Spec: docs/specs/003-currency-management.md
CREATE TABLE IF NOT EXISTS treasury.currencies (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- ISO 4217 fields
    code CHAR(3) NOT NULL,                    -- ISO 4217 alphabetic code (USD, EUR, GBP)
    numeric_code CHAR(3),                      -- ISO 4217 numeric code (840, 978, 826)
    name VARCHAR(100) NOT NULL,                -- Official currency name
    minor_units SMALLINT NOT NULL DEFAULT 2,   -- Decimal places (2 for USD, 0 for JPY)
    
    -- Additional metadata
    symbol VARCHAR(10),                        -- Currency symbol ($, €, £)
    symbol_position VARCHAR(10) DEFAULT 'before', -- before/after amount
    country_codes TEXT[],                      -- Array of ISO 3166 country codes
    is_active BOOLEAN NOT NULL DEFAULT true,   -- Whether currency is currently active
    is_crypto BOOLEAN NOT NULL DEFAULT false,  -- Whether this is a cryptocurrency
    
    -- Status management
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, inactive, deprecated, deleted
    activated_at TIMESTAMP WITH TIME ZONE,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    version INTEGER NOT NULL DEFAULT 1,
    
    -- Constraints
    CONSTRAINT uk_currencies_code UNIQUE (code),
    CONSTRAINT uk_currencies_numeric_code UNIQUE (numeric_code),
    CONSTRAINT chk_currencies_code_format CHECK (code ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_currencies_numeric_code_format CHECK (numeric_code IS NULL OR numeric_code ~ '^[0-9]{3}$'),
    CONSTRAINT chk_currencies_minor_units CHECK (minor_units >= 0 AND minor_units <= 8),
    CONSTRAINT chk_currencies_status CHECK (status IN ('active', 'inactive', 'deprecated', 'deleted'))
);

-- Indexes for performance
CREATE INDEX idx_currencies_code ON treasury.currencies(code) WHERE status != 'deleted';
CREATE INDEX idx_currencies_numeric_code ON treasury.currencies(numeric_code) WHERE numeric_code IS NOT NULL AND status != 'deleted';
CREATE INDEX idx_currencies_status ON treasury.currencies(status);
CREATE INDEX idx_currencies_country_codes ON treasury.currencies USING GIN(country_codes);
CREATE INDEX idx_currencies_is_active ON treasury.currencies(is_active) WHERE is_active = true;

-- Trigger for updated_at
CREATE TRIGGER update_currencies_updated_at 
    BEFORE UPDATE ON treasury.currencies
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Note: Currency deletion is not expected in production as currencies are 
-- permanent reference data. Foreign keys should use ON DELETE RESTRICT to 
-- prevent accidental deletion. The reference tracking pattern shown here
-- would be more appropriate for transient treasury features like:
-- - Temporary trading strategies
-- - Short-term investment positions  
-- - Expired financial instruments
-- - Cancelled trade orders
-- For those features, a reference tracking table would provide visibility
-- before deletion. For currencies, we rely on FK constraints.
```

### API Design

#### Protocol Buffer Definition

```protobuf
// Currency management service
// Spec: docs/specs/003-currency-management.md
syntax = "proto3";

package treasury;

import "google/protobuf/timestamp.proto";
import "google/protobuf/field_mask.proto";

option go_package = "example.com/go-mono-repo/proto/treasury";

// Currency represents an ISO 4217 compliant currency
message Currency {
    string id = 1;                              // UUID
    string code = 2;                            // ISO 4217 code (USD, EUR)
    string numeric_code = 3;                    // ISO 4217 numeric code
    string name = 4;                            // Official name
    int32 minor_units = 5;                      // Decimal places
    string symbol = 6;                          // Currency symbol
    string symbol_position = 7;                 // before/after
    repeated string country_codes = 8;          // ISO 3166 codes
    bool is_active = 9;                         // Active status
    bool is_crypto = 10;                        // Cryptocurrency flag
    CurrencyStatus status = 11;                 // Current status
    google.protobuf.Timestamp activated_at = 12;
    google.protobuf.Timestamp deactivated_at = 13;
    google.protobuf.Timestamp created_at = 14;
    google.protobuf.Timestamp updated_at = 15;
    string created_by = 16;
    string updated_by = 17;
    int32 version = 18;                         // Optimistic locking
}

enum CurrencyStatus {
    CURRENCY_STATUS_UNSPECIFIED = 0;
    CURRENCY_STATUS_ACTIVE = 1;
    CURRENCY_STATUS_INACTIVE = 2;
    CURRENCY_STATUS_DEPRECATED = 3;
    CURRENCY_STATUS_DELETED = 4;
}

// Request/Response messages
message CreateCurrencyRequest {
    string code = 1;                            // Required
    string numeric_code = 2;                    // Optional
    string name = 3;                            // Required
    int32 minor_units = 4;                      // Default 2
    string symbol = 5;
    repeated string country_codes = 6;
    bool is_crypto = 7;
}

message CreateCurrencyResponse {
    Currency currency = 1;
}

message GetCurrencyRequest {
    oneof identifier {
        string code = 1;                        // Primary lookup
        string numeric_code = 2;                // Alternative lookup
        string id = 3;                          // UUID lookup
    }
}

message GetCurrencyResponse {
    Currency currency = 1;
}

message UpdateCurrencyRequest {
    string code = 1;                            // Required (identifies currency)
    google.protobuf.FieldMask update_mask = 2;  // Fields to update
    string name = 3;
    int32 minor_units = 4;
    string symbol = 5;
    repeated string country_codes = 6;
    CurrencyStatus status = 7;
    int32 version = 8;                          // For optimistic locking
}

message UpdateCurrencyResponse {
    Currency currency = 1;
}

message DeactivateCurrencyRequest {
    string code = 1;                            // Required
    CurrencyStatus status = 2;                  // New status (inactive/deprecated/deleted)
    string updated_by = 3;                      // User making the change
    int32 version = 4;                          // For optimistic locking
}

message DeactivateCurrencyResponse {
    bool success = 1;
    Currency currency = 2;                      // Updated currency
}

message ListCurrenciesRequest {
    CurrencyStatus status = 1;                  // Filter by status
    bool is_active = 2;                         // Filter by active flag
    bool is_crypto = 3;                         // Filter cryptocurrencies
    string country_code = 4;                    // Filter by country
    int32 page_size = 5;                        // Pagination
    string page_token = 6;                      // Pagination token
}

message ListCurrenciesResponse {
    repeated Currency currencies = 1;
    string next_page_token = 2;
    int32 total_count = 3;
}

message BulkCreateCurrenciesRequest {
    repeated CreateCurrencyRequest currencies = 1;
    bool skip_duplicates = 2;                   // Skip existing currencies
    bool update_existing = 3;                   // Update if exists
}

message BulkCreateCurrenciesResponse {
    int32 created_count = 1;
    int32 updated_count = 2;
    int32 skipped_count = 3;
    repeated string errors = 4;
}

// Currency service definition
service CurrencyService {
    // Create a new currency
    // Spec: docs/specs/003-currency-management.md#story-1-create-new-currency
    rpc CreateCurrency(CreateCurrencyRequest) returns (CreateCurrencyResponse);
    
    // Get currency information
    // Spec: docs/specs/003-currency-management.md#story-2-query-currency-information
    rpc GetCurrency(GetCurrencyRequest) returns (GetCurrencyResponse);
    
    // Update currency metadata
    // Spec: docs/specs/003-currency-management.md#story-3-update-currency-metadata
    rpc UpdateCurrency(UpdateCurrencyRequest) returns (UpdateCurrencyResponse);
    
    // Deactivate currency (soft delete only - hard delete not supported)
    // Spec: docs/specs/003-currency-management.md#story-4-deactivate-currency
    rpc DeactivateCurrency(DeactivateCurrencyRequest) returns (DeactivateCurrencyResponse);
    
    // List currencies with filters
    // Spec: docs/specs/003-currency-management.md#story-2-query-currency-information
    rpc ListCurrencies(ListCurrenciesRequest) returns (ListCurrenciesResponse);
    
    // Bulk create currencies
    // Spec: docs/specs/003-currency-management.md#story-5-bulk-currency-operations
    rpc BulkCreateCurrencies(BulkCreateCurrenciesRequest) returns (BulkCreateCurrenciesResponse);
}
```

### Implementation Structure

```go
// CurrencyManager handles currency operations
// Spec: docs/specs/003-currency-management.md
type CurrencyManager struct {
    db     *sql.DB
    logger *log.Logger
}

// Currency represents a currency entity
// Spec: docs/specs/003-currency-management.md
type Currency struct {
    ID            uuid.UUID
    Code          string    // ISO 4217 code
    NumericCode   *string   // Optional numeric code
    Name          string
    MinorUnits    int
    Symbol        *string
    SymbolPosition string
    CountryCodes  []string
    IsActive      bool
    IsCrypto      bool
    Status        CurrencyStatus
    ActivatedAt   *time.Time
    DeactivatedAt *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
    CreatedBy     string
    UpdatedBy     string
    Version       int
}

// CreateCurrency creates a new currency
// Spec: docs/specs/003-currency-management.md#story-1-create-new-currency
func (cm *CurrencyManager) CreateCurrency(ctx context.Context, req *pb.CreateCurrencyRequest) (*Currency, error) {
    // Validate ISO code format
    if !isValidISOCode(req.Code) {
        return nil, status.Error(codes.InvalidArgument, "invalid ISO code format")
    }
    
    // Check for duplicates
    exists, err := cm.currencyExists(ctx, req.Code)
    if err != nil {
        return nil, status.Error(codes.Internal, "failed to check currency existence")
    }
    if exists {
        return nil, status.Error(codes.AlreadyExists, "currency already exists")
    }
    
    // Insert currency
    currency := &Currency{
        ID:           uuid.New(),
        Code:         req.Code,
        Name:         req.Name,
        MinorUnits:   defaultIfZero(req.MinorUnits, 2),
        Symbol:       req.Symbol,
        CountryCodes: req.CountryCodes,
        IsCrypto:     req.IsCrypto,
        Status:       CurrencyStatusActive,
        IsActive:     true,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
        Version:      1,
    }
    
    // Execute insert
    err = cm.insertCurrency(ctx, currency)
    if err != nil {
        return nil, status.Error(codes.Internal, "failed to create currency")
    }
    
    return currency, nil
}

// DeactivateCurrency updates currency status
// Spec: docs/specs/003-currency-management.md#story-4-deactivate-currency
func (cm *CurrencyManager) DeactivateCurrency(ctx context.Context, req *pb.DeactivateCurrencyRequest) error {
    // Note: Hard deletion is not supported as currencies are permanent reference data
    // Foreign keys with ON DELETE RESTRICT will prevent deletion of referenced currencies
    
    // Update status to inactive/deprecated/deleted
    query := `
        UPDATE treasury.currencies 
        SET status = $1, 
            is_active = false,
            deactivated_at = CURRENT_TIMESTAMP,
            updated_at = CURRENT_TIMESTAMP,
            updated_by = $2,
            version = version + 1
        WHERE code = $3 AND version = $4
    `
    
    result, err := cm.db.ExecContext(ctx, query, req.Status, req.UpdatedBy, req.Code, req.Version)
    if err != nil {
        return status.Error(codes.Internal, "failed to update currency status")
    }
    
    if rows, _ := result.RowsAffected(); rows == 0 {
        return status.Error(codes.Aborted, "version conflict or currency not found")
    }
    
    return nil
}
```

### Error Handling

| Error Code | Description | HTTP Status | Example |
|------------|-------------|-------------|---------|
| INVALID_ARGUMENT | Invalid currency code format | 400 | "XY" instead of "XYZ" |
| ALREADY_EXISTS | Currency code already exists | 409 | Duplicate USD entry |
| NOT_FOUND | Currency not found | 404 | Unknown currency code |
| FAILED_PRECONDITION | Cannot delete due to references | 412 | Currency used in accounts |
| INTERNAL | Database or system error | 500 | Connection failure |
| ABORTED | Optimistic locking conflict | 409 | Concurrent update detected |

### ISO 4217 Compliance

The system will maintain compliance with ISO 4217 standards:

1. **Code Format**: 3 uppercase letters (regex: `^[A-Z]{3}$`)
2. **Numeric Code**: 3 digits, unique per currency
3. **Minor Units**: Standard decimal places (0-8)
4. **Reserved Codes**: 
   - XTS: Testing
   - XXX: No currency
   - XAU, XAG, XPT, XPD: Precious metals

### Migration Scripts

```sql
-- Migration: 000003_create_currencies_table.up.sql
-- Spec: docs/specs/003-currency-management.md

BEGIN;

-- Create currencies table
CREATE TABLE IF NOT EXISTS treasury.currencies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code CHAR(3) NOT NULL,
    numeric_code CHAR(3),
    name VARCHAR(100) NOT NULL,
    minor_units SMALLINT NOT NULL DEFAULT 2,
    symbol VARCHAR(10),
    symbol_position VARCHAR(10) DEFAULT 'before',
    country_codes TEXT[],
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_crypto BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    activated_at TIMESTAMP WITH TIME ZONE,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    version INTEGER NOT NULL DEFAULT 1,
    
    CONSTRAINT uk_currencies_code UNIQUE (code),
    CONSTRAINT uk_currencies_numeric_code UNIQUE (numeric_code),
    CONSTRAINT chk_currencies_code_format CHECK (code ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_currencies_numeric_code_format CHECK (numeric_code IS NULL OR numeric_code ~ '^[0-9]{3}$'),
    CONSTRAINT chk_currencies_minor_units CHECK (minor_units >= 0 AND minor_units <= 8),
    CONSTRAINT chk_currencies_status CHECK (status IN ('active', 'inactive', 'deprecated', 'deleted'))
);

-- Create indexes
CREATE INDEX idx_currencies_code ON treasury.currencies(code) WHERE status != 'deleted';
CREATE INDEX idx_currencies_numeric_code ON treasury.currencies(numeric_code) 
    WHERE numeric_code IS NOT NULL AND status != 'deleted';
CREATE INDEX idx_currencies_status ON treasury.currencies(status);
CREATE INDEX idx_currencies_country_codes ON treasury.currencies USING GIN(country_codes);
CREATE INDEX idx_currencies_is_active ON treasury.currencies(is_active) WHERE is_active = true;

-- Add trigger
CREATE TRIGGER update_currencies_updated_at 
    BEFORE UPDATE ON treasury.currencies
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Insert common currencies
INSERT INTO treasury.currencies (code, numeric_code, name, minor_units, symbol, country_codes) VALUES
    ('USD', '840', 'United States Dollar', 2, '$', ARRAY['US']),
    ('EUR', '978', 'Euro', 2, '€', ARRAY['EU']),
    ('GBP', '826', 'Pound Sterling', 2, '£', ARRAY['GB']),
    ('JPY', '392', 'Japanese Yen', 0, '¥', ARRAY['JP']),
    ('CHF', '756', 'Swiss Franc', 2, 'CHF', ARRAY['CH']),
    ('CAD', '124', 'Canadian Dollar', 2, '$', ARRAY['CA']),
    ('AUD', '036', 'Australian Dollar', 2, '$', ARRAY['AU']),
    ('CNY', '156', 'Chinese Yuan', 2, '¥', ARRAY['CN']);

COMMIT;
```

## Implementation Plan

### Phase 1: Foundation (Day 1-2)
- [ ] Create migration files for currency table
- [ ] Update protobuf definitions in treasury_service.proto
- [ ] Generate protobuf code
- [ ] Implement CurrencyManager struct
- [ ] Add currency configuration to config.go

### Phase 2: Core CRUD Operations (Day 3-4)
- [ ] Implement CreateCurrency with validation
- [ ] Implement GetCurrency with multiple lookup options
- [ ] Implement UpdateCurrency with optimistic locking
- [ ] Implement DeleteCurrency with reference checking
- [ ] Add proper error handling for all operations

### Phase 3: Advanced Features (Day 5-6)
- [ ] Implement ListCurrencies with filtering and pagination
- [ ] Implement BulkCreateCurrencies for initial loading
- [ ] Add reference tracking mechanism
- [ ] Implement soft delete functionality
- [ ] Add currency validation helpers

### Phase 4: Testing & Documentation (Day 7)
- [ ] Unit tests for all operations
- [ ] Integration tests with database
- [ ] Load testing for bulk operations
- [ ] Update service documentation
- [ ] Create operational runbooks

## Dependencies

### Service Dependencies
- Database connection from spec 001-database-connection.md
- Migration system from spec 002-database-migrations.md
- Health check integration for monitoring

### External Dependencies
- ISO 4217 currency code list for validation
- PostgreSQL 16 for storage
- gRPC/protobuf for API

### Data Dependencies
- Initial currency dataset from ISO 4217
- Country code mappings from ISO 3166

## Security Considerations

### Authentication & Authorization
- All write operations require admin role
- Read operations available to all authenticated users
- Audit trail includes user identity for all changes
- Service-to-service authentication for internal calls

### Data Privacy
- No sensitive financial data in currency table
- Audit logs retained per compliance requirements
- Currency metadata is public information

### Input Validation
- Strict validation of ISO codes
- SQL injection prevention through parameterized queries
- Rate limiting on bulk operations
- Input sanitization for all text fields

## Testing Strategy

### Unit Tests
- [ ] Currency code validation
- [ ] Reference checking logic
- [ ] Optimistic locking behavior
- [ ] Status transition rules
- [ ] Pagination logic

### Integration Tests
- [ ] Full CRUD cycle
- [ ] Concurrent updates
- [ ] Reference constraint enforcement
- [ ] Bulk import with duplicates
- [ ] Database transaction rollback

### Acceptance Tests
- [ ] Create major world currencies
- [ ] Update currency metadata
- [ ] Delete with and without references
- [ ] List with various filters
- [ ] Bulk import ISO dataset

## Monitoring & Observability

### Metrics
- Currency operation latency (p50, p95, p99)
- Currency cache hit rate
- Failed operation count by error type
- Active currency count
- Bulk import performance

### Logs
- All currency modifications with before/after state
- Failed operations with error details
- Reference check results
- Bulk import progress and results

### Alerts
- Currency creation failures > 5/minute
- Reference check timeout > 5 seconds
- Database connection failures
- Unexpected currency deletion

## Documentation Updates

Upon implementation, update:
- [ ] Treasury service README with currency management section
- [ ] CLAUDE.md with currency operation examples
- [ ] API documentation with request/response examples
- [ ] Runbook for currency data management
- [ ] ISO compliance documentation

## Open Questions

1. Should we support custom currency codes for internal use (e.g., loyalty points)?
2. Do we need currency groupings (e.g., G10, emerging markets)?
3. Should historical currencies be supported (e.g., pre-Euro currencies)?
4. What is the strategy for cryptocurrency decimal places (up to 18)?
5. Should we track currency activation/deactivation history?

## Design Note: Reference Tracking Pattern

While this spec uses simple foreign key constraints with ON DELETE RESTRICT for currencies (as they are permanent reference data), future treasury features that involve more transient data should consider implementing a reference tracking pattern. This would be appropriate for:

- **Trading Positions**: Track references before closing positions
- **Investment Strategies**: Ensure no active trades before deletion
- **Financial Instruments**: Check for outstanding obligations
- **Trade Orders**: Verify no pending executions
- **Temporary Accounts**: Ensure zero balance before removal

For these features, a reference tracking table would provide:
1. Visibility into dependencies before deletion
2. Detailed error messages about what blocks deletion
3. Ability to implement business rules (e.g., "can only delete if balance is zero")
4. Audit trail of deletion attempts

This pattern trades simplicity for safety and visibility, which is often the right choice for financial systems handling transient but critical data.

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-12 | Use CHAR(3) for ISO codes | Fixed length, standard format | Team |
| 2025-01-12 | Include numeric codes | Some systems require them | Team |
| 2025-01-12 | Soft delete option | Maintain referential integrity | Team |
| 2025-01-12 | Version field for optimistic locking | Prevent lost updates | Team |
| 2025-01-12 | Support cryptocurrencies | Modern treasury requirement | Team |

## References

- [ISO 4217 Currency Codes](https://www.iso.org/iso-4217-currency-codes.html)
- [Database Connection Spec](./001-database-connection.md)
- [Database Migration Spec](./002-database-migrations.md)
- [Protobuf Patterns](../../../../docs/PROTOBUF_PATTERNS.md)
- [Service Development Guide](../../../../docs/SERVICE_DEVELOPMENT.md)

## Appendix

### Sample Currency Data

```json
{
  "currencies": [
    {
      "code": "USD",
      "numeric_code": "840",
      "name": "United States Dollar",
      "minor_units": 2,
      "symbol": "$",
      "symbol_position": "before",
      "country_codes": ["US", "AS", "EC", "GU", "MH", "FM", "MP", "PW", "PR", "TC", "VI", "UM"],
      "is_active": true,
      "is_crypto": false,
      "status": "CURRENCY_STATUS_ACTIVE"
    },
    {
      "code": "BTC",
      "numeric_code": null,
      "name": "Bitcoin",
      "minor_units": 8,
      "symbol": "₿",
      "symbol_position": "before",
      "country_codes": [],
      "is_active": true,
      "is_crypto": true,
      "status": "CURRENCY_STATUS_ACTIVE"
    }
  ]
}
```

### Reference Check Query Example

```sql
-- Check for currency references across all tables
WITH reference_counts AS (
    SELECT 'accounts' as table_name, 
           'currency_code' as column_name,
           COUNT(*) as ref_count
    FROM treasury.accounts 
    WHERE currency_code = 'USD'
    
    UNION ALL
    
    SELECT 'transactions' as table_name,
           'currency_code' as column_name, 
           COUNT(*) as ref_count
    FROM treasury.transactions 
    WHERE currency_code = 'USD'
    
    UNION ALL
    
    SELECT 'exchange_rates' as table_name,
           'base_currency' as column_name,
           COUNT(*) as ref_count
    FROM treasury.exchange_rates 
    WHERE base_currency = 'USD'
)
SELECT table_name, column_name, ref_count
FROM reference_counts
WHERE ref_count > 0;
```

### Currency Validation Rules

1. **ISO Code**: Must be 3 uppercase letters
2. **Numeric Code**: Must be 3 digits, unique across all currencies
3. **Minor Units**: 0-8, typically 2 for most currencies
4. **Country Codes**: Valid ISO 3166-1 alpha-2 codes
5. **Status Transitions**: 
   - active → inactive → deprecated → deleted
   - No backward transitions except inactive → active
6. **Deletion Rules**:
   - Hard delete only if no references
   - Soft delete sets status to 'deleted'
   - Deleted currencies excluded from normal queries