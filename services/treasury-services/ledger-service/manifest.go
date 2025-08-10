package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	pb "example.com/go-mono-repo/proto/ledger"
)

// Build-time variables (injected via -ldflags)
var (
	// These will be set at build time
	GitCommit   = "unknown"
	GitBranch   = "unknown"
	BuildTime   = "unknown"
	BuildUser   = "unknown"
	GitDirty    = "false"
)

// ManifestServer handles the Manifest service endpoint
// Spec: docs/specs/001-manifest.md
type ManifestServer struct {
	pb.UnimplementedManifestServer
	startTime      time.Time
	manifestCache  *pb.ManifestResponse
	config         *Config
}

// NewManifestServer creates a new manifest server with cached data
// Spec: docs/specs/001-manifest.md
func NewManifestServer(cfg *Config, startTime time.Time) *ManifestServer {
	return &ManifestServer{
		startTime:     startTime,
		config:        cfg,
		manifestCache: computeManifest(cfg, startTime),
	}
}

// GetManifest returns service metadata
// Spec: docs/specs/001-manifest.md
func (s *ManifestServer) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
	// Clone the cached response to avoid mutations
	response := &pb.ManifestResponse{
		Identity:     s.manifestCache.Identity,
		BuildInfo:    s.manifestCache.BuildInfo,
		RuntimeInfo:  s.manifestCache.RuntimeInfo,
		Metadata:     s.manifestCache.Metadata,
		Capabilities: s.manifestCache.Capabilities,
	}
	
	// Update dynamic fields
	if response.RuntimeInfo != nil {
		response.RuntimeInfo.UptimeSeconds = int64(time.Since(s.startTime).Seconds())
	}
	
	return response, nil
}

// computeManifest builds the manifest at startup
// Spec: docs/specs/001-manifest.md#runtime-info
func computeManifest(cfg *Config, startTime time.Time) *pb.ManifestResponse {
	commit, branch, isDirty := getGitInfo()
	
	buildTime := BuildTime
	if buildTime == "unknown" {
		buildTime = time.Now().Format(time.RFC3339)
	}
	
	builder := BuildUser
	if builder == "unknown" {
		if user := os.Getenv("USER"); user != "" {
			builder = user
		} else {
			builder = "local-dev"
		}
	}
	
	// Add service tier to labels if configured
	labels := make(map[string]string)
	for k, v := range cfg.ServiceLabels {
		labels[k] = v
	}
	if cfg.ServiceTier != "" {
		labels["tier"] = cfg.ServiceTier
	}
	
	return &pb.ManifestResponse{
		Identity: &pb.ServiceIdentity{
			Name:        cfg.ServiceName,
			Version:     cfg.ServiceVersion,
			ApiVersion:  cfg.APIVersion,
			Description: cfg.ServiceDescription,
		},
		BuildInfo: &pb.BuildInfo{
			CommitHash: commit,
			Branch:     branch,
			BuildTime:  buildTime,
			Builder:    builder,
			IsDirty:    isDirty,
		},
		RuntimeInfo: &pb.RuntimeInfo{
			InstanceId:  getInstanceID(),
			Hostname:    getHostname(),
			StartedAt:   startTime.Format(time.RFC3339),
			Environment: cfg.Environment,
			Region:      cfg.Region,
			// Uptime will be calculated dynamically
		},
		Metadata: &pb.ServiceMetadata{
			Owner:            cfg.ServiceOwner,
			RepositoryUrl:    cfg.RepoURL,
			DocumentationUrl: cfg.DocsURL,
			SupportContact:   cfg.SupportContact,
			Labels:           labels,
		},
		Capabilities: &pb.ServiceCapabilities{
			ApiVersions: []string{cfg.APIVersion},
			Protocols:   []string{"grpc", "grpc-web"},
			Features:    cfg.EnabledFeatures,
			Dependencies: []*pb.ServiceDependency{
				// Add dependencies as needed
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

// getGitInfo retrieves git information at runtime (fallback for dev)
func getGitInfo() (commit, branch string, isDirty bool) {
	// If build-time values are set, use them
	if GitCommit != "unknown" {
		return GitCommit, GitBranch, GitDirty == "true"
	}
	
	// Try to get git info at runtime (for local development)
	if commitBytes, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		commit = strings.TrimSpace(string(commitBytes))
	} else {
		commit = "development"
	}
	
	if branchBytes, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		branch = strings.TrimSpace(string(branchBytes))
	} else {
		branch = "local"
	}
	
	if statusBytes, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		isDirty = len(statusBytes) > 0
	}
	
	return commit, branch, isDirty
}

// GetManifestCache returns the cached manifest data (useful for startup logging)
func (s *ManifestServer) GetManifestCache() *pb.ManifestResponse {
	return s.manifestCache
}