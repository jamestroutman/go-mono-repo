package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the treasury service
// Spec: docs/specs/002-configuration-management.md
type Config struct {
	// Service Identity
	ServiceName        string `envconfig:"SERVICE_NAME" default:"treasury-service"`
	ServiceVersion     string `envconfig:"SERVICE_VERSION" default:"1.0.0"`
	ServiceDescription string `envconfig:"SERVICE_DESCRIPTION" default:"Treasury service for financial operations"`
	APIVersion         string `envconfig:"API_VERSION" default:"v1"`

	// Runtime Configuration
	Port        int    `envconfig:"PORT" default:"50052"`
	Environment string `envconfig:"ENVIRONMENT" default:"dev"`
	Region      string `envconfig:"REGION" default:"local"`

	// Service Metadata
	ServiceOwner   string `envconfig:"SERVICE_OWNER" default:"treasury-team@example.com"`
	RepoURL        string `envconfig:"REPO_URL" default:"https://github.com/example/go-mono-repo"`
	DocsURL        string `envconfig:"DOCS_URL" default:"https://docs.example.com/treasury-service"`
	SupportContact string `envconfig:"SUPPORT_CONTACT" default:"treasury-support@example.com"`
	ServiceTier    string `envconfig:"SERVICE_TIER" default:"1"`

	// Features
	EnabledFeatures []string `envconfig:"ENABLED_FEATURES" default:"base,manifest"`

	// Logging
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat string `envconfig:"LOG_FORMAT" default:"json"`

	// Labels - will be parsed from SERVICE_LABELS env var
	ServiceLabels map[string]string `envconfig:"-"`
	RawLabels     string            `envconfig:"SERVICE_LABELS" default:"team:treasury,domain:treasury"`

	// Database Configuration
	// Spec: docs/specs/001-database-connection.md
	Database DatabaseConfig `envconfig:"-"`

	// Dependency Services
	LedgerServiceHost string `envconfig:"LEDGER_SERVICE_HOST" default:"localhost"`
	LedgerServicePort int    `envconfig:"LEDGER_SERVICE_PORT" default:"50051"`
}

// DatabaseConfig holds database connection parameters
// Spec: docs/specs/001-database-connection.md
type DatabaseConfig struct {
	Host                  string        `envconfig:"DB_HOST" default:"localhost"`
	Port                  int           `envconfig:"DB_PORT" default:"5432"`
	Database              string        `envconfig:"DB_NAME" default:"treasury_db"`
	User                  string        `envconfig:"DB_USER" default:"treasury_user"`
	Password              string        `envconfig:"DB_PASSWORD" default:"treasury_pass"`
	Schema                string        `envconfig:"DB_SCHEMA" default:"public"`
	SSLMode               string        `envconfig:"DB_SSL_MODE" default:"disable"`
	MaxConnections        int           `envconfig:"DB_MAX_CONNECTIONS" default:"25"`
	MaxIdleConnections    int           `envconfig:"DB_MAX_IDLE_CONNECTIONS" default:"5"`
	ConnectionMaxLifetime time.Duration `envconfig:"-"`
	ConnectionMaxIdleTime time.Duration `envconfig:"-"`
	HealthCheckInterval   time.Duration `envconfig:"-"`
	PingTimeout           time.Duration `envconfig:"-"`
}

// LoadConfig loads configuration from environment variables and .env file
// Spec: docs/specs/002-configuration-management.md#configuration-loading-function
func LoadConfig() (*Config, error) {
	// Try to load .env file from multiple locations
	// 1. First try the service directory (when running from monorepo root)
	envPaths := []string{
		"services/treasury-services/treasury-service/.env",
		".env", // Fallback to current directory
	}
	
	var loaded bool
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded environment from: %s", path)
			loaded = true
			break
		} else if !os.IsNotExist(err) {
			log.Printf("Warning: Error loading %s: %v", path, err)
		}
	}
	
	if !loaded {
		log.Printf("No .env file found, using environment variables and defaults")
	}

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}

	// Parse labels from comma-separated key:value pairs
	cfg.ServiceLabels = parseLabels(cfg.RawLabels)

	// Load database configuration
	// Spec: docs/specs/001-database-connection.md#story-1-database-configuration-management
	if err := envconfig.Process("", &cfg.Database); err != nil {
		return nil, fmt.Errorf("failed to process database config: %w", err)
	}

	// Parse database durations from environment
	cfg.Database.ConnectionMaxLifetime = parseDurationFromEnv("DB_CONNECTION_MAX_LIFETIME", 3600*time.Second)
	cfg.Database.ConnectionMaxIdleTime = parseDurationFromEnv("DB_CONNECTION_MAX_IDLE_TIME", 900*time.Second)
	cfg.Database.HealthCheckInterval = parseDurationFromEnv("DB_HEALTH_CHECK_INTERVAL", 30*time.Second)
	cfg.Database.PingTimeout = parseDurationFromEnv("DB_PING_TIMEOUT", 5*time.Second)

	// Log loaded configuration for debugging
	log.Printf("Loaded configuration: %s", cfg.String())

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

// parseDurationFromEnv parses a duration from an environment variable
// Spec: docs/specs/001-database-connection.md
func parseDurationFromEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	// If the value is just a number, treat it as seconds
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try to parse as a duration string
	if duration, err := time.ParseDuration(value); err == nil {
		return duration
	}

	log.Printf("Warning: Invalid duration value for %s: %s, using default: %v", key, value, defaultValue)
	return defaultValue
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
	sb.WriteString(fmt.Sprintf("  Database: %s:%d/%s (user: %s, pool: %d/%d)\n", 
		c.Database.Host, c.Database.Port, c.Database.Database, 
		c.Database.User, c.Database.MaxIdleConnections, c.Database.MaxConnections))
	sb.WriteString(fmt.Sprintf("  Ledger Service: %s:%d\n", c.LedgerServiceHost, c.LedgerServicePort))
	return sb.String()
}

// GetConnectionString returns PostgreSQL connection string
// Spec: docs/specs/001-database-connection.md
func (dc *DatabaseConfig) GetConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=%s",
		dc.Host, dc.Port, dc.User, dc.Password, dc.Database, dc.SSLMode, dc.Schema)
}

// BuildConfig holds build-time information
// Spec: docs/specs/001-manifest.md#build-info
type BuildConfig struct {
	CommitHash string
	Branch     string
	BuildTime  string
	Builder    string
	IsDirty    bool
}

// GetBuildConfig returns build configuration
// These values would typically be injected at build time via ldflags
func GetBuildConfig() *BuildConfig {
	return &BuildConfig{
		CommitHash: getEnvOrDefault("BUILD_COMMIT", "unknown"),
		Branch:     getEnvOrDefault("BUILD_BRANCH", "unknown"),
		BuildTime:  getEnvOrDefault("BUILD_TIME", "unknown"),
		Builder:    getEnvOrDefault("BUILD_USER", os.Getenv("USER")),
		IsDirty:    strings.ToLower(getEnvOrDefault("BUILD_DIRTY", "false")) == "true",
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}