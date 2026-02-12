package registry

import (
	"context"
	"testing"

	"github.com/misfitdev/proto-mcp/go/mcp"
)

func TestRegisterAliasRoutesCalls(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	tool := &mcp.Tool{
		Name:        "CreateIssue",
		Description: "Create a GitHub issue.",
		SchemaSource: &mcp.Tool_BsrRef{
			BsrRef: "buf.build/misfitdev/github/tucker.mcproto.github.v1.CreateIssueRequest:v1",
		},
	}

	reg.Register(tool, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		return &mcp.ToolResult{
			Content: []*mcp.ToolContent{
				{
					Content: &mcp.ToolContent_Text{
						Text: "ok",
					},
				},
			},
		}, nil
	})

	for _, name := range []string{"CreateIssue", "create_issue"} {
		resp, err := reg.Call(context.Background(), name, nil)
		if err != nil {
			t.Fatalf("Call failed for %s: %v", name, err)
		}
		if resp == nil || len(resp.Content) == 0 || resp.Content[0].GetText() != "ok" {
			t.Fatalf("Unexpected response for %s: %#v", name, resp)
		}
	}
}

func TestListUsesCanonicalName(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	tool := &mcp.Tool{
		Name:        "CreateIssue",
		Description: "Create a GitHub issue.",
		SchemaSource: &mcp.Tool_BsrRef{
			BsrRef: "buf.build/misfitdev/github/tucker.mcproto.github.v1.CreateIssueRequest:v1",
		},
	}
	reg.Register(tool, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		return &mcp.ToolResult{}, nil
	})

	tools := reg.List("")
	// List now returns both the canonical tool and alias clones.
	// CreateIssue auto-registers alias create_issue, so we expect 2 entries.
	if len(tools) != 2 {
		t.Fatalf("Expected 2 tools (canonical + alias), got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tool := range tools {
		names[tool.Name] = true
	}
	if !names["CreateIssue"] {
		t.Fatal("Expected canonical name CreateIssue in results")
	}
	if !names["create_issue"] {
		t.Fatal("Expected alias create_issue in results")
	}
}
