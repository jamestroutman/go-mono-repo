# Service Development Guide

## Creating a New Service

This guide walks through the complete process of adding a new microservice to the monorepo.

## Step-by-Step Process

### 1. Plan Your Service

Before creating a service, determine:
- **Domain**: Which business domain does it belong to?
- **Name**: Clear, descriptive service name
- **Port**: Unique port number (check existing services)
- **Dependencies**: What other services will it interact with?

### 2. Create Service Structure

```bash
# Create service directory
mkdir -p services/{domain}/{service-name}/proto

# Navigate to service directory
cd services/{domain}/{service-name}
```

### 3. Initialize Go Module

```bash
# Initialize the module
go mod init github.com/{org}/{service-name}

# Add to workspace from repo root
go work use ./services/{domain}/{service-name}
```

### 4. Define Protocol Buffers

Create `proto/{service}_service.proto`:

```protobuf
syntax = "proto3";

package {package_name};

option go_package = "example.com/go-mono-repo/proto/{package_name}";

// Standard Manifest service
service Manifest {
    rpc GetManifest (ManifestRequest) returns (ManifestResponse) {}
}

message ManifestRequest {}

message ManifestResponse {
    string message = 1;
    string version = 2;
}

// Your service definition
service {ServiceName} {
    // Add your RPC methods here
}
```

### 5. Implement Service

Create `main.go`:

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

    pb "example.com/go-mono-repo/proto/{package_name}"
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
)

type server struct {
    pb.Unimplemented{ServiceName}Server
    pb.UnimplementedManifestServer
}

func (s *server) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
    return &pb.ManifestResponse{
        Message: "{service-name}",
        Version: "1.0.0",
    }, nil
}

// Implement your service methods here

func main() {
    port := ":{port_number}"
    
    lis, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    s := grpc.NewServer()
    
    // Register services
    pb.RegisterManifestServer(s, &server{})
    pb.Register{ServiceName}Server(s, &server{})
    
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

### 6. Add Makefile Target

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

### 7. Update Documentation

Update `README.md` with:
- Service description in the Services section
- Port assignment
- Testing instructions

Update `CLAUDE.md` with:
- New make command
- grpcurl test command
- Port information

### 8. Sync and Test

```bash
# Sync workspace modules
go work sync

# Generate proto and run service
make {service-name}

# Test the service (in another terminal)
grpcurl -plaintext localhost:{port} {package}.Manifest/GetManifest
```

## Service Implementation Patterns

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

Create `main_test.go`:

```go
package main

import (
    "context"
    "testing"
    
    pb "example.com/go-mono-repo/proto/{package_name}"
    "github.com/stretchr/testify/assert"
)

func TestGetManifest(t *testing.T) {
    s := &server{}
    
    resp, err := s.GetManifest(context.Background(), &pb.ManifestRequest{})
    
    assert.NoError(t, err)
    assert.Equal(t, "{service-name}", resp.Message)
    assert.Equal(t, "1.0.0", resp.Version)
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

1. **Always implement the Manifest service** for service discovery
2. **Use graceful shutdown** to handle termination signals
3. **Enable gRPC reflection** in development for easier debugging
4. **Implement proper error handling** with appropriate gRPC status codes
5. **Add structured logging** for better observability
6. **Write tests** for all RPC methods
7. **Document your service** in README and inline comments
8. **Use interceptors** for cross-cutting concerns
9. **Implement health checks** for production readiness
10. **Keep services focused** on a single domain