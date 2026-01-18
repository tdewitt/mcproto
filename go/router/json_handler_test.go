package router

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONHandler_Initialize(t *testing.T) {
	handler := &JSONHandler{}
	
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
