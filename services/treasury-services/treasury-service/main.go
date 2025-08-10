package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	pb "example.com/go-mono-repo/proto/treasury"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedManifestServer
	manifestCache *pb.ManifestResponse
	startTime     time.Time
}

// GetManifest returns service metadata
// Spec: docs/specs/001-manifest.md
func (s *server) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
	// Update dynamic fields
	response := *s.manifestCache
	response.RuntimeInfo.UptimeSeconds = int64(time.Since(s.startTime).Seconds())
	
	return &response, nil
}

// computeManifest builds the manifest at startup
// Spec: docs/specs/001-manifest.md#runtime-info
func computeManifest(config *ManifestConfig, buildConfig *BuildConfig, startTime time.Time) *pb.ManifestResponse {
	hostname, _ := os.Hostname()
	instanceID := fmt.Sprintf("%s-%d", hostname, os.Getpid())
	
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
	
	return &pb.ManifestResponse{
		Identity: &pb.ServiceIdentity{
			Name:        config.ServiceName,
			Version:     config.ServiceVersion,
			ApiVersion:  config.APIVersion,
			Description: config.Description,
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
			Owner:            config.Owner,
			RepositoryUrl:    config.RepoURL,
			DocumentationUrl: config.DocsURL,
			SupportContact:   config.SupportContact,
			Labels: map[string]string{
				"service": config.ServiceName,
				"version": config.ServiceVersion,
				"env":     config.Environment,
			},
		},
		Capabilities: &pb.ServiceCapabilities{
			ApiVersions: []string{config.APIVersion},
			Protocols:   []string{"grpc"},
			Features:    []string{}, // Add feature flags as needed
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

func main() {
	port := ":50052"
	
	// Load configuration
	config := LoadManifestConfig()
	buildConfig := GetBuildConfig()
	startTime := time.Now()
	
	// Log startup information
	log.Printf("Starting %s version %s", config.ServiceName, config.ServiceVersion)
	log.Printf("Environment: %s, Region: %s", config.Environment, config.Region)
	
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Compute manifest once at startup
	manifestCache := computeManifest(config, buildConfig, startTime)
	
	s := grpc.NewServer()
	pb.RegisterManifestServer(s, &server{
		manifestCache: manifestCache,
		startTime:     startTime,
	})
	
	reflection.Register(s)
	
	fmt.Printf("%s starting on port %s\n", config.ServiceName, port)
	fmt.Printf("Build: %s@%s (dirty: %v)\n", 
		manifestCache.BuildInfo.CommitHash[:7], 
		manifestCache.BuildInfo.Branch,
		manifestCache.BuildInfo.IsDirty)
	
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}