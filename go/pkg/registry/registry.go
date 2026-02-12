package registry

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

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
		if err := r.RegisterAlias(tool.Name, alias); err != nil {
			log.Printf("WARNING: Failed to register alias %q for tool %q: %v", alias, tool.Name, err)
		}
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

type toolMatch struct {
	tool  *mcp.Tool
	score int
}

func (r *UnifiedRegistry) List(query string) []*mcp.Tool {
	queryLower := strings.ToLower(query)
	matches := make([]toolMatch, 0)

	for name, entry := range r.tools {
		score := r.calculateRelevanceScore(name, entry.Tool.Description, queryLower)

		// Check aliases for matches
		aliases := r.canonicalAliases[name]
		aliasScore := 0
		for _, alias := range aliases {
			if s := r.calculateRelevanceScore(alias, "", queryLower); s > aliasScore {
				aliasScore = s
			}
		}
		if aliasScore > score {
			score = aliasScore
		}

		// If query is empty, include all tools with base score
		if query == "" {
			score = 1
		}

		if score > 0 {
			// Return tool with primary name only (not aliases)
			// Aliases are available via metadata if needed
			matches = append(matches, toolMatch{
				tool:  entry.Tool,
				score: score,
			})
		}
	}

	// Sort by relevance score (descending)
	for i := 0; i < len(matches)-1; i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].score > matches[i].score {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}

	result := make([]*mcp.Tool, len(matches))
	for i, m := range matches {
		result[i] = m.tool
	}
	return result
}

func (r *UnifiedRegistry) calculateRelevanceScore(name, description, queryLower string) int {
	if queryLower == "" {
		return 0
	}

	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)

	// Exact match on name
	if nameLower == queryLower {
		return 100
	}

	// Name starts with query
	if strings.HasPrefix(nameLower, queryLower) {
		return 80
	}

	// Name contains query
	if strings.Contains(nameLower, queryLower) {
		return 50
	}

	// Description contains query
	if strings.Contains(descLower, queryLower) {
		return 20
	}

	return 0
}

// GetTool looks up a tool by name, resolving aliases if necessary.
// Returns the ToolEntry and true if found, or a zero ToolEntry and false otherwise.
func (r *UnifiedRegistry) GetTool(name string) (ToolEntry, bool) {
	if canonical, ok := r.aliases[name]; ok {
		name = canonical
	}
	entry, ok := r.tools[name]
	return entry, ok
}

func (r *UnifiedRegistry) Call(ctx context.Context, name string, args []byte) (*mcp.ToolResult, error) {
	originalName := name
	if canonical, ok := r.aliases[name]; ok {
		name = canonical
	}
	entry, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool %q not found", originalName)
	}
	start := time.Now()
	resp, err := entry.Handler(ctx, args)
	log.Printf("registry.call tool=%s input_name=%s duration_ms=%d error=%t", name, originalName, time.Since(start).Milliseconds(), err != nil)
	return resp, err
}

func (r *UnifiedRegistry) CallByBsrRef(ctx context.Context, bsrRef string, args []byte) (*mcp.ToolResult, error) {
	if bsrRef == "" {
		return nil, fmt.Errorf("bsr_ref is required")
	}
	for _, entry := range r.tools {
		if toolBsrRef(entry.Tool) == bsrRef {
			return entry.Handler(ctx, args)
		}
	}
	return nil, fmt.Errorf("no tool found for bsr_ref %q", bsrRef)
}

func toolBsrRef(tool *mcp.Tool) string {
	if tool == nil {
		return ""
	}
	if ref, ok := tool.SchemaSource.(*mcp.Tool_BsrRef); ok {
		return ref.BsrRef
	}
	return ""
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
