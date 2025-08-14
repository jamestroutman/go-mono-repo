package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the payroll service
// Spec: docs/specs/001-manifest.md
type Config struct {
	// Service Identity
	ServiceName        string `envconfig:"SERVICE_NAME" default:"payroll-service"`
	ServiceVersion     string `envconfig:"SERVICE_VERSION" default:"1.0.0"`
	ServiceDescription string `envconfig:"SERVICE_DESCRIPTION" default:"Payroll processing and management service"`
	APIVersion         string `envconfig:"API_VERSION" default:"v1"`

	// Runtime Configuration
	Port        int    `envconfig:"PORT" default:"50053"`
	Environment string `envconfig:"ENVIRONMENT" default:"dev"`
	Region      string `envconfig:"REGION" default:"local"`

	// Service Metadata
	ServiceOwner   string `envconfig:"SERVICE_OWNER" default:"payroll-team@example.com"`
	RepoURL        string `envconfig:"REPO_URL" default:"https://github.com/example/go-mono-repo"`
	DocsURL        string `envconfig:"DOCS_URL" default:"/services/payroll-services/payroll-service/docs"`
	SupportContact string `envconfig:"SUPPORT_CONTACT" default:"payroll-team@example.com"`
	ServiceTier    string `envconfig:"SERVICE_TIER" default:"1"`

	// Features
	EnabledFeatures []string `envconfig:"ENABLED_FEATURES" default:"hello-world,health-check,liveness-check,manifest"`

	// Logging
	LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
	LogFormat string `envconfig:"LOG_FORMAT" default:"json"`

	// Labels - will be parsed from SERVICE_LABELS env var
	ServiceLabels map[string]string `envconfig:"-"`
	RawLabels     string            `envconfig:"SERVICE_LABELS" default:"team:payroll,domain:payroll"`
	
	// Tracing Configuration
	// Spec: docs/specs/004-opentelemetry-tracing.md#configuration-integration
	Tracing TracingConfig `envconfig:""`
	
	// Internal - not from env
	EnvFilePath string `envconfig:"-"`
}

// TracingConfig holds tracing configuration for the service
// Spec: docs/specs/004-opentelemetry-tracing.md#configuration-integration
type TracingConfig struct {
	Enabled        bool    `envconfig:"TRACING_ENABLED" default:"true"`
	SentryDSN      string  `envconfig:"SENTRY_DSN" default:""`
	SampleRate     float64 `envconfig:"TRACE_SAMPLE_RATE" default:"0.01"`  // 1% default for production safety
	Environment    string  `envconfig:"TRACE_ENVIRONMENT" default:""`       // Defaults to main Environment field
	ServiceName    string  `envconfig:"TRACE_SERVICE_NAME" default:""`      // Defaults to main ServiceName field
	ServiceVersion string  `envconfig:"TRACE_SERVICE_VERSION" default:""`   // Defaults to main ServiceVersion field
}

// GetEnvironment returns the tracing environment or falls back to provided default
func (c *TracingConfig) GetEnvironment(fallback string) string {
	if c.Environment != "" {
		return c.Environment
	}
	return fallback
}

// GetServiceName returns the tracing service name or falls back to provided default
func (c *TracingConfig) GetServiceName(fallback string) string {
	if c.ServiceName != "" {
		return c.ServiceName
	}
	return fallback
}

// GetServiceVersion returns the tracing service version or falls back to provided default
func (c *TracingConfig) GetServiceVersion(fallback string) string {
	if c.ServiceVersion != "" {
		return c.ServiceVersion
	}
	return fallback
}

// LoadConfig loads configuration from environment variables and .env file
// Spec: docs/specs/001-manifest.md
func LoadConfig() (*Config, error) {
	// Try to load .env file from multiple locations
	// 1. First try the service directory (when running from monorepo root)
	envPaths := []string{
		"services/payroll-services/payroll-service/.env",
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

	// Store the loaded path for later logging if needed
	if loadedPath != "" {
		cfg.EnvFilePath = loadedPath
	}

	// Set default values for tracing configuration from main config if not explicitly set
	// Spec: docs/specs/004-opentelemetry-tracing.md#configuration-integration
	if cfg.Tracing.Environment == "" {
		cfg.Tracing.Environment = cfg.Environment
	}
	if cfg.Tracing.ServiceName == "" {
		cfg.Tracing.ServiceName = cfg.ServiceName
	}
	if cfg.Tracing.ServiceVersion == "" {
		cfg.Tracing.ServiceVersion = cfg.ServiceVersion
	}

	return &cfg, nil
}

// parseLabels parses comma-separated key:value pairs into a map
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
func (c *Config) GetPort() string {
	return fmt.Sprintf(":%d", c.Port)
}

// Validate checks if the configuration is valid
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
// These values would typically be injected at build time via environment variables
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