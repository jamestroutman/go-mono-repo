# Payroll Service

## Overview
The Payroll Service manages payroll-related operations within the monorepo architecture. This service is designed to handle employee compensation, tax calculations, and payment processing as a dedicated business domain.

## Specifications

- [001 - Service Initialization](./specs/001-service-initialization.md) - Initial service setup with hello-world API

## Architecture Decision Records

- [Coming soon]

## Runbooks

- [Deployment Guide](./runbooks/deployment.md) - Coming soon
- [Troubleshooting Guide](./runbooks/troubleshooting.md) - Coming soon

## API Documentation

See the [proto file](../proto/payroll_service.proto) for detailed API documentation.

### Available Endpoints

- **Manifest Service**
  - `GetManifest` - Returns service identification information
  
- **Health Service**
  - `GetHealth` - Returns current health status
  - `GetLiveness` - Returns liveness status with uptime
  
- **Payroll Service**
  - `HelloWorld` - Test endpoint that returns a greeting message

## Development

This service runs within the devcontainer environment. See [DEVCONTAINER.md](/docs/DEVCONTAINER.md) for setup.

### Running the Service

```bash
# From devcontainer terminal
# Using Makefile
make payroll-service

# Or run directly
go run ./services/payroll-services/payroll-service/
```

### Configuration

- **Port**: 50053
- **Protocol**: gRPC with reflection enabled

## Testing

```bash
# Run unit tests (from devcontainer)
go test ./services/payroll-services/payroll-service/...

# Test with grpcurl (from devcontainer)
grpcurl -plaintext localhost:50053 payroll.Manifest/GetManifest
grpcurl -plaintext localhost:50053 payroll.Health/GetHealth
grpcurl -plaintext localhost:50053 payroll.Health/GetLiveness
grpcurl -plaintext -d '{"name": "Developer"}' localhost:50053 payroll.PayrollService/HelloWorld
```

## Future Features

The following features are planned for future development:
- Employee management
- Payroll calculation engine
- Tax computation
- Payment processing
- Reporting and analytics
- Integration with accounting systems

## Dependencies

### Current Dependencies
- gRPC and Protocol Buffers
- Go standard library

### Future Dependencies
- Database connection (PostgreSQL)
- Authentication service
- Notification service

## Contributing

When adding new features:
1. Start with a specification in `docs/specs/`
2. Get spec approved before implementation
3. Reference spec in code comments
4. Update this README with new capabilities