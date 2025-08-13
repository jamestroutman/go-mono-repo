package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	"clarity/treasury-services/ledger-service/pkg/migration"
	"github.com/codenotary/immudb/pkg/client"
	"github.com/joho/godotenv"
)

const version = "1.0.0"

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}
	
	// Define commands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	
	command := os.Args[1]
	
	// Setup flags
	flagSet := flag.NewFlagSet(command, flag.ExitOnError)
	_ = flagSet.String("config", "", "Config file path") // Reserved for future use
	dryRun := flagSet.Bool("dry-run", false, "Show what would be executed")
	migrationsPath := flagSet.String("migrations", "./migrations", "Migration files path")
	serviceName := flagSet.String("service", "ledger", "Service name")
	timeout := flagSet.Duration("timeout", 30*time.Second, "Migration timeout")
	verbose := flagSet.Bool("verbose", false, "Enable verbose logging")
	
	// Parse remaining args
	flagSet.Parse(os.Args[2:])
	
	// Setup logging
	if !*verbose {
		log.SetFlags(0)
	}
	
	// Connect to ImmuDB
	immuClient, err := connectToImmuDB()
	if err != nil {
		log.Fatalf("Failed to connect to ImmuDB: %v", err)
	}
	defer immuClient.Logout(context.Background())
	
	// Create migration config
	config := &migration.MigrationConfig{
		MigrationsPath: *migrationsPath,
		DryRun:         *dryRun,
		Timeout:        *timeout,
		TableName:      fmt.Sprintf("%s_schema_migrations", *serviceName),
		ServiceName:    *serviceName,
	}
	
	// Create migration manager
	manager := migration.NewMigrationManager(immuClient, config)
	
	ctx := context.Background()
	
	switch command {
	case "up":
		if err := runMigrations(ctx, manager, *dryRun); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		
	case "status":
		if err := showStatus(ctx, manager); err != nil {
			log.Fatalf("Failed to get status: %v", err)
		}
		
	case "validate":
		if err := validateMigrations(manager); err != nil {
			log.Fatalf("Validation failed: %v", err)
		}
		
	case "create":
		if flagSet.NArg() < 1 {
			log.Fatal("Usage: migrate create <name>")
		}
		name := flagSet.Arg(0)
		if err := createMigration(manager, name); err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		
	case "version":
		fmt.Printf("ledger-service migration tool v%s\n", version)
		
	case "help", "--help", "-h":
		printUsage()
		
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func connectToImmuDB() (client.ImmuClient, error) {
	// Get configuration from environment
	database := getEnv("IMMUDB_DATABASE", "defaultdb")
	username := getEnv("IMMUDB_USERNAME", "immudb")
	password := getEnv("IMMUDB_PASSWORD", "immudb")
	
	// Create client with options
	opts := client.DefaultOptions().
		WithAddress(getEnv("IMMUDB_HOST", "immudb")).
		WithPort(getEnvInt("IMMUDB_PORT", 3322))
	
	// Create client with proper options
	immuClient := client.NewClient().WithOptions(opts)
	
	// Open session with options
	err := immuClient.OpenSession(context.Background(), []byte(username), []byte(password), database)
	if err != nil {
		return nil, fmt.Errorf("failed to open session: %w", err)
	}
	
	return immuClient, nil
}

func runMigrations(ctx context.Context, manager *migration.MigrationManager, dryRun bool) error {
	if dryRun {
		log.Println("Running in DRY RUN mode - no changes will be made")
	}
	
	log.Println("Running database migrations...")
	
	if err := manager.Run(ctx); err != nil {
		return err
	}
	
	if !dryRun {
		log.Println("Migrations completed successfully")
	}
	
	return nil
}

func showStatus(ctx context.Context, manager *migration.MigrationManager) error {
	status, err := manager.Status(ctx)
	if err != nil {
		return err
	}
	
	fmt.Println("\nMigration Status for ledger-service")
	fmt.Println("====================================")
	fmt.Printf("Database: %s @ %s:%d\n", 
		getEnv("IMMUDB_DATABASE", "defaultdb"),
		getEnv("IMMUDB_HOST", "localhost"),
		getEnvInt("IMMUDB_PORT", 3322))
	config := manager.GetConfig()
	fmt.Printf("Tracking Table: %s_schema_migrations\n", config.ServiceName)
	fmt.Printf("Migration Path: %s\n", config.MigrationsPath)
	
	if len(status.Applied) > 0 {
		fmt.Printf("\nApplied Migrations (%d):\n", len(status.Applied))
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, m := range status.Applied {
			fmt.Fprintf(w, "  ✓ %03d_%s\t(%s, %dms)\n", 
				m.Version, m.Name, 
				m.ExecutedAt.Format("2006-01-02 15:04:05"),
				m.ExecutionTime)
		}
		w.Flush()
	} else {
		fmt.Println("\nNo applied migrations")
	}
	
	if len(status.Pending) > 0 {
		fmt.Printf("\nPending Migrations (%d):\n", len(status.Pending))
		for _, m := range status.Pending {
			fmt.Printf("  • %03d_%s\n", m.Version, m.Name)
		}
	} else {
		fmt.Println("\nNo pending migrations")
	}
	
	fmt.Println("\nSummary:")
	fmt.Printf("  Service:  %s\n", config.ServiceName)
	fmt.Printf("  Applied:  %d\n", len(status.Applied))
	fmt.Printf("  Pending:  %d\n", len(status.Pending))
	fmt.Printf("  Total:    %d\n", status.Total)
	if status.LastRun != nil {
		fmt.Printf("  Last Run: %s\n", status.LastRun.Format("2006-01-02 15:04:05"))
	}
	
	return nil
}

func validateMigrations(manager *migration.MigrationManager) error {
	log.Println("Validating migration files...")
	
	if err := manager.Validate(); err != nil {
		return err
	}
	
	log.Println("All migration files are valid")
	return nil
}

func createMigration(manager *migration.MigrationManager, name string) error {
	// Sanitize name - replace spaces with underscores, remove special chars
	sanitized := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
		   (r >= '0' && r <= '9') || r == '_' {
			sanitized += string(r)
		} else if r == ' ' || r == '-' {
			sanitized += "_"
		}
	}
	
	if sanitized == "" {
		return fmt.Errorf("invalid migration name")
	}
	
	return manager.CreateMigration(sanitized)
}

func printUsage() {
	fmt.Println("ledger-service database migration tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  migrate [command]")
	fmt.Println()
	fmt.Println("Available Commands:")
	fmt.Println("  up          Run pending migrations")
	fmt.Println("  status      Show migration status")
	fmt.Println("  validate    Validate migration files")
	fmt.Println("  create      Create new migration file")
	fmt.Println("  version     Show migration tool version")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --config string     Config file path")
	fmt.Println("  --dry-run          Show what would be executed")
	fmt.Println("  --migrations string Migration files path (default \"./migrations\")")
	fmt.Println("  --service string    Service name (default \"ledger\")")
	fmt.Println("  --timeout duration  Migration timeout (default 30s)")
	fmt.Println("  --verbose          Enable verbose logging")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # From repo root")
	fmt.Println("  go run ./services/treasury-services/ledger-service/cmd/migrate up")
	fmt.Println("  go run ./services/treasury-services/ledger-service/cmd/migrate status")
	fmt.Println()
	fmt.Println("  # Using make commands (preferred)")
	fmt.Println("  make migrate-ledger")
	fmt.Println("  make migrate-ledger-status")
	fmt.Println("  make migration-ledger-new NAME=add_indexes")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var i int
		if _, err := fmt.Sscanf(value, "%d", &i); err == nil {
			return i
		}
	}
	return defaultValue
}