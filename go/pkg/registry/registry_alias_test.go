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
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}
	// After deduplication fix, List returns canonical name, not alias
	if tools[0].Name != "CreateIssue" {
		t.Fatalf("Expected canonical name CreateIssue, got %s", tools[0].Name)
	}
}
