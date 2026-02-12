package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/misfitdev/proto-mcp/go/mcp"
)

func noopHandler(_ context.Context, _ []byte) (*mcp.ToolResult, error) {
	return &mcp.ToolResult{}, nil
}

// registerN adds n tools named "tool_000" .. "tool_{n-1}" to the registry.
// All tools are lowercase so they produce no aliases (snake_case == original).
func registerN(reg *UnifiedRegistry, n int) {
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("tool_%03d", i)
		reg.RegisterWithCategory(&mcp.Tool{
			Name:        name,
			Description: fmt.Sprintf("Test tool %d", i),
		}, noopHandler, "test", nil)
	}
}

func TestListPaginated_DefaultPageSize(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	registerN(reg, 120)

	tools, nextCursor := reg.ListPaginated("", 0, "")
	if len(tools) != DefaultPageSize {
		t.Fatalf("expected %d tools on first page, got %d", DefaultPageSize, len(tools))
	}
	if nextCursor == "" {
		t.Fatal("expected a non-empty nextCursor when more results exist")
	}
}

func TestListPaginated_CustomPageSize(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	registerN(reg, 20)

	tools, nextCursor := reg.ListPaginated("", 10, "")
	if len(tools) != 10 {
		t.Fatalf("expected 10 tools, got %d", len(tools))
	}
	if nextCursor == "" {
		t.Fatal("expected non-empty nextCursor when 20 > 10")
	}
}

func TestListPaginated_WalkAllPages(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	total := 73
	registerN(reg, total)

	pageSize := 25
	var allNames []string
	cursor := ""

	for {
		tools, next := reg.ListPaginated("", pageSize, cursor)
		for _, tool := range tools {
			allNames = append(allNames, tool.Name)
		}

		if next == "" {
			break
		}
		cursor = next
	}

	if len(allNames) != total {
		t.Fatalf("expected %d total tools across all pages, got %d", total, len(allNames))
	}

	// Verify no duplicates.
	seen := make(map[string]bool)
	for _, name := range allNames {
		if seen[name] {
			t.Fatalf("duplicate tool name across pages: %s", name)
		}
		seen[name] = true
	}
}

func TestListPaginated_ExactFit(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	registerN(reg, 10)

	tools, nextCursor := reg.ListPaginated("", 10, "")
	if len(tools) != 10 {
		t.Fatalf("expected 10 tools, got %d", len(tools))
	}
	if nextCursor != "" {
		t.Fatalf("expected empty nextCursor when results exactly fill the page, got %q", nextCursor)
	}
}

func TestListPaginated_EmptyRegistry(t *testing.T) {
	reg := NewUnifiedRegistry(nil)

	tools, nextCursor := reg.ListPaginated("", 10, "")
	if len(tools) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(tools))
	}
	if nextCursor != "" {
		t.Fatalf("expected empty nextCursor for empty registry, got %q", nextCursor)
	}
}

func TestListPaginated_InvalidCursor(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	registerN(reg, 10)

	// A cursor that does not match any tool name returns from the beginning.
	tools, _ := reg.ListPaginated("", 5, "nonexistent_cursor")
	if len(tools) != 5 {
		t.Fatalf("expected 5 tools (fallback to start), got %d", len(tools))
	}
}

func TestListPaginated_LastPagePartial(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	registerN(reg, 7)

	// First page: 5 tools.
	page1, cursor := reg.ListPaginated("", 5, "")
	if len(page1) != 5 {
		t.Fatalf("expected 5 tools on first page, got %d", len(page1))
	}
	if cursor == "" {
		t.Fatal("expected non-empty cursor after first page")
	}

	// Second page: remaining 2 tools.
	page2, cursor2 := reg.ListPaginated("", 5, cursor)
	if len(page2) != 2 {
		t.Fatalf("expected 2 tools on second page, got %d", len(page2))
	}
	if cursor2 != "" {
		t.Fatalf("expected empty cursor on last page, got %q", cursor2)
	}
}

func TestListPaginated_WithQuery(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	// Register some tools with specific names.
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("alpha_%d", i)
		reg.RegisterWithCategory(&mcp.Tool{
			Name:        name,
			Description: "Alpha tool",
		}, noopHandler, "test", nil)
	}
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("beta_%d", i)
		reg.RegisterWithCategory(&mcp.Tool{
			Name:        name,
			Description: "Beta tool",
		}, noopHandler, "test", nil)
	}

	// Search for "alpha" -- should return only the 10 alpha tools.
	tools, nextCursor := reg.ListPaginated("alpha", 5, "")
	if len(tools) != 5 {
		t.Fatalf("expected 5 alpha tools on first page, got %d", len(tools))
	}
	if nextCursor == "" {
		t.Fatal("expected non-empty cursor")
	}

	tools2, nextCursor2 := reg.ListPaginated("alpha", 5, nextCursor)
	if len(tools2) != 5 {
		t.Fatalf("expected 5 alpha tools on second page, got %d", len(tools2))
	}
	if nextCursor2 != "" {
		t.Fatalf("expected empty cursor on last page, got %q", nextCursor2)
	}
}

func TestHandleListTools_Paginated(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	registerN(reg, 100)

	// Request page_size=10, no cursor.
	args := []byte(`{"page_size": 10}`)
	result, err := reg.handleListTools(args)
	if err != nil {
		t.Fatalf("handleListTools failed: %v", err)
	}

	var resp struct {
		TotalCount int    `json:"total_count"`
		NextCursor string `json:"next_cursor"`
		Tools      []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].GetText()), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.TotalCount != 10 {
		t.Fatalf("expected total_count=10, got %d", resp.TotalCount)
	}
	if resp.NextCursor == "" {
		t.Fatal("expected non-empty next_cursor")
	}

	// Request second page using cursor from first.
	args2 := []byte(fmt.Sprintf(`{"page_size": 10, "cursor": %q}`, resp.NextCursor))
	result2, err := reg.handleListTools(args2)
	if err != nil {
		t.Fatalf("handleListTools page 2 failed: %v", err)
	}

	var resp2 struct {
		TotalCount int    `json:"total_count"`
		NextCursor string `json:"next_cursor"`
		Tools      []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal([]byte(result2.Content[0].GetText()), &resp2); err != nil {
		t.Fatalf("failed to unmarshal page 2 response: %v", err)
	}

	if resp2.TotalCount != 10 {
		t.Fatalf("expected total_count=10 on page 2, got %d", resp2.TotalCount)
	}

	// Verify no overlap between pages.
	page1Names := make(map[string]bool)
	for _, t := range resp.Tools {
		page1Names[t.Name] = true
	}
	for _, tool := range resp2.Tools {
		if page1Names[tool.Name] {
			t.Fatalf("tool %q appeared on both page 1 and page 2", tool.Name)
		}
	}
}

func TestHandleListTools_DefaultPageSize(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	registerN(reg, 100)

	// No page_size specified; should use DefaultPageSize (50).
	args := []byte(`{}`)
	result, err := reg.handleListTools(args)
	if err != nil {
		t.Fatalf("handleListTools failed: %v", err)
	}

	var resp struct {
		TotalCount int `json:"total_count"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].GetText()), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.TotalCount != DefaultPageSize {
		t.Fatalf("expected total_count=%d (DefaultPageSize), got %d", DefaultPageSize, resp.TotalCount)
	}
}
