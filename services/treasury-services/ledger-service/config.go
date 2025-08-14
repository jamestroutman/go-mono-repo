package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"clarity/treasury-services/ledger-service/pkg/migration"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the ledger service
// Spec: docs/specs/002-configuration-management.md
type Config struct {
	// Service Identity
	ServiceName        string `envconfig:"SERVICE_NAME" default:"ledger-service"`
	ServiceVersion     string `envconfig:"SERVICE_VERSION" default:"1.0.0"`
	ServiceDescription string `envconfig:"SERVICE_DESCRIPTION" default:"Ledger service for managing financial accounts and transactions"`
	APIVersion         string `envconfig:"API_VERSION" default:"v1"`

	// Runtime Configuration
	Port        int    `envconfig:"PORT" default:"50051"`
	Environment string `envconfig:"ENVIRONMENT" default:"dev"`
	Region      string `envconfig:"REGION" default:"local"`

	// Service Metadata
	ServiceOwner   string `envconfig:"SERVICE_OWNER" default:"platform-team@example.com"`
	RepoURL        string `envconfig:"REPO_URL" default:"https://github.com/example/go-mono-repo"`
	DocsURL        string `envconfig:"DOCS_URL" default:"https://docs.example.com/ledger-service"`
	SupportContact string `envconfig:"SUPPORT_CONTACT" default:"default@example.com"`
	ServiceTier    string `envconfig:"SERVICE_TIER" default:"1"`

	// Features
	EnabledFeatures []string `envconfig:"ENABLED_FEATURES" default:"base,manifest"`

	// Logging
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat string `envconfig:"LOG_FORMAT" default:"json"`

	// Labels - will be parsed from SERVICE_LABELS env var
	ServiceLabels map[string]string `envconfig:"-"`
	RawLabels     string            `envconfig:"SERVICE_LABELS" default:"team:platform,domain:treasury"`

	// ImmuDB Configuration
	// Spec: docs/specs/001-immudb-connection.md
	ImmuDB *ImmuDBConfig `envconfig:"-"`
	
	// Migration Configuration
	// Spec: docs/specs/002-database-migrations.md
	Migration *migration.MigrationConfig `envconfig:"-"`
	
	// Tracing Configuration
	// Spec: docs/specs/004-opentelemetry-tracing.md
	Tracing *TracingConfig `envconfig:"-"`
	
	// Internal - not from env
	EnvFilePath string `envconfig:"-"`
}

// TracingConfig holds tracing configuration for the service
// Spec: docs/specs/004-opentelemetry-tracing.md
type TracingConfig struct {
	Enabled        bool    
	SentryDSN      string  
	SampleRate     float64 
	Environment    string  
	ServiceName    string  
	ServiceVersion string  
}

// ImmuDBConfig holds ImmuDB connection parameters
// Spec: docs/specs/001-immudb-connection.md
type ImmuDBConfig struct {
	Host                  string
	Port                  int
	Database              string
	Username              string
	Password              string
	MaxConnections        int
	MaxIdleConnections    int
	ConnectionMaxLifetime time.Duration
	ConnectionMaxIdleTime time.Duration
	VerifyTransactions    bool
	ServerSigningPubKey   string
	ClientKeyPath         string
	ClientCertPath        string
	HealthCheckInterval   time.Duration
	PingTimeout           time.Duration
	ChunkSize             int
	MaxRecvMsgSize        int
}

// LoadConfig loads configuration from environment variables and .env file
// Spec: docs/specs/002-configuration-management.md#configuration-loading-function
func LoadConfig() (*Config, error) {
	// Try to load .env file from multiple locations
	// 1. First try the service directory (when running from monorepo root)
	envPaths := []string{
		"services/treasury-services/ledger-service/.env",
		".env", // Fallback to current directory
	}
	
	var loaded bool
	var loadedPath string
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			loadedPath = path
			loaded = true
			break
		} else if !os.IsNotExist(err) {
			// Only log actual errors, not missing files
			log.Printf("Warning: Error loading %s: %v", path, err)
		}
	}
	
	if !loaded {
		// This is expected in production, so don't log unless debugging
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}

	// Parse labels from comma-separated key:value pairs
	cfg.ServiceLabels = parseLabels(cfg.RawLabels)

	// Load ImmuDB configuration
	// Spec: docs/specs/001-immudb-connection.md#story-1-immudb-configuration-management
	immuDBConfig, err := LoadImmuDBConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load ImmuDB config: %w", err)
	}
	cfg.ImmuDB = immuDBConfig
	
	// Load Migration configuration
	// Spec: docs/specs/002-database-migrations.md
	migrationConfig := LoadMigrationConfig()
	cfg.Migration = migrationConfig

	// Load Tracing configuration
	// Spec: docs/specs/004-opentelemetry-tracing.md
	tracingConfig := LoadTracingConfig(&cfg)
	cfg.Tracing = tracingConfig

	// Store the loaded path for later logging if needed
	if loadedPath != "" {
		cfg.EnvFilePath = loadedPath
	}

	return &cfg, nil
}

// parseLabels parses comma-separated key:value pairs into a map
// Spec: docs/specs/002-configuration-management.md
func parseLabels(rawLabels string) map[string]string {
	labels := make(map[string]string)
	if rawLabels == "" {
		return labels
	}

	pairs := strings.Split(rawLabels, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) == 2 {
			labels[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return labels
}

// GetPort returns the port as a string with colon prefix
// Spec: docs/specs/002-configuration-management.md#helper-functions
func (c *Config) GetPort() string {
	return fmt.Sprintf(":%d", c.Port)
}

// Validate checks if the configuration is valid
// Spec: docs/specs/002-configuration-management.md#configuration-validation
func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port number: %d", c.Port)
	}

	if c.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}

	if c.ServiceVersion == "" {
		return fmt.Errorf("service version is required")
	}

	validEnvironments := map[string]bool{
		"dev":     true,
		"staging": true,
		"prod":    true,
		"local":   true,
	}
	if !validEnvironments[c.Environment] {
		return fmt.Errorf("invalid environment: %s (must be dev, staging, prod, or local)", c.Environment)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	return nil
}

// String returns a string representation of the config (for debugging)
// Spec: docs/specs/002-configuration-management.md#helper-functions
func (c *Config) String() string {
	var sb strings.Builder
	sb.WriteString("Configuration:\n")
	sb.WriteString(fmt.Sprintf("  Service: %s v%s\n", c.ServiceName, c.ServiceVersion))
	sb.WriteString(fmt.Sprintf("  Environment: %s\n", c.Environment))
	sb.WriteString(fmt.Sprintf("  Region: %s\n", c.Region))
	sb.WriteString(fmt.Sprintf("  Port: %d\n", c.Port))
	sb.WriteString(fmt.Sprintf("  Log Level: %s\n", c.LogLevel))
	sb.WriteString(fmt.Sprintf("  Features: %v\n", c.EnabledFeatures))
	sb.WriteString(fmt.Sprintf("  Labels: %v\n", c.ServiceLabels))
	if c.ImmuDB != nil {
		sb.WriteString(fmt.Sprintf("  ImmuDB: %s:%d/%s\n", c.ImmuDB.Host, c.ImmuDB.Port, c.ImmuDB.Database))
	}
	return sb.String()
}

// LoadImmuDBConfig loads ImmuDB configuration from environment
// Spec: docs/specs/001-immudb-connection.md#story-1-immudb-configuration-management
func LoadImmuDBConfig() (*ImmuDBConfig, error) {
	cfg := &ImmuDBConfig{
		// Note: Default to 'immudb' which is the container service name
		// Override with IMMUDB_HOST env var if needed
		Host:               getEnvString("IMMUDB_HOST", "immudb"),
		Port:               getEnvInt("IMMUDB_PORT", 3322),
		Database:           getEnvString("IMMUDB_DATABASE", "ledgerdb"),
		Username:           getEnvString("IMMUDB_USERNAME", "ledger_user"),
		Password:           getEnvString("IMMUDB_PASSWORD", "ledger_pass"),
		MaxConnections:     getEnvInt("IMMUDB_MAX_CONNECTIONS", 25),
		MaxIdleConnections: getEnvInt("IMMUDB_MAX_IDLE_CONNECTIONS", 5),
		VerifyTransactions: getEnvBool("IMMUDB_VERIFY_TRANSACTIONS", true),
		ServerSigningPubKey: getEnvString("IMMUDB_SERVER_SIGNING_PUB_KEY", ""),
		ClientKeyPath:      getEnvString("IMMUDB_CLIENT_KEY_PATH", ""),
		ClientCertPath:     getEnvString("IMMUDB_CLIENT_CERT_PATH", ""),
		ChunkSize:          getEnvInt("IMMUDB_CHUNK_SIZE", 64),
		MaxRecvMsgSize:     getEnvInt("IMMUDB_MAX_RECV_MSG_SIZE", 4194304),
	}

	// Parse durations
	cfg.ConnectionMaxLifetime = time.Duration(getEnvInt("IMMUDB_CONNECTION_MAX_LIFETIME", 3600)) * time.Second
	cfg.ConnectionMaxIdleTime = time.Duration(getEnvInt("IMMUDB_CONNECTION_MAX_IDLE_TIME", 900)) * time.Second
	cfg.HealthCheckInterval = time.Duration(getEnvInt("IMMUDB_HEALTH_CHECK_INTERVAL", 30)) * time.Second
	cfg.PingTimeout = time.Duration(getEnvInt("IMMUDB_PING_TIMEOUT", 5)) * time.Second

	// Validate configuration
	if cfg.Host == "" {
		return nil, fmt.Errorf("IMMUDB_HOST is required")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid IMMUDB_PORT: %d", cfg.Port)
	}
	if cfg.Database == "" {
		return nil, fmt.Errorf("IMMUDB_DATABASE is required")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("IMMUDB_USERNAME is required")
	}
	// Don't validate password as it could be empty in some environments

	// Configuration loaded successfully - will be logged in startup banner

	return cfg, nil
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

// LoadMigrationConfig loads migration configuration from environment
// Spec: docs/specs/002-database-migrations.md
func LoadMigrationConfig() *migration.MigrationConfig {
	return &migration.MigrationConfig{
		MigrationsPath: getEnvString("LEDGER_MIGRATION_PATH", "./migrations"),
		RunOnBoot:      getEnvBool("LEDGER_MIGRATION_RUN_ON_BOOT", false),
		DryRun:         false, // Never dry run in production
		Timeout:        time.Duration(getEnvInt("LEDGER_MIGRATION_TIMEOUT", 30)) * time.Second,
		TableName:      getEnvString("LEDGER_MIGRATION_TABLE", "ledger_schema_migrations"),
		ServiceName:    "ledger",
	}
}

// LoadTracingConfig loads tracing configuration from environment
// Spec: docs/specs/004-opentelemetry-tracing.md
func LoadTracingConfig(cfg *Config) *TracingConfig {
	return &TracingConfig{
		Enabled:        getEnvBool("TRACING_ENABLED", true),
		SentryDSN:      getEnvString("SENTRY_DSN", ""),
		SampleRate:     getEnvFloat("TRACE_SAMPLE_RATE", 0.01),
		Environment:    getEnvString("TRACE_ENVIRONMENT", cfg.Environment),
		ServiceName:    getEnvString("TRACE_SERVICE_NAME", cfg.ServiceName),
		ServiceVersion: getEnvString("TRACE_SERVICE_VERSION", cfg.ServiceVersion),
	}
}

// getEnvFloat gets a float64 value from environment or returns default
func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}
