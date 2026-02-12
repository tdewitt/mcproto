package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/misfitdev/proto-mcp/go/mcp"
	"github.com/misfitdev/proto-mcp/go/pkg/notion"
	"google.golang.org/protobuf/proto"
)

const notionBsrBase = "buf.build/mcpb/notion/tucker.mcproto.notion.v1."
const notionBsrVersion = "main"

func (r *UnifiedRegistry) PopulateNotionTools(client *notion.Client) error {
	if client == nil {
		return fmt.Errorf("notion client is nil")
	}

	tools := []struct {
		name        string
		description string
		bsrRef      string
		handler     ToolHandler
	}{
		{
			name:        "NotionSearch",
			description: "Search Notion pages and databases by query string.",
			bsrRef:      notionBsrBase + "SearchRequest:" + notionBsrVersion,
			handler:     makeNotionSearchHandler(client),
		},
		{
			name:        "NotionGetPage",
			description: "Get a Notion page and its properties by ID.",
			bsrRef:      notionBsrBase + "GetPageRequest:" + notionBsrVersion,
			handler:     makeNotionGetPageHandler(client),
		},
		{
			name:        "NotionCreatePage",
			description: "Create a new page in a Notion parent page or database. Use NotionSearch to find parent IDs.",
			bsrRef:      notionBsrBase + "CreatePageRequest:" + notionBsrVersion,
			handler:     makeNotionCreatePageHandler(client),
		},
		{
			name:        "NotionUpdatePage",
			description: "Update properties on an existing Notion page.",
			bsrRef:      notionBsrBase + "UpdatePageRequest:" + notionBsrVersion,
			handler:     makeNotionUpdatePageHandler(client),
		},
		{
			name:        "NotionMovePage",
			description: "Move a Notion page to a different parent. Use NotionGetPage to verify both source and destination pages exist.",
			bsrRef:      notionBsrBase + "MovePageRequest:" + notionBsrVersion,
			handler:     makeNotionMovePageHandler(client),
		},
		{
			name:        "NotionArchivePage",
			description: "Archive or unarchive a Notion page.",
			bsrRef:      notionBsrBase + "ArchivePageRequest:" + notionBsrVersion,
			handler:     makeNotionArchivePageHandler(client),
		},
		{
			name:        "NotionCreateDatabase",
			description: "Create a new database as a child of a Notion page.",
			bsrRef:      notionBsrBase + "CreateDatabaseRequest:" + notionBsrVersion,
			handler:     makeNotionCreateDatabaseHandler(client),
		},
		{
			name:        "NotionQueryDatabase",
			description: "Query a Notion database with filters and sorts. Use NotionSearch to find database IDs.",
			bsrRef:      notionBsrBase + "QueryDatabaseRequest:" + notionBsrVersion,
			handler:     makeNotionQueryDatabaseHandler(client),
		},
		{
			name:        "NotionGetBlockChildren",
			description: "Get child blocks of a Notion page or block.",
			bsrRef:      notionBsrBase + "GetBlockChildrenRequest:" + notionBsrVersion,
			handler:     makeNotionGetBlockChildrenHandler(client),
		},
		{
			name:        "NotionAppendBlocks",
			description: "Append child blocks to a Notion page or block. Use NotionGetBlockChildren to inspect existing blocks.",
			bsrRef:      notionBsrBase + "AppendBlocksRequest:" + notionBsrVersion,
			handler:     makeNotionAppendBlocksHandler(client),
		},
		{
			name:        "NotionCreateComment",
			description: "Create a comment on a Notion page. Requires parent_page_id or discussion_id, plus rich_text content.",
			bsrRef:      notionBsrBase + "CreateCommentRequest:" + notionBsrVersion,
			handler:     makeNotionCreateCommentHandler(client),
		},
		{
			name:        "NotionGetComments",
			description: "Get comments on a Notion page or block.",
			bsrRef:      notionBsrBase + "GetCommentsRequest:" + notionBsrVersion,
			handler:     makeNotionGetCommentsHandler(client),
		},
		{
			name:        "NotionListUsers",
			description: "List users in the Notion workspace.",
			bsrRef:      notionBsrBase + "ListUsersRequest:" + notionBsrVersion,
			handler:     makeNotionListUsersHandler(client),
		},
		{
			name:        "NotionGetUser",
			description: "Get a Notion user by ID.",
			bsrRef:      notionBsrBase + "GetUserRequest:" + notionBsrVersion,
			handler:     makeNotionGetUserHandler(client),
		},
		{
			name:        "NotionGetSelf",
			description: "Get the authenticated Notion bot user identity.",
			bsrRef:      notionBsrBase + "GetSelfRequest:" + notionBsrVersion,
			handler:     makeNotionGetSelfHandler(client),
		},
		{
			name:        "NotionListTeams",
			description: "List teams in the Notion workspace (Enterprise).",
			bsrRef:      notionBsrBase + "ListTeamsRequest:" + notionBsrVersion,
			handler:     makeNotionListTeamsHandler(client),
		},
	}

	for _, t := range tools {
		r.RegisterWithCategory(&mcp.Tool{
			Name:        t.name,
			Description: t.description,
			SchemaSource: &mcp.Tool_BsrRef{
				BsrRef: t.bsrRef,
			},
		}, t.handler, "notion", []string{"notion", "knowledge-base"})
	}

	return nil
}

func makeNotionSearchHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.SearchRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.Search(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionGetPageHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.GetPageRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetPageId()) == "" {
			return nil, fmt.Errorf("page_id is required")
		}
		resp, err := client.GetPage(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionCreatePageHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.CreatePageRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if req.GetParent() == nil {
			return nil, fmt.Errorf("parent is required")
		}
		resp, err := client.CreatePage(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionUpdatePageHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.UpdatePageRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetPageId()) == "" {
			return nil, fmt.Errorf("page_id is required")
		}
		resp, err := client.UpdatePage(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionMovePageHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.MovePageRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetPageId()) == "" {
			return nil, fmt.Errorf("page_id is required")
		}
		if req.GetParent() == nil {
			return nil, fmt.Errorf("parent is required")
		}
		resp, err := client.MovePage(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionArchivePageHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.ArchivePageRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetPageId()) == "" {
			return nil, fmt.Errorf("page_id is required")
		}
		resp, err := client.ArchivePage(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionCreateDatabaseHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.CreateDatabaseRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetParentPageId()) == "" {
			return nil, fmt.Errorf("parent_page_id is required")
		}
		if strings.TrimSpace(req.GetTitle()) == "" {
			return nil, fmt.Errorf("title is required")
		}
		resp, err := client.CreateDatabase(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionQueryDatabaseHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.QueryDatabaseRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetDatabaseId()) == "" {
			return nil, fmt.Errorf("database_id is required")
		}
		resp, err := client.QueryDatabase(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionGetBlockChildrenHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.GetBlockChildrenRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetBlockId()) == "" {
			return nil, fmt.Errorf("block_id is required")
		}
		resp, err := client.GetBlockChildren(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionAppendBlocksHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.AppendBlocksRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetBlockId()) == "" {
			return nil, fmt.Errorf("block_id is required")
		}
		if len(req.GetChildren()) == 0 {
			return nil, fmt.Errorf("children is required")
		}
		resp, err := client.AppendBlocks(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionCreateCommentHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.CreateCommentRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetParentPageId()) == "" && strings.TrimSpace(req.GetDiscussionId()) == "" {
			return nil, fmt.Errorf("parent_page_id or discussion_id is required")
		}
		if strings.TrimSpace(req.GetRichText()) == "" {
			return nil, fmt.Errorf("rich_text is required")
		}
		resp, err := client.CreateComment(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionGetCommentsHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.GetCommentsRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetBlockId()) == "" {
			return nil, fmt.Errorf("block_id is required")
		}
		resp, err := client.GetComments(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionListUsersHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.ListUsersRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.ListUsers(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionGetUserHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.GetUserRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		if strings.TrimSpace(req.GetUserId()) == "" {
			return nil, fmt.Errorf("user_id is required")
		}
		resp, err := client.GetUser(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionGetSelfHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.GetSelfRequest
		if err := proto.Unmarshal(args, &req); err != nil {
			return nil, fmt.Errorf("unmarshal request: %w", err)
		}
		resp, err := client.GetSelf(ctx, &req)
		if err != nil {
			return nil, err
		}
		return toolJSON(resp)
	}
}

func makeNotionListTeamsHandler(client *notion.Client) ToolHandler {
	return func(ctx context.Context, args []byte) (*mcp.ToolResult, error) {
		var req notion.ListTeamsRequest
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
