package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/bsr"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"github.com/misfitdev/proto-mcp/go/stdio"
)

func main() {
	client := bsr.NewClient()
	reg := bsr.NewRegistry(client)
	
	reader := stdio.NewReader(os.Stdin)
	writer := stdio.NewWriter(os.Stdout)

	// In this demo, the server "knows" about a public tool on BSR
	toolRef := "buf.build/bufbuild/registry/buf.registry.module.v1.Module:main"

	// We also need a UnifiedRegistry for tool management in dynamic-server
	toolReg := registry.NewUnifiedRegistry()
	toolReg.Register(&mcp.Tool{
		Name:        "get_module_info",
		Description: "Dynamically resolved BSR tool (Module)",
		SchemaSource: &mcp.Tool_BsrRef{
			BsrRef: toolRef,
		},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		return nil, nil
	})

	for {
		msg, err := reader.ReadMessage()
		if err != nil {
			return
		}

		switch payload := msg.Payload.(type) {
		case *mcp.MCPMessage_ListToolsRequest:
			// Offer the dynamic tool
			resp := &mcp.MCPMessage{
				Id: msg.Id,
				Payload: &mcp.MCPMessage_ListToolsResponse{
					ListToolsResponse: &mcp.ListToolsResponse{
						Tools: toolReg.List(""),
					},
				},
			}
			writer.WriteMessage(resp)

		case *mcp.MCPMessage_CallToolRequest:
			// Resolve the schema to unpack the arguments
			_, err := reg.Resolve(context.Background(), toolRef)
			if err != nil {
				log.Fatalf("Server failed to resolve schema: %v", err)
			}

			args, err := reg.Unpack(payload.CallToolRequest.Arguments)
			if err != nil {
				log.Fatalf("Server failed to unpack arguments: %v", err)
			}

			// Let's just print that we received it.
			fmt.Fprintln(os.Stderr, "DEBUG: [Server] Executing dynamic tool with:", args.Descriptor().FullName())

			// Send back a success response
			resp := &mcp.MCPMessage{
				Id: msg.Id,
				Payload: &mcp.MCPMessage_CallToolResponse{
					CallToolResponse: &mcp.CallToolResponse{
						Result: &mcp.CallToolResponse_Success{
							Success: &mcp.ToolResult{
								Content: []*mcp.ToolContent{
									{
										Content: &mcp.ToolContent_Text{
											Text: "Dynamically processed Module object",
										},
									},
								},
							},
						},
					},
				},
			}
			writer.WriteMessage(resp)
		}
	}
}
