# Protocol Buffer Patterns

## Overview

This document outlines the patterns and conventions for working with Protocol Buffers in our Go monorepo.

## File Organization

### Proto File Location
```
services/
└── {domain}/
    └── {service-name}/
        └── proto/
            └── {service}_service.proto
```

### Generated Code Location
```
proto/
└── {package}/
    ├── {service}_service.pb.go      # Message types
    └── {service}_service_grpc.pb.go # gRPC service stubs
```

## Package Naming Convention

### Proto Package
```protobuf
syntax = "proto3";

package treasury;  // Short, lowercase package name
```

### Go Package
```protobuf
option go_package = "example.com/go-mono-repo/proto/treasury";
```

The pattern is: `{root-module}/proto/{proto-package}`

## Service Definition Patterns

### Standard Service Structure

Every service should implement these standard endpoints:

```protobuf
service Manifest {
    rpc GetManifest (ManifestRequest) returns (ManifestResponse) {}
}

service Health {
    rpc Check (HealthCheckRequest) returns (HealthCheckResponse) {}
    rpc Watch (HealthCheckRequest) returns (stream HealthCheckResponse) {}
}
```

### Core Service Pattern

```protobuf
service {ServiceName} {
    // Query operations (no side effects)
    rpc Get{Entity} (Get{Entity}Request) returns ({Entity}Response) {}
    rpc List{Entities} (List{Entities}Request) returns (List{Entities}Response) {}
    
    // Command operations (with side effects)
    rpc Create{Entity} (Create{Entity}Request) returns (Create{Entity}Response) {}
    rpc Update{Entity} (Update{Entity}Request) returns (Update{Entity}Response) {}
    rpc Delete{Entity} (Delete{Entity}Request) returns (Delete{Entity}Response) {}
    
    // Streaming operations
    rpc Watch{Entities} (Watch{Entities}Request) returns (stream {Entity}Event) {}
}
```

## Message Design Patterns

### Request/Response Pairs

Always use dedicated request and response messages:

```protobuf
message GetAccountRequest {
    string account_id = 1;
}

message GetAccountResponse {
    Account account = 1;
}
```

### Common Fields

#### Timestamps
```protobuf
import "google/protobuf/timestamp.proto";

message Transaction {
    google.protobuf.Timestamp created_at = 1;
    google.protobuf.Timestamp updated_at = 2;
}
```

#### Money/Currency
```protobuf
message Money {
    int64 units = 1;      // Whole units
    int32 nanos = 2;      // Nano units (10^-9)
    string currency = 3;  // ISO 4217 code
}
```

#### Pagination
```protobuf
message ListRequest {
    int32 page_size = 1;
    string page_token = 2;
}

message ListResponse {
    repeated Item items = 1;
    string next_page_token = 2;
    int32 total_count = 3;
}
```

#### Field Masks
```protobuf
import "google/protobuf/field_mask.proto";

message UpdateRequest {
    Resource resource = 1;
    google.protobuf.FieldMask update_mask = 2;
}
```

## Enum Patterns

### Enum Definition
```protobuf
enum Status {
    STATUS_UNSPECIFIED = 0;  // Always include UNSPECIFIED as 0
    STATUS_PENDING = 1;
    STATUS_ACTIVE = 2;
    STATUS_INACTIVE = 3;
}
```

### Enum Naming
- Prefix enum values with the enum type name
- Use SCREAMING_SNAKE_CASE for values
- Always include an UNSPECIFIED value as 0

## Versioning Strategies

### API Versioning
```protobuf
package treasury.v1;  // Version in package name

option go_package = "example.com/go-mono-repo/proto/treasury/v1";
```

### Field Evolution

#### Adding Fields
```protobuf
message Account {
    string id = 1;
    string name = 2;
    // Safe to add new fields with unique numbers
    string email = 3;  // Added in v1.1
}
```

#### Deprecating Fields
```protobuf
message Account {
    string id = 1;
    string name = 2;
    string old_field = 3 [deprecated = true];
}
```

## Error Handling

### Standard Error Response
```protobuf
message ErrorDetail {
    string code = 1;      // Machine-readable error code
    string message = 2;   // Human-readable message
    map<string, string> metadata = 3;
}
```

### gRPC Status Codes
Use appropriate gRPC status codes:
- `OK` - Success
- `INVALID_ARGUMENT` - Client error in request
- `NOT_FOUND` - Resource doesn't exist
- `ALREADY_EXISTS` - Resource already exists
- `PERMISSION_DENIED` - Authorization failure
- `INTERNAL` - Server error

## Code Generation

### Makefile Pattern
```makefile
proto-{service}:
	@export PATH="$$PATH:$$(go env GOPATH)/bin" && \
		protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
		--go-grpc_out=. --go-grpc_opt=module=example.com/go-mono-repo \
		services/{domain}/{service}/proto/*.proto
```

### Import in Go Code
```go
import (
    pb "example.com/go-mono-repo/proto/treasury"
)
```

## Best Practices

### DO's
1. ✅ Use semantic field names
2. ✅ Include field comments for documentation
3. ✅ Use well-known types (Timestamp, Duration, etc.)
4. ✅ Design for forward compatibility
5. ✅ Use field numbers 1-15 for frequently used fields
6. ✅ Group related fields using nested messages

### DON'Ts
1. ❌ Reuse field numbers
2. ❌ Change field types
3. ❌ Rename fields (use deprecation instead)
4. ❌ Use required fields (proto3 doesn't support them)
5. ❌ Use map fields for ordered data
6. ❌ Use extremely large message sizes

## Testing Protobuf Services

### Unit Testing
```go
func TestGetManifest(t *testing.T) {
    server := &server{}
    req := &pb.ManifestRequest{}
    
    resp, err := server.GetManifest(context.Background(), req)
    
    assert.NoError(t, err)
    assert.Equal(t, "treasury-service", resp.Message)
}
```

### Integration Testing with grpcurl
```bash
# Test service endpoint
grpcurl -plaintext localhost:50052 treasury.Manifest/GetManifest

# List available services
grpcurl -plaintext localhost:50052 list

# Describe service
grpcurl -plaintext localhost:50052 describe treasury.Manifest
```

## Documentation

### Proto Comments
```protobuf
// Treasury service manages financial accounts and transactions.
service Treasury {
    // GetAccount retrieves account details by ID.
    // Returns NOT_FOUND if the account doesn't exist.
    rpc GetAccount (GetAccountRequest) returns (GetAccountResponse) {}
}

// Account represents a financial account.
message Account {
    // Unique identifier for the account.
    string id = 1;
    
    // Human-readable account name.
    string name = 2;
    
    // Current account balance.
    Money balance = 3;
}
```

### Generated Documentation
Consider using tools like:
- `protoc-gen-doc` for HTML/Markdown documentation
- `buf` for schema linting and breaking change detection
- `grpc-gateway` for REST API generation