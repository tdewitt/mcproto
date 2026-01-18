package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/misfitdev/proto-mcp/go/mcp"
	grpc_pkg "github.com/misfitdev/proto-mcp/go/pkg/grpc"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"google.golang.org/grpc"
)

func main() {
	transport := flag.String("transport", "grpc", "Transport to use (grpc or stdio)")
	mode := flag.String("mode", "proto", "Protocol mode (explicit, search, or proto)")
	flag.Parse()

	reg := registry.NewUnifiedRegistry()
	reg.PopulateETLTools()

	fmt.Fprintf(os.Stderr, "Starting Showdown Server [Mode: %s, Transport: %s]\n", *mode, *transport)

	if *transport == "grpc" {
		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
		mcp.RegisterMCPServiceServer(s, grpc_pkg.NewServer(reg))
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	} else {
		// Stdio Transport
		// For the showdown, we will use the ProtocolRouter
		// But we need to handle the specific 'mode' for benchmarking.
		// For now, let's just use gRPC as the primary showdown transport.
		log.Fatal("Stdio transport for multi-mode showdown not fully implemented yet")
	}
}
