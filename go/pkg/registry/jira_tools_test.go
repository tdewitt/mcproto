package registry

import (
	"context"
	"net/http"
	"testing"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/jira"
)

func TestPopulateJiraTools_DoesNotOverwriteExistingToolNames(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	reg.Register(&mcp.Tool{Name: "CreateIssue"}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		return mcpText("github"), nil
	})

	client, err := jira.NewClientWithConfig("https://jira.example", "user@example.com", "token", &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	if err := reg.PopulateJiraTools(client); err != nil {
		t.Fatalf("PopulateJiraTools failed: %v", err)
	}

	if _, ok := reg.tools["CreateIssue"]; !ok {
		t.Fatalf("expected existing CreateIssue tool to remain registered")
	}
	if _, ok := reg.tools["JiraCreateIssue"]; !ok {
		t.Fatalf("expected JiraCreateIssue tool to be registered")
	}
}
