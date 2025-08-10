package main

import (
	"os"
	"strings"
)

// ManifestConfig configuration from environment
// Spec: docs/specs/001-manifest.md
type ManifestConfig struct {
	ServiceName    string
	ServiceVersion string
	APIVersion     string
	Description    string
	Environment    string
	Region         string
	Owner          string
	RepoURL        string
	DocsURL        string
	SupportContact string
}

// LoadManifestConfig loads configuration from environment variables
// Spec: docs/specs/001-manifest.md#configuration
func LoadManifestConfig() *ManifestConfig {
	return &ManifestConfig{
		ServiceName:    getEnvOrDefault("SERVICE_NAME", "treasury-service"),
		ServiceVersion: getEnvOrDefault("SERVICE_VERSION", "1.0.0"),
		APIVersion:     getEnvOrDefault("API_VERSION", "v1"),
		Description:    getEnvOrDefault("SERVICE_DESCRIPTION", "Treasury service for financial operations"),
		Environment:    getEnvOrDefault("ENVIRONMENT", "dev"),
		Region:         getEnvOrDefault("REGION", "local"),
		Owner:          getEnvOrDefault("SERVICE_OWNER", "treasury-team@example.com"),
		RepoURL:        getEnvOrDefault("REPO_URL", "https://github.com/example/go-mono-repo"),
		DocsURL:        getEnvOrDefault("DOCS_URL", "https://example.atlassian.net/wiki/spaces/PLATFORM/pages/treasury"),
		SupportContact: getEnvOrDefault("SUPPORT_CONTACT", "treasury-support@example.com"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
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