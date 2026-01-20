package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/bsr"
)

const bsrOwnerFilter = "mcpb"

type SearchCandidate struct {
	Owner          string   `json:"owner"`
	Repository     string   `json:"repository"`
	Message        string   `json:"message"`
	BsrRef         string   `json:"bsr_ref"`
	LocalToolNames []string `json:"local_tool_names,omitempty"`
}

// PopulateDiscoveryTools adds the meta-discovery tool to the registry.
func (r *UnifiedRegistry) PopulateDiscoveryTools() {
	const base = "buf.build/mcpb/discovery"
	searchRef := base + "/misfit.discovery.v1.SearchRegistryRequest:main"

	r.Register(&mcp.Tool{
		Name:         "search_registry",
		Description:  "Search for available tool blueprints in the mcpb registry by keyword.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: searchRef},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		query := extractSearchQuery(args)
		candidates, err := r.SearchRegistry(ctx, query)
		if err != nil {
			return nil, err
		}

		payload, err := json.Marshal(map[string]interface{}{
			"query":      query,
			"candidates": candidates,
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
	})
}

func (r *UnifiedRegistry) SearchRegistry(ctx context.Context, query string) ([]SearchCandidate, error) {
	if r.bsrClient == nil {
		return nil, fmt.Errorf("BSR client is not configured")
	}

	repos, err := r.bsrClient.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	toolNamesByRef := r.toolNamesByBsrRef()
	candidates := make([]SearchCandidate, 0, len(repos))

	for _, repo := range repos {
		if repo.Owner != bsrOwnerFilter {
			continue
		}

		fds, err := r.bsrClient.FetchDescriptorSet(ctx, &bsr.BSRRef{
			Owner: repo.Owner, Repository: repo.Repository, Version: "main",
		})
		if err != nil {
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

				if !isLikelyTool || foundCount >= 3 {
					continue
				}

				fullName := fmt.Sprintf("%s.%s", pkg, name)
				bsrRef := fmt.Sprintf(
					"buf.build/%s/%s/%s:main",
					repo.Owner,
					repo.Repository,
					fullName,
				)

				candidate := SearchCandidate{
					Owner:      repo.Owner,
					Repository: repo.Repository,
					Message:    fullName,
					BsrRef:     bsrRef,
				}
				if names := toolNamesByRef[bsrRef]; len(names) > 0 {
					candidate.LocalToolNames = names
				}

				candidates = append(candidates, candidate)
				foundCount++
			}
		}
	}

	return candidates, nil
}

func extractSearchQuery(args []byte) string {
	const fallback = "analytics"
	if len(args) == 0 {
		return fallback
	}

	var payload struct {
		Query string `json:"query"`
	}
	if args[0] == '{' {
		if err := json.Unmarshal(args, &payload); err == nil && payload.Query != "" {
			return payload.Query
		}
	}

	if len(args) > 2 {
		query := strings.TrimSpace(string(args[2:]))
		if query != "" {
			return query
		}
	}

	return fallback
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
