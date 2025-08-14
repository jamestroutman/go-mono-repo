package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "example.com/go-mono-repo/proto/payroll"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedPayrollServiceServer
	pb.UnimplementedManifestServer
	pb.UnimplementedHealthServer
	startTime time.Time
}

// GetManifest implements the Manifest service
// Spec: docs/specs/001-manifest.md
func (s *server) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
	return &pb.ManifestResponse{
		Message: "payroll-service",
		Version: "1.0.0",
	}, nil
}

// GetHealth implements health check
// Spec: docs/specs/003-health-check-liveness.md
func (s *server) GetHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Healthy: true,
		Status:  "serving",
		Details: map[string]string{
			"service": "payroll-service",
			"version": "1.0.0",
		},
	}, nil
}

// GetLiveness implements liveness check
// Spec: docs/specs/003-health-check-liveness.md
func (s *server) GetLiveness(ctx context.Context, req *pb.LivenessRequest) (*pb.LivenessResponse, error) {
	return &pb.LivenessResponse{
		Alive:         true,
		UptimeSeconds: int64(time.Since(s.startTime).Seconds()),
	}, nil
}

// HelloWorld implements the hello world endpoint
// Spec: services/payroll-services/payroll-service/docs/specs/001-service-initialization.md
func (s *server) HelloWorld(ctx context.Context, req *pb.HelloWorldRequest) (*pb.HelloWorldResponse, error) {
	name := req.Name
	if name == "" {
		name = "World"
	}
	return &pb.HelloWorldResponse{
		Message: fmt.Sprintf("Hello, %s! From Payroll Service", name),
	}, nil
}

func main() {
	port := ":50053"

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	srv := &server{
		startTime: time.Now(),
	}

	s := grpc.NewServer()

	// Register services
	pb.RegisterManifestServer(s, srv)
	pb.RegisterHealthServer(s, srv)
	pb.RegisterPayrollServiceServer(s, srv)

	// Enable reflection for debugging
	reflection.Register(s)

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		fmt.Println("\nShutting down gracefully...")
		s.GracefulStop()
	}()

	fmt.Printf("Payroll Service starting on port %s\n", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}