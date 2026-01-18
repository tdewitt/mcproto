package grpc

import (
	"context"

	"github.com/misfitdev/proto-mcp/go/mcp"
)

// Server implements the mcp.MCPServiceServer interface.
type Server struct {
	mcp.UnimplementedMCPServiceServer
	// TODO: Add ToolRegistry here in Phase 2
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Initialize(ctx context.Context, req *mcp.InitializeRequest) (*mcp.InitializeResponse, error) {
	return &mcp.InitializeResponse{
		ProtocolVersion: "1.0.0",
		Capabilities: &mcp.ServerCapabilities{
			Tools: &mcp.ToolCapabilities{
				SupportsListChanged: true,
			},
		},
		Metadata: map[string]string{
			"server": "proto-mcp-go-grpc",
		},
	}, nil
}

func (s *Server) ListTools(ctx context.Context, req *mcp.ListToolsRequest) (*mcp.ListToolsResponse, error) {
	// Minimal implementation for now, will be populated in Phase 2
	return &mcp.ListToolsResponse{}, nil
}

func (s *Server) CallTool(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResponse, error) {
	// Minimal implementation for now
	return &mcp.CallToolResponse{}, nil
}

func (s *Server) ListResources(ctx context.Context, req *mcp.ListResourcesRequest) (*mcp.ListResourcesResponse, error) {
	return &mcp.ListResourcesResponse{}, nil
}

func (s *Server) ReadResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResponse, error) {
	return &mcp.ReadResourceResponse{}, nil
}
