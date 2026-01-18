package grpc

import (
	"context"

	"github.com/misfitdev/proto-mcp/go/mcp"
)

// Server implements the mcp.MCPServiceServer interface.
type Server struct {
	mcp.UnimplementedMCPServiceServer
	registry *mcp.UnifiedRegistry
}

func NewServer(r *mcp.UnifiedRegistry) *Server {
	return &Server{
		registry: r,
	}
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
	// The gRPC ListTools can also support search if we add a field, 
	// but for now we'll just use the registry.
	return &mcp.ListToolsResponse{
		Tools: s.registry.List(""),
	}, nil
}

func (s *Server) CallTool(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResponse, error) {
	result, err := s.registry.Call(ctx, req.Name, req.Arguments.Value)
	if err != nil {
		return &mcp.CallToolResponse{
			Result: &mcp.CallToolResponse_Error{
				Error: &mcp.Error{
					Code:    -32603,
					Message: err.Error(),
				},
			},
		}, nil
	}

	return &mcp.CallToolResponse{
		Result: &mcp.CallToolResponse_Success{
			Success: result,
		},
	}, nil
}

func (s *Server) ListResources(ctx context.Context, req *mcp.ListResourcesRequest) (*mcp.ListResourcesResponse, error) {
	return &mcp.ListResourcesResponse{}, nil
}

func (s *Server) ReadResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResponse, error) {
	return &mcp.ReadResourceResponse{}, nil
}
