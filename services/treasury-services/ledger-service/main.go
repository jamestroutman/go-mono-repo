package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "example.com/go-mono-repo/proto/ledger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type manifestServer struct {
	pb.UnimplementedManifestServer
}

func (s *manifestServer) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
	return &pb.ManifestResponse{
		Message: "ledger-service",
	}, nil
}

func main() {
	port := ":50051"
	
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterManifestServer(grpcServer, &manifestServer{})
	
	// Register reflection service for debugging
	reflection.Register(grpcServer)

	fmt.Printf("Ledger service starting on port %s...\n", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}