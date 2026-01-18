package mcp

import (
	"context"
	"strings"

	"github.com/misfitdev/proto-mcp/go/mcp"
)

// ToolHandler is a function that executes a tool call.
type ToolHandler func(ctx context.Context, args []byte) (*mcp.ToolResult, error)

// ToolEntry represents a tool in the registry.
type ToolEntry struct {
	Tool    *mcp.Tool
	Handler ToolHandler
}

// UnifiedRegistry is a transport-agnostic registry of tools.
type UnifiedRegistry struct {
	tools map[string]ToolEntry
}

func NewUnifiedRegistry() *UnifiedRegistry {
	return &UnifiedRegistry{
		tools: make(map[string]ToolEntry),
	}
}

func (r *UnifiedRegistry) Register(tool *mcp.Tool, handler ToolHandler) {
	r.tools[tool.Name] = ToolEntry{
		Tool:    tool,
		Handler: handler,
	}
}

func (r *UnifiedRegistry) List(query string) []*mcp.Tool {
	var result []*mcp.Tool
	for _, entry := range r.tools {
		if query == "" || strings.Contains(strings.ToLower(entry.Tool.Name), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(entry.Tool.Description), strings.ToLower(query)) {
			result = append(result, entry.Tool)
		}
	}
	return result
}

func (r *UnifiedRegistry) Call(ctx context.Context, name string, args []byte) (*mcp.ToolResult, error) {
	entry, ok := r.tools[name]
	if !ok {
		return nil, nil // or error
	}
	return entry.Handler(ctx, args)
}
