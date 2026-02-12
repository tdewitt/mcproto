package registry

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/bsr"
	"google.golang.org/protobuf/proto"
)

// ToolHandler is a function that executes a tool call.
type ToolHandler func(ctx context.Context, args []byte) (*mcp.ToolResult, error)

// ToolEntry represents a tool in the registry with optional metadata.
type ToolEntry struct {
	Tool     *mcp.Tool
	Handler  ToolHandler
	Category string   // Integration grouping: "jira", "linear", "notion", "github", "etl", "discovery", "mock"
	Tags     []string // Additional searchable tags
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

// Register adds a tool with no category. Kept for backward compatibility.
func (r *UnifiedRegistry) Register(tool *mcp.Tool, handler ToolHandler) {
	r.RegisterWithCategory(tool, handler, "", nil)
}

// RegisterWithCategory adds a tool with an integration category and optional tags.
func (r *UnifiedRegistry) RegisterWithCategory(tool *mcp.Tool, handler ToolHandler, category string, tags []string) {
	r.tools[tool.Name] = ToolEntry{
		Tool:     tool,
		Handler:  handler,
		Category: category,
		Tags:     tags,
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

// scoredTool holds a tool and its relevance score for sorted output.
type scoredTool struct {
	tool  *mcp.Tool
	score int
}

// List returns tools matching the query string. It supports three modes:
//
//   - Empty query: returns all tools with a baseline score.
//   - Category filter: queries starting with "category:" or "integration:"
//     return only tools in that category (e.g. "category:jira").
//   - Free text: relevance-scored search across name, aliases, tags, and description.
//
// Results always include the canonical tool entry. When a tool has aliases,
// cloned entries with alias names are also included. Results are sorted by
// relevance score descending, then alphabetically for determinism.
func (r *UnifiedRegistry) List(query string) []*mcp.Tool {
	query = strings.TrimSpace(query)
	queryLower := strings.ToLower(query)

	// Parse category/integration filter prefix.
	var categoryFilter string
	if strings.HasPrefix(queryLower, "category:") {
		categoryFilter = strings.TrimSpace(queryLower[len("category:"):])
		queryLower = ""
	} else if strings.HasPrefix(queryLower, "integration:") {
		categoryFilter = strings.TrimSpace(queryLower[len("integration:"):])
		queryLower = ""
	}

	var results []scoredTool

	for name, entry := range r.tools {
		// Apply category filter when present.
		if categoryFilter != "" && strings.ToLower(entry.Category) != categoryFilter {
			continue
		}

		aliases := r.canonicalAliases[name]
		score := r.scoreMatch(name, entry, aliases, queryLower)

		// Skip non-matching tools when a text query is active.
		if score == 0 && queryLower != "" {
			continue
		}

		// For empty text queries (including category-only filters), give baseline score.
		if queryLower == "" {
			score = 1
		}

		// Always include the canonical tool entry.
		results = append(results, scoredTool{tool: entry.Tool, score: score})

		// Include alias copies so callers see snake_case names too.
		for _, alias := range aliases {
			results = append(results, scoredTool{
				tool:  cloneToolWithName(entry.Tool, alias),
				score: score,
			})
		}
	}

	// Sort by score descending, then name ascending for stability.
	sort.Slice(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].tool.Name < results[j].tool.Name
	})

	out := make([]*mcp.Tool, len(results))
	for i, s := range results {
		out[i] = s.tool
	}
	return out
}

// scoreMatch computes a relevance score for a tool given a lowercase query.
// Returns 0 when there is no match. Higher scores indicate better matches.
//
// Scoring tiers:
//
//	100 - exact name match (canonical or alias)
//	 80 - name starts with query
//	 60 - name contains query
//	 40 - tag or category match
//	 20 - description contains query
func (r *UnifiedRegistry) scoreMatch(name string, entry ToolEntry, aliases []string, queryLower string) int {
	if queryLower == "" {
		return 0
	}

	nameLower := strings.ToLower(name)

	// Exact name match (canonical).
	if nameLower == queryLower {
		return 100
	}
	// Exact name match (alias).
	for _, alias := range aliases {
		if strings.ToLower(alias) == queryLower {
			return 100
		}
	}

	// Name prefix match.
	best := 0
	if strings.HasPrefix(nameLower, queryLower) {
		best = 80
	}
	for _, alias := range aliases {
		if strings.HasPrefix(strings.ToLower(alias), queryLower) && best < 80 {
			best = 80
		}
	}
	if best > 0 {
		return best
	}

	// Name contains match.
	if strings.Contains(nameLower, queryLower) {
		return 60
	}
	for _, alias := range aliases {
		if strings.Contains(strings.ToLower(alias), queryLower) {
			return 60
		}
	}

	// Category or tag match.
	if strings.Contains(strings.ToLower(entry.Category), queryLower) {
		return 40
	}
	for _, tag := range entry.Tags {
		if strings.Contains(strings.ToLower(tag), queryLower) {
			return 40
		}
	}

	// Description match.
	if strings.Contains(strings.ToLower(entry.Tool.Description), queryLower) {
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

// Categories returns a sorted, deduplicated list of category names present in
// the registry. Empty-category tools are excluded.
func (r *UnifiedRegistry) Categories() []string {
	seen := make(map[string]bool)
	for _, entry := range r.tools {
		if entry.Category != "" {
			seen[entry.Category] = true
		}
	}
	cats := make([]string, 0, len(seen))
	for c := range seen {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	return cats
}

// CountByCategory returns a map of category name to the number of canonical
// tools registered under that category. Tools with no category are counted
// under "uncategorized".
func (r *UnifiedRegistry) CountByCategory() map[string]int {
	counts := make(map[string]int)
	for _, entry := range r.tools {
		cat := entry.Category
		if cat == "" {
			cat = "uncategorized"
		}
		counts[cat]++
	}
	return counts
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
// All mock tools are tagged with category "mock" so they can be filtered out
// of real search results.
func (r *UnifiedRegistry) GenerateMockCatalog() {
	prefixes := []string{"git", "fs", "db", "net", "sys", "cloud", "ai", "test", "build", "deploy"}
	verbs := []string{"read", "write", "list", "query", "exec", "sync", "scan", "analyze", "delete", "create"}

	// Use GetModuleRequest as a placeholder BSR ref for all mock tools.
	toolRef := "buf.build/bufbuild/registry/buf.registry.module.v1.Module:main"

	for i := 0; i < 1000; i++ {
		prefix := prefixes[(i/100)%len(prefixes)]
		verb := verbs[(i/10)%len(verbs)]
		name := fmt.Sprintf("%s_%s_%d", prefix, verb, i)
		desc := fmt.Sprintf("Mock tool for %s operation on %s service (Instance %d)", verb, prefix, i)

		r.RegisterWithCategory(&mcp.Tool{
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
		}, "mock", []string{"mock", "demo", prefix})
	}
}
