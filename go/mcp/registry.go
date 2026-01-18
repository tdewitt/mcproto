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

// GenerateMockCatalog populates the registry with 1,000 coding-themed tools.
func (r *UnifiedRegistry) GenerateMockCatalog() {
	prefixes := []string{"git", "fs", "db", "net", "sys", "cloud", "ai", "test", "build", "deploy"}
	verbs := []string{"read", "write", "list", "query", "exec", "sync", "scan", "analyze", "delete", "create"}

	// Use GetModuleRequest as a placeholder BSR ref for all mock tools
	toolRef := "buf.build/bufbuild/registry/buf.registry.module.v1.Module:main"

	for i := 0; i < 1000; i++ {
		prefix := prefixes[(i/100)%len(prefixes)]
		verb := verbs[(i/10)%len(verbs)]
		name := fmt.Sprintf("%s_%s_%d", prefix, verb, i)
		desc := fmt.Sprintf("Mock tool for %s operation on %s service (Instance %d)", verb, prefix, i)

		r.Register(&mcp.Tool{
			Name:        name,
			Description: desc,
			SchemaSource: &mcp.Tool_BsrRef{
				BsrRef: toolRef,
			},
		}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
			return &mcp.ToolResult{
				Content: []*mcp.ToolContent{
					{
						Content: &mcp.ToolContent_Text{
							Text: fmt.Sprintf("Executed %s successfully.", name),
						},
					},
				},
			}, nil
		})
	}
}
