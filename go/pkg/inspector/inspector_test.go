package inspector

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

func TestInspect(t *testing.T) {
	if os.Getenv("BE_MOCK_SERVER") == "1" {
		runMockServer()
		return
	}

	ctx := context.Background()

	tools, err := Inspect(ctx, os.Args[0], "-test.run=TestInspect", "NORMAL")
	if err != nil {
		t.Fatalf("Inspect failed: %v", err)
	}

	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}
}

func TestInspect_Error(t *testing.T) {
	if os.Getenv("BE_MOCK_SERVER") == "1" {
		return
	}
	ctx := context.Background()
	_, err := Inspect(ctx, "nonexistent-command")
	if err == nil {
		t.Error("Expected error for nonexistent command, got nil")
	}
}

func TestInspect_MCPError(t *testing.T) {
	if os.Getenv("BE_MOCK_SERVER") == "1" {
		runMockServer()
		return
	}
	ctx := context.Background()
	_, err := Inspect(ctx, os.Args[0], "-test.run=TestInspect_MCPError", "ERR")
	if err == nil {
		t.Error("Expected MCP error, got nil")
	}
}

func runMockServer() {
	scanner := bufio.NewScanner(os.Stdin)
	// 1. Recv initialize
	if !scanner.Scan() {
		return
	}
	var req map[string]interface{}
	json.Unmarshal(scanner.Bytes(), &req)

	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      req["id"],
		"result": map[string]interface{}{
			"protocolVersion": "2024-11-05",
		},
	}
	data, _ := json.Marshal(resp)
	fmt.Printf("%s\n", data)

	// 2. Recv tools/list
	if !scanner.Scan() {
		return
	}
	json.Unmarshal(scanner.Bytes(), &req)

	if os.Getenv("MOCK_MODE") == "ERR" {
		resp = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"error": map[string]interface{}{
				"code":    -32000,
				"message": "Internal error",
			},
		}
	} else {
		resp = map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result": map[string]interface{}{
				"tools": []map[string]interface{}{
					{"name": "test_tool", "description": "A test tool", "inputSchema": map[string]interface{}{"type": "object"}},
				},
			},
		}
	}
	data, _ = json.Marshal(resp)
	fmt.Printf("%s\n", data)
}

func init() {
	if len(os.Args) > 1 {
		last := os.Args[len(os.Args)-1]
		if last == "NORMAL" {
			os.Setenv("BE_MOCK_SERVER", "1")
			os.Setenv("MOCK_MODE", "NORMAL")
		} else if last == "ERR" {
			os.Setenv("BE_MOCK_SERVER", "1")
			os.Setenv("MOCK_MODE", "ERR")
		}
	}
}
