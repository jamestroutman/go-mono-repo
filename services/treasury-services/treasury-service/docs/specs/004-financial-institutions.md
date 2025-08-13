# Financial Institutions Management Specification

> **Status**: Draft  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-12  
> **Author(s)**: Engineering Team  
> **Reviewer(s)**: Treasury Team, Platform Team  
> **Confluence**: https://example.atlassian.net/wiki/spaces/TREASURY/pages/004/Financial+Institutions  

## Executive Summary

The Treasury Service requires a comprehensive financial institutions management system to serve as the foundation for treasury account relationships and banking operations. This specification defines a unified financial institutions database with full CRUD operations, routing number support, international banking codes, and referential integrity constraints to ensure data consistency across the system.

## Problem Statement

### Current State
The Treasury Service lacks a centralized financial institutions management system, making it impossible to properly track and manage banking relationships. Without a unified institutions database with proper routing and identification codes, the service cannot accurately represent banking partners, manage treasury accounts, or maintain consistency across financial operations.

### Desired State
The Treasury Service will implement a comprehensive financial institutions management system with standardized bank identification (routing numbers, SWIFT codes), full CRUD operations with appropriate authorization, referential integrity enforcement to prevent orphaned references, complete audit trail for all institution modifications, and support for both domestic and international institutions.

## Scope

### In Scope
- Financial institutions table schema with routing numbers and SWIFT codes
- Full CRUD operations (Create, Read, Update, Delete) via gRPC
- Referential integrity checks for safe deletion
- US routing number (ABA) validation
- SWIFT/BIC code validation for international banks
- Institution status management (active/inactive/suspended)
- Institution type classification (bank, credit union, investment bank, etc.)
- Address and contact information management
- Audit fields for tracking changes
- Bulk operations for initial data loading
- Institution validation rules and constraints
- Migration scripts for institutions table creation

### Out of Scope
- Treasury account management (separate specification)
- Wire transfer routing and processing
- Banking fee structures and negotiations
- Institution credit ratings and risk assessment
- Real-time institution status verification
- Integration with external banking APIs
- Multi-tenancy institution configurations
- Document management (agreements, contracts)

## User Stories

### Story 1: Create New Financial Institution
**As a** Treasury administrator  
**I want to** add new financial institutions to the system  
**So that** I can establish banking relationships and manage treasury accounts  

**Acceptance Criteria:**
- [ ] Can create institution with name, routing number, and type
- [ ] Validation ensures routing number format (9 digits for US)
- [ ] SWIFT code validation for international institutions
- [ ] Institution code must be unique
- [ ] Address fields properly structured
- [ ] Audit fields automatically populated
- [ ] Duplicate routing numbers allowed (branches)
- [ ] Duplicate check on institution code

### Story 2: Query Financial Institution Information
**As a** Treasury system user  
**I want to** retrieve institution information by various criteria  
**So that** I can properly process banking transactions  

**Acceptance Criteria:**
- [ ] Can get institution by code (primary lookup)
- [ ] Can get institution by routing number
- [ ] Can get institution by SWIFT code
- [ ] Can list all active institutions
- [ ] Can filter institutions by type
- [ ] Can filter institutions by country
- [ ] Response includes all metadata fields
- [ ] Pagination support for list operations

### Story 3: Update Institution Information
**As a** Treasury administrator  
**I want to** update financial institution details  
**So that** I can maintain accurate banking information  

**Acceptance Criteria:**
- [ ] Can update institution name and details
- [ ] Can update address and contact information
- [ ] Can update routing numbers and SWIFT codes
- [ ] Can update status (active/inactive/suspended)
- [ ] Cannot modify institution code (immutable)
- [ ] Audit trail tracks all changes
- [ ] Version field prevents concurrent modifications
- [ ] Validation on all updated fields

### Story 4: Deactivate Financial Institution
**As a** Treasury administrator  
**I want to** deactivate institutions no longer in use  
**So that** I can maintain accurate institution status  

**Acceptance Criteria:**
- [ ] Can update institution status to inactive/suspended
- [ ] Check for active treasury accounts before deactivation
- [ ] Foreign key constraints prevent hard deletion
- [ ] Soft delete option available (status = deleted)
- [ ] Deactivated institutions excluded from active lists
- [ ] Audit trail records status changes with user and timestamp
- [ ] Historical data with deactivated institutions remains intact

### Story 5: Bulk Institution Operations
**As a** System administrator  
**I want to** load multiple institutions at once  
**So that** I can efficiently initialize the system  

**Acceptance Criteria:**
- [ ] Can import institutions from CSV/JSON
- [ ] Bulk create validates all entries before applying
- [ ] Transaction ensures all-or-nothing operation
- [ ] Progress reporting for large imports
- [ ] Duplicate handling options (skip/update/error)
- [ ] Import summary with success/failure counts
- [ ] Routing number validation for all entries

## Technical Design

### Architecture Overview
The financial institutions management system will be implemented as part of the Treasury Service, using PostgreSQL for storage with strong consistency guarantees, gRPC APIs following established patterns, and integration with the existing migration and health check systems.

### Database Schema

```sql
-- Financial institutions table
-- Spec: docs/specs/004-financial-institutions.md
CREATE TABLE IF NOT EXISTS treasury.financial_institutions (
    -- Primary key
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Institution identifiers
    code VARCHAR(50) NOT NULL,                    -- Internal unique code (e.g., "JPMORGAN", "BOFA")
    name VARCHAR(255) NOT NULL,                   -- Official institution name
    short_name VARCHAR(100),                      -- Common/display name
    
    -- US Banking identifiers (moved to separate table for multiple routing support)
    
    -- International identifiers
    swift_code VARCHAR(11),                       -- SWIFT/BIC code (8 or 11 chars)
    iban_prefix VARCHAR(4),                       -- IBAN country prefix
    bank_code VARCHAR(20),                        -- National bank code
    branch_code VARCHAR(20),                      -- Branch identifier
    
    -- Institution details
    institution_type VARCHAR(50) NOT NULL,        -- bank, credit_union, investment_bank, etc.
    country_code CHAR(2) NOT NULL,                -- ISO 3166-1 alpha-2 country code
    primary_currency CHAR(3),                     -- Default currency (ISO 4217)
    
    -- Address information
    street_address_1 VARCHAR(255),
    street_address_2 VARCHAR(255),
    city VARCHAR(100),
    state_province VARCHAR(100),
    postal_code VARCHAR(20),
    
    -- Contact information
    phone_number VARCHAR(50),
    fax_number VARCHAR(50),
    email_address VARCHAR(255),
    website_url VARCHAR(255),
    
    -- Operational details
    time_zone VARCHAR(50),                        -- Institution's primary time zone
    business_hours JSONB,                         -- Structured business hours
    holiday_calendar VARCHAR(50),                 -- Reference to holiday calendar
    
    -- Regulatory information
    regulatory_id VARCHAR(50),                    -- Federal/national regulatory ID
    tax_id VARCHAR(50),                          -- Tax identification number
    licenses JSONB,                              -- Banking licenses and registrations
    
    -- Status management
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, inactive, suspended, deleted
    is_active BOOLEAN NOT NULL DEFAULT true,
    activated_at TIMESTAMP WITH TIME ZONE,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    suspension_reason TEXT,
    
    -- Metadata
    capabilities JSONB,                          -- Supported services/features
    notes TEXT,                                  -- Internal notes
    external_references JSONB,                   -- External system IDs
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    version INTEGER NOT NULL DEFAULT 1,
    
    -- Constraints
    CONSTRAINT uk_institutions_code UNIQUE (code),
    CONSTRAINT uk_institutions_swift UNIQUE (swift_code),
    CONSTRAINT chk_institutions_swift_format CHECK (swift_code IS NULL OR swift_code ~ '^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$'),
    CONSTRAINT chk_institutions_country_format CHECK (country_code ~ '^[A-Z]{2}$'),
    CONSTRAINT chk_institutions_status CHECK (status IN ('active', 'inactive', 'suspended', 'deleted')),
    CONSTRAINT chk_institutions_type CHECK (institution_type IN ('bank', 'credit_union', 'investment_bank', 'central_bank', 'savings_bank', 'online_bank', 'other'))
);

-- Routing numbers table (supports multiple routing numbers per institution)
-- Spec: docs/specs/004-financial-institutions.md
CREATE TABLE IF NOT EXISTS treasury.institution_routing_numbers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES treasury.financial_institutions(id) ON DELETE CASCADE,
    routing_number CHAR(9) NOT NULL,
    routing_type VARCHAR(50) NOT NULL DEFAULT 'standard', -- standard, wire, ach, etc.
    is_primary BOOLEAN NOT NULL DEFAULT false,
    description VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_routing_format CHECK (routing_number ~ '^[0-9]{9}$'),
    CONSTRAINT chk_routing_type CHECK (routing_type IN ('standard', 'wire', 'ach', 'fedwire', 'other')),
    CONSTRAINT uk_routing_number_type UNIQUE (institution_id, routing_number, routing_type)
);

-- Indexes for performance
CREATE INDEX idx_institutions_code ON treasury.financial_institutions(code) WHERE status != 'deleted';
CREATE INDEX idx_routing_numbers ON treasury.institution_routing_numbers(routing_number);
CREATE INDEX idx_routing_institution ON treasury.institution_routing_numbers(institution_id);
CREATE INDEX idx_routing_primary ON treasury.institution_routing_numbers(institution_id, is_primary) WHERE is_primary = true;
CREATE INDEX idx_institutions_swift ON treasury.financial_institutions(swift_code) WHERE swift_code IS NOT NULL AND status != 'deleted';
CREATE INDEX idx_institutions_country ON treasury.financial_institutions(country_code);
CREATE INDEX idx_institutions_type ON treasury.financial_institutions(institution_type);
CREATE INDEX idx_institutions_status ON treasury.financial_institutions(status);
CREATE INDEX idx_institutions_is_active ON treasury.financial_institutions(is_active) WHERE is_active = true;

-- Trigger for updated_at
CREATE TRIGGER update_institutions_updated_at 
    BEFORE UPDATE ON treasury.financial_institutions
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Reference tracking table for deletion safety
CREATE TABLE IF NOT EXISTS treasury.institution_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES treasury.financial_institutions(id),
    table_name VARCHAR(100) NOT NULL,
    column_name VARCHAR(100) NOT NULL,
    reference_count INTEGER NOT NULL DEFAULT 0,
    last_checked TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_institution_references UNIQUE (institution_id, table_name, column_name)
);
```

### API Design

#### Protocol Buffer Definition

```protobuf
// Financial institutions management service
// Spec: docs/specs/004-financial-institutions.md
syntax = "proto3";

package treasury;

import "google/protobuf/timestamp.proto";
import "google/protobuf/field_mask.proto";
import "google/protobuf/struct.proto";

option go_package = "example.com/go-mono-repo/proto/treasury";

// RoutingNumber represents a routing number for an institution
message RoutingNumber {
    string id = 1;                              // UUID
    string routing_number = 2;                  // 9-digit routing number
    string routing_type = 3;                    // standard, wire, ach, fedwire, other
    bool is_primary = 4;                        // Primary routing number flag
    string description = 5;                     // Optional description
    google.protobuf.Timestamp created_at = 6;
    google.protobuf.Timestamp updated_at = 7;
}

// FinancialInstitution represents a banking institution
message FinancialInstitution {
    string id = 1;                              // UUID
    string code = 2;                            // Unique institution code
    string name = 3;                            // Official name
    string short_name = 4;                      // Display name
    
    // US Banking identifiers (supports multiple routing numbers)
    repeated RoutingNumber routing_numbers = 5;  // List of routing numbers
    
    // International identifiers
    string swift_code = 6;                      // SWIFT/BIC code
    string iban_prefix = 7;                     // IBAN prefix
    string bank_code = 8;                       // National bank code
    string branch_code = 9;                     // Branch code (not for separate branch entities)
    
    // Institution details
    InstitutionType institution_type = 10;
    string country_code = 11;                   // ISO 3166-1 alpha-2
    string primary_currency = 12;               // ISO 4217 code
    
    // Address
    Address address = 13;
    
    // Contact
    ContactInfo contact = 14;
    
    // Operational
    string time_zone = 15;
    google.protobuf.Struct business_hours = 16;
    string holiday_calendar = 17;
    
    // Regulatory
    string regulatory_id = 18;
    string tax_id = 19;
    google.protobuf.Struct licenses = 20;
    
    // Status
    InstitutionStatus status = 21;
    bool is_active = 22;
    google.protobuf.Timestamp activated_at = 23;
    google.protobuf.Timestamp deactivated_at = 24;
    string suspension_reason = 25;
    
    // Metadata
    google.protobuf.Struct capabilities = 26;
    string notes = 27;
    google.protobuf.Struct external_references = 28;
    
    // Audit
    google.protobuf.Timestamp created_at = 29;
    google.protobuf.Timestamp updated_at = 30;
    string created_by = 31;
    string updated_by = 32;
    int32 version = 33;
}

enum InstitutionType {
    INSTITUTION_TYPE_UNSPECIFIED = 0;
    INSTITUTION_TYPE_BANK = 1;
    INSTITUTION_TYPE_CREDIT_UNION = 2;
    INSTITUTION_TYPE_INVESTMENT_BANK = 3;
    INSTITUTION_TYPE_CENTRAL_BANK = 4;
    INSTITUTION_TYPE_SAVINGS_BANK = 5;
    INSTITUTION_TYPE_ONLINE_BANK = 6;
    INSTITUTION_TYPE_OTHER = 7;
}

enum InstitutionStatus {
    INSTITUTION_STATUS_UNSPECIFIED = 0;
    INSTITUTION_STATUS_ACTIVE = 1;
    INSTITUTION_STATUS_INACTIVE = 2;
    INSTITUTION_STATUS_SUSPENDED = 3;
    INSTITUTION_STATUS_DELETED = 4;
}

message Address {
    string street_address_1 = 1;
    string street_address_2 = 2;
    string city = 3;
    string state_province = 4;
    string postal_code = 5;
    string country_code = 6;
}

message ContactInfo {
    string phone_number = 1;
    string fax_number = 2;
    string email_address = 3;
    string website_url = 4;
}

// Request/Response messages
message CreateInstitutionRequest {
    string code = 1;                            // Required, unique
    string name = 2;                            // Required
    string short_name = 3;
    
    // Support for multiple routing numbers
    message RoutingNumberInput {
        string routing_number = 1;
        string routing_type = 2;                // standard, wire, ach, fedwire, other
        bool is_primary = 3;
        string description = 4;
    }
    repeated RoutingNumberInput routing_numbers = 4; // US banks routing numbers
    
    string swift_code = 5;                      // Required for international
    string bank_code = 6;
    string branch_code = 7;
    InstitutionType institution_type = 8;       // Required
    string country_code = 9;                    // Required
    string primary_currency = 10;
    Address address = 11;
    ContactInfo contact = 12;
    string time_zone = 13;
    google.protobuf.Struct capabilities = 14;
    string notes = 15;
}

message CreateInstitutionResponse {
    FinancialInstitution institution = 1;
}

message GetInstitutionRequest {
    oneof identifier {
        string code = 1;                        // Primary lookup
        string routing_number = 2;              // US routing lookup (finds institution with this routing)
        string swift_code = 3;                  // International lookup
        string id = 4;                          // UUID lookup
    }
}

message GetInstitutionResponse {
    FinancialInstitution institution = 1;
}

message UpdateInstitutionRequest {
    string code = 1;                            // Required (identifies institution)
    google.protobuf.FieldMask update_mask = 2;  // Fields to update
    string name = 3;
    string short_name = 4;
    
    // For updating routing numbers
    message RoutingNumberUpdate {
        string routing_number = 1;
        string routing_type = 2;
        bool is_primary = 3;
        string description = 4;
    }
    repeated RoutingNumberUpdate routing_numbers = 5; // Replace all routing numbers
    
    string swift_code = 6;
    Address address = 7;
    ContactInfo contact = 8;
    InstitutionStatus status = 9;
    google.protobuf.Struct capabilities = 10;
    string notes = 11;
    int32 version = 12;                         // For optimistic locking
}

message UpdateInstitutionResponse {
    FinancialInstitution institution = 1;
}

message DeleteInstitutionRequest {
    string code = 1;                            // Required
    bool force = 2;                             // Force deletion despite references
    string deleted_by = 3;                      // User making deletion
}

message DeleteInstitutionResponse {
    bool success = 1;
    repeated string blocking_references = 2;    // What prevents deletion
}

message ListInstitutionsRequest {
    InstitutionStatus status = 1;               // Filter by status
    InstitutionType institution_type = 2;       // Filter by type
    string country_code = 3;                    // Filter by country
    bool is_active = 4;                         // Filter by active flag
    int32 page_size = 5;                        // Pagination
    string page_token = 6;                      // Pagination token
}

message ListInstitutionsResponse {
    repeated FinancialInstitution institutions = 1;
    string next_page_token = 2;
    int32 total_count = 3;
}

message CheckInstitutionReferencesRequest {
    string code = 1;                            // Institution to check
}

message CheckInstitutionReferencesResponse {
    message Reference {
        string table_name = 1;
        string column_name = 2;
        int32 count = 3;
    }
    repeated Reference references = 1;
    bool can_delete = 2;                        // True if no references
}

message BulkCreateInstitutionsRequest {
    repeated CreateInstitutionRequest institutions = 1;
    bool skip_duplicates = 2;                   // Skip existing institutions
    bool update_existing = 3;                   // Update if exists
}

message BulkCreateInstitutionsResponse {
    int32 created_count = 1;
    int32 updated_count = 2;
    int32 skipped_count = 3;
    repeated string errors = 4;
}

// Financial institution service definition
service FinancialInstitutionService {
    // Create a new financial institution
    // Spec: docs/specs/004-financial-institutions.md#story-1-create-new-financial-institution
    rpc CreateInstitution(CreateInstitutionRequest) returns (CreateInstitutionResponse);
    
    // Get institution information
    // Spec: docs/specs/004-financial-institutions.md#story-2-query-financial-institution-information
    rpc GetInstitution(GetInstitutionRequest) returns (GetInstitutionResponse);
    
    // Update institution metadata
    // Spec: docs/specs/004-financial-institutions.md#story-3-update-institution-information
    rpc UpdateInstitution(UpdateInstitutionRequest) returns (UpdateInstitutionResponse);
    
    // Delete institution (with reference checking)
    // Spec: docs/specs/004-financial-institutions.md#story-4-deactivate-financial-institution
    rpc DeleteInstitution(DeleteInstitutionRequest) returns (DeleteInstitutionResponse);
    
    // List institutions with filters
    // Spec: docs/specs/004-financial-institutions.md#story-2-query-financial-institution-information
    rpc ListInstitutions(ListInstitutionsRequest) returns (ListInstitutionsResponse);
    
    // Check for references before deletion
    // Spec: docs/specs/004-financial-institutions.md#story-4-deactivate-financial-institution
    rpc CheckInstitutionReferences(CheckInstitutionReferencesRequest) returns (CheckInstitutionReferencesResponse);
    
    // Bulk create institutions
    // Spec: docs/specs/004-financial-institutions.md#story-5-bulk-institution-operations
    rpc BulkCreateInstitutions(BulkCreateInstitutionsRequest) returns (BulkCreateInstitutionsResponse);
}
```

### Implementation Structure

```go
// InstitutionManager handles financial institution operations
// Spec: docs/specs/004-financial-institutions.md
type InstitutionManager struct {
    db     *sql.DB
    logger *log.Logger
}

// RoutingNumber represents a routing number for an institution
type RoutingNumber struct {
    ID             uuid.UUID
    InstitutionID  uuid.UUID
    RoutingNumber  string
    RoutingType    string // standard, wire, ach, fedwire, other
    IsPrimary      bool
    Description    *string
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

// FinancialInstitution represents an institution entity
// Spec: docs/specs/004-financial-institutions.md
type FinancialInstitution struct {
    ID                uuid.UUID
    Code              string
    Name              string
    ShortName         *string
    RoutingNumbers    []RoutingNumber  // Multiple routing numbers
    SwiftCode         *string
    IbanPrefix        *string
    BankCode          *string
    BranchCode        *string
    InstitutionType   InstitutionType
    CountryCode       string
    PrimaryCurrency   *string
    Address           Address
    Contact           ContactInfo
    TimeZone          *string
    BusinessHours     map[string]interface{}
    HolidayCalendar   *string
    RegulatoryID      *string
    TaxID             *string
    Licenses          map[string]interface{}
    Status            InstitutionStatus
    IsActive          bool
    ActivatedAt       *time.Time
    DeactivatedAt     *time.Time
    SuspensionReason  *string
    Capabilities      map[string]interface{}
    Notes             *string
    ExternalRefs      map[string]interface{}
    CreatedAt         time.Time
    UpdatedAt         time.Time
    CreatedBy         string
    UpdatedBy         string
    Version           int
}

// CreateInstitution creates a new financial institution
// Spec: docs/specs/004-financial-institutions.md#story-1-create-new-financial-institution
func (im *InstitutionManager) CreateInstitution(ctx context.Context, req *pb.CreateInstitutionRequest) (*FinancialInstitution, error) {
    // Validate institution code uniqueness
    exists, err := im.institutionExists(ctx, req.Code)
    if err != nil {
        return nil, status.Error(codes.Internal, "failed to check institution existence")
    }
    if exists {
        return nil, status.Error(codes.AlreadyExists, "institution code already exists")
    }
    
    // Validate routing numbers for US banks
    if req.CountryCode == "US" && len(req.RoutingNumbers) > 0 {
        for _, rn := range req.RoutingNumbers {
            if !isValidRoutingNumber(rn.RoutingNumber) {
                return nil, status.Errorf(codes.InvalidArgument, "invalid routing number format: %s", rn.RoutingNumber)
            }
        }
    }
    
    // Validate SWIFT code format
    if req.SwiftCode != "" && !isValidSwiftCode(req.SwiftCode) {
        return nil, status.Error(codes.InvalidArgument, "invalid SWIFT code format")
    }
    
    // Create institution
    institution := &FinancialInstitution{
        ID:              uuid.New(),
        Code:            req.Code,
        Name:            req.Name,
        ShortName:       req.ShortName,
        SwiftCode:       req.SwiftCode,
        InstitutionType: req.InstitutionType,
        CountryCode:     req.CountryCode,
        Status:          InstitutionStatusActive,
        IsActive:        true,
        CreatedAt:       time.Now(),
        UpdatedAt:       time.Now(),
        Version:         1,
    }
    
    // Begin transaction
    tx, err := im.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, status.Error(codes.Internal, "failed to begin transaction")
    }
    defer tx.Rollback()
    
    // Insert institution
    err = im.insertInstitution(ctx, tx, institution)
    if err != nil {
        return nil, status.Error(codes.Internal, "failed to create institution")
    }
    
    // Insert routing numbers
    for _, rn := range req.RoutingNumbers {
        routingNum := &RoutingNumber{
            ID:            uuid.New(),
            InstitutionID: institution.ID,
            RoutingNumber: rn.RoutingNumber,
            RoutingType:   rn.RoutingType,
            IsPrimary:     rn.IsPrimary,
            Description:   rn.Description,
            CreatedAt:     time.Now(),
            UpdatedAt:     time.Now(),
        }
        
        err = im.insertRoutingNumber(ctx, tx, routingNum)
        if err != nil {
            return nil, status.Error(codes.Internal, "failed to create routing number")
        }
        
        institution.RoutingNumbers = append(institution.RoutingNumbers, *routingNum)
    }
    
    // Commit transaction
    if err = tx.Commit(); err != nil {
        return nil, status.Error(codes.Internal, "failed to commit transaction")
    }
    
    return institution, nil
}

// DeleteInstitution handles institution deletion with reference checking
// Spec: docs/specs/004-financial-institutions.md#story-4-deactivate-financial-institution
func (im *InstitutionManager) DeleteInstitution(ctx context.Context, req *pb.DeleteInstitutionRequest) error {
    // Check for references
    if !req.Force {
        refs, err := im.checkReferences(ctx, req.Code)
        if err != nil {
            return status.Error(codes.Internal, "failed to check references")
        }
        if len(refs) > 0 {
            return status.Errorf(codes.FailedPrecondition, 
                "cannot delete institution with %d references", len(refs))
        }
    }
    
    // Soft delete by updating status
    query := `
        UPDATE treasury.financial_institutions 
        SET status = 'deleted', 
            is_active = false,
            deactivated_at = CURRENT_TIMESTAMP,
            updated_at = CURRENT_TIMESTAMP,
            updated_by = $1,
            version = version + 1
        WHERE code = $2 AND status != 'deleted'
    `
    
    result, err := im.db.ExecContext(ctx, query, req.DeletedBy, req.Code)
    if err != nil {
        return status.Error(codes.Internal, "failed to delete institution")
    }
    
    if rows, _ := result.RowsAffected(); rows == 0 {
        return status.Error(codes.NotFound, "institution not found or already deleted")
    }
    
    return nil
}

// Validation helpers
func isValidRoutingNumber(routing string) bool {
    // US routing number: 9 digits with check digit validation
    if len(routing) != 9 {
        return false
    }
    
    // Check digit algorithm (ABA)
    weights := []int{3, 7, 1, 3, 7, 1, 3, 7, 1}
    sum := 0
    for i, weight := range weights {
        digit, err := strconv.Atoi(string(routing[i]))
        if err != nil {
            return false
        }
        sum += digit * weight
    }
    
    return sum%10 == 0
}

func isValidSwiftCode(swift string) bool {
    // SWIFT code: 8 or 11 characters
    // Format: AAAABBCC or AAAABBCCDDD
    // AAAA: Bank code
    // BB: Country code
    // CC: Location code
    // DDD: Optional branch code
    pattern := `^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$`
    matched, _ := regexp.MatchString(pattern, swift)
    return matched
}
```

### Error Handling

| Error Code | Description | HTTP Status | Example |
|------------|-------------|-------------|---------|
| INVALID_ARGUMENT | Invalid routing/SWIFT format | 400 | Invalid routing number |
| ALREADY_EXISTS | Institution code exists | 409 | Duplicate JPMORGAN code |
| NOT_FOUND | Institution not found | 404 | Unknown institution |
| FAILED_PRECONDITION | Cannot delete due to references | 412 | Active treasury accounts |
| INTERNAL | Database or system error | 500 | Connection failure |
| ABORTED | Optimistic locking conflict | 409 | Concurrent update |

### Migration Scripts

```sql
-- Migration: 000004_create_financial_institutions_table.up.sql
-- Spec: docs/specs/004-financial-institutions.md

BEGIN;

-- Create financial institutions table
CREATE TABLE IF NOT EXISTS treasury.financial_institutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(100),
    swift_code VARCHAR(11),
    iban_prefix VARCHAR(4),
    bank_code VARCHAR(20),
    branch_code VARCHAR(20),
    institution_type VARCHAR(50) NOT NULL,
    country_code CHAR(2) NOT NULL,
    primary_currency CHAR(3),
    street_address_1 VARCHAR(255),
    street_address_2 VARCHAR(255),
    city VARCHAR(100),
    state_province VARCHAR(100),
    postal_code VARCHAR(20),
    phone_number VARCHAR(50),
    fax_number VARCHAR(50),
    email_address VARCHAR(255),
    website_url VARCHAR(255),
    time_zone VARCHAR(50),
    business_hours JSONB,
    holiday_calendar VARCHAR(50),
    regulatory_id VARCHAR(50),
    tax_id VARCHAR(50),
    licenses JSONB,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_active BOOLEAN NOT NULL DEFAULT true,
    activated_at TIMESTAMP WITH TIME ZONE,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    suspension_reason TEXT,
    capabilities JSONB,
    notes TEXT,
    external_references JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    version INTEGER NOT NULL DEFAULT 1,
    
    CONSTRAINT uk_institutions_code UNIQUE (code),
    CONSTRAINT uk_institutions_swift UNIQUE (swift_code),
    CONSTRAINT chk_institutions_swift_format CHECK (swift_code IS NULL OR swift_code ~ '^[A-Z]{6}[A-Z0-9]{2}([A-Z0-9]{3})?$'),
    CONSTRAINT chk_institutions_country_format CHECK (country_code ~ '^[A-Z]{2}$'),
    CONSTRAINT chk_institutions_status CHECK (status IN ('active', 'inactive', 'suspended', 'deleted')),
    CONSTRAINT chk_institutions_type CHECK (institution_type IN ('bank', 'credit_union', 'investment_bank', 'central_bank', 'savings_bank', 'online_bank', 'other'))
);

-- Create indexes
CREATE INDEX idx_institutions_code ON treasury.financial_institutions(code) WHERE status != 'deleted';
CREATE INDEX idx_institutions_swift ON treasury.financial_institutions(swift_code) 
    WHERE swift_code IS NOT NULL AND status != 'deleted';
CREATE INDEX idx_institutions_country ON treasury.financial_institutions(country_code);
CREATE INDEX idx_institutions_type ON treasury.financial_institutions(institution_type);
CREATE INDEX idx_institutions_status ON treasury.financial_institutions(status);
CREATE INDEX idx_institutions_is_active ON treasury.financial_institutions(is_active) WHERE is_active = true;

-- Add trigger
CREATE TRIGGER update_institutions_updated_at 
    BEFORE UPDATE ON treasury.financial_institutions
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create routing numbers table (supports multiple routing numbers per institution)
CREATE TABLE IF NOT EXISTS treasury.institution_routing_numbers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES treasury.financial_institutions(id) ON DELETE CASCADE,
    routing_number CHAR(9) NOT NULL,
    routing_type VARCHAR(50) NOT NULL DEFAULT 'standard', -- standard, wire, ach, fedwire, other
    is_primary BOOLEAN NOT NULL DEFAULT false,
    description VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT chk_routing_format CHECK (routing_number ~ '^[0-9]{9}$'),
    CONSTRAINT chk_routing_type CHECK (routing_type IN ('standard', 'wire', 'ach', 'fedwire', 'other')),
    CONSTRAINT uk_routing_number_type UNIQUE (institution_id, routing_number, routing_type)
);

-- Create indexes for routing numbers
CREATE INDEX idx_routing_numbers ON treasury.institution_routing_numbers(routing_number);
CREATE INDEX idx_routing_institution ON treasury.institution_routing_numbers(institution_id);
CREATE INDEX idx_routing_primary ON treasury.institution_routing_numbers(institution_id, is_primary) WHERE is_primary = true;

-- Add trigger for routing numbers updated_at
CREATE TRIGGER update_routing_numbers_updated_at 
    BEFORE UPDATE ON treasury.institution_routing_numbers
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Create reference tracking table
CREATE TABLE IF NOT EXISTS treasury.institution_references (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id UUID NOT NULL REFERENCES treasury.financial_institutions(id),
    table_name VARCHAR(100) NOT NULL,
    column_name VARCHAR(100) NOT NULL,
    reference_count INTEGER NOT NULL DEFAULT 0,
    last_checked TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uk_institution_references UNIQUE (institution_id, table_name, column_name)
);

-- Insert sample institutions
INSERT INTO treasury.financial_institutions 
    (id, code, name, short_name, swift_code, institution_type, country_code, primary_currency) 
VALUES
    ('a1111111-1111-1111-1111-111111111111', 'JPMORGAN', 'JPMorgan Chase Bank, N.A.', 'Chase', 'CHASUS33', 'bank', 'US', 'USD'),
    ('a2222222-2222-2222-2222-222222222222', 'BOFA', 'Bank of America, N.A.', 'Bank of America', 'BOFAUS3N', 'bank', 'US', 'USD'),
    ('a3333333-3333-3333-3333-333333333333', 'WELLS', 'Wells Fargo Bank, N.A.', 'Wells Fargo', 'WFBIUS6S', 'bank', 'US', 'USD'),
    ('a4444444-4444-4444-4444-444444444444', 'CITI', 'Citibank, N.A.', 'Citibank', 'CITIUS33', 'bank', 'US', 'USD'),
    ('a5555555-5555-5555-5555-555555555555', 'HSBC', 'HSBC Bank USA, N.A.', 'HSBC', 'MRMDUS33', 'bank', 'US', 'USD'),
    ('a6666666-6666-6666-6666-666666666666', 'BARCLAYS', 'Barclays Bank PLC', 'Barclays', 'BARCGB22', 'bank', 'GB', 'GBP'),
    ('a7777777-7777-7777-7777-777777777777', 'DEUTSCHE', 'Deutsche Bank AG', 'Deutsche Bank', 'DEUTDEFF', 'investment_bank', 'DE', 'EUR'),
    ('a8888888-8888-8888-8888-888888888888', 'BNP', 'BNP Paribas SA', 'BNP Paribas', 'BNPAFRPP', 'bank', 'FR', 'EUR');

-- Insert sample routing numbers for US banks
INSERT INTO treasury.institution_routing_numbers 
    (institution_id, routing_number, routing_type, is_primary, description)
VALUES
    -- JPMorgan Chase routing numbers
    ('a1111111-1111-1111-1111-111111111111', '021000021', 'standard', true, 'New York'),
    ('a1111111-1111-1111-1111-111111111111', '322271627', 'standard', false, 'California'),
    ('a1111111-1111-1111-1111-111111111111', '021000021', 'wire', false, 'Wire transfers'),
    
    -- Bank of America routing numbers
    ('a2222222-2222-2222-2222-222222222222', '026009593', 'wire', true, 'Wire transfers'),
    ('a2222222-2222-2222-2222-222222222222', '121000358', 'standard', false, 'California'),
    ('a2222222-2222-2222-2222-222222222222', '051000017', 'standard', false, 'Virginia'),
    
    -- Wells Fargo routing numbers
    ('a3333333-3333-3333-3333-333333333333', '121000248', 'standard', true, 'California'),
    ('a3333333-3333-3333-3333-333333333333', '121042882', 'wire', false, 'Wire transfers'),
    ('a3333333-3333-3333-3333-333333333333', '102000076', 'standard', false, 'Colorado'),
    
    -- Citibank routing numbers
    ('a4444444-4444-4444-4444-444444444444', '021000089', 'standard', true, 'New York'),
    ('a4444444-4444-4444-4444-444444444444', '321171184', 'standard', false, 'California'),
    ('a4444444-4444-4444-4444-444444444444', '021000089', 'wire', false, 'Wire transfers'),
    
    -- HSBC routing numbers
    ('a5555555-5555-5555-5555-555555555555', '021001088', 'standard', true, 'New York'),
    ('a5555555-5555-5555-5555-555555555555', '021001088', 'wire', false, 'Wire transfers');

COMMIT;
```

## Implementation Plan

### Phase 1: Foundation (Day 1-2)
- [ ] Create migration files for institutions table
- [ ] Update protobuf definitions in treasury_service.proto
- [ ] Generate protobuf code
- [ ] Implement InstitutionManager struct
- [ ] Add institution configuration to config.go

### Phase 2: Core CRUD Operations (Day 3-4)
- [ ] Implement CreateInstitution with validation
- [ ] Implement GetInstitution with multiple lookup options
- [ ] Implement UpdateInstitution with optimistic locking
- [ ] Implement DeleteInstitution with reference checking
- [ ] Add proper error handling for all operations

### Phase 3: Advanced Features (Day 5-6)
- [ ] Implement ListInstitutions with filtering and pagination
- [ ] Implement BulkCreateInstitutions for initial loading
- [ ] Add reference tracking mechanism
- [ ] Implement routing number validation
- [ ] Add SWIFT code validation

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
- Currency management from spec 003-currency-management.md
- Health check integration for monitoring

### External Dependencies
- Federal Reserve routing number database
- SWIFT BIC directory for validation
- PostgreSQL 16 for storage
- gRPC/protobuf for API

### Data Dependencies
- Initial institution dataset from Federal Reserve
- SWIFT codes from SWIFT organization
- Country codes from ISO 3166

## Security Considerations

### Authentication & Authorization
- All write operations require admin role
- Read operations available to all authenticated users
- Audit trail includes user identity for all changes
- Service-to-service authentication for internal calls

### Data Privacy
- Routing numbers are public information
- Internal account numbers must be encrypted
- Contact information requires access control
- Audit logs retained per compliance requirements

### Input Validation
- Strict validation of routing numbers
- SWIFT code format validation
- SQL injection prevention through parameterized queries
- Rate limiting on bulk operations

## Testing Strategy

### Unit Tests
- [ ] Routing number validation
- [ ] SWIFT code validation
- [ ] Reference checking logic
- [ ] Optimistic locking behavior
- [ ] Status transition rules

### Integration Tests
- [ ] Full CRUD cycle
- [ ] Concurrent updates
- [ ] Reference constraint enforcement
- [ ] Bulk import with duplicates
- [ ] Database transaction rollback

### Acceptance Tests
- [ ] Create major US banks
- [ ] Create international banks
- [ ] Update institution metadata
- [ ] Delete with and without references
- [ ] List with various filters
- [ ] Bulk import institution dataset

## Monitoring & Observability

### Metrics
- Institution operation latency (p50, p95, p99)
- Institution lookup cache hit rate
- Failed operation count by error type
- Active institution count
- Bulk import performance

### Logs
- All institution modifications with before/after state
- Failed operations with error details
- Reference check results
- Bulk import progress and results

### Alerts
- Institution creation failures > 5/minute
- Reference check timeout > 5 seconds
- Database connection failures
- Unexpected institution deletion

## Documentation Updates

Upon implementation, update:
- [ ] Treasury service README with institution management section
- [ ] CLAUDE.md with institution operation examples
- [ ] API documentation with request/response examples
- [ ] Runbook for institution data management
- [ ] Integration guide for treasury accounts

## Open Questions - Resolved

1. ~~Should we support bank branch management as separate entities?~~ **No** - Branches are not necessary
2. ~~Do we need to track institution mergers and acquisitions?~~ **No** - Not required at this time
3. ~~Should we integrate with external services for real-time validation?~~ **No** - Not at this time
4. ~~What level of address validation is required?~~ **None** - No validation required at this time
5. ~~Should we support multiple routing numbers per institution?~~ **Yes** - Implemented with separate routing numbers table for safety
6. ~~Do we need to track correspondent banking relationships?~~ **No** - Not at this time

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-12 | Use VARCHAR(50) for institution code | Flexibility for various formats | Team |
| 2025-01-12 | Include reference tracking table | Safe deletion with visibility | Team |
| 2025-01-12 | Support both routing and SWIFT | Global operations requirement | Team |
| 2025-01-12 | Soft delete only | Maintain data integrity | Team |
| 2025-01-12 | Version field for optimistic locking | Prevent lost updates | Team |
| 2025-01-12 | Multiple routing numbers per institution | Banks often have different routing numbers by region/purpose | User |
| 2025-01-12 | No branch management | Keep it simple, branches not needed | User |
| 2025-01-12 | No address validation | Reduce complexity for initial implementation | User |

## References

- [Currency Management Spec](./003-currency-management.md)
- [Database Connection Spec](./001-database-connection.md)
- [Database Migration Spec](./002-database-migrations.md)
- [Protobuf Patterns](../../../../docs/PROTOBUF_PATTERNS.md)
- [Service Development Guide](../../../../docs/SERVICE_DEVELOPMENT.md)
- [Federal Reserve Routing Numbers](https://www.frbservices.org/EPaymentsDirectory/search.html)
- [SWIFT BIC Search](https://www.swift.com/bic-search)

## Appendix

### Sample Institution Data

```json
{
  "institutions": [
    {
      "code": "JPMORGAN",
      "name": "JPMorgan Chase Bank, N.A.",
      "short_name": "Chase",
      "routing_numbers": [
        {
          "routing_number": "021000021",
          "routing_type": "standard",
          "is_primary": true,
          "description": "New York"
        },
        {
          "routing_number": "322271627",
          "routing_type": "standard",
          "is_primary": false,
          "description": "California"
        },
        {
          "routing_number": "021000021",
          "routing_type": "wire",
          "is_primary": false,
          "description": "Wire transfers"
        }
      ],
      "swift_code": "CHASUS33",
      "institution_type": "INSTITUTION_TYPE_BANK",
      "country_code": "US",
      "primary_currency": "USD",
      "address": {
        "street_address_1": "270 Park Avenue",
        "city": "New York",
        "state_province": "NY",
        "postal_code": "10017",
        "country_code": "US"
      },
      "contact": {
        "phone_number": "+1-212-270-6000",
        "website_url": "https://www.jpmorganchase.com"
      },
      "capabilities": {
        "wire_transfer": true,
        "ach": true,
        "swift": true,
        "fedwire": true,
        "sepa": false
      },
      "status": "INSTITUTION_STATUS_ACTIVE",
      "is_active": true
    },
    {
      "code": "BARCLAYS",
      "name": "Barclays Bank PLC",
      "short_name": "Barclays",
      "routing_numbers": [],
      "swift_code": "BARCGB22",
      "institution_type": "INSTITUTION_TYPE_BANK",
      "country_code": "GB",
      "primary_currency": "GBP",
      "address": {
        "street_address_1": "1 Churchill Place",
        "city": "London",
        "postal_code": "E14 5HP",
        "country_code": "GB"
      },
      "capabilities": {
        "swift": true,
        "sepa": true,
        "chaps": true,
        "bacs": true
      },
      "status": "INSTITUTION_STATUS_ACTIVE",
      "is_active": true
    }
  ]
}
```

### Routing Number Validation Algorithm

```go
// ValidateRoutingNumber checks US ABA routing number validity
func ValidateRoutingNumber(routing string) error {
    if len(routing) != 9 {
        return fmt.Errorf("routing number must be 9 digits")
    }
    
    // ABA check digit algorithm
    weights := []int{3, 7, 1, 3, 7, 1, 3, 7, 1}
    sum := 0
    
    for i, weight := range weights {
        digit, err := strconv.Atoi(string(routing[i]))
        if err != nil {
            return fmt.Errorf("routing number must contain only digits")
        }
        sum += digit * weight
    }
    
    if sum%10 != 0 {
        return fmt.Errorf("invalid routing number check digit")
    }
    
    return nil
}
```

### Reference Check Query

```sql
-- Check for institution references across all tables
WITH reference_counts AS (
    SELECT 'treasury_accounts' as table_name, 
           'institution_id' as column_name,
           COUNT(*) as ref_count
    FROM treasury.treasury_accounts 
    WHERE institution_id = (SELECT id FROM treasury.financial_institutions WHERE code = 'JPMORGAN')
    
    UNION ALL
    
    SELECT 'wire_transfers' as table_name,
           'sending_institution_id' as column_name, 
           COUNT(*) as ref_count
    FROM treasury.wire_transfers 
    WHERE sending_institution_id = (SELECT id FROM treasury.financial_institutions WHERE code = 'JPMORGAN')
    
    UNION ALL
    
    SELECT 'wire_transfers' as table_name,
           'receiving_institution_id' as column_name,
           COUNT(*) as ref_count
    FROM treasury.wire_transfers 
    WHERE receiving_institution_id = (SELECT id FROM treasury.financial_institutions WHERE code = 'JPMORGAN')
)
SELECT table_name, column_name, ref_count
FROM reference_counts
WHERE ref_count > 0;
```

### Institution Validation Rules

1. **Institution Code**: Unique identifier, uppercase alphanumeric
2. **Routing Number**: 9 digits with valid check digit (US banks)
3. **SWIFT Code**: 8 or 11 characters following SWIFT format
4. **Country Code**: Valid ISO 3166-1 alpha-2 code
5. **Status Transitions**: 
   - active → inactive → deleted
   - active → suspended → active/inactive
   - No backward transitions except suspended → active
6. **Deletion Rules**:
   - Check for active treasury accounts
   - Check for pending transactions
   - Soft delete sets status to 'deleted'
   - Deleted institutions excluded from normal queries