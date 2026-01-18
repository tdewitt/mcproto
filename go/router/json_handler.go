package router

import (
	"encoding/json"
	"fmt"
	"io"
)

type JSONHandler struct{}

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func (h *JSONHandler) Handle(rw io.ReadWriter) error {
	dec := json.NewDecoder(rw)
	var req jsonRPCRequest
	if err := dec.Decode(&req); err != nil {
		return fmt.Errorf("failed to decode JSON-RPC request: %w", err)
	}

	if req.Method != "initialize" {
		return fmt.Errorf("unsupported method: %s", req.Method)
	}

	// Minimal initialize response
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"serverInfo": map[string]interface{}{
				"name":    "proto-mcp-dual-server",
				"version": "0.1.0",
			},
		},
	}

	enc := json.NewEncoder(rw)
	return enc.Encode(resp)
}
