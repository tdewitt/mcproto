package main

import (
	"fmt"
	"log"
	"net"

	"github.com/misfitdev/proto-mcp/go/pkg/github"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	srv, err := github.NewServer()
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	github.RegisterGitHubServiceServer(s, srv)
	fmt.Println("GitHub MCP Server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
