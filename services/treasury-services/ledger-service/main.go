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

	pb "example.com/go-mono-repo/proto/ledger"
	"clarity/treasury-services/ledger-service/account"
	"clarity/treasury-services/ledger-service/pkg/migration"
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
	
	// Initialize ImmuDB connection
	// Spec: docs/specs/001-immudb-connection.md
	var immuDBManager *ImmuDBManager
	if cfg.ImmuDB != nil {
		log.Println("Initializing ImmuDB connection...")
		immuDBManager = NewImmuDBManager(cfg.ImmuDB)
		
		// Attempt to connect with graceful degradation
		// Spec: docs/specs/001-immudb-connection.md#story-5-graceful-degradation
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		if err := immuDBManager.Connect(ctx); err != nil {
			log.Printf("Warning: Failed to connect to ImmuDB: %v", err)
			log.Printf("Service will continue without database persistence")
			// Service continues without ImmuDB per graceful degradation spec
		} else {
			// Add ImmuDB health checker
			immuDBChecker := NewImmuDBChecker(immuDBManager)
			healthServer.AddDependencyChecker(immuDBChecker)
			log.Println("ImmuDB connection established and health check registered")
			
			// Add migration health checker and run migrations if configured
			// Spec: docs/specs/002-database-migrations.md
			if cfg.Migration != nil {
				// Keep the migration path relative to the service directory
				// The service runs from its own directory
				
				migrationChecker := migration.NewMigrationChecker(immuDBManager.GetClient(), cfg.Migration)
				healthServer.AddDependencyChecker(migrationChecker)
				
				// Run migrations on boot if configured
				// Spec: docs/specs/002-database-migrations.md#story-3-on-boot-migration-execution
				if cfg.Migration.RunOnBoot {
					log.Println("Running database migrations on boot...")
					migCtx, migCancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer migCancel()
					
					if err := migrationChecker.RunPendingMigrations(migCtx); err != nil {
						log.Fatalf("Failed to run migrations: %v", err)
					}
					log.Println("Database migrations completed successfully")
				}
				
				// Log migration status
				summary := migrationChecker.GetMigrationSummary(context.Background())
				log.Printf("Migration status: %s", summary)
			}
		}
	} else {
		log.Println("ImmuDB configuration not found, running in memory-only mode")
	}
	
	// Log configuration and manifest info at startup
	fmt.Println("=================================")
	fmt.Println("    LEDGER SERVICE STARTING     ")
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
	if cfg.ImmuDB != nil {
		fmt.Printf("ImmuDB: %s:%d/%s\n", cfg.ImmuDB.Host, cfg.ImmuDB.Port, cfg.ImmuDB.Database)
	}
	fmt.Println("=================================")
	
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterManifestServer(grpcServer, manifestServer)
	pb.RegisterHealthServer(grpcServer, healthServer)
	
	// Register Account Service if ImmuDB is connected
	// Spec: docs/specs/003-account-management.md
	if immuDBManager != nil && immuDBManager.GetClient() != nil {
		accountServer := account.NewServer(immuDBManager.GetClient())
		pb.RegisterAccountServiceServer(grpcServer, accountServer)
		log.Println("Account management service registered")
	} else {
		log.Println("Account management service not available (ImmuDB not connected)")
	}
	
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
		
		// Disconnect from ImmuDB if connected
		// Spec: docs/specs/001-immudb-connection.md#story-2-connection-pool-management
		if immuDBManager != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := immuDBManager.Disconnect(ctx); err != nil {
				log.Printf("Warning: Error disconnecting from ImmuDB: %v", err)
			}
		}
		
		grpcServer.GracefulStop()
	}()

	log.Printf("Ledger service ready on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}