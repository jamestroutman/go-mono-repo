package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "example.com/go-mono-repo/proto/treasury"
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

func main() {
	// Load configuration
	// Spec: docs/specs/002-configuration-management.md#usage-in-maingo
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Validate configuration
	// Spec: docs/specs/002-configuration-management.md#configuration-validation
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	
	// Setup logging
	setupLogging(cfg)
	
	startTime := time.Now()
	port := cfg.GetPort()
	
	// Create manifest server with cached data
	// Spec: docs/specs/001-manifest.md
	manifestServer := NewManifestServer(cfg, startTime)
	
	// Create health server
	// Spec: docs/specs/003-health-check-liveness.md
	healthServer := NewHealthServer(startTime)
	healthServer.SetConfigLoaded(true) // Mark config as loaded after successful validation
	
	// Log configuration and manifest info at startup
	fmt.Println("=================================")
	fmt.Println("   TREASURY SERVICE STARTING    ")
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
	pb.RegisterManifestServer(grpcServer, manifestServer)
	pb.RegisterHealthServer(grpcServer, healthServer)
	
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

	log.Printf("Treasury service ready on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}