package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/bsr"
)

const bsrOwnerFilter = "mcpb"

// maxMatchesPerRepo caps the number of candidate messages extracted from a
// single BSR repository during search. This prevents a single large proto
// package from dominating results and keeps response sizes reasonable for
// AI clients that consume the output in their context window.
const maxMatchesPerRepo = 8

type SearchCandidate struct {
	Owner          string   `json:"owner"`
	Repository     string   `json:"repository"`
	Message        string   `json:"message"`
	Description    string   `json:"description"`
	BsrRef         string   `json:"bsr_ref"`
	Callable       bool     `json:"callable"`
	LocalToolNames []string `json:"local_tool_names,omitempty"`
}

// PopulateDiscoveryTools adds the meta-discovery tools to the registry.
func (r *UnifiedRegistry) PopulateDiscoveryTools() {
	const base = "buf.build/mcpb/discovery"
	searchRef := base + "/tucker.mcproto.discovery.v1.SearchRegistryRequest:main"
	listRef := base + "/tucker.mcproto.discovery.v1.ListToolsRequest:main"

	// search_registry - searches the remote BSR for tool blueprints.
	r.RegisterWithCategory(&mcp.Tool{
		Name:         "search_registry",
		Description:  "Search for tool blueprints in the mcpb registry by keyword. Example queries: 'github', 'jira', 'linear', 'notion', 'analytics'.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: searchRef},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		query := extractSearchQuery(args)
		candidates, err := r.SearchRegistry(ctx, query)
		if err != nil {
			return nil, err
		}

		payload, err := json.Marshal(map[string]interface{}{
			"query":                query,
			"total_count":          len(candidates),
			"categories_available": r.Categories(),
			"candidates":           candidates,
		})
		if err != nil {
			return nil, err
		}

		return &mcp.ToolResult{
			Content: []*mcp.ToolContent{
				{
					Content: &mcp.ToolContent_Text{
						Text: string(payload),
					},
				},
			},
		}, nil
	}, "discovery", []string{"discovery", "search", "meta"})

	// list_tools - lists locally registered tools by category or query.
	r.RegisterWithCategory(&mcp.Tool{
		Name:         "list_tools",
		Description:  "List locally registered tools. Filter by category (e.g. 'jira', 'linear', 'notion', 'github', 'etl', 'mock') or search by keyword. Returns tool names, descriptions, and categories.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: listRef},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		return r.handleListTools(args)
	}, "discovery", []string{"discovery", "list", "meta"})
}

// listToolsRequest is the expected JSON input for the list_tools tool.
type listToolsRequest struct {
	Category string `json:"category"`
	Query    string `json:"query"`
	PageSize int    `json:"page_size"`
	Cursor   string `json:"cursor"`
}

// listToolEntry is a single tool in the list_tools response.
type listToolEntry struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category,omitempty"`
}

func (r *UnifiedRegistry) handleListTools(args []byte) (*mcp.ToolResult, error) {
	var req listToolsRequest
	if len(args) > 0 {
		_ = json.Unmarshal(args, &req)
	}

	// Build the query for List(). Category filter takes precedence.
	query := ""
	if strings.TrimSpace(req.Category) != "" {
		query = "category:" + strings.TrimSpace(req.Category)
	} else if strings.TrimSpace(req.Query) != "" {
		query = strings.TrimSpace(req.Query)
	}

	tools, nextCursor := r.ListPaginated(query, req.PageSize, req.Cursor)

	entries := make([]listToolEntry, 0, len(tools))
	for _, t := range tools {
		cat := ""
		if entry, ok := r.GetTool(t.Name); ok {
			cat = entry.Category
		}
		entries = append(entries, listToolEntry{
			Name:        t.Name,
			Description: t.Description,
			Category:    cat,
		})
	}

	payload, err := json.Marshal(map[string]interface{}{
		"query":                query,
		"total_count":          len(entries),
		"next_cursor":          nextCursor,
		"categories_available": r.Categories(),
		"counts_by_category":   r.CountByCategory(),
		"tools":                entries,
	})
	if err != nil {
		return nil, err
	}

	return &mcp.ToolResult{
		Content: []*mcp.ToolContent{
			{
				Content: &mcp.ToolContent_Text{
					Text: string(payload),
				},
			},
		},
	}, nil
}

func (r *UnifiedRegistry) SearchRegistry(ctx context.Context, query string) ([]SearchCandidate, error) {
	start := time.Now()
	if r.bsrClient == nil {
		return nil, fmt.Errorf("BSR client is not configured")
	}

	query = strings.TrimSpace(query)
	repos, err := r.bsrClient.Search(ctx, query)
	if err != nil {
		log.Printf("registry.search_registry query=%q error=%v duration_ms=%d", query, err, time.Since(start).Milliseconds())
		return nil, err
	}

	toolNamesByRef := r.toolNamesByBsrRef()
	candidates := make([]SearchCandidate, 0, len(repos))
	seen := make(map[string]bool)

	for _, repo := range repos {
		if repo.Owner != bsrOwnerFilter {
			continue
		}

		fds, err := r.bsrClient.FetchDescriptorSet(ctx, &bsr.BSRRef{
			Owner: repo.Owner, Repository: repo.Repository, Version: "main",
		})
		if err != nil {
			log.Printf("registry.search_registry query=%q repo=%s/%s fetch_error=%v", query, repo.Owner, repo.Repository, err)
			continue
		}

		foundCount := 0
		for _, f := range fds.File {
			pkg := f.GetPackage()
			if strings.HasPrefix(pkg, "google.protobuf") ||
				strings.HasPrefix(pkg, "buf.validate") ||
				strings.HasPrefix(pkg, "google.api") {
				continue
			}

			for _, mt := range f.MessageType {
				name := mt.GetName()
				isLikelyTool := strings.Contains(name, "Request") ||
					strings.Contains(name, "Event") ||
					strings.Contains(name, "Task") ||
					strings.Contains(name, "Call")

				if !isLikelyTool || foundCount >= maxMatchesPerRepo {
					continue
				}

				fullName := fmt.Sprintf("%s.%s", pkg, name)
				bsrRef := fmt.Sprintf(
					"buf.build/%s/%s/%s:main",
					repo.Owner,
					repo.Repository,
					fullName,
				)

				localNames := toolNamesByRef[bsrRef]
				candidate := SearchCandidate{
					Owner:       repo.Owner,
					Repository:  repo.Repository,
					Message:     fullName,
					Description: descriptionFromMessageName(name),
					BsrRef:      bsrRef,
					Callable:    len(localNames) > 0,
				}
				if len(localNames) > 0 {
					candidate.LocalToolNames = localNames
				}

				if !seen[bsrRef] {
					candidates = append(candidates, candidate)
					seen[bsrRef] = true
				}
				foundCount++
			}
		}
	}

	log.Printf("registry.search_registry query=%q repos_searched=%d candidates=%d duration_ms=%d",
		query, len(repos), len(candidates), time.Since(start).Milliseconds())
	return candidates, nil
}

// extractSearchQuery parses the query string from tool arguments.
// Accepts JSON {"query": "..."} or returns empty string if no query provided.
func extractSearchQuery(args []byte) string {
	if len(args) == 0 {
		return ""
	}

	var payload struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &payload); err == nil {
		return strings.TrimSpace(payload.Query)
	}

	// Graceful fallback: treat raw bytes as a plain string query.
	return strings.TrimSpace(string(args))
}

func (r *UnifiedRegistry) toolNamesByBsrRef() map[string][]string {
	names := make(map[string][]string)
	for name, entry := range r.tools {
		bsrRef := toolBsrRef(entry.Tool)
		if bsrRef == "" {
			continue
		}
		names[bsrRef] = append(names[bsrRef], name)
	}
	return names
}

// descriptionFromMessageName derives a human-readable description from a proto
// message name by splitting on camelCase boundaries and stripping common
// suffixes like "Request", "Event", "Task", "Call". For example,
// "SearchIssuesRequest" becomes "Search issues".
func descriptionFromMessageName(name string) string {
	// Split camelCase into words.
	var words []string
	start := 0
	for i := 1; i < len(name); i++ {
		if name[i] >= 'A' && name[i] <= 'Z' {
			words = append(words, name[start:i])
			start = i
		}
	}
	words = append(words, name[start:])

	// Strip trailing suffix words that are proto conventions, not semantics.
	suffixes := map[string]bool{"Request": true, "Event": true, "Task": true, "Call": true}
	if len(words) > 1 && suffixes[words[len(words)-1]] {
		words = words[:len(words)-1]
	}

	if len(words) == 0 {
		return name
	}

	// Lowercase all words except the first, then join.
	result := words[0]
	for _, w := range words[1:] {
		result += " " + strings.ToLower(w)
	}
	return result
}
