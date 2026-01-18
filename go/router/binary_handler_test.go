package router

import (
	"bytes"
	"testing"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/stdio"
)

func TestBinaryHandler(t *testing.T) {
	handler := &BinaryHandler{}
	
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
