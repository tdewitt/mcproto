package router

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func TestJSONHandler_ListTools_EmptyRegistry(t *testing.T) {
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

	// With an empty registry, only the 3 meta-tools should be present.
	if len(tools) != 3 {
		t.Fatalf("Expected 3 meta-tools, got %d", len(tools))
	}

	expectedNames := []string{"search_registry", "resolve_schema", "call_tool"}
	for i, name := range expectedNames {
		tool := tools[i].(map[string]interface{})
		if tool["name"] != name {
			t.Errorf("Expected tool[%d] name %q, got %v", i, name, tool["name"])
		}
	}
}

func TestJSONHandler_ListTools_WithRegisteredTools(t *testing.T) {
	reg := registry.NewUnifiedRegistry(nil)
	reg.Register(&mcp.Tool{
		Name:        "my_tool",
		Description: "A custom tool.",
		SchemaSource: &mcp.Tool_BsrRef{
			BsrRef: "buf.build/mcpb/test/test.v1.MyRequest:main",
		},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		return nil, nil
	})

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

	// 3 meta-tools + 1 registered tool = 4
	if len(tools) != 4 {
		t.Fatalf("Expected 4 tools (3 meta + 1 registered), got %d", len(tools))
	}

	// Meta-tools come first.
	if tools[0].(map[string]interface{})["name"] != "search_registry" {
		t.Errorf("Expected first tool to be search_registry, got %v", tools[0].(map[string]interface{})["name"])
	}

	// Registered tool appears after meta-tools.
	lastTool := tools[3].(map[string]interface{})
	if lastTool["name"] != "my_tool" {
		t.Errorf("Expected last tool to be my_tool, got %v", lastTool["name"])
	}
	if lastTool["bsr_ref"] != "buf.build/mcpb/test/test.v1.MyRequest:main" {
		t.Errorf("Expected bsr_ref on registered tool, got %v", lastTool["bsr_ref"])
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

func TestJSONHandler_DirectToolCall_WithResolver(t *testing.T) {
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
						Text: "direct-call-ok",
					},
				},
			},
		}, nil
	})

	md := analytics.File_analytics_proto.Messages().ByName("ExtractRequest")
	resolver := fakeResolver{mt: dynamicpb.NewMessageType(md)}

	handler := NewJSONHandler(reg, resolver)
	// Call demo_tool directly by name instead of going through call_tool meta-tool.
	input := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"demo_tool","arguments":{}}}`
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

	if resp["error"] != nil {
		t.Fatalf("Expected no error, got %v", resp["error"])
	}

	result := resp["result"].(map[string]interface{})
	content := result["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("Expected 1 content entry, got %d", len(content))
	}

	entry := content[0].(map[string]interface{})
	if entry["text"] != "direct-call-ok" {
		t.Fatalf("Expected content text 'direct-call-ok', got %v", entry["text"])
	}
}

func TestJSONHandler_DirectToolCall_WithoutResolver(t *testing.T) {
	reg := registry.NewUnifiedRegistry(nil)
	reg.Register(&mcp.Tool{
		Name:        "json_tool",
		Description: "Tool that accepts raw JSON.",
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		// Handler receives raw JSON bytes when no resolver is available.
		return &mcp.ToolResult{
			Content: []*mcp.ToolContent{
				{
					Content: &mcp.ToolContent_Text{
						Text: fmt.Sprintf("got: %s", string(args)),
					},
				},
			},
		}, nil
	})

	handler := NewJSONHandler(reg, nil)
	input := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"json_tool","arguments":{"key":"value"}}}`
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

	if resp["error"] != nil {
		t.Fatalf("Expected no error, got %v", resp["error"])
	}

	result := resp["result"].(map[string]interface{})
	content := result["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("Expected 1 content entry, got %d", len(content))
	}

	entry := content[0].(map[string]interface{})
	text := entry["text"].(string)
	if !strings.Contains(text, `"key":"value"`) {
		t.Fatalf("Expected JSON args to be passed through, got %v", text)
	}
}

func TestJSONHandler_DirectToolCall_UnknownTool(t *testing.T) {
	reg := registry.NewUnifiedRegistry(nil)
	handler := NewJSONHandler(reg, nil)

	input := `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"nonexistent_tool","arguments":{}}}`
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

	if resp["error"] == nil {
		t.Fatal("Expected error for unknown tool, got nil")
	}

	errObj := resp["error"].(map[string]interface{})
	msg := errObj["message"].(string)
	if !strings.Contains(msg, "nonexistent_tool") {
		t.Errorf("Expected error message to contain tool name, got %q", msg)
	}
}

type fakeResolver struct {
	mt protoreflect.MessageType
}

func (f fakeResolver) Resolve(ctx context.Context, refStr string) (protoreflect.MessageType, error) {
	return f.mt, nil
}
