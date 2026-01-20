package router

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/mcp/analytics"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestJSONHandler_Initialize(t *testing.T) {
	reg := registry.NewUnifiedRegistry(nil)
	handler := NewJSONHandler(reg, nil)

	input := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2024-11-05"}}`
	output := &bytes.Buffer{}

	rw := &combinedReadWriter{
		Reader: strings.NewReader(input),
		Writer: output,
	}

	if err := handler.Handle(rw); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp["id"].(float64) != 1 {
		t.Errorf("Expected response id 1, got %v", resp["id"])
	}

	result := resp["result"].(map[string]interface{})
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocolVersion 2024-11-05, got %v", result["protocolVersion"])
	}
}

func TestJSONHandler_ListTools(t *testing.T) {
	reg := registry.NewUnifiedRegistry(nil)
	handler := NewJSONHandler(reg, nil)
	input := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	output := &bytes.Buffer{}

	rw := &combinedReadWriter{
		Reader: strings.NewReader(input),
		Writer: output,
	}

	if err := handler.Handle(rw); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	result := resp["result"].(map[string]interface{})
	tools := result["tools"].([]interface{})
	if len(tools) != 3 {
		t.Fatalf("Expected 3 tools, got %d", len(tools))
	}

	tool := tools[0].(map[string]interface{})
	if tool["name"] != "search_registry" {
		t.Fatalf("Expected tool name search_registry, got %v", tool["name"])
	}
}

func TestJSONHandler_CallTool(t *testing.T) {
	reg := registry.NewUnifiedRegistry(nil)
	reg.Register(&mcp.Tool{
		Name:        "demo_tool",
		Description: "Demo tool.",
		SchemaSource: &mcp.Tool_BsrRef{
			BsrRef: "buf.build/mcpb/analytics/misfit.analytics.v1.ExtractRequest:main",
		},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		return &mcp.ToolResult{
			Content: []*mcp.ToolContent{
				{
					Content: &mcp.ToolContent_Text{
						Text: "ok",
					},
				},
			},
		}, nil
	})

	md := analytics.File_analytics_proto.Messages().ByName("ExtractRequest")
	resolver := fakeResolver{mt: dynamicpb.NewMessageType(md)}

	handler := NewJSONHandler(reg, resolver)
	input := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"call_tool","arguments":{"bsr_ref":"buf.build/mcpb/analytics/misfit.analytics.v1.ExtractRequest:main","tool_name":"demo_tool","arguments":{}}}}`
	output := &bytes.Buffer{}

	rw := &combinedReadWriter{
		Reader: strings.NewReader(input),
		Writer: output,
	}

	if err := handler.Handle(rw); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	result := resp["result"].(map[string]interface{})
	content := result["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("Expected 1 content entry, got %d", len(content))
	}

	entry := content[0].(map[string]interface{})
	if entry["text"] != "ok" {
		t.Fatalf("Expected content text ok, got %v", entry["text"])
	}
}

type fakeResolver struct {
	mt protoreflect.MessageType
}

func (f fakeResolver) Resolve(ctx context.Context, refStr string) (protoreflect.MessageType, error) {
	return f.mt, nil
}
