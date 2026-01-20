package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/bsr"
	"google.golang.org/protobuf/proto"
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
	tools            map[string]ToolEntry
	aliases          map[string]string
	canonicalAliases map[string][]string
	bsrClient        *bsr.Client
}

func NewUnifiedRegistry(c *bsr.Client) *UnifiedRegistry {
	return &UnifiedRegistry{
		tools:            make(map[string]ToolEntry),
		aliases:          make(map[string]string),
		canonicalAliases: make(map[string][]string),
		bsrClient:        c,
	}
}

func (r *UnifiedRegistry) Register(tool *mcp.Tool, handler ToolHandler) {
	r.tools[tool.Name] = ToolEntry{
		Tool:    tool,
		Handler: handler,
	}
	if alias := snakeCaseName(tool.Name); alias != tool.Name {
		_ = r.RegisterAlias(tool.Name, alias)
	}
}

func (r *UnifiedRegistry) RegisterAlias(canonical string, alias string) error {
	if canonical == "" || alias == "" {
		return fmt.Errorf("canonical and alias names must be non-empty")
	}
	if _, ok := r.tools[canonical]; !ok {
		return fmt.Errorf("canonical tool %q is not registered", canonical)
	}
	if _, ok := r.tools[alias]; ok {
		return fmt.Errorf("alias %q conflicts with an existing tool name", alias)
	}
	if existing, ok := r.aliases[alias]; ok {
		return fmt.Errorf("alias %q already mapped to %q", alias, existing)
	}
	r.aliases[alias] = canonical
	r.canonicalAliases[canonical] = append(r.canonicalAliases[canonical], alias)
	return nil
}

func (r *UnifiedRegistry) List(query string) []*mcp.Tool {
	var result []*mcp.Tool
	queryLower := strings.ToLower(query)
	for name, entry := range r.tools {
		matches := query == "" ||
			strings.Contains(strings.ToLower(name), queryLower) ||
			strings.Contains(strings.ToLower(entry.Tool.Description), queryLower)

		aliases := r.canonicalAliases[name]
		if !matches {
			for _, alias := range aliases {
				if strings.Contains(strings.ToLower(alias), queryLower) {
					matches = true
					break
				}
			}
		}
		if !matches {
			continue
		}

		if len(aliases) == 0 {
			result = append(result, entry.Tool)
			continue
		}

		for _, alias := range aliases {
			result = append(result, cloneToolWithName(entry.Tool, alias))
		}
	}
	return result
}

func (r *UnifiedRegistry) Call(ctx context.Context, name string, args []byte) (*mcp.ToolResult, error) {
	if canonical, ok := r.aliases[name]; ok {
		name = canonical
	}
	entry, ok := r.tools[name]
	if !ok {
		return nil, nil // or error
	}
	return entry.Handler(ctx, args)
}

func cloneToolWithName(tool *mcp.Tool, name string) *mcp.Tool {
	if tool == nil {
		return nil
	}
	clone := proto.Clone(tool).(*mcp.Tool)
	clone.Name = name
	return clone
}

func snakeCaseName(name string) string {
	if name == "" {
		return name
	}
	hasUpper := false
	for i := 0; i < len(name); i++ {
		if name[i] >= 'A' && name[i] <= 'Z' {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return name
	}
	var b strings.Builder
	b.Grow(len(name) + 4)
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch == '_' {
			b.WriteByte(ch)
			continue
		}
		if ch >= 'A' && ch <= 'Z' {
			if i > 0 && name[i-1] != '_' {
				b.WriteByte('_')
			}
			b.WriteByte(ch + ('a' - 'A'))
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
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
