package main

import (
	"fmt"
	"log"
	"net"

	"github.com/misfitdev/proto-mcp/go/mcp"
	mcp_grpc "github.com/misfitdev/proto-mcp/go/pkg/grpc"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	mcp.RegisterMCPServiceServer(s, mcp_grpc.NewServer())
	fmt.Println("gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
