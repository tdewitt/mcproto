package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/jira"
	"google.golang.org/protobuf/proto"
)

const jiraBsrBase = "buf.build/mcpb/jira/tucker.mcproto.jira.v1."
const jiraBsrVersion = "main"

func (r *UnifiedRegistry) PopulateJiraTools(client *jira.Client) error {
	if client == nil {
		return fmt.Errorf("jira client is nil")
	}

	tools := []struct {
		name        string
		description string
		bsrRef      string
		handler     ToolHandler
		metadata    map[string]string
	}{
		{
			name:        "JiraSearchIssues",
			description: "Search Jira issues using JQL. Returns matching issues with selected fields.",
			bsrRef:      jiraBsrBase + "SearchIssuesRequest:" + jiraBsrVersion,
			handler:     makeSearchIssuesHandler(client),
			metadata: map[string]string{
				"category":     "Jira",
				"integration":  "issue-tracking",
				"capabilities": "search,read",
			},
		},
		{
			name:        "JiraGetIssue",
			description: "Get details of a specific Jira issue by key (e.g., PROJ-123).",
			bsrRef:      jiraBsrBase + "GetIssueRequest:" + jiraBsrVersion,
			handler:     makeGetIssueHandler(client),
		},
		{
			name:        "JiraCreateIssue",
			description: "Create a new Jira issue in a project with specified type, summary, and fields.",
			bsrRef:      jiraBsrBase + "CreateIssueRequest:" + jiraBsrVersion,
			handler:     makeCreateIssueHandler(client),
		},
		{
			name:        "JiraUpdateIssue",
			description: "Update fields on an existing Jira issue.",
			bsrRef:      jiraBsrBase + "UpdateIssueRequest:" + jiraBsrVersion,
			handler:     makeUpdateIssueHandler(client),
		},
		{
			name:        "JiraTransitionIssue",
			description: "Transition a Jira issue to a new status. Use GetTransitions first to get available transition IDs.",
			bsrRef:      jiraBsrBase + "TransitionIssueRequest:" + jiraBsrVersion,
			handler:     makeTransitionIssueHandler(client),
		},
		{
			name:        "JiraAddComment",
			description: "Add a comment to a Jira issue.",
			bsrRef:      jiraBsrBase + "AddCommentRequest:" + jiraBsrVersion,
			handler:     makeAddCommentHandler(client),
		},
		{
			name:        "JiraAssignIssue",
			description: "Assign a Jira issue to a user by account ID. Use SearchUsers to find account IDs.",
			bsrRef:      jiraBsrBase + "AssignIssueRequest:" + jiraBsrVersion,
			handler:     makeAssignIssueHandler(client),
		},
		{
			name:        "JiraGetTransitions",
			description: "Get available status transitions for an issue. Returns transition IDs needed for TransitionIssue.",
			bsrRef:      jiraBsrBase + "GetTransitionsRequest:" + jiraBsrVersion,
			handler:     makeGetTransitionsHandler(client),
		},
		{
			name:        "JiraSearchUsers",
			description: "Search for Jira users by name or email. Returns account IDs for assignment.",
			bsrRef:      jiraBsrBase + "SearchUsersRequest:" + jiraBsrVersion,
			handler:     makeSearchUsersHandler(client),
		},
		{
			name:        "JsmGetServiceDesks",
			description: "List Jira Service Management service desks visible to the authenticated user.",
			bsrRef:      jiraBsrBase + "GetServiceDesksRequest:" + jiraBsrVersion,
			handler:     makeGetServiceDesksHandler(client),
		},
		{
			name:        "JsmGetRequestTypes",
			description: "List request types for a service desk in Jira Service Management.",
			bsrRef:      jiraBsrBase + "GetRequestTypesRequest:" + jiraBsrVersion,
			handler:     makeGetRequestTypesHandler(client),
		},
		{
			name:        "JsmCreateRequest",
			description: "Create a new Jira Service Management customer request.",
			bsrRef:      jiraBsrBase + "CreateRequestRequest:" + jiraBsrVersion,
			handler:     makeCreateRequestHandler(client),
		},
		{
			name:        "JsmGetRequest",
			description: "Get Jira Service Management request details by issue ID or key.",
			bsrRef:      jiraBsrBase + "GetRequestRequest:" + jiraBsrVersion,
			handler:     makeGetRequestHandler(client),
		},
		{
			name:        "JsmAddRequestComment",
			description: "Add a Jira Service Management request comment with public/internal visibility.",
			bsrRef:      jiraBsrBase + "AddRequestCommentRequest:" + jiraBsrVersion,
			handler:     makeAddRequestCommentHandler(client),
		},
		{
			name:        "JsmGetOrganizations",
			description: "List Jira Service Management organizations.",
			bsrRef:      jiraBsrBase + "GetOrganizationsRequest:" + jiraBsrVersion,
			handler:     makeGetOrganizationsHandler(client),
		},
		{
			name:        "JsmGetCustomers",
			description: "List customers in a Jira Service Management organization.",
			bsrRef:      jiraBsrBase + "GetCustomersRequest:" + jiraBsrVersion,
			handler:     makeGetCustomersHandler(client),
		},
		{
			name:        "JsmGetSlaInfo",
			description: "Get SLA information for a Jira Service Management request.",
			bsrRef:      jiraBsrBase + "GetSlaInfoRequest:" + jiraBsrVersion,
			handler:     makeGetSlaInfoHandler(client),
		},
	}

	for _, t := range tools {
		tool := &mcp.Tool{
			Name:        t.name,
			Description: t.description,
			SchemaSource: &mcp.Tool_BsrRef{
				BsrRef: t.bsrRef,
			},
		}
		// Add metadata if present
		if len(t.metadata) > 0 {
			tool.Metadata = t.metadata
		} else {
			// Default metadata for tools without explicit metadata
			tool.Metadata = map[string]string{
				"category":    "Jira",
				"integration": "issue-tracking",
			}
		}
		r.Register(tool, t.handler)
	}

	return nil
}

func makeSearchIssuesHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.SearchIssuesRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetJql()) == "" {
			return nil, fmt.Errorf("jql is required")
		}

		result, err := client.SearchIssues(ctx, req.GetJql(), jira.SearchOptions{
			Fields:     req.GetFields(),
			StartAt:    int(req.GetStartAt()),
			MaxResults: int(req.GetMaxResults()),
			Expand:     req.GetExpand(),
		})
		if err != nil {
			return nil, err
		}

		return toolJSON(result)
	}
}

func makeGetIssueHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.GetIssueRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueKey()) == "" {
			return nil, fmt.Errorf("issue_key is required")
		}

		issue, err := client.GetIssue(ctx, req.GetIssueKey())
		if err != nil {
			return nil, err
		}

		return toolJSON(issue)
	}
}

func makeCreateIssueHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.CreateIssueRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetProjectKey()) == "" {
			return nil, fmt.Errorf("project_key is required")
		}
		if strings.TrimSpace(req.GetIssueType()) == "" {
			return nil, fmt.Errorf("issue_type is required")
		}
		if strings.TrimSpace(req.GetSummary()) == "" {
			return nil, fmt.Errorf("summary is required")
		}

		resp, err := client.CreateIssue(ctx, &req)
		if err != nil {
			return nil, err
		}

		return toolJSON(resp)
	}
}

func makeUpdateIssueHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.UpdateIssueRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueKey()) == "" {
			return nil, fmt.Errorf("issue_key is required")
		}

		resp, err := client.UpdateIssue(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeTransitionIssueHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.TransitionIssueRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueKey()) == "" {
			return nil, fmt.Errorf("issue_key is required")
		}
		if strings.TrimSpace(req.GetTransitionId()) == "" {
			return nil, fmt.Errorf("transition_id is required")
		}

		resp, err := client.TransitionIssue(ctx, &req)
		if err != nil {
			return nil, err
		}

		return toolJSON(resp)
	}
}

func makeAddCommentHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.AddCommentRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueKey()) == "" {
			return nil, fmt.Errorf("issue_key is required")
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

func makeAssignIssueHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.AssignIssueRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueKey()) == "" {
			return nil, fmt.Errorf("issue_key is required")
		}

		resp, err := client.AssignIssue(ctx, &req)
		if err != nil {
			return nil, err
		}

		return toolJSON(resp)
	}
}

func makeGetTransitionsHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.GetTransitionsRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueKey()) == "" {
			return nil, fmt.Errorf("issue_key is required")
		}

		transitions, err := client.GetTransitions(ctx, req.GetIssueKey())
		if err != nil {
			return nil, err
		}

		return toolJSON(transitions)
	}
}

func makeSearchUsersHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.SearchUsersRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetQuery()) == "" {
			return nil, fmt.Errorf("query is required")
		}

		users, err := client.SearchUsers(ctx, req.GetQuery(), req.GetMaxResults())
		if err != nil {
			return nil, err
		}

		return toolJSON(users)
	}
}

func makeGetServiceDesksHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.GetServiceDesksRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.GetServiceDesks(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeGetRequestTypesHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.GetRequestTypesRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetServiceDeskId()) == "" {
			return nil, fmt.Errorf("service_desk_id is required")
		}
		resp, err := client.GetRequestTypes(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeCreateRequestHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.CreateRequestRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetServiceDeskId()) == "" {
			return nil, fmt.Errorf("service_desk_id is required")
		}
		if strings.TrimSpace(req.GetRequestTypeId()) == "" {
			return nil, fmt.Errorf("request_type_id is required")
		}
		resp, err := client.CreateRequest(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeGetRequestHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.GetRequestRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueIdOrKey()) == "" {
			return nil, fmt.Errorf("issue_id_or_key is required")
		}
		resp, err := client.GetRequest(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeAddRequestCommentHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.AddRequestCommentRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueIdOrKey()) == "" {
			return nil, fmt.Errorf("issue_id_or_key is required")
		}
		if strings.TrimSpace(req.GetBody()) == "" {
			return nil, fmt.Errorf("body is required")
		}
		resp, err := client.AddRequestComment(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeGetOrganizationsHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.GetOrganizationsRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.GetOrganizations(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeGetCustomersHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.GetCustomersRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetOrganizationId()) == "" {
			return nil, fmt.Errorf("organization_id is required")
		}
		resp, err := client.GetCustomers(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeGetSlaInfoHandler(client *jira.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req jira.GetSlaInfoRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetIssueIdOrKey()) == "" {
			return nil, fmt.Errorf("issue_id_or_key is required")
		}
		resp, err := client.GetSlaInfo(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func toolJSON(v any) (*mcp.ToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal tool response: %w", err)
	}
	return mcpText(string(b)), nil
}
