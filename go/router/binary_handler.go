package router

import (
	"context"
	"fmt"
	"io"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/stdio"
)

type BinaryHandler struct {
	registry *mcp.UnifiedRegistry
}

func NewBinaryHandler(r *mcp.UnifiedRegistry) *BinaryHandler {
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
			writer.WriteMessage(resp)

		case *mcp.MCPMessage_ListToolsRequest:
			resp := &mcp.MCPMessage{
				Id: msg.Id,
				Payload: &mcp.MCPMessage_ListToolsResponse{
					ListToolsResponse: &mcp.ListToolsResponse{
						Tools: h.registry.List(payload.ListToolsRequest.Query),
					},
				},
			}
			writer.WriteMessage(resp)

		case *mcp.MCPMessage_CallToolRequest:
			result, err := h.registry.Call(context.Background(), payload.CallToolRequest.Name, payload.CallToolRequest.Arguments.Value)
			
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
			writer.WriteMessage(resp)
		}
	}
}
