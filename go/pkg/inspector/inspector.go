package inspector

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type ListToolsResult struct {
	Tools []Tool `json:"tools"`
}

type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func Inspect(ctx context.Context, command string, args ...string) ([]Tool, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close() // Close stdin pipe on stdout pipe failure
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, err
	}
	defer cmd.Process.Kill()

	// 1. Initialize
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "mcproto-inspector",
				"version": "0.1.0",
			},
		},
	}
	if err := send(stdin, initReq); err != nil {
		return nil, err
	}

	var initResp JSONRPCResponse
	if err := recv(stdout, &initResp); err != nil {
		return nil, err
	}

	// 2. List Tools
	listReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	}
	if err := send(stdin, listReq); err != nil {
		return nil, err
	}

	var listResp JSONRPCResponse
	if err := recv(stdout, &listResp); err != nil {
		return nil, err
	}

	if listResp.Error != nil {
		return nil, fmt.Errorf("MCP error: %s", listResp.Error.Message)
	}

	var result ListToolsResult
	if err := json.Unmarshal(listResp.Result, &result); err != nil {
		return nil, err
	}

	return result.Tools, nil
}

func send(w io.Writer, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

func recv(r io.Reader, msg interface{}) error {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		return io.EOF
	}
	return json.Unmarshal(scanner.Bytes(), msg)
}
