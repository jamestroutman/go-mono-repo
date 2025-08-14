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

	pb "example.com/go-mono-repo/proto/payroll"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// setupLogging configures logging based on config
func setupLogging(cfg *Config) {
	// For now, use standard log package
	// In production, you might want to use a structured logger like zap or logrus
	if cfg.LogLevel == "debug" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}
}

type server struct {
	pb.UnimplementedPayrollServiceServer
	pb.UnimplementedHealthServer
	startTime time.Time
	config    *Config
}

// GetHealth implements health check
// Spec: docs/specs/003-health-check-liveness.md
func (s *server) GetHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Healthy: true,
		Status:  "serving",
		Details: map[string]string{
			"service": s.config.ServiceName,
			"version": s.config.ServiceVersion,
		},
	}, nil
}

// GetLiveness implements liveness check
// Spec: docs/specs/003-health-check-liveness.md
func (s *server) GetLiveness(ctx context.Context, req *pb.LivenessRequest) (*pb.LivenessResponse, error) {
	return &pb.LivenessResponse{
		Alive:         true,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
	}, nil
}

// HelloWorld implements the hello world endpoint
// Spec: services/payroll-services/payroll-service/docs/specs/001-service-initialization.md
func (s *server) HelloWorld(ctx context.Context, req *pb.HelloWorldRequest) (*pb.HelloWorldResponse, error) {
	name := req.Name
	if name == "" {
		name = "World"
	}
	return &pb.HelloWorldResponse{
		Message: fmt.Sprintf("Hello, %s! From Payroll Service", name),
	}, nil
}

func main() {
	// Load configuration
	log.Printf("Loading configuration...")
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Validate configuration
	log.Printf("Validating configuration...")
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	
	// Setup logging
	setupLogging(cfg)
	
	startTime := time.Now()
	port := cfg.GetPort()
	
	// Initialize server
	srv := &server{
		startTime: startTime,
		config:    cfg,
	}
	
	// Create manifest server with cached data
	// Spec: docs/specs/002-manifest-implementation.md
	manifestServer := NewManifestServer(cfg, startTime)
	
	// Log configuration and manifest info at startup
	fmt.Println("=================================")
	fmt.Println("   PAYROLL SERVICE STARTING     ")
	fmt.Println("=================================")
	fmt.Printf("Service: %s v%s\n", cfg.ServiceName, cfg.ServiceVersion)
	fmt.Printf("Environment: %s\n", cfg.Environment)
	fmt.Printf("Region: %s\n", cfg.Region)
	fmt.Printf("Port: %d\n", cfg.Port)
	manifestCache := manifestServer.GetManifestCache()
	fmt.Printf("Instance ID: %s\n", manifestCache.RuntimeInfo.InstanceId)
	fmt.Printf("Git Commit: %s\n", manifestCache.BuildInfo.CommitHash)
	fmt.Printf("Git Branch: %s\n", manifestCache.BuildInfo.Branch)
	fmt.Printf("Log Level: %s\n", cfg.LogLevel)
	fmt.Printf("Features: %v\n", cfg.EnabledFeatures)
	fmt.Println("=================================")
	
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	
	// Register services
	pb.RegisterManifestServer(grpcServer, manifestServer)
	log.Printf("Manifest service registered")
	
	pb.RegisterHealthServer(grpcServer, srv)
	log.Printf("Health check service registered")
	
	pb.RegisterPayrollServiceServer(grpcServer, srv)
	log.Printf("Payroll service registered with gRPC server")
	
	// Register reflection service for debugging
	reflection.Register(grpcServer)
	log.Printf("gRPC reflection enabled")
	
	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Payroll service ready on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}