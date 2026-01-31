package router

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"github.com/misfitdev/proto-mcp/go/stdio"
)

type BinaryHandler struct {
	registry *registry.UnifiedRegistry
}

func NewBinaryHandler(r *registry.UnifiedRegistry) *BinaryHandler {
	return &BinaryHandler{registry: r}
}

func (h *BinaryHandler) Handle(rw io.ReadWriter) error {
	reader := stdio.NewReader(rw)
	writer := stdio.NewWriter(rw)

	for {
		msg, err := reader.ReadMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("binary handler read error: %w", err)
		}

		switch payload := msg.Payload.(type) {
		case *mcp.MCPMessage_InitializeRequest:
			resp := &mcp.MCPMessage{
				Id: msg.Id,
				Payload: &mcp.MCPMessage_InitializeResponse{
					InitializeResponse: &mcp.InitializeResponse{
						ProtocolVersion: "1.0.0",
					},
				},
			}
			if err := writer.WriteMessage(resp); err != nil {
				return fmt.Errorf("failed to write initialize response: %w", err)
			}

		case *mcp.MCPMessage_ListToolsRequest:
			resp := &mcp.MCPMessage{
				Id: msg.Id,
				Payload: &mcp.MCPMessage_ListToolsResponse{
					ListToolsResponse: &mcp.ListToolsResponse{
						Tools: h.registry.List(payload.ListToolsRequest.Query),
					},
				},
			}
			if err := writer.WriteMessage(resp); err != nil {
				return fmt.Errorf("failed to write list tools response: %w", err)
			}

		case *mcp.MCPMessage_CallToolRequest:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			result, err := h.registry.Call(ctx, payload.CallToolRequest.Name, payload.CallToolRequest.Arguments.Value)
			cancel()

			var responsePayload mcp.CallToolResponse
			if err != nil {
				responsePayload.Result = &mcp.CallToolResponse_Error{
					Error: &mcp.Error{
						Code:    -32603,
						Message: err.Error(),
					},
				}
			} else {
				responsePayload.Result = &mcp.CallToolResponse_Success{
					Success: result,
				}
			}

			resp := &mcp.MCPMessage{
				Id: msg.Id,
				Payload: &mcp.MCPMessage_CallToolResponse{
					CallToolResponse: &responsePayload,
				},
			}
			if err := writer.WriteMessage(resp); err != nil {
				return fmt.Errorf("failed to write call tool response: %w", err)
			}

		default:
			return fmt.Errorf("unsupported message type: %T", msg.Payload)
		}
	}
}
