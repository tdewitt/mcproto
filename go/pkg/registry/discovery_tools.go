package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/bsr"
)

// PopulateDiscoveryTools adds the meta-discovery tool to the registry.
func (r *UnifiedRegistry) PopulateDiscoveryTools() {
	const base = "buf.build/mcpb/discovery"
	searchRef := base + "/misfit.discovery.v1.SearchRegistryRequest:main"

	r.Register(&mcp.Tool{
		Name:        "search_registry",
		Description: "Search for available tool blueprints in the global registry by keyword.",
		SchemaSource: &mcp.Tool_BsrRef{BsrRef: searchRef},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		// --- DEMO-SPECIFIC HEURISTIC START ---
		// NOTE: In a production implementation, we would properly unpack 'args' using
		// the BSR Registry to get the 'query' field. For this spike demo, we are 
		// using a basic heuristic to extract the query string from the raw bytes.
		query := "analytics"
		if len(args) > 2 {
			query = string(args[2:])
		}
		// --- DEMO-SPECIFIC HEURISTIC END ---

		repos, err := r.bsrClient.Search(ctx, query)
		if err != nil {
			return nil, err
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Global Search Results for '%s':\n", query))
		
		for i, repo := range repos {
			// --- NOISE FILTER START ---
			// DEMO ONLY: We are manually filtering and ranking results to improve AI relevance.
			// In a real system, this would be handled by:
			// 1. Semantic search ranking on the BSR side.
			// 2. Trust-root configuration (e.g., only searching specific orgs).
			// 3. Custom Protobuf Options to explicitly mark 'Tool' messages.
			
			sb.WriteString(fmt.Sprintf("%d. Repository: buf.build/%s/%s\n", i+1, repo.Owner, repo.Repository))
			
			fds, err := r.bsrClient.FetchDescriptorSet(ctx, &bsr.BSRRef{
				Owner: repo.Owner, Repository: repo.Repository, Version: "main",
			})
			
			if err == nil {
				foundCount := 0
				sb.WriteString("   Top Tool Blueprints:\n")
				for _, f := range fds.File {
					pkg := f.GetPackage()
					
					// FILTER: Exclude standard library dependencies which are 'noise' for tool discovery
				if strings.HasPrefix(pkg, "google.protobuf") || 
				   strings.HasPrefix(pkg, "buf.validate") || 
				   strings.HasPrefix(pkg, "google.api") {
						continue
					}

					for _, mt := range f.MessageType {
						name := mt.GetName()
						
						// RANKING HEURISTIC: Prioritize messages that look like Actions or Events.
						// We look for common patterns in tool-centric Protobuf design.
						isLikelyTool := strings.Contains(name, "Request") || 
								strings.Contains(name, "Event") || 
								strings.Contains(name, "Task") || 
								strings.Contains(name, "Call")

						if isLikelyTool && foundCount < 3 {
							fullName := fmt.Sprintf("%s.%s", pkg, name)
							sb.WriteString(fmt.Sprintf("   - %s (ref: buf.build/%s/%s/%s:main)\n", name, repo.Owner, repo.Repository, fullName))
							foundCount++
						}
					}
				}
				if foundCount == 0 {
					sb.WriteString("   (No high-confidence tool blueprints found in this repository)\n")
				}
			}
			sb.WriteString("\n")
			// --- NOISE FILTER END ---
		}

		return &mcp.ToolResult{
			Content: []*mcp.ToolContent{
				{
					Content: &mcp.ToolContent_Text{
						Text: sb.String(),
					},
				},
			},
		}, nil
	})
}