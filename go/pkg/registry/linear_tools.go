package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/linear"
	"google.golang.org/protobuf/proto"
)

const linearBsrBase = "buf.build/mcpb/linear/tucker.mcproto.linear.v1."
const linearBsrVersion = "main"

func (r *UnifiedRegistry) PopulateLinearTools(client *linear.Client) error {
	if client == nil {
		return fmt.Errorf("linear client is nil")
	}

	tools := []struct {
		name        string
		description string
		bsrRef      string
		handler     ToolHandler
	}{
		{
			name:        "LinearListIssues",
			description: "List Linear issues with optional filters for team, assignee, state, label, project, and cycle.",
			bsrRef:      linearBsrBase + "ListIssuesRequest:" + linearBsrVersion,
			handler:     makeLinearListIssuesHandler(client),
		},
		{
			name:        "LinearGetIssue",
			description: "Get a Linear issue by ID or identifier (e.g., ENG-123).",
			bsrRef:      linearBsrBase + "GetIssueRequest:" + linearBsrVersion,
			handler:     makeLinearGetIssueHandler(client),
		},
		{
			name:        "LinearCreateIssue",
			description: "Create a new issue in a Linear team.",
			bsrRef:      linearBsrBase + "CreateIssueRequest:" + linearBsrVersion,
			handler:     makeLinearCreateIssueHandler(client),
		},
		{
			name:        "LinearUpdateIssue",
			description: "Update fields on an existing Linear issue.",
			bsrRef:      linearBsrBase + "UpdateIssueRequest:" + linearBsrVersion,
			handler:     makeLinearUpdateIssueHandler(client),
		},
		{
			name:        "LinearDeleteIssue",
			description: "Archive/delete a Linear issue.",
			bsrRef:      linearBsrBase + "DeleteIssueRequest:" + linearBsrVersion,
			handler:     makeLinearDeleteIssueHandler(client),
		},
		{
			name:        "LinearSearchIssues",
			description: "Full-text search Linear issues.",
			bsrRef:      linearBsrBase + "SearchIssuesRequest:" + linearBsrVersion,
			handler:     makeLinearSearchIssuesHandler(client),
		},
		{
			name:        "LinearListProjects",
			description: "List Linear projects.",
			bsrRef:      linearBsrBase + "ListProjectsRequest:" + linearBsrVersion,
			handler:     makeLinearListProjectsHandler(client),
		},
		{
			name:        "LinearGetProject",
			description: "Get a Linear project by ID.",
			bsrRef:      linearBsrBase + "GetProjectRequest:" + linearBsrVersion,
			handler:     makeLinearGetProjectHandler(client),
		},
		{
			name:        "LinearCreateProjectUpdate",
			description: "Post a status update to a Linear project.",
			bsrRef:      linearBsrBase + "CreateProjectUpdateRequest:" + linearBsrVersion,
			handler:     makeLinearCreateProjectUpdateHandler(client),
		},
		{
			name:        "LinearListProjectUpdates",
			description: "List status updates for a Linear project.",
			bsrRef:      linearBsrBase + "ListProjectUpdatesRequest:" + linearBsrVersion,
			handler:     makeLinearListProjectUpdatesHandler(client),
		},
		{
			name:        "LinearListTeams",
			description: "List Linear teams in the workspace.",
			bsrRef:      linearBsrBase + "ListTeamsRequest:" + linearBsrVersion,
			handler:     makeLinearListTeamsHandler(client),
		},
		{
			name:        "LinearListMembers",
			description: "List workspace members in Linear.",
			bsrRef:      linearBsrBase + "ListMembersRequest:" + linearBsrVersion,
			handler:     makeLinearListMembersHandler(client),
		},
		{
			name:        "LinearAddComment",
			description: "Add a comment to a Linear issue.",
			bsrRef:      linearBsrBase + "AddCommentRequest:" + linearBsrVersion,
			handler:     makeLinearAddCommentHandler(client),
		},
		{
			name:        "LinearListComments",
			description: "List comments on a Linear issue.",
			bsrRef:      linearBsrBase + "ListCommentsRequest:" + linearBsrVersion,
			handler:     makeLinearListCommentsHandler(client),
		},
		{
			name:        "LinearListLabels",
			description: "List issue labels in Linear, optionally filtered by team.",
			bsrRef:      linearBsrBase + "ListLabelsRequest:" + linearBsrVersion,
			handler:     makeLinearListLabelsHandler(client),
		},
		{
			name:        "LinearListCycles",
			description: "List cycles for a Linear team.",
			bsrRef:      linearBsrBase + "ListCyclesRequest:" + linearBsrVersion,
			handler:     makeLinearListCyclesHandler(client),
		},
		{
			name:        "LinearListDocuments",
			description: "List documents in Linear.",
			bsrRef:      linearBsrBase + "ListDocumentsRequest:" + linearBsrVersion,
			handler:     makeLinearListDocumentsHandler(client),
		},
		{
			name:        "LinearGetDocument",
			description: "Get a Linear document by ID.",
			bsrRef:      linearBsrBase + "GetDocumentRequest:" + linearBsrVersion,
			handler:     makeLinearGetDocumentHandler(client),
		},
		{
			name:        "LinearSearchDocuments",
			description: "Search Linear documents by query string.",
			bsrRef:      linearBsrBase + "SearchDocumentsRequest:" + linearBsrVersion,
			handler:     makeLinearSearchDocumentsHandler(client),
		},
		{
			name:        "LinearListWorkflowStates",
			description: "List workflow states for a Linear team.",
			bsrRef:      linearBsrBase + "ListWorkflowStatesRequest:" + linearBsrVersion,
			handler:     makeLinearListWorkflowStatesHandler(client),
		},
		{
			name:        "LinearGetViewer",
			description: "Get the authenticated Linear user's profile.",
			bsrRef:      linearBsrBase + "GetViewerRequest:" + linearBsrVersion,
			handler:     makeLinearGetViewerHandler(client),
		},
	}

	for _, t := range tools {
		r.Register(&mcp.Tool{
			Name:        t.name,
			Description: t.description,
			SchemaSource: &mcp.Tool_BsrRef{
				BsrRef: t.bsrRef,
			},
		}, t.handler)
	}

	return nil
}

func makeLinearListIssuesHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListIssuesRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.ListIssues(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearGetIssueHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.GetIssueRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetId()) == "" {
			return nil, fmt.Errorf("id is required")
		}
		resp, err := client.GetIssue(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearCreateIssueHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.CreateIssueRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetTeamId()) == "" {
			return nil, fmt.Errorf("team_id is required")
		}
		if strings.TrimSpace(req.GetTitle()) == "" {
			return nil, fmt.Errorf("title is required")
		}
		resp, err := client.CreateIssue(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearUpdateIssueHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.UpdateIssueRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetId()) == "" {
			return nil, fmt.Errorf("id is required")
		}
		resp, err := client.UpdateIssue(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearDeleteIssueHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.DeleteIssueRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetId()) == "" {
			return nil, fmt.Errorf("id is required")
		}
		resp, err := client.DeleteIssue(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearSearchIssuesHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.SearchIssuesRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetQuery()) == "" {
			return nil, fmt.Errorf("query is required")
		}
		resp, err := client.SearchIssues(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearListProjectsHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListProjectsRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.ListProjects(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearGetProjectHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.GetProjectRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetId()) == "" {
			return nil, fmt.Errorf("id is required")
		}
		resp, err := client.GetProject(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearCreateProjectUpdateHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.CreateProjectUpdateRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetProjectId()) == "" {
			return nil, fmt.Errorf("project_id is required")
		}
		if strings.TrimSpace(req.GetBody()) == "" {
			return nil, fmt.Errorf("body is required")
		}
		resp, err := client.CreateProjectUpdate(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearListProjectUpdatesHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListProjectUpdatesRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetProjectId()) == "" {
			return nil, fmt.Errorf("project_id is required")
		}
		resp, err := client.ListProjectUpdates(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearListTeamsHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListTeamsRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.ListTeams(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearListMembersHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListMembersRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.ListMembers(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearAddCommentHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.AddCommentRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueId()) == "" {
			return nil, fmt.Errorf("issue_id is required")
		}
		if strings.TrimSpace(req.GetBody()) == "" {
			return nil, fmt.Errorf("body is required")
		}
		resp, err := client.AddComment(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearListCommentsHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListCommentsRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueId()) == "" {
			return nil, fmt.Errorf("issue_id is required")
		}
		resp, err := client.ListComments(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearListLabelsHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListLabelsRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.ListLabels(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearListCyclesHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListCyclesRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		// team_id is optional; when omitted, lists cycles across all teams
		resp, err := client.ListCycles(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearListDocumentsHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListDocumentsRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.ListDocuments(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearGetDocumentHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.GetDocumentRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetId()) == "" {
			return nil, fmt.Errorf("id is required")
		}
		resp, err := client.GetDocument(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearSearchDocumentsHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.SearchDocumentsRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetQuery()) == "" {
			return nil, fmt.Errorf("query is required")
		}
		resp, err := client.SearchDocuments(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearListWorkflowStatesHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.ListWorkflowStatesRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		// team_id is optional; when omitted, lists workflow states across all teams
		resp, err := client.ListWorkflowStates(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeLinearGetViewerHandler(client *linear.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req linear.GetViewerRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.GetViewer(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}
