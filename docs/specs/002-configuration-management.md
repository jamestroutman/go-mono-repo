# Configuration Management Specification

> **Status**: Approved  
> **Version**: 1.0.0  
> **Last Updated**: 2025-01-10  
> **Author(s)**: Platform Team  
> **Reviewer(s)**: Engineering Team  

## Executive Summary

This specification defines the standard configuration management pattern for all services in the monorepo. It ensures consistent configuration loading from environment variables and `.env` files, with proper fallback mechanisms and debugging capabilities.

## Problem Statement

### Current State
Services need consistent configuration management that works both in development (with `.env` files) and production (with environment variables), while handling the monorepo structure where services can be run from different working directories.

### Desired State
A standardized configuration pattern that:
- Loads from `.env` files in development
- Uses environment variables in production
- Works regardless of the working directory
- Provides clear debugging information
- Validates configuration at startup

## Scope

### In Scope
- Environment variable loading with defaults
- `.env` file support for local development
- Configuration validation
- Multi-path `.env` file discovery
- Configuration debugging and logging
- Type-safe configuration structs

### Out of Scope
- Dynamic configuration reloading
- Configuration from external services (Consul, etcd)
- Encrypted configuration values
- Configuration versioning

## Technical Design

### Architecture Overview
Each service implements a `Config` struct with environment variable tags and a `LoadConfig()` function that handles loading from multiple sources with proper precedence.

### Configuration Loading Order
1. `.env` file (if exists)
2. Environment variables
3. Default values in struct tags

### Implementation Pattern

#### 1. Config Struct Definition

```go
// Config holds all configuration for the service
type Config struct {
    // Service Identity
    ServiceName        string `envconfig:"SERVICE_NAME" default:"service-name"`
    ServiceVersion     string `envconfig:"SERVICE_VERSION" default:"1.0.0"`
    ServiceDescription string `envconfig:"SERVICE_DESCRIPTION" default:"Service description"`
    APIVersion         string `envconfig:"API_VERSION" default:"v1"`

    // Runtime Configuration
    Port        int    `envconfig:"PORT" default:"50051"`
    Environment string `envconfig:"ENVIRONMENT" default:"dev"`
    Region      string `envconfig:"REGION" default:"local"`

    // Service Metadata
    ServiceOwner   string `envconfig:"SERVICE_OWNER" default:"team@example.com"`
    RepoURL        string `envconfig:"REPO_URL" default:"https://github.com/example/repo"`
    DocsURL        string `envconfig:"DOCS_URL" default:"https://docs.example.com"`
    SupportContact string `envconfig:"SUPPORT_CONTACT" default:"support@example.com"`
    ServiceTier    string `envconfig:"SERVICE_TIER" default:"1"`

    // Features
    EnabledFeatures []string `envconfig:"ENABLED_FEATURES" default:"base,manifest"`

    // Logging
    LogLevel  string `envconfig:"LOG_LEVEL" default:"info"`
    LogFormat string `envconfig:"LOG_FORMAT" default:"json"`

    // Custom Labels
    ServiceLabels map[string]string `envconfig:"-"`
    RawLabels     string            `envconfig:"SERVICE_LABELS" default:""`
}
```

#### 2. Configuration Loading Function

```go
// LoadConfig loads configuration from environment variables and .env file
func LoadConfig() (*Config, error) {
    // Try to load .env file from multiple locations
    // This handles running from different working directories in monorepo
    envPaths := []string{
        "services/{domain}/{service-name}/.env",  // From monorepo root
        ".env",                                    // From service directory
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

    // Parse any complex fields (e.g., labels from comma-separated values)
    cfg.ServiceLabels = parseLabels(cfg.RawLabels)

    // Log loaded configuration for debugging
    log.Printf("Loaded configuration: %s", cfg.String())

    return &cfg, nil
}
```

#### 3. Configuration Validation

```go
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
        return fmt.Errorf("invalid environment: %s", c.Environment)
    }

    return nil
}
```

#### 4. Helper Functions

```go
// GetPort returns the port as a string with colon prefix
func (c *Config) GetPort() string {
    return fmt.Sprintf(":%d", c.Port)
}

// String returns a string representation for debugging
func (c *Config) String() string {
    var sb strings.Builder
    sb.WriteString("Configuration:\n")
    sb.WriteString(fmt.Sprintf("  Service: %s v%s\n", c.ServiceName, c.ServiceVersion))
    sb.WriteString(fmt.Sprintf("  Environment: %s\n", c.Environment))
    sb.WriteString(fmt.Sprintf("  Port: %d\n", c.Port))
    // Add other fields as needed
    return sb.String()
}
```

### Environment File (.env) Structure

```bash
# Service Identity
SERVICE_NAME=ledger-service
SERVICE_VERSION=1.0.0
SERVICE_DESCRIPTION=Service description
API_VERSION=v1

# Runtime Configuration
PORT=50051
ENVIRONMENT=dev
REGION=local

# Service Metadata
SERVICE_OWNER=platform-team@example.com
REPO_URL=https://github.com/example/repo
DOCS_URL=https://docs.example.com/service
SUPPORT_CONTACT=support@example.com
SERVICE_TIER=1

# Features
ENABLED_FEATURES=base,manifest,feature-x

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Labels (comma-separated key:value pairs)
SERVICE_LABELS=team:platform,domain:treasury
```

### Usage in main.go

```go
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
    
    // Use configuration
    port := cfg.GetPort()
    log.Printf("Starting %s on port %s", cfg.ServiceName, port)
    
    // ... rest of service initialization
}
```

## Implementation Guidelines

### 1. Required Dependencies

Add to your service's `go.mod`:
```go
require (
    github.com/joho/godotenv v1.5.1
    github.com/kelseyhightower/envconfig v1.4.0
)
```

### 2. File Structure

```
services/
  {domain}/
    {service-name}/
      .env           # Development configuration (git-ignored)
      .env.example   # Example configuration (committed)
      config.go      # Configuration implementation
      main.go        # Uses LoadConfig()
```

### 3. .gitignore Entry

Ensure `.env` files are git-ignored:
```gitignore
# Environment files
.env
.env.local
```

### 4. Example Configuration File

Always provide `.env.example` with all available configuration options:
```bash
cp .env.example .env
# Edit .env with your local values
```

## Troubleshooting

### Common Issues

#### 1. .env File Not Loading

**Problem**: Service uses default values despite .env file existing.

**Solution**: Check the `envPaths` array in `LoadConfig()` includes the correct path for your service when running from the monorepo root.

**Debug**: Look for log output:
```
Loaded environment from: services/domain/service/.env
```

#### 2. Environment Variable Not Being Read

**Problem**: Environment variable set but not reflected in config.

**Solution**: Ensure the struct field has the correct `envconfig` tag and the environment variable name matches.

#### 3. Invalid Configuration Not Caught

**Problem**: Service starts with invalid configuration.

**Solution**: Add validation rules to the `Validate()` method for all critical fields.

## Best Practices

1. **Always provide defaults**: Every field should have a sensible default value
2. **Validate early**: Call `Validate()` immediately after loading
3. **Log configuration**: Always log the loaded configuration (excluding secrets)
4. **Use .env.example**: Commit an example file showing all available options
5. **Handle multiple paths**: Support running from both monorepo root and service directory
6. **Clear error messages**: Provide specific error messages for validation failures
7. **Type safety**: Use appropriate types (int for ports, bool for flags, etc.)

## Security Considerations

1. **Never commit .env files**: Always git-ignore actual .env files
2. **No secrets in defaults**: Default values should never contain real credentials
3. **Mask sensitive values**: When logging configuration, mask sensitive fields
4. **Validate inputs**: Always validate configuration to prevent injection attacks

## Testing Strategy

### Unit Tests

```go
func TestLoadConfig(t *testing.T) {
    // Set test environment variables
    os.Setenv("SERVICE_NAME", "test-service")
    os.Setenv("PORT", "8080")
    
    cfg, err := LoadConfig()
    assert.NoError(t, err)
    assert.Equal(t, "test-service", cfg.ServiceName)
    assert.Equal(t, 8080, cfg.Port)
}

func TestConfigValidation(t *testing.T) {
    cfg := &Config{
        Port: 70000, // Invalid port
    }
    
    err := cfg.Validate()
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "invalid port")
}
```

## Migration Guide

To migrate an existing service to this configuration pattern:

1. Create `config.go` with the Config struct and LoadConfig function
2. Update struct fields to match your service's needs
3. Create `.env.example` with all configuration options
4. Update `main.go` to use `LoadConfig()` and `Validate()`
5. Update the service's path in `envPaths` array
6. Test both from monorepo root and service directory

## References

- [godotenv documentation](https://github.com/joho/godotenv)
- [envconfig documentation](https://github.com/kelseyhightower/envconfig)
- [Twelve-Factor App Configuration](https://12factor.net/config)
- [Manifest Service Specification](./001-manifest.md) - Uses this configuration pattern

## Decision Log

| Date | Decision | Rationale | Made By |
|------|----------|-----------|---------|
| 2025-01-10 | Use envconfig + godotenv | Industry standard, simple, type-safe | Platform Team |
| 2025-01-10 | Support multiple .env paths | Monorepo requires flexible path resolution | Platform Team |
| 2025-01-10 | Log configuration at startup | Aids debugging in development and production | Platform Team |
| 2025-01-10 | Validate configuration | Fail fast on invalid configuration | Platform Team |