package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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
	return sb.String()
}
