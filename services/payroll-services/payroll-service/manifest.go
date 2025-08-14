package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	pb "example.com/go-mono-repo/proto/payroll"
)

// ManifestServer implements the Manifest service
type ManifestServer struct {
	pb.UnimplementedManifestServer
	manifestCache *pb.ManifestResponse
	startTime     time.Time
}

// NewManifestServer creates a new manifest server with cached data
// Spec: docs/specs/001-manifest.md
func NewManifestServer(cfg *Config, startTime time.Time) *ManifestServer {
	return &ManifestServer{
		manifestCache: computeManifest(cfg, GetBuildConfig(), startTime),
		startTime:     startTime,
	}
}

// GetManifest returns service metadata
// Spec: docs/specs/001-manifest.md
func (s *ManifestServer) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
	// Create a copy of the cached manifest and update dynamic fields
	response := *s.manifestCache
	response.RuntimeInfo.UptimeSeconds = int64(time.Since(s.startTime).Seconds())
	
	return &response, nil
}

// GetManifestCache returns the cached manifest for use in startup logs
func (s *ManifestServer) GetManifestCache() *pb.ManifestResponse {
	return s.manifestCache
}

// computeManifest builds the manifest at startup
// Spec: docs/specs/001-manifest.md#runtime-info
func computeManifest(config *Config, buildConfig *BuildConfig, startTime time.Time) *pb.ManifestResponse {
	hostname := getHostname()
	instanceID := generateInstanceID()
	
	// Get git information if available
	commitHash := buildConfig.CommitHash
	branch := buildConfig.Branch
	isDirty := buildConfig.IsDirty
	
	// Try to get git info dynamically if not set
	if commitHash == "unknown" {
		if hash, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
			commitHash = strings.TrimSpace(string(hash))
		}
	}
	
	if branch == "unknown" {
		if br, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
			branch = strings.TrimSpace(string(br))
		}
	}
	
	// Check for uncommitted changes
	if !isDirty {
		if status, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
			isDirty = len(strings.TrimSpace(string(status))) > 0
		}
	}
	
	// Build labels from config
	labels := make(map[string]string)
	for k, v := range config.ServiceLabels {
		labels[k] = v
	}
	labels["service"] = config.ServiceName
	labels["version"] = config.ServiceVersion
	labels["env"] = config.Environment
	
	return &pb.ManifestResponse{
		Identity: &pb.ServiceIdentity{
			Name:        config.ServiceName,
			Version:     config.ServiceVersion,
			ApiVersion:  config.APIVersion,
			Description: config.ServiceDescription,
		},
		BuildInfo: &pb.BuildInfo{
			CommitHash: commitHash,
			Branch:     branch,
			BuildTime:  buildConfig.BuildTime,
			Builder:    buildConfig.Builder,
			IsDirty:    isDirty,
		},
		RuntimeInfo: &pb.RuntimeInfo{
			InstanceId:    instanceID,
			Hostname:      hostname,
			StartedAt:     startTime.Format(time.RFC3339),
			Environment:   config.Environment,
			Region:        config.Region,
			UptimeSeconds: 0, // Will be calculated dynamically
		},
		Metadata: &pb.ServiceMetadata{
			Owner:            config.ServiceOwner,
			RepositoryUrl:    config.RepoURL,
			DocumentationUrl: config.DocsURL,
			SupportContact:   config.SupportContact,
			Labels:           labels,
		},
		Capabilities: &pb.ServiceCapabilities{
			ApiVersions: []string{config.APIVersion},
			Protocols:   []string{"grpc"},
			Features:    config.EnabledFeatures,
			Dependencies: []*pb.ServiceDependency{
				// No dependencies for now - will add as service evolves
			},
		},
	}
}

// getHostname returns the hostname or a default value
func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// generateInstanceID creates a unique identifier for this service instance
func generateInstanceID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("payroll-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("payroll-%s", hex.EncodeToString(bytes))
}