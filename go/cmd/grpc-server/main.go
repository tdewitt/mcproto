package main

import (
	"fmt"
	"log"
	"net"

	"github.com/misfitdev/proto-mcp/go/mcp"
	grpc_pkg "github.com/misfitdev/proto-mcp/go/pkg/grpc"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	reg := registry.NewUnifiedRegistry()
	reg.GenerateMockCatalog()
	mcp.RegisterMCPServiceServer(s, grpc_pkg.NewServer(reg))
	fmt.Println("gRPC server listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
