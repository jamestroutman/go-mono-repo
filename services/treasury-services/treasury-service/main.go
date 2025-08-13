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

	"github.com/jamestroutman/treasury-service/currency"
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
	
	// Create database manager
	// Spec: docs/specs/001-database-connection.md
	dbManager := NewDatabaseManager(&cfg.Database)
	
	// Connect to database synchronously with timeout
	// Spec: docs/specs/001-database-connection.md#story-4-graceful-degradation
	ctx := context.Background()
	log.Printf("Attempting to connect to database...")
	if err := dbManager.ConnectWithRetry(ctx, 5); err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
		log.Printf("Service will continue without database connection (degraded mode)")
	} else {
		// Database connected successfully, handle migrations
		// Spec: docs/specs/002-database-migrations.md#story-1-automated-migration-on-startup
		if cfg.Migration.AutoMigrate {
			log.Printf("Running database migrations...")
			migrationManager, err := NewMigrationManager(dbManager.GetDB(), &cfg.Migration)
			if err != nil {
				log.Printf("Warning: Failed to create migration manager: %v", err)
			} else {
				if err := migrationManager.Migrate(ctx); err != nil {
					log.Printf("Error: Failed to run migrations: %v", err)
					// In production, you might want to fail the service here
					// For now, continue in degraded mode
				} else {
					log.Printf("Database migrations completed successfully")
				}
				// Store migration manager for health checks
				dbManager.SetMigrationManager(migrationManager)
			}
		} else {
			log.Printf("Auto-migration disabled, skipping migrations")
		}
	}
	
	// Initialize currency server if database is available
	// Spec: docs/specs/003-currency-management.md
	var currencyServer *currency.Server
	if dbManager.GetDB() != nil {
		currencyManager := currency.NewManager(dbManager.GetDB())
		currencyServer = currency.NewServer(currencyManager)
		log.Printf("Currency service initialized")
	}
	
	// Initialize institution server if database is available
	// Spec: docs/specs/004-financial-institutions.md
	var institutionServer *InstitutionServer
	if dbManager.GetDB() != nil {
		institutionManager := NewInstitutionManager(dbManager.GetDB())
		institutionServer = NewInstitutionServer(institutionManager)
		log.Printf("Financial institution service initialized")
	}
	
	// Create manifest server with cached data
	// Spec: docs/specs/001-manifest.md
	manifestServer := NewManifestServer(cfg, startTime)
	
	// Create health server with database checker
	// Spec: docs/specs/003-health-check-liveness.md
	healthServer := NewHealthServerWithDB(startTime, dbManager, cfg)
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
	
	// Register currency service if available
	// Spec: docs/specs/003-currency-management.md
	if currencyServer != nil {
		pb.RegisterCurrencyServiceServer(grpcServer, currencyServer)
		log.Printf("Currency service registered with gRPC server")
	}
	
	// Register financial institution service if available
	// Spec: docs/specs/004-financial-institutions.md
	if institutionServer != nil {
		pb.RegisterFinancialInstitutionServiceServer(grpcServer, institutionServer)
		log.Printf("Financial institution service registered with gRPC server")
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
		
		// Close database connection
		// Spec: docs/specs/001-database-connection.md
		if err := dbManager.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
		
		grpcServer.GracefulStop()
	}()

	log.Printf("Treasury service ready on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}