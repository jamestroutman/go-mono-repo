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

	"example.com/go-mono-repo/common/tracing"
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
	startTime time.Time
	config    *Config
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
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	
	// Setup logging
	setupLogging(cfg)
	
	// Initialize tracing
	// Spec: docs/specs/004-opentelemetry-tracing.md#3-service-integration-pattern
	tracingCfg := tracing.TracingConfig{
		Enabled:        cfg.Tracing.Enabled,
		SentryDSN:      cfg.Tracing.SentryDSN,
		SampleRate:     cfg.Tracing.SampleRate,
		Environment:    cfg.Tracing.GetEnvironment(cfg.Environment),
		ServiceName:    cfg.Tracing.GetServiceName(cfg.ServiceName),
		ServiceVersion: cfg.Tracing.GetServiceVersion(cfg.ServiceVersion),
	}
	
	cleanup, err := tracing.InitializeTracing(tracingCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer cleanup()
	
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
	
	// Create health server
	// Spec: docs/specs/003-health-check-liveness.md
	healthServer := NewHealthServer(cfg, startTime)
	healthServer.SetConfigLoaded(true) // Mark config as loaded after successful validation
	
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
	if cfg.EnvFilePath != "" {
		fmt.Printf("Config File: %s\n", cfg.EnvFilePath)
	}
	fmt.Println("=================================")
	
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create gRPC server with tracing interceptors
	// Spec: docs/specs/004-opentelemetry-tracing.md#2-grpc-interceptors
	unaryInterceptor, streamInterceptor := tracing.NewServerInterceptors()
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(unaryInterceptor),
		grpc.StreamInterceptor(streamInterceptor),
	)
	pb.RegisterManifestServer(grpcServer, manifestServer)
	pb.RegisterHealthServer(grpcServer, healthServer)
	pb.RegisterPayrollServiceServer(grpcServer, srv)
	
	// Mark gRPC as ready after registration
	// Spec: docs/specs/003-health-check-liveness.md
	healthServer.SetGRPCReady(true)
	
	// Register reflection service for debugging
	reflection.Register(grpcServer)
	
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