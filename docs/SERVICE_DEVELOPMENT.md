# Service Development Guide

## Creating a New Service

This guide walks through the complete process of adding a new microservice to the monorepo following our spec-driven development approach.

## Prerequisites

- VS Code with Dev Containers extension installed
- Docker Desktop or Docker Engine running
- Understanding of the [Spec-Driven Development](./SPEC_DRIVEN_DEVELOPMENT.md) process
- Familiarity with [Protobuf Patterns](./PROTOBUF_PATTERNS.md)

### Development Environment Setup

1. **Open the repository in VS Code**
2. **Reopen in Container** when prompted (or use Command Palette: "Dev Containers: Reopen in Container")
3. **Wait for container build** - all tools will be automatically installed
4. **Verify environment**: 
   ```bash
   go version  # Should show Go 1.24+
   protoc --version  # Should show libprotoc version
   ```

For detailed devcontainer setup, see [DEVCONTAINER.md](./DEVCONTAINER.md)

## Step-by-Step Process

### 1. Write the Specification (REQUIRED)

**All new services and features MUST start with an approved specification.**

#### Create Initial Service Spec

```bash
# Create service documentation structure
mkdir -p services/{domain}/{service-name}/docs/specs

# Copy the spec template
cp docs/SPEC_TEMPLATE.md services/{domain}/{service-name}/docs/specs/001-service-initialization.md
```

#### Define Service Requirements

In your spec, clearly define:
- **Problem Statement**: Why this service is needed
- **Service Boundaries**: What this service will and won't do
- **API Contract**: Initial RPC methods and messages
- **Dependencies**: Other services it will interact with
- **Port Assignment**: Unique port number (check existing services)
- **Success Metrics**: How to measure if the service is successful

#### Get Spec Approved

- Create a pull request with your spec
- Get technical and stakeholder review
- Update spec based on feedback
- Mark spec as "Approved" before proceeding

### 2. Plan Your Service Implementation

Based on your approved spec, determine:
- **Domain**: Which business domain does it belong to?
- **Name**: Clear, descriptive service name (from spec)
- **Port**: Unique port number (from spec)
- **Dependencies**: What other services will it interact with? (from spec)

### 3. Create Service Structure

```bash
# Create service directory
mkdir -p services/{domain}/{service-name}/proto

# Navigate to service directory
cd services/{domain}/{service-name}
```

### 4. Initialize Go Module

```bash
# Initialize the module
go mod init github.com/{org}/{service-name}

# Add to workspace from repo root
go work use ./services/{domain}/{service-name}
```

### 5. Define Protocol Buffers

Create `proto/{service}_service.proto` based on your approved spec:

```protobuf
syntax = "proto3";

package {package_name};

option go_package = "example.com/go-mono-repo/proto/{package_name}";

// Service definition based on approved specification
// Spec: docs/specs/001-service-initialization.md

// Standard Manifest service (Required)
// Spec: docs/specs/001-manifest.md
service Manifest {
    rpc GetManifest (ManifestRequest) returns (ManifestResponse) {}
}

message ManifestRequest {}

message ManifestResponse {
    string message = 1;
    string version = 2;
}

// Health service (Required)
// Spec: docs/specs/003-health-check-liveness.md
service Health {
    rpc GetHealth (HealthRequest) returns (HealthResponse) {}
    rpc GetLiveness (LivenessRequest) returns (LivenessResponse) {}
}

message HealthRequest {}
message HealthResponse {
    bool healthy = 1;
    string status = 2;
    map<string, string> details = 3;
}

message LivenessRequest {}
message LivenessResponse {
    bool alive = 1;
    int64 uptime_seconds = 2;
}

// Your service definition
// Spec: docs/specs/001-service-initialization.md#api-design
service {ServiceName} {
    // Add your RPC methods here as defined in the spec
}
```

### 6. Implement Service

Create `main.go` implementing the methods defined in your spec:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"

    pb "example.com/go-mono-repo/proto/{package_name}"
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
)

type server struct {
    pb.Unimplemented{ServiceName}Server
    pb.UnimplementedManifestServer
    pb.UnimplementedHealthServer
    startTime time.Time
}

// GetManifest implements the Manifest service
// Spec: docs/specs/001-manifest.md
func (s *server) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
    return &pb.ManifestResponse{
        Message: "{service-name}",
        Version: "1.0.0",
    }, nil
}

// GetHealth implements health check
// Spec: docs/specs/003-health-check-liveness.md
func (s *server) GetHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
    return &pb.HealthResponse{
        Healthy: true,
        Status:  "serving",
        Details: map[string]string{
            "service": "{service-name}",
            "version": "1.0.0",
        },
    }, nil
}

// GetLiveness implements liveness check
// Spec: docs/specs/003-health-check-liveness.md
func (s *server) GetLiveness(ctx context.Context, req *pb.LivenessRequest) (*pb.LivenessResponse, error) {
    return &pb.LivenessResponse{
        Alive:          true,
        UptimeSeconds:  int64(time.Since(s.startTime).Seconds()),
    }, nil
}

// Implement your service methods here
// Each method should reference its governing spec section

func main() {
    port := ":{port_number}"
    
    lis, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    srv := &server{
        startTime: time.Now(),
    }

    s := grpc.NewServer()
    
    // Register services
    pb.RegisterManifestServer(s, srv)
    pb.RegisterHealthServer(s, srv)
    pb.Register{ServiceName}Server(s, srv)
    
    // Enable reflection for debugging
    reflection.Register(s)
    
    // Graceful shutdown
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
        <-sigChan
        fmt.Println("\nShutting down gracefully...")
        s.GracefulStop()
    }()
    
    fmt.Printf("{Service} starting on port %s\n", port)
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### 7. Add Makefile Target

Update the root `Makefile`:

```makefile
{service-name}:
	@echo "Starting {service-name}..."
	@echo "Generating protobuf code for {service-name}..."
	@export PATH="$$PATH:$$(go env GOPATH)/bin" && \
		protoc --go_out=. --go_opt=module=example.com/go-mono-repo \
		--go-grpc_out=. --go-grpc_opt=module=example.com/go-mono-repo \
		services/{domain}/{service-name}/proto/{service}_service.proto
	@echo "✓ Protobuf code generated"
	@echo "Running {service-name}..."
	@go run ./services/{domain}/{service-name}/main.go
```

### 8. Create Service Documentation

#### Create Documentation Structure

```bash
# From within the devcontainer terminal
# Create documentation directories
mkdir -p services/{domain}/{service-name}/docs/{specs,adrs,runbooks}

# Create service README
cat > services/{domain}/{service-name}/docs/README.md << 'EOF'
# {Service Name}

## Overview
[Brief description of the service]

## Specifications

- [001 - Service Initialization](./specs/001-service-initialization.md) - Initial service setup and core functionality

## Architecture Decision Records

- [Coming soon]

## Runbooks

- [Deployment Guide](./runbooks/deployment.md)
- [Troubleshooting Guide](./runbooks/troubleshooting.md)

## API Documentation

See the [proto file](../proto/{service}_service.proto) for detailed API documentation.

## Development

This service runs within the devcontainer environment. See [DEVCONTAINER.md](/docs/DEVCONTAINER.md) for setup.

## Testing

```bash
# Run unit tests (from devcontainer)
go test ./...

# Test with grpcurl (from devcontainer)
grpcurl -plaintext localhost:{port} {package}.Manifest/GetManifest
grpcurl -plaintext localhost:{port} {package}.Health/GetHealth
grpcurl -plaintext localhost:{port} {package}.Health/GetLiveness
```
EOF
```

### 9. Update Root Documentation

Update `README.md` with:
- Service description in the Services section
- Port assignment
- Testing instructions

Update `CLAUDE.md` with:
- New make command
- grpcurl test command
- Port information

### 10. Sync and Test

```bash
# Sync workspace modules
go work sync

# Generate proto and run service
make {service-name}

# Test the service (in another terminal)
grpcurl -plaintext localhost:{port} {package}.Manifest/GetManifest
```

## Service Implementation Patterns

### Linking Specifications to Code

**IMPORTANT**: All implementations MUST reference their governing specifications in code comments.

#### In Service Methods

```go
// CreateResource implements resource creation
// Spec: docs/specs/002-resource-management.md#story-1-create-resource
func (s *server) CreateResource(ctx context.Context, req *pb.CreateResourceRequest) (*pb.CreateResourceResponse, error) {
    // Validate input per spec requirements
    if err := validateCreateRequest(req); err != nil {
        return nil, status.Error(codes.InvalidArgument, err.Error())
    }
    
    // Implementation following spec...
}
```

#### In Tests

```go
// TestCreateResource verifies resource creation meets spec requirements
// Spec: docs/specs/002-resource-management.md#acceptance-criteria
func TestCreateResource(t *testing.T) {
    // Test each acceptance criterion from the spec
    t.Run("validates required fields", func(t *testing.T) {
        // Spec: Criterion 1 - Must validate required fields
    })
    
    t.Run("returns proper error codes", func(t *testing.T) {
        // Spec: Criterion 2 - Must return appropriate gRPC status codes
    })
}
```

### Configuration Management

```go
type Config struct {
    Port        string
    DatabaseURL string
    LogLevel    string
}

func loadConfig() *Config {
    return &Config{
        Port:        getEnv("PORT", "50053"),
        DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost/db"),
        LogLevel:    getEnv("LOG_LEVEL", "info"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### Logging

```go
import (
    "log/slog"
)

func setupLogger(level string) {
    var logLevel slog.Level
    switch level {
    case "debug":
        logLevel = slog.LevelDebug
    case "warn":
        logLevel = slog.LevelWarn
    case "error":
        logLevel = slog.LevelError
    default:
        logLevel = slog.LevelInfo
    }
    
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: logLevel,
    }))
    
    slog.SetDefault(logger)
}
```

### Error Handling

```go
import (
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

func (s *server) GetResource(ctx context.Context, req *pb.GetResourceRequest) (*pb.GetResourceResponse, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "resource ID is required")
    }
    
    resource, err := s.db.GetResource(req.Id)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            return nil, status.Error(codes.NotFound, "resource not found")
        }
        slog.Error("Failed to get resource", "error", err, "id", req.Id)
        return nil, status.Error(codes.Internal, "internal server error")
    }
    
    return &pb.GetResourceResponse{Resource: resource}, nil
}
```

### Middleware/Interceptors

```go
func loggingInterceptor() grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
        start := time.Now()
        
        resp, err := handler(ctx, req)
        
        duration := time.Since(start)
        statusCode := codes.OK
        if err != nil {
            statusCode = status.Code(err)
        }
        
        slog.Info("gRPC request",
            "method", info.FullMethod,
            "duration", duration,
            "status", statusCode,
        )
        
        return resp, err
    }
}

// In main()
s := grpc.NewServer(
    grpc.UnaryInterceptor(loggingInterceptor()),
)
```

## Testing

### Unit Tests

Create `main_test.go` that validates your spec's acceptance criteria:

```go
package main

import (
    "context"
    "testing"
    
    pb "example.com/go-mono-repo/proto/{package_name}"
    "github.com/stretchr/testify/assert"
)

// TestGetManifest verifies manifest endpoint meets requirements
// Spec: docs/specs/001-manifest.md
func TestGetManifest(t *testing.T) {
    s := &server{}
    
    resp, err := s.GetManifest(context.Background(), &pb.ManifestRequest{})
    
    assert.NoError(t, err)
    assert.Equal(t, "{service-name}", resp.Message)
    assert.Equal(t, "1.0.0", resp.Version)
}

// TestHealthCheck verifies health endpoint meets requirements  
// Spec: docs/specs/003-health-check-liveness.md
func TestHealthCheck(t *testing.T) {
    s := &server{startTime: time.Now()}
    
    t.Run("health check returns healthy status", func(t *testing.T) {
        resp, err := s.GetHealth(context.Background(), &pb.HealthRequest{})
        assert.NoError(t, err)
        assert.True(t, resp.Healthy)
        assert.Equal(t, "serving", resp.Status)
    })
    
    t.Run("liveness check returns alive status", func(t *testing.T) {
        resp, err := s.GetLiveness(context.Background(), &pb.LivenessRequest{})
        assert.NoError(t, err)
        assert.True(t, resp.Alive)
        assert.GreaterOrEqual(t, resp.UptimeSeconds, int64(0))
    })
}
```

### Integration Tests

```go
func TestServiceIntegration(t *testing.T) {
    // Start test server
    lis := bufconn.Listen(1024 * 1024)
    s := grpc.NewServer()
    pb.RegisterManifestServer(s, &server{})
    
    go func() {
        if err := s.Serve(lis); err != nil {
            t.Fatalf("Server exited with error: %v", err)
        }
    }()
    
    // Create client
    ctx := context.Background()
    conn, err := grpc.DialContext(ctx, "bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
    assert.NoError(t, err)
    defer conn.Close()
    
    client := pb.NewManifestClient(conn)
    
    // Test request
    resp, err := client.GetManifest(ctx, &pb.ManifestRequest{})
    assert.NoError(t, err)
    assert.NotNil(t, resp)
}
```

## Deployment

### Dockerfile

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy workspace files
COPY go.work go.work.sum ./
COPY go.mod go.sum ./

# Copy service module
COPY services/{domain}/{service-name}/go.mod services/{domain}/{service-name}/go.sum ./services/{domain}/{service-name}/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the service
RUN CGO_ENABLED=0 GOOS=linux go build -o service ./services/{domain}/{service-name}

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/service .

EXPOSE {port}

CMD ["./service"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  {service-name}:
    build:
      context: .
      dockerfile: services/{domain}/{service-name}/Dockerfile
    ports:
      - "{port}:{port}"
    environment:
      - LOG_LEVEL=info
    networks:
      - services-network

networks:
  services-network:
    driver: bridge
```

## Monitoring

### Health Checks

```go
import (
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
)

// In main()
healthServer := health.NewServer()
grpc_health_v1.RegisterHealthServer(s, healthServer)

// Set service status
healthServer.SetServingStatus("{service-name}", grpc_health_v1.HealthCheckResponse_SERVING)
```

### Metrics

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    requestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "grpc_requests_total",
            Help: "Total number of gRPC requests",
        },
        []string{"method", "status"},
    )
)

func init() {
    prometheus.MustRegister(requestsTotal)
}

// Start metrics server
go func() {
    http.Handle("/metrics", promhttp.Handler())
    log.Fatal(http.ListenAndServe(":9090", nil))
}()
```

## Common Issues and Solutions

### Issue: Proto generation fails
**Solution**: Ensure protoc and Go plugins are installed:
```bash
make install-reqs
```

### Issue: Module not found
**Solution**: Sync workspace modules:
```bash
go work sync
```

### Issue: Port already in use
**Solution**: Check for running services:
```bash
lsof -i :{port}
```

### Issue: Import errors
**Solution**: Ensure correct module path in proto `go_package` option

## Best Practices

1. **Start with a specification** - Never implement without an approved spec
2. **Reference specs in code** - Every implementation should link to its governing spec
3. **Always implement the Manifest service** for service discovery (Spec: docs/specs/001-manifest.md)
4. **Always implement Health/Liveness checks** for production readiness (Spec: docs/specs/003-health-check-liveness.md)
5. **Use graceful shutdown** to handle termination signals
6. **Enable gRPC reflection** in development for easier debugging
7. **Implement proper error handling** with appropriate gRPC status codes
8. **Add structured logging** for better observability
9. **Write tests** that verify spec acceptance criteria
10. **Document your service** following the documentation structure
11. **Use interceptors** for cross-cutting concerns
12. **Keep services focused** on a single domain
13. **Update specs when requirements change** - Keep specs as living documents