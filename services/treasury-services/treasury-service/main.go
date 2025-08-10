package main

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "example.com/go-mono-repo/proto/treasury"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	pb.UnimplementedManifestServer
}

func (s *server) GetManifest(ctx context.Context, req *pb.ManifestRequest) (*pb.ManifestResponse, error) {
	return &pb.ManifestResponse{
		Message: "treasury-service",
	}, nil
}

func main() {
	port := ":50052"
	
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterManifestServer(s, &server{})
	
	reflection.Register(s)
	
	fmt.Printf("Treasury service starting on port %s\n", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}