package router

import (
	"context"
	"fmt"
	"io"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"github.com/misfitdev/proto-mcp/go/stdio"
	"google.golang.org/protobuf/proto"
)

// MaxSessionMemory is the per-connection memory budget in bytes.
// Once cumulative message bytes exceed this limit the handler returns an error
// and the connection is closed.
const MaxSessionMemory = 256 * 1024 * 1024 // 256MB per session

type BinaryHandler struct {
	registry   *registry.UnifiedRegistry
	memoryUsed int64
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

		// Track cumulative message size against the session budget.
		msgSize := int64(proto.Size(msg))
		h.memoryUsed += msgSize
		if h.memoryUsed > MaxSessionMemory {
			errResp := &mcp.MCPMessage{
				Id: msg.Id,
				Payload: &mcp.MCPMessage_CallToolResponse{
					CallToolResponse: &mcp.CallToolResponse{
						Result: &mcp.CallToolResponse_Error{
							Error: &mcp.Error{
								Code:    -32603,
								Message: fmt.Sprintf("session memory limit exceeded (%d bytes used, limit %d bytes)", h.memoryUsed, MaxSessionMemory),
							},
						},
					},
				},
			}
			_ = writer.WriteMessage(errResp)
			return fmt.Errorf("session memory limit exceeded: %d bytes", h.memoryUsed)
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
			if err := writer.WriteMessage(resp); err != nil {
				return fmt.Errorf("failed to write call tool response: %w", err)
			}

		default:
			resp := &mcp.MCPMessage{
				Id: msg.Id,
				Payload: &mcp.MCPMessage_CallToolResponse{
					CallToolResponse: &mcp.CallToolResponse{
						Result: &mcp.CallToolResponse_Error{
							Error: &mcp.Error{
								Code:    -32601,
								Message: fmt.Sprintf("unsupported message type: %T", payload),
							},
						},
					},
				},
			}
			if err := writer.WriteMessage(resp); err != nil {
				return fmt.Errorf("failed to write error response for unknown payload: %w", err)
			}
		}
	}
}
