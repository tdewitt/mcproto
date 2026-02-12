package router

import (
	"bytes"
	"strings"
	"testing"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/registry"
	"github.com/misfitdev/proto-mcp/go/stdio"
)

func TestBinaryHandler(t *testing.T) {
	registry := registry.NewUnifiedRegistry(nil)
	handler := NewBinaryHandler(registry)

	msg := &mcp.MCPMessage{
		Id: 1,
		Payload: &mcp.MCPMessage_InitializeRequest{
			InitializeRequest: &mcp.InitializeRequest{
				ProtocolVersion: "1.0.0",
			},
		},
	}

	input := &bytes.Buffer{}
	writer := stdio.NewWriter(input)
	if err := writer.WriteMessage(msg); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	output := &bytes.Buffer{}
	rw := &combinedReadWriter{
		Reader: input,
		Writer: output,
	}

	// Run handler in background or manually stop it
	// BinaryHandler.Handle loops until EOF
	if err := handler.Handle(rw); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	reader := stdio.NewReader(output)
	resp, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read echoed message: %v", err)
	}

	if resp.Id != msg.Id {
		t.Errorf("Expected echoed ID %d, got %d", msg.Id, resp.Id)
	}
}

func TestBinaryHandler_MemoryUsedTracking(t *testing.T) {
	reg := registry.NewUnifiedRegistry(nil)
	handler := NewBinaryHandler(reg)

	// Send a valid message and verify memoryUsed is incremented.
	msg := &mcp.MCPMessage{
		Id: 1,
		Payload: &mcp.MCPMessage_InitializeRequest{
			InitializeRequest: &mcp.InitializeRequest{
				ProtocolVersion: "1.0.0",
			},
		},
	}

	input := &bytes.Buffer{}
	writer := stdio.NewWriter(input)
	if err := writer.WriteMessage(msg); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	output := &bytes.Buffer{}
	rw := &combinedReadWriter{
		Reader: input,
		Writer: output,
	}

	if err := handler.Handle(rw); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}

	if handler.memoryUsed <= 0 {
		t.Fatalf("expected memoryUsed > 0 after processing a message, got %d", handler.memoryUsed)
	}
}

func TestBinaryHandler_SessionMemoryLimit(t *testing.T) {
	reg := registry.NewUnifiedRegistry(nil)
	handler := NewBinaryHandler(reg)

	// Pre-set memoryUsed to just below the limit so the next message triggers it.
	handler.memoryUsed = MaxSessionMemory - 1

	msg := &mcp.MCPMessage{
		Id: 99,
		Payload: &mcp.MCPMessage_InitializeRequest{
			InitializeRequest: &mcp.InitializeRequest{
				ProtocolVersion: "1.0.0",
			},
		},
	}

	input := &bytes.Buffer{}
	writer := stdio.NewWriter(input)
	if err := writer.WriteMessage(msg); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	output := &bytes.Buffer{}
	rw := &combinedReadWriter{
		Reader: input,
		Writer: output,
	}

	err := handler.Handle(rw)
	if err == nil {
		t.Fatal("expected error when session memory limit is exceeded")
	}
	if !strings.Contains(err.Error(), "session memory limit exceeded") {
		t.Fatalf("expected memory limit error, got: %v", err)
	}

	// The handler should have written an error response before closing.
	if output.Len() == 0 {
		t.Fatal("expected an error response to be written to the output")
	}

	reader := stdio.NewReader(output)
	resp, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("failed to read error response: %v", err)
	}

	callResp := resp.GetCallToolResponse()
	if callResp == nil {
		t.Fatal("expected CallToolResponse in error message")
	}
	errResult := callResp.GetError()
	if errResult == nil {
		t.Fatal("expected error result in response")
	}
	if !strings.Contains(errResult.Message, "session memory limit exceeded") {
		t.Fatalf("expected memory limit message in response, got: %s", errResult.Message)
	}
}
