package notion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
)

const (
	defaultTimeout     = 15 * time.Second
	defaultPageSize    = 100
	defaultNotionVersion = "2022-06-28"
)

// Client is a pure-HTTP REST client for the Notion API.
type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
	apiVersion string
}

// NewClient creates a Client from the NOTION_TOKEN environment variable.
// Optionally reads NOTION_API_VERSION (defaults to 2022-06-28).
func NewClient() (*Client, error) {
	token := strings.TrimSpace(os.Getenv("NOTION_TOKEN"))
	if token == "" {
		return nil, fmt.Errorf("NOTION_TOKEN is required")
	}
	apiVersion := strings.TrimSpace(os.Getenv("NOTION_API_VERSION"))
	if apiVersion == "" {
		apiVersion = defaultNotionVersion
	}
	return NewClientWithConfig(token, &http.Client{Timeout: defaultTimeout}, "", apiVersion)
}

// NewClientWithConfig creates a Client with explicit configuration.
// If baseURL is empty it defaults to https://api.notion.com.
// If apiVersion is empty it defaults to 2022-06-28.
func NewClientWithConfig(token string, httpClient *http.Client, baseURL string, apiVersion string) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("notion token is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.notion.com"
	}
	if strings.TrimSpace(apiVersion) == "" {
		apiVersion = defaultNotionVersion
	}
	return &Client{
		httpClient: httpClient,
		token:      token,
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiVersion: apiVersion,
	}, nil
}

// ---------------------------------------------------------------------------
// Public Methods
// ---------------------------------------------------------------------------

// Search performs a POST /v1/search against the Notion API.
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
	if req == nil {
		req = &SearchRequest{}
	}

	body := map[string]any{}
	if q := strings.TrimSpace(req.GetQuery()); q != "" {
		body["query"] = q
	}
	if fo := strings.TrimSpace(req.GetFilterObject()); fo != "" {
		body["filter"] = map[string]any{
			"value":    fo,
			"property": "object",
		}
	}
	if s := req.GetSort(); s != nil {
		sortMap := map[string]any{}
		if dir := strings.TrimSpace(s.GetDirection()); dir != "" {
			sortMap["direction"] = dir
		}
		if ts := strings.TrimSpace(s.GetTimestamp()); ts != "" {
			sortMap["timestamp"] = ts
		}
		if len(sortMap) > 0 {
			body["sort"] = sortMap
		}
	}
	ps := int(req.GetPageSize())
	if ps <= 0 {
		ps = defaultPageSize
	}
	body["page_size"] = ps
	if cursor := strings.TrimSpace(req.GetStartCursor()); cursor != "" {
		body["start_cursor"] = cursor
	}

	var raw map[string]any
	if err := c.do(ctx, http.MethodPost, "/v1/search", body, &raw); err != nil {
		return nil, err
	}

	out := &SearchResponse{
		HasMore:    getBool(raw, "has_more"),
		NextCursor: getString(raw, "next_cursor"),
	}
	for _, item := range toSlice(getAny(raw, "results")) {
		m := toMap(item)
		out.Results = append(out.Results, toProtoSearchResult(m))
	}
	return out, nil
}

// GetPage retrieves a single page via GET /v1/pages/{page_id}.
func (c *Client) GetPage(ctx context.Context, req *GetPageRequest) (*GetPageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get page request is required")
	}
	pageID := strings.TrimSpace(req.GetPageId())
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	var raw map[string]any
	if err := c.do(ctx, http.MethodGet, "/v1/pages/"+url.PathEscape(pageID), nil, &raw); err != nil {
		return nil, err
	}
	return &GetPageResponse{Page: toProtoPage(raw)}, nil
}

// CreatePage creates a new page via POST /v1/pages.
func (c *Client) CreatePage(ctx context.Context, req *CreatePageRequest) (*CreatePageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create page request is required")
	}
	if req.GetParent() == nil {
		return nil, fmt.Errorf("parent is required")
	}

	body := map[string]any{}

	// Build parent
	parent := req.GetParent()
	parentMap := map[string]any{}
	switch {
	case strings.TrimSpace(parent.GetDatabaseId()) != "":
		parentMap["type"] = "database_id"
		parentMap["database_id"] = parent.GetDatabaseId()
	case strings.TrimSpace(parent.GetPageId()) != "":
		parentMap["type"] = "page_id"
		parentMap["page_id"] = parent.GetPageId()
	default:
		return nil, fmt.Errorf("parent must specify database_id or page_id")
	}
	body["parent"] = parentMap

	// Properties
	if props := req.GetProperties(); props != nil {
		body["properties"] = props.AsMap()
	}

	// Children blocks
	if children := req.GetChildren(); len(children) > 0 {
		childSlice := make([]any, 0, len(children))
		for _, ch := range children {
			childSlice = append(childSlice, ch.AsMap())
		}
		body["children"] = childSlice
	}

	// Icon
	if emoji := strings.TrimSpace(req.GetIconEmoji()); emoji != "" {
		body["icon"] = map[string]any{
			"type":  "emoji",
			"emoji": emoji,
		}
	}

	// Cover
	if coverURL := strings.TrimSpace(req.GetCoverUrl()); coverURL != "" {
		body["cover"] = map[string]any{
			"type": "external",
			"external": map[string]any{
				"url": coverURL,
			},
		}
	}

	var raw map[string]any
	if err := c.do(ctx, http.MethodPost, "/v1/pages", body, &raw); err != nil {
		return nil, err
	}
	return &CreatePageResponse{Page: toProtoPage(raw)}, nil
}

// UpdatePage updates page properties via PATCH /v1/pages/{page_id}.
func (c *Client) UpdatePage(ctx context.Context, req *UpdatePageRequest) (*UpdatePageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update page request is required")
	}
	pageID := strings.TrimSpace(req.GetPageId())
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	body := map[string]any{}

	if props := req.GetProperties(); props != nil {
		body["properties"] = props.AsMap()
	}
	if emoji := strings.TrimSpace(req.GetIconEmoji()); emoji != "" {
		body["icon"] = map[string]any{
			"type":  "emoji",
			"emoji": emoji,
		}
	}
	if coverURL := strings.TrimSpace(req.GetCoverUrl()); coverURL != "" {
		body["cover"] = map[string]any{
			"type": "external",
			"external": map[string]any{
				"url": coverURL,
			},
		}
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("at least one field is required for update")
	}

	var raw map[string]any
	if err := c.do(ctx, http.MethodPatch, "/v1/pages/"+url.PathEscape(pageID), body, &raw); err != nil {
		return nil, err
	}
	return &UpdatePageResponse{Page: toProtoPage(raw)}, nil
}

// MovePage changes a page's parent via PATCH /v1/pages/{page_id}.
func (c *Client) MovePage(ctx context.Context, req *MovePageRequest) (*MovePageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("move page request is required")
	}
	pageID := strings.TrimSpace(req.GetPageId())
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}
	if req.GetParent() == nil {
		return nil, fmt.Errorf("parent is required")
	}

	parent := req.GetParent()
	parentMap := map[string]any{}
	switch {
	case strings.TrimSpace(parent.GetDatabaseId()) != "":
		parentMap["type"] = "database_id"
		parentMap["database_id"] = parent.GetDatabaseId()
	case strings.TrimSpace(parent.GetPageId()) != "":
		parentMap["type"] = "page_id"
		parentMap["page_id"] = parent.GetPageId()
	default:
		return nil, fmt.Errorf("parent must specify database_id or page_id")
	}

	body := map[string]any{"parent": parentMap}

	var raw map[string]any
	if err := c.do(ctx, http.MethodPatch, "/v1/pages/"+url.PathEscape(pageID), body, &raw); err != nil {
		return nil, err
	}
	return &MovePageResponse{Page: toProtoPage(raw)}, nil
}

// ArchivePage archives or unarchives a page via PATCH /v1/pages/{page_id}.
func (c *Client) ArchivePage(ctx context.Context, req *ArchivePageRequest) (*ArchivePageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("archive page request is required")
	}
	pageID := strings.TrimSpace(req.GetPageId())
	if pageID == "" {
		return nil, fmt.Errorf("page_id is required")
	}

	body := map[string]any{"archived": req.GetArchived()}

	var raw map[string]any
	if err := c.do(ctx, http.MethodPatch, "/v1/pages/"+url.PathEscape(pageID), body, &raw); err != nil {
		return nil, err
	}
	return &ArchivePageResponse{Page: toProtoPage(raw)}, nil
}

// CreateDatabase creates an inline database via POST /v1/databases.
func (c *Client) CreateDatabase(ctx context.Context, req *CreateDatabaseRequest) (*CreateDatabaseResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create database request is required")
	}
	parentPageID := strings.TrimSpace(req.GetParentPageId())
	if parentPageID == "" {
		return nil, fmt.Errorf("parent_page_id is required")
	}
	if strings.TrimSpace(req.GetTitle()) == "" {
		return nil, fmt.Errorf("title is required")
	}

	body := map[string]any{
		"parent": map[string]any{
			"type":    "page_id",
			"page_id": parentPageID,
		},
		"title": []map[string]any{
			{
				"type": "text",
				"text": map[string]any{
					"content": req.GetTitle(),
				},
			},
		},
	}

	if props := req.GetProperties(); props != nil {
		body["properties"] = props.AsMap()
	}

	var raw map[string]any
	if err := c.do(ctx, http.MethodPost, "/v1/databases", body, &raw); err != nil {
		return nil, err
	}

	propsMap := getMap(raw, "properties")
	return &CreateDatabaseResponse{
		Id:         getString(raw, "id"),
		Url:        getString(raw, "url"),
		Title:      extractTitle(raw),
		Properties: protoStructFromMap(propsMap),
	}, nil
}

// QueryDatabase queries a database via POST /v1/databases/{database_id}/query.
func (c *Client) QueryDatabase(ctx context.Context, req *QueryDatabaseRequest) (*QueryDatabaseResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("query database request is required")
	}
	dbID := strings.TrimSpace(req.GetDatabaseId())
	if dbID == "" {
		return nil, fmt.Errorf("database_id is required")
	}

	body := map[string]any{}
	if filter := req.GetFilter(); filter != nil {
		body["filter"] = filter.AsMap()
	}
	if sorts := req.GetSorts(); len(sorts) > 0 {
		sortSlice := make([]map[string]any, 0, len(sorts))
		for _, s := range sorts {
			sm := map[string]any{}
			if prop := strings.TrimSpace(s.GetProperty()); prop != "" {
				sm["property"] = prop
			}
			if ts := strings.TrimSpace(s.GetTimestamp()); ts != "" {
				sm["timestamp"] = ts
			}
			if dir := strings.TrimSpace(s.GetDirection()); dir != "" {
				sm["direction"] = dir
			}
			if len(sm) > 0 {
				sortSlice = append(sortSlice, sm)
			}
		}
		if len(sortSlice) > 0 {
			body["sorts"] = sortSlice
		}
	}
	ps := int(req.GetPageSize())
	if ps <= 0 {
		ps = defaultPageSize
	}
	body["page_size"] = ps
	if cursor := strings.TrimSpace(req.GetStartCursor()); cursor != "" {
		body["start_cursor"] = cursor
	}

	path := "/v1/databases/" + url.PathEscape(dbID) + "/query"
	var raw map[string]any
	if err := c.do(ctx, http.MethodPost, path, body, &raw); err != nil {
		return nil, err
	}

	out := &QueryDatabaseResponse{
		HasMore:    getBool(raw, "has_more"),
		NextCursor: getString(raw, "next_cursor"),
	}
	for _, item := range toSlice(getAny(raw, "results")) {
		m := toMap(item)
		out.Results = append(out.Results, toProtoPage(m))
	}
	return out, nil
}

// GetBlockChildren retrieves child blocks via GET /v1/blocks/{block_id}/children.
func (c *Client) GetBlockChildren(ctx context.Context, req *GetBlockChildrenRequest) (*GetBlockChildrenResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get block children request is required")
	}
	blockID := strings.TrimSpace(req.GetBlockId())
	if blockID == "" {
		return nil, fmt.Errorf("block_id is required")
	}

	values := url.Values{}
	ps := int(req.GetPageSize())
	if ps <= 0 {
		ps = defaultPageSize
	}
	values.Set("page_size", strconv.Itoa(ps))
	if cursor := strings.TrimSpace(req.GetStartCursor()); cursor != "" {
		values.Set("start_cursor", cursor)
	}

	path := "/v1/blocks/" + url.PathEscape(blockID) + "/children?" + values.Encode()
	var raw map[string]any
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	out := &GetBlockChildrenResponse{
		HasMore:    getBool(raw, "has_more"),
		NextCursor: getString(raw, "next_cursor"),
	}
	for _, item := range toSlice(getAny(raw, "results")) {
		m := toMap(item)
		out.Results = append(out.Results, toProtoBlock(m))
	}
	return out, nil
}

// AppendBlocks appends child blocks via PATCH /v1/blocks/{block_id}/children.
func (c *Client) AppendBlocks(ctx context.Context, req *AppendBlocksRequest) (*AppendBlocksResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("append blocks request is required")
	}
	blockID := strings.TrimSpace(req.GetBlockId())
	if blockID == "" {
		return nil, fmt.Errorf("block_id is required")
	}
	if len(req.GetChildren()) == 0 {
		return nil, fmt.Errorf("children is required")
	}

	childSlice := make([]any, 0, len(req.GetChildren()))
	for _, ch := range req.GetChildren() {
		childSlice = append(childSlice, ch.AsMap())
	}
	body := map[string]any{"children": childSlice}

	path := "/v1/blocks/" + url.PathEscape(blockID) + "/children"
	var raw map[string]any
	if err := c.do(ctx, http.MethodPatch, path, body, &raw); err != nil {
		return nil, err
	}

	out := &AppendBlocksResponse{}
	for _, item := range toSlice(getAny(raw, "results")) {
		m := toMap(item)
		out.Results = append(out.Results, toProtoBlock(m))
	}
	return out, nil
}

// CreateComment creates a comment via POST /v1/comments.
func (c *Client) CreateComment(ctx context.Context, req *CreateCommentRequest) (*CreateCommentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create comment request is required")
	}
	if strings.TrimSpace(req.GetRichText()) == "" {
		return nil, fmt.Errorf("rich_text is required")
	}
	if strings.TrimSpace(req.GetParentPageId()) == "" && strings.TrimSpace(req.GetDiscussionId()) == "" {
		return nil, fmt.Errorf("parent_page_id or discussion_id is required")
	}

	body := map[string]any{
		"rich_text": makeRichText(req.GetRichText()),
	}
	if pageID := strings.TrimSpace(req.GetParentPageId()); pageID != "" {
		body["parent"] = map[string]any{"page_id": pageID}
	}
	if discID := strings.TrimSpace(req.GetDiscussionId()); discID != "" {
		body["discussion_id"] = discID
	}

	var raw map[string]any
	if err := c.do(ctx, http.MethodPost, "/v1/comments", body, &raw); err != nil {
		return nil, err
	}
	return &CreateCommentResponse{Comment: toProtoComment(raw)}, nil
}

// GetComments retrieves comments via GET /v1/comments.
func (c *Client) GetComments(ctx context.Context, req *GetCommentsRequest) (*GetCommentsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get comments request is required")
	}
	blockID := strings.TrimSpace(req.GetBlockId())
	if blockID == "" {
		return nil, fmt.Errorf("block_id is required")
	}

	values := url.Values{}
	values.Set("block_id", blockID)
	ps := int(req.GetPageSize())
	if ps <= 0 {
		ps = defaultPageSize
	}
	values.Set("page_size", strconv.Itoa(ps))
	if cursor := strings.TrimSpace(req.GetStartCursor()); cursor != "" {
		values.Set("start_cursor", cursor)
	}

	path := "/v1/comments?" + values.Encode()
	var raw map[string]any
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	out := &GetCommentsResponse{
		HasMore:    getBool(raw, "has_more"),
		NextCursor: getString(raw, "next_cursor"),
	}
	for _, item := range toSlice(getAny(raw, "results")) {
		m := toMap(item)
		out.Comments = append(out.Comments, toProtoComment(m))
	}
	return out, nil
}

// ListUsers lists workspace users via GET /v1/users.
func (c *Client) ListUsers(ctx context.Context, req *ListUsersRequest) (*ListUsersResponse, error) {
	if req == nil {
		req = &ListUsersRequest{}
	}

	values := url.Values{}
	ps := int(req.GetPageSize())
	if ps <= 0 {
		ps = defaultPageSize
	}
	values.Set("page_size", strconv.Itoa(ps))
	if cursor := strings.TrimSpace(req.GetStartCursor()); cursor != "" {
		values.Set("start_cursor", cursor)
	}

	path := "/v1/users?" + values.Encode()
	var raw map[string]any
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	out := &ListUsersResponse{
		HasMore:    getBool(raw, "has_more"),
		NextCursor: getString(raw, "next_cursor"),
	}
	for _, item := range toSlice(getAny(raw, "results")) {
		m := toMap(item)
		out.Users = append(out.Users, toProtoUser(m))
	}
	return out, nil
}

// GetUser retrieves a single user via GET /v1/users/{user_id}.
func (c *Client) GetUser(ctx context.Context, req *GetUserRequest) (*GetUserResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get user request is required")
	}
	userID := strings.TrimSpace(req.GetUserId())
	if userID == "" {
		return nil, fmt.Errorf("user_id is required")
	}

	var raw map[string]any
	if err := c.do(ctx, http.MethodGet, "/v1/users/"+url.PathEscape(userID), nil, &raw); err != nil {
		return nil, err
	}
	return &GetUserResponse{User: toProtoUser(raw)}, nil
}

// GetSelf retrieves the bot user via GET /v1/users/me.
func (c *Client) GetSelf(ctx context.Context, req *GetSelfRequest) (*GetSelfResponse, error) {
	if req == nil {
		req = &GetSelfRequest{}
	}

	var raw map[string]any
	if err := c.do(ctx, http.MethodGet, "/v1/users/me", nil, &raw); err != nil {
		return nil, err
	}
	return &GetSelfResponse{User: toProtoUser(raw)}, nil
}

// ListTeams lists teams via GET /v1/teams.
func (c *Client) ListTeams(ctx context.Context, req *ListTeamsRequest) (*ListTeamsResponse, error) {
	if req == nil {
		req = &ListTeamsRequest{}
	}

	values := url.Values{}
	ps := int(req.GetPageSize())
	if ps <= 0 {
		ps = defaultPageSize
	}
	values.Set("page_size", strconv.Itoa(ps))
	if cursor := strings.TrimSpace(req.GetStartCursor()); cursor != "" {
		values.Set("start_cursor", cursor)
	}

	path := "/v1/teams?" + values.Encode()
	var raw map[string]any
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	out := &ListTeamsResponse{
		HasMore:    getBool(raw, "has_more"),
		NextCursor: getString(raw, "next_cursor"),
	}
	for _, item := range toSlice(getAny(raw, "results")) {
		m := toMap(item)
		out.Teams = append(out.Teams, &NotionTeam{
			Id:          getString(m, "id"),
			Name:        getString(m, "name"),
			Description: getString(m, "description"),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// HTTP transport
// ---------------------------------------------------------------------------

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal notion request: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build notion request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Notion-Version", c.apiVersion)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notion request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		requestID := strings.TrimSpace(resp.Header.Get("X-Request-Id"))

		// Detect rate limiting and surface retry-after header
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			return fmt.Errorf(
				"notion rate limit exceeded: retry_after=%s request_id=%s path=%s",
				retryAfter,
				requestID,
				path,
			)
		}

		return fmt.Errorf(
			"notion %s %s failed: status=%d request_id=%s response_bytes=%d",
			method,
			path,
			resp.StatusCode,
			requestID,
			len(raw),
		)
	}

	if result == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	// Limit response body to 10MB to prevent memory exhaustion
	const maxResponseSize = 10 * 1024 * 1024
	limitedBody := io.LimitReader(resp.Body, maxResponseSize)

	if err := json.NewDecoder(limitedBody).Decode(result); err != nil {
		return fmt.Errorf("decode notion response: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Proto conversion helpers
// ---------------------------------------------------------------------------

func toProtoPage(m map[string]any) *Page {
	if len(m) == 0 {
		return nil
	}

	page := &Page{
		Id:             getString(m, "id"),
		Url:            getString(m, "url"),
		CreatedTime:    getString(m, "created_time"),
		LastEditedTime: getString(m, "last_edited_time"),
		Archived:       getBool(m, "archived"),
		Parent:         toProtoParent(getMap(m, "parent")),
		Properties:     protoStructFromMap(getMap(m, "properties")),
	}

	// Icon: can be emoji or external
	if iconMap := getMap(m, "icon"); len(iconMap) > 0 {
		switch getString(iconMap, "type") {
		case "emoji":
			page.Icon = getString(iconMap, "emoji")
		case "external":
			page.Icon = getString(getMap(iconMap, "external"), "url")
		case "file":
			page.Icon = getString(getMap(iconMap, "file"), "url")
		}
	}

	// Cover: can be external or file
	if coverMap := getMap(m, "cover"); len(coverMap) > 0 {
		switch getString(coverMap, "type") {
		case "external":
			page.Cover = getString(getMap(coverMap, "external"), "url")
		case "file":
			page.Cover = getString(getMap(coverMap, "file"), "url")
		}
	}

	return page
}

func toProtoBlock(m map[string]any) *Block {
	if len(m) == 0 {
		return nil
	}

	blockType := getString(m, "type")
	block := &Block{
		Id:             getString(m, "id"),
		Type:           blockType,
		CreatedTime:    getString(m, "created_time"),
		LastEditedTime: getString(m, "last_edited_time"),
		HasChildren:    getBool(m, "has_children"),
		Archived:       getBool(m, "archived"),
	}

	// Type-specific payload lives under the type key
	if blockType != "" {
		if data := getMap(m, blockType); len(data) > 0 {
			block.Data = protoStructFromMap(data)
		}
	}

	return block
}

func toProtoUser(m map[string]any) *NotionUser {
	if len(m) == 0 {
		return nil
	}

	email := getString(m, "email")
	if email == "" {
		// Notion nests email under person.email
		if person := getMap(m, "person"); len(person) > 0 {
			email = getString(person, "email")
		}
	}

	return &NotionUser{
		Id:        getString(m, "id"),
		Type:      getString(m, "type"),
		Name:      getString(m, "name"),
		AvatarUrl: getString(m, "avatar_url"),
		Email:     email,
	}
}

func toProtoParent(m map[string]any) *Parent {
	if len(m) == 0 {
		return nil
	}
	return &Parent{
		Type:       getString(m, "type"),
		DatabaseId: getString(m, "database_id"),
		PageId:     getString(m, "page_id"),
	}
}

func toProtoSearchResult(m map[string]any) *SearchResult {
	if len(m) == 0 {
		return nil
	}

	// Extract parent_id from the parent object
	parentID := ""
	if p := getMap(m, "parent"); len(p) > 0 {
		if id := getString(p, "database_id"); id != "" {
			parentID = id
		} else if id := getString(p, "page_id"); id != "" {
			parentID = id
		} else if id := getString(p, "workspace"); id != "" {
			parentID = id
		}
	}

	// Extract title from properties
	title := ""
	objectType := getString(m, "object")
	if objectType == "database" {
		// Databases store title as a top-level title array
		if titleArr := toSlice(getAny(m, "title")); len(titleArr) > 0 {
			first := toMap(titleArr[0])
			title = getString(first, "plain_text")
		}
	} else {
		// Pages store title in properties
		title = extractTitle(getMap(m, "properties"))
	}

	return &SearchResult{
		Id:             getString(m, "id"),
		Object:         objectType,
		Url:            getString(m, "url"),
		Title:          title,
		ParentId:       parentID,
		LastEditedTime: getString(m, "last_edited_time"),
	}
}

func toProtoComment(m map[string]any) *CommentEntry {
	if len(m) == 0 {
		return nil
	}

	// Extract parent_id from parent object
	parentID := ""
	if p := getMap(m, "parent"); len(p) > 0 {
		if id := getString(p, "page_id"); id != "" {
			parentID = id
		}
	}

	// Extract created_by id
	createdByID := ""
	if cb := getMap(m, "created_by"); len(cb) > 0 {
		createdByID = getString(cb, "id")
	}

	// Extract plain text from rich_text array
	richText := ""
	if rtArr := toSlice(getAny(m, "rich_text")); len(rtArr) > 0 {
		var parts []string
		for _, rt := range rtArr {
			rtMap := toMap(rt)
			if pt := getString(rtMap, "plain_text"); pt != "" {
				parts = append(parts, pt)
			}
		}
		richText = strings.Join(parts, "")
	}

	return &CommentEntry{
		Id:           getString(m, "id"),
		DiscussionId: getString(m, "discussion_id"),
		ParentId:     parentID,
		CreatedTime:  getString(m, "created_time"),
		CreatedById:  createdByID,
		RichText:     richText,
	}
}

// ---------------------------------------------------------------------------
// Title extraction helper
// ---------------------------------------------------------------------------

// extractTitle handles Notion's title property format.
// Notion pages store titles in a "title" type property. The property name
// varies (commonly "Name" or "title"), so we scan all properties for one
// whose type is "title".
// Format: {"Name": {"type": "title", "title": [{"plain_text": "My Title"}]}}
func extractTitle(properties map[string]any) string {
	if len(properties) == 0 {
		return ""
	}
	for _, v := range properties {
		prop := toMap(v)
		if getString(prop, "type") != "title" {
			continue
		}
		titleArr := toSlice(getAny(prop, "title"))
		if len(titleArr) == 0 {
			continue
		}
		var parts []string
		for _, item := range titleArr {
			m := toMap(item)
			if pt := getString(m, "plain_text"); pt != "" {
				parts = append(parts, pt)
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "")
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Rich text helpers
// ---------------------------------------------------------------------------

func makeRichText(text string) []map[string]any {
	return []map[string]any{
		{
			"type": "text",
			"text": map[string]any{
				"content": text,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// JSON helper functions
// ---------------------------------------------------------------------------

func getAny(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func getMap(m map[string]any, key string) map[string]any {
	return toMap(getAny(m, key))
}

func getString(m map[string]any, key string) string {
	v := getAny(m, key)
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return ""
	}
}

func getBool(m map[string]any, key string) bool {
	v := getAny(m, key)
	b, ok := v.(bool)
	return ok && b
}

func getFloat64(m map[string]any, key string) float64 {
	v := getAny(m, key)
	switch t := v.(type) {
	case float64:
		return t
	case int64:
		return float64(t)
	case int:
		return float64(t)
	case json.Number:
		out, _ := t.Float64()
		return out
	default:
		return 0
	}
}

func toMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func toSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func protoStructFromMap(m map[string]any) *structpb.Struct {
	if len(m) == 0 {
		return nil
	}
	out, err := structpb.NewStruct(m)
	if err != nil {
		// Critical: structpb conversion failed, likely malformed API response
		// This indicates data corruption and should be surfaced to operations
		fmt.Fprintf(os.Stderr, "CRITICAL: notion proto conversion failed: %v (map keys: %v)\n", err, mapKeys(m))
		return nil
	}
	return out
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
