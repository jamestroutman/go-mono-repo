package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	pb "example.com/go-mono-repo/proto/treasury"
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
	// Create a new response with updated uptime
	response := &pb.ManifestResponse{
		Identity: s.manifestCache.Identity,
		BuildInfo: s.manifestCache.BuildInfo,
		RuntimeInfo: &pb.RuntimeInfo{
			InstanceId:    s.manifestCache.RuntimeInfo.InstanceId,
			Hostname:      s.manifestCache.RuntimeInfo.Hostname,
			StartedAt:     s.manifestCache.RuntimeInfo.StartedAt,
			UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
			Region:        s.manifestCache.RuntimeInfo.Region,
			Environment:   s.manifestCache.RuntimeInfo.Environment,
		},
		Metadata: s.manifestCache.Metadata,
		Capabilities: s.manifestCache.Capabilities,
	}
	
	return response, nil
}

// GetManifestCache returns the cached manifest for use in startup logs
func (s *ManifestServer) GetManifestCache() *pb.ManifestResponse {
	return s.manifestCache
}

// computeManifest builds the manifest at startup
// Spec: docs/specs/001-manifest.md#runtime-info
func computeManifest(config *Config, buildConfig *BuildConfig, startTime time.Time) *pb.ManifestResponse {
	hostname := getHostname()
	instanceID := getInstanceID()
	
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
				{
					Name:       "ledger-service",
					Version:    ">=1.0.0",
					IsOptional: false,
				},
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

// getInstanceID generates a unique instance identifier
func getInstanceID() string {
	hostname := getHostname()
	pid := os.Getpid()
	return fmt.Sprintf("%s-%d-%d", hostname, pid, time.Now().Unix())
}