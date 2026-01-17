package stdio

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"google.golang.org/protobuf/proto"
)

func TestReadMessage(t *testing.T) {
	// Create a sample message
	msg := &mcp.MCPMessage{
		Id: 1,
		Payload: &mcp.MCPMessage_InitializeRequest{
			InitializeRequest: &mcp.InitializeRequest{
				ProtocolVersion: "1.0.0",
			},
		},
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	// Create a buffer with length prefix + data
	var buf bytes.Buffer
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, uint32(len(data)))
	buf.Write(lenBytes)
	buf.Write(data)

	// Test ReadMessage
	reader := NewReader(&buf)
	readMsg, err := reader.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if readMsg.Id != msg.Id {
		t.Errorf("Expected ID %d, got %d", msg.Id, readMsg.Id)
	}
}

func TestReadMessage_Errors(t *testing.T) {
	// Empty buffer (EOF)
	reader := NewReader(&bytes.Buffer{})
	_, err := reader.ReadMessage()
	if err == nil {
		t.Error("Expected error on empty buffer, got nil")
	}

	// Incomplete length prefix
	var buf bytes.Buffer
	buf.Write([]byte{0, 0, 0}) // only 3 bytes
	reader = NewReader(&buf)
	_, err = reader.ReadMessage()
	if err == nil {
		t.Error("Expected error on incomplete length prefix, got nil")
	}

	// Incomplete message body
	buf.Reset()
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, 10) // claim 10 bytes
	buf.Write(lenBytes)
	buf.Write([]byte{1, 2, 3}) // only 3 bytes
	reader = NewReader(&buf)
	_, err = reader.ReadMessage()
	if err == nil {
		t.Error("Expected error on incomplete message body, got nil")
	}
}
