package stdio

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"google.golang.org/protobuf/proto"
)

func TestWriteMessage(t *testing.T) {
	var buf bytes.Buffer
	writer := NewWriter(&buf)

	msg := &mcp.MCPMessage{
		Id: 1,
		Payload: &mcp.MCPMessage_InitializeRequest{
			InitializeRequest: &mcp.InitializeRequest{
				ProtocolVersion: "1.0.0",
			},
		},
	}

	if err := writer.WriteMessage(msg); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Verify length prefix
	data := buf.Bytes()
	if len(data) < 4 {
		t.Fatalf("Buffer too short: %d", len(data))
	}
	length := binary.BigEndian.Uint32(data[:4])

	// Verify body
	body := data[4:]
	if uint32(len(body)) != length {
		t.Errorf("Expected body length %d, got %d", length, len(body))
	}

	readMsg := &mcp.MCPMessage{}
	if err := proto.Unmarshal(body, readMsg); err != nil {
		t.Fatalf("Failed to unmarshal body: %v", err)
	}

	if readMsg.Id != msg.Id {
		t.Errorf("Expected ID %d, got %d", msg.Id, readMsg.Id)
	}
}
