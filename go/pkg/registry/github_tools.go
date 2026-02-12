package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/misfitdev/proto-mcp/go/mcp"
	ghpb "github.com/misfitdev/proto-mcp/go/pkg/github"
	"google.golang.org/protobuf/proto"
)

const githubBsrBase = "buf.build/mcpb/github/tucker.mcproto.github.v1."
const githubBsrVersion = "main"

// PopulateGitHubTools registers a minimal GitHub tool surface backed by the real GitHub API.
// It intentionally focuses on discovery-relevant operations for the demo.
func (r *UnifiedRegistry) PopulateGitHubTools(s *ghpb.Server) {
	if s == nil {
		return
	}

	// SearchRepositories
	r.RegisterWithCategory(&mcp.Tool{
		Name:        "SearchRepositories",
		Description: "Search GitHub repositories (backed by go-github).",
		SchemaSource: &mcp.Tool_BsrRef{
			BsrRef: githubBsrBase + "SearchRepositoriesRequest:" + githubBsrVersion,
		},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		req := &ghpb.SearchRepositoriesRequest{}
		if err := proto.Unmarshal(args, req); err != nil {
			return nil, fmt.Errorf("invalid SearchRepositoriesRequest: %w", err)
		}

		resp, err := s.SearchRepositories(ctx, req)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		b.WriteString("Search results:\n")
		for i, repo := range resp.Repositories {
			if i >= 5 {
				break
			}
			b.WriteString(fmt.Sprintf("- %s (%s): %s\n", repo.GetFullName(), repo.GetHtmlUrl(), strings.TrimSpace(repo.GetDescription())))
		}

		return mcpText(strings.TrimSpace(b.String())), nil
	}, "github", []string{"github", "source-control"})

	// CreateIssue
	r.RegisterWithCategory(&mcp.Tool{
		Name:        "CreateIssue",
		Description: "Create a GitHub issue in a repository (requires GITHUB_PERSONAL_ACCESS_TOKEN).",
		SchemaSource: &mcp.Tool_BsrRef{
			BsrRef: githubBsrBase + "CreateIssueRequest:" + githubBsrVersion,
		},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		req := &ghpb.CreateIssueRequest{}
		if err := proto.Unmarshal(args, req); err != nil {
			return nil, fmt.Errorf("invalid CreateIssueRequest: %w", err)
		}

		title := req.GetTitle()
		if strings.TrimSpace(title) == "" {
			title = "proto-mcp issue (auto)"
		}
		req.Title = title

		resp, err := s.CreateIssue(ctx, req)
		if err != nil {
			return nil, err
		}

		issue := resp.GetIssue()
		output := fmt.Sprintf("Created issue #%d: %s\nURL: %s\nTitle: %s\nBody: %s",
			issue.GetNumber(),
			issue.GetHtmlUrl(),
			issue.GetHtmlUrl(),
			issue.GetTitle(),
			strings.TrimSpace(issue.GetBody()),
		)
		return mcpText(output), nil
	}, "github", []string{"github", "source-control"})

	// ListIssues
	r.RegisterWithCategory(&mcp.Tool{
		Name:        "ListIssues",
		Description: "List issues in a GitHub repository with optional filtering by state, sort, and direction.",
		SchemaSource: &mcp.Tool_BsrRef{
			BsrRef: githubBsrBase + "ListIssuesRequest:" + githubBsrVersion,
		},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		req := &ghpb.ListIssuesRequest{}
		if err := proto.Unmarshal(args, req); err != nil {
			return nil, fmt.Errorf("invalid ListIssuesRequest: %w", err)
		}

		resp, err := s.ListIssues(ctx, req)
		if err != nil {
			return nil, err
		}

		var b strings.Builder
		b.WriteString(fmt.Sprintf("Issues for %s/%s:\n", req.GetOwner(), req.GetRepo()))
		for _, issue := range resp.GetIssues() {
			b.WriteString(fmt.Sprintf("- #%d [%s] %s (%s)\n",
				issue.GetNumber(),
				issue.GetState(),
				issue.GetTitle(),
				issue.GetHtmlUrl(),
			))
		}
		if len(resp.GetIssues()) == 0 {
			b.WriteString("No issues found.")
		}

		return mcpText(strings.TrimSpace(b.String())), nil
	}, "github", []string{"github", "source-control"})

	// CreateOrUpdateFile
	r.RegisterWithCategory(&mcp.Tool{
		Name:        "CreateOrUpdateFile",
		Description: "Create or update a file in a GitHub repository. Provide sha to update an existing file.",
		SchemaSource: &mcp.Tool_BsrRef{
			BsrRef: githubBsrBase + "CreateOrUpdateFileRequest:" + githubBsrVersion,
		},
	}, func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		req := &ghpb.CreateOrUpdateFileRequest{}
		if err := proto.Unmarshal(args, req); err != nil {
			return nil, fmt.Errorf("invalid CreateOrUpdateFileRequest: %w", err)
		}

		resp, err := s.CreateOrUpdateFile(ctx, req)
		if err != nil {
			return nil, err
		}

		content := resp.GetContent()
		commit := resp.GetCommit()
		output := fmt.Sprintf("File committed: %s\nPath: %s\nSHA: %s\nCommit: %s\nURL: %s",
			content.GetName(),
			content.GetPath(),
			content.GetSha(),
			commit.GetSha(),
			content.GetHtmlUrl(),
		)
		return mcpText(output), nil
	}, "github", []string{"github", "source-control"})
}

func mcpText(text string) *mcp.ToolResult {
	return &mcp.ToolResult{
		Content: []*mcp.ToolContent{
			{
				Content: &mcp.ToolContent_Text{
					Text: text,
				},
			},
		},
	}
}
