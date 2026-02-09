package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout  = 15 * time.Second
	defaultPageSize = 50
	linearAPIURL    = "https://api.linear.app/graphql"
)

// issueFields is the inline GraphQL field selection used for issue queries.
const issueFields = `
	id identifier title description priority priorityLabel
	state { id name }
	assignee { id name }
	team { id name }
	project { id name }
	labels { nodes { id name } }
	cycle { id }
	createdAt updatedAt url
`

// projectFields is the inline GraphQL field selection used for project queries.
const projectFields = `
	id name description state progress targetDate startDate createdAt updatedAt url
`

// projectUpdateFields is the inline GraphQL field selection for project updates.
const projectUpdateFields = `
	id body health user { id name } createdAt url
`

// documentFields is the inline GraphQL field selection for documents.
const documentFields = `
	id title content icon color project { id } createdAt updatedAt url
`

// commentFields is the inline GraphQL field selection for comments.
const commentFields = `
	id body user { id name } createdAt updatedAt url
`

type Client struct {
	httpClient *http.Client
	apiKey     string
	apiURL     string // defaults to linearAPIURL, overridable for tests
}

func NewClient() (*Client, error) {
	apiKey := strings.TrimSpace(os.Getenv("LINEAR_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("LINEAR_API_KEY is required")
	}
	return NewClientWithConfig(apiKey, &http.Client{Timeout: defaultTimeout})
}

func NewClientWithConfig(apiKey string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("linear api key is required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		httpClient: httpClient,
		apiKey:     apiKey,
		apiURL:     linearAPIURL,
	}, nil
}

// SetAPIURL overrides the API URL (used for testing).
func (c *Client) SetAPIURL(url string) { c.apiURL = url }

// ---------------------------------------------------------------------------
// 1. ListIssues
// ---------------------------------------------------------------------------

func (c *Client) ListIssues(ctx context.Context, req *ListIssuesRequest) (*ListIssuesResponse, error) {
	if req == nil {
		req = &ListIssuesRequest{}
	}
	first := pageSize(req.GetFirst())

	filter := map[string]any{}
	if v := strings.TrimSpace(req.GetTeamId()); v != "" {
		filter["team"] = map[string]any{"id": map[string]any{"eq": v}}
	}
	if v := strings.TrimSpace(req.GetAssigneeId()); v != "" {
		filter["assignee"] = map[string]any{"id": map[string]any{"eq": v}}
	}
	if v := strings.TrimSpace(req.GetStateId()); v != "" {
		filter["state"] = map[string]any{"id": map[string]any{"eq": v}}
	}
	if v := strings.TrimSpace(req.GetLabelId()); v != "" {
		filter["labels"] = map[string]any{"id": map[string]any{"eq": v}}
	}
	if v := strings.TrimSpace(req.GetProjectId()); v != "" {
		filter["project"] = map[string]any{"id": map[string]any{"eq": v}}
	}
	if v := strings.TrimSpace(req.GetCycleId()); v != "" {
		filter["cycle"] = map[string]any{"id": map[string]any{"eq": v}}
	}

	vars := map[string]any{"first": first}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}
	if len(filter) > 0 {
		vars["filter"] = filter
	}

	query := `query($first: Int!, $after: String, $filter: IssueFilter) {
		issues(first: $first, after: $after, filter: $filter) {
			nodes {` + issueFields + `}
			pageInfo { hasNextPage endCursor }
		}
	}`

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	issues := getMap(data, "issues")
	nodes := toSlice(getAny(issues, "nodes"))
	out := &ListIssuesResponse{
		PageInfo: toProtoPageInfo(getMap(issues, "pageInfo")),
	}
	for _, n := range nodes {
		out.Issues = append(out.Issues, toProtoIssue(toMap(n)))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 2. GetIssue
// ---------------------------------------------------------------------------

func (c *Client) GetIssue(ctx context.Context, req *GetIssueRequest) (*GetIssueResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get issue request is required")
	}
	id := strings.TrimSpace(req.GetId())
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}

	// Linear's issue(id:) query accepts both UUIDs and identifiers (e.g., "ENG-123")
	query := `query($id: String!) {
		issue(id: $id) {` + issueFields + `}
	}`
	data, err := c.do(ctx, query, map[string]any{"id": id})
	if err != nil {
		return nil, err
	}
	m := getMap(data, "issue")
	if len(m) == 0 {
		return nil, fmt.Errorf("issue not found: %s", id)
	}
	return &GetIssueResponse{Issue: toProtoIssue(m)}, nil
}

// ---------------------------------------------------------------------------
// 3. CreateIssue
// ---------------------------------------------------------------------------

func (c *Client) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*CreateIssueResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create issue request is required")
	}
	if strings.TrimSpace(req.GetTeamId()) == "" {
		return nil, fmt.Errorf("team_id is required")
	}
	if strings.TrimSpace(req.GetTitle()) == "" {
		return nil, fmt.Errorf("title is required")
	}

	input := map[string]any{
		"teamId": req.GetTeamId(),
		"title":  req.GetTitle(),
	}
	if v := strings.TrimSpace(req.GetDescription()); v != "" {
		input["description"] = v
	}
	if v := strings.TrimSpace(req.GetAssigneeId()); v != "" {
		input["assigneeId"] = v
	}
	if v := strings.TrimSpace(req.GetStateId()); v != "" {
		input["stateId"] = v
	}
	if req.GetPriority() != 0 {
		input["priority"] = req.GetPriority()
	}
	if ids := req.GetLabelIds(); len(ids) > 0 {
		input["labelIds"] = ids
	}
	if v := strings.TrimSpace(req.GetProjectId()); v != "" {
		input["projectId"] = v
	}
	if v := strings.TrimSpace(req.GetCycleId()); v != "" {
		input["cycleId"] = v
	}

	query := `mutation($input: IssueCreateInput!) {
		issueCreate(input: $input) {
			success
			issue {` + issueFields + `}
		}
	}`
	data, err := c.do(ctx, query, map[string]any{"input": input})
	if err != nil {
		return nil, err
	}

	result := getMap(data, "issueCreate")
	return &CreateIssueResponse{
		Success: getBool(result, "success"),
		Issue:   toProtoIssue(getMap(result, "issue")),
	}, nil
}

// ---------------------------------------------------------------------------
// 4. UpdateIssue
// ---------------------------------------------------------------------------

func (c *Client) UpdateIssue(ctx context.Context, req *UpdateIssueRequest) (*UpdateIssueResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update issue request is required")
	}
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, fmt.Errorf("id is required")
	}

	input := map[string]any{}
	if v := strings.TrimSpace(req.GetTitle()); v != "" {
		input["title"] = v
	}
	if v := strings.TrimSpace(req.GetDescription()); v != "" {
		input["description"] = v
	}
	if v := strings.TrimSpace(req.GetAssigneeId()); v != "" {
		input["assigneeId"] = v
	}
	if v := strings.TrimSpace(req.GetStateId()); v != "" {
		input["stateId"] = v
	}
	if req.GetPriority() != 0 {
		input["priority"] = req.GetPriority()
	}
	if ids := req.GetLabelIds(); len(ids) > 0 {
		input["labelIds"] = ids
	}
	if v := strings.TrimSpace(req.GetProjectId()); v != "" {
		input["projectId"] = v
	}
	if v := strings.TrimSpace(req.GetCycleId()); v != "" {
		input["cycleId"] = v
	}
	if len(input) == 0 {
		return nil, fmt.Errorf("at least one field is required for update")
	}

	query := `mutation($id: String!, $input: IssueUpdateInput!) {
		issueUpdate(id: $id, input: $input) {
			success
			issue {` + issueFields + `}
		}
	}`
	data, err := c.do(ctx, query, map[string]any{"id": req.GetId(), "input": input})
	if err != nil {
		return nil, err
	}

	result := getMap(data, "issueUpdate")
	return &UpdateIssueResponse{
		Success: getBool(result, "success"),
		Issue:   toProtoIssue(getMap(result, "issue")),
	}, nil
}

// ---------------------------------------------------------------------------
// 5. DeleteIssue
// ---------------------------------------------------------------------------

func (c *Client) DeleteIssue(ctx context.Context, req *DeleteIssueRequest) (*DeleteIssueResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("delete issue request is required")
	}
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, fmt.Errorf("id is required")
	}

	query := `mutation($id: String!) {
		issueDelete(id: $id) { success }
	}`
	data, err := c.do(ctx, query, map[string]any{"id": req.GetId()})
	if err != nil {
		return nil, err
	}

	result := getMap(data, "issueDelete")
	return &DeleteIssueResponse{
		Success: getBool(result, "success"),
	}, nil
}

// ---------------------------------------------------------------------------
// 6. SearchIssues
// ---------------------------------------------------------------------------

func (c *Client) SearchIssues(ctx context.Context, req *SearchIssuesRequest) (*SearchIssuesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("search issues request is required")
	}
	if strings.TrimSpace(req.GetQuery()) == "" {
		return nil, fmt.Errorf("query is required")
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{
		"query": req.GetQuery(),
		"first": first,
	}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	gql := `query($query: String!, $first: Int!, $after: String) {
		issueSearch(query: $query, first: $first, after: $after) {
			nodes {` + issueFields + `}
			pageInfo { hasNextPage endCursor }
		}
	}`

	data, err := c.do(ctx, gql, vars)
	if err != nil {
		return nil, err
	}

	search := getMap(data, "issueSearch")
	nodes := toSlice(getAny(search, "nodes"))
	out := &SearchIssuesResponse{
		PageInfo: toProtoPageInfo(getMap(search, "pageInfo")),
	}
	for _, n := range nodes {
		out.Issues = append(out.Issues, toProtoIssue(toMap(n)))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 7. ListProjects
// ---------------------------------------------------------------------------

func (c *Client) ListProjects(ctx context.Context, req *ListProjectsRequest) (*ListProjectsResponse, error) {
	if req == nil {
		req = &ListProjectsRequest{}
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{"first": first}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	query := `query($first: Int!, $after: String) {
		projects(first: $first, after: $after) {
			nodes {` + projectFields + `}
			pageInfo { hasNextPage endCursor }
		}
	}`

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	projects := getMap(data, "projects")
	nodes := toSlice(getAny(projects, "nodes"))
	out := &ListProjectsResponse{
		PageInfo: toProtoPageInfo(getMap(projects, "pageInfo")),
	}
	for _, n := range nodes {
		out.Projects = append(out.Projects, toProtoProject(toMap(n)))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 8. GetProject
// ---------------------------------------------------------------------------

func (c *Client) GetProject(ctx context.Context, req *GetProjectRequest) (*GetProjectResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get project request is required")
	}
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, fmt.Errorf("id is required")
	}

	query := `query($id: String!) {
		project(id: $id) {` + projectFields + `}
	}`
	data, err := c.do(ctx, query, map[string]any{"id": req.GetId()})
	if err != nil {
		return nil, err
	}

	return &GetProjectResponse{
		Project: toProtoProject(getMap(data, "project")),
	}, nil
}

// ---------------------------------------------------------------------------
// 9. CreateProjectUpdate
// ---------------------------------------------------------------------------

func (c *Client) CreateProjectUpdate(ctx context.Context, req *CreateProjectUpdateRequest) (*CreateProjectUpdateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create project update request is required")
	}
	if strings.TrimSpace(req.GetProjectId()) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	if strings.TrimSpace(req.GetBody()) == "" {
		return nil, fmt.Errorf("body is required")
	}

	input := map[string]any{
		"projectId": req.GetProjectId(),
		"body":      req.GetBody(),
	}
	if v := strings.TrimSpace(req.GetHealth()); v != "" {
		input["health"] = v
	}

	query := `mutation($input: ProjectUpdateCreateInput!) {
		projectUpdateCreate(input: $input) {
			success
			projectUpdate {` + projectUpdateFields + `}
		}
	}`
	data, err := c.do(ctx, query, map[string]any{"input": input})
	if err != nil {
		return nil, err
	}

	result := getMap(data, "projectUpdateCreate")
	return &CreateProjectUpdateResponse{
		Success:       getBool(result, "success"),
		ProjectUpdate: toProtoProjectUpdate(getMap(result, "projectUpdate")),
	}, nil
}

// ---------------------------------------------------------------------------
// 10. ListProjectUpdates
// ---------------------------------------------------------------------------

func (c *Client) ListProjectUpdates(ctx context.Context, req *ListProjectUpdatesRequest) (*ListProjectUpdatesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("list project updates request is required")
	}
	if strings.TrimSpace(req.GetProjectId()) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{
		"first":     first,
		"projectId": req.GetProjectId(),
	}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	query := `query($first: Int!, $after: String, $projectId: String!) {
		projectUpdates(first: $first, after: $after, filter: {project: {id: {eq: $projectId}}}) {
			nodes {` + projectUpdateFields + `}
			pageInfo { hasNextPage endCursor }
		}
	}`

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	updates := getMap(data, "projectUpdates")
	nodes := toSlice(getAny(updates, "nodes"))
	out := &ListProjectUpdatesResponse{
		PageInfo: toProtoPageInfo(getMap(updates, "pageInfo")),
	}
	for _, n := range nodes {
		out.ProjectUpdates = append(out.ProjectUpdates, toProtoProjectUpdate(toMap(n)))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 11. ListTeams
// ---------------------------------------------------------------------------

func (c *Client) ListTeams(ctx context.Context, req *ListTeamsRequest) (*ListTeamsResponse, error) {
	if req == nil {
		req = &ListTeamsRequest{}
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{"first": first}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	query := `query($first: Int!, $after: String) {
		teams(first: $first, after: $after) {
			nodes { id name key description }
			pageInfo { hasNextPage endCursor }
		}
	}`

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	teams := getMap(data, "teams")
	nodes := toSlice(getAny(teams, "nodes"))
	out := &ListTeamsResponse{
		PageInfo: toProtoPageInfo(getMap(teams, "pageInfo")),
	}
	for _, n := range nodes {
		m := toMap(n)
		out.Teams = append(out.Teams, &Team{
			Id:          getString(m, "id"),
			Name:        getString(m, "name"),
			Key:         getString(m, "key"),
			Description: getString(m, "description"),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 12. ListMembers
// ---------------------------------------------------------------------------

func (c *Client) ListMembers(ctx context.Context, req *ListMembersRequest) (*ListMembersResponse, error) {
	if req == nil {
		req = &ListMembersRequest{}
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{"first": first}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	query := `query($first: Int!, $after: String) {
		users(first: $first, after: $after) {
			nodes { id name displayName email active admin }
			pageInfo { hasNextPage endCursor }
		}
	}`

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	users := getMap(data, "users")
	nodes := toSlice(getAny(users, "nodes"))
	out := &ListMembersResponse{
		PageInfo: toProtoPageInfo(getMap(users, "pageInfo")),
	}
	for _, n := range nodes {
		out.Members = append(out.Members, toProtoMember(toMap(n)))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 13. AddComment
// ---------------------------------------------------------------------------

func (c *Client) AddComment(ctx context.Context, req *AddCommentRequest) (*AddCommentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("add comment request is required")
	}
	if strings.TrimSpace(req.GetIssueId()) == "" {
		return nil, fmt.Errorf("issue_id is required")
	}
	if strings.TrimSpace(req.GetBody()) == "" {
		return nil, fmt.Errorf("body is required")
	}

	query := `mutation($issueId: String!, $body: String!) {
		commentCreate(input: {issueId: $issueId, body: $body}) {
			success
			comment {` + commentFields + `}
		}
	}`
	data, err := c.do(ctx, query, map[string]any{
		"issueId": req.GetIssueId(),
		"body":    req.GetBody(),
	})
	if err != nil {
		return nil, err
	}

	result := getMap(data, "commentCreate")
	return &AddCommentResponse{
		Success: getBool(result, "success"),
		Comment: toProtoComment(getMap(result, "comment")),
	}, nil
}

// ---------------------------------------------------------------------------
// 14. ListComments
// ---------------------------------------------------------------------------

func (c *Client) ListComments(ctx context.Context, req *ListCommentsRequest) (*ListCommentsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("list comments request is required")
	}
	if strings.TrimSpace(req.GetIssueId()) == "" {
		return nil, fmt.Errorf("issue_id is required")
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{
		"issueId": req.GetIssueId(),
		"first":   first,
	}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	query := `query($issueId: String!, $first: Int!, $after: String) {
		issue(id: $issueId) {
			comments(first: $first, after: $after) {
				nodes {` + commentFields + `}
				pageInfo { hasNextPage endCursor }
			}
		}
	}`

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	issue := getMap(data, "issue")
	comments := getMap(issue, "comments")
	nodes := toSlice(getAny(comments, "nodes"))
	out := &ListCommentsResponse{
		PageInfo: toProtoPageInfo(getMap(comments, "pageInfo")),
	}
	for _, n := range nodes {
		out.Comments = append(out.Comments, toProtoComment(toMap(n)))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 15. ListLabels
// ---------------------------------------------------------------------------

func (c *Client) ListLabels(ctx context.Context, req *ListLabelsRequest) (*ListLabelsResponse, error) {
	if req == nil {
		req = &ListLabelsRequest{}
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{"first": first}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	filterClause := ""
	if teamID := strings.TrimSpace(req.GetTeamId()); teamID != "" {
		vars["teamId"] = teamID
		filterClause = `, filter: {team: {id: {eq: $teamId}}}`
	}

	// Build query dynamically based on whether teamId filter is present.
	var query string
	if filterClause != "" {
		query = `query($first: Int!, $after: String, $teamId: String!) {
			issueLabels(first: $first, after: $after` + filterClause + `) {
				nodes { id name color parent { id } team { id } }
				pageInfo { hasNextPage endCursor }
			}
		}`
	} else {
		query = `query($first: Int!, $after: String) {
			issueLabels(first: $first, after: $after) {
				nodes { id name color parent { id } team { id } }
				pageInfo { hasNextPage endCursor }
			}
		}`
	}

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	labels := getMap(data, "issueLabels")
	nodes := toSlice(getAny(labels, "nodes"))
	out := &ListLabelsResponse{
		PageInfo: toProtoPageInfo(getMap(labels, "pageInfo")),
	}
	for _, n := range nodes {
		m := toMap(n)
		out.Labels = append(out.Labels, &Label{
			Id:       getString(m, "id"),
			Name:     getString(m, "name"),
			Color:    getString(m, "color"),
			ParentId: getString(getMap(m, "parent"), "id"),
			TeamId:   getString(getMap(m, "team"), "id"),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 16. ListCycles
// ---------------------------------------------------------------------------

func (c *Client) ListCycles(ctx context.Context, req *ListCyclesRequest) (*ListCyclesResponse, error) {
	if req == nil {
		req = &ListCyclesRequest{}
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{"first": first}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	filterClause := ""
	if teamID := strings.TrimSpace(req.GetTeamId()); teamID != "" {
		vars["teamId"] = teamID
		filterClause = `, filter: {team: {id: {eq: $teamId}}}`
	}

	var query string
	if filterClause != "" {
		query = `query($first: Int!, $after: String, $teamId: String!) {
			cycles(first: $first, after: $after` + filterClause + `) {
				nodes { id name number startsAt endsAt team { id } progress }
				pageInfo { hasNextPage endCursor }
			}
		}`
	} else {
		query = `query($first: Int!, $after: String) {
			cycles(first: $first, after: $after) {
				nodes { id name number startsAt endsAt team { id } progress }
				pageInfo { hasNextPage endCursor }
			}
		}`
	}

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	cycles := getMap(data, "cycles")
	nodes := toSlice(getAny(cycles, "nodes"))
	out := &ListCyclesResponse{
		PageInfo: toProtoPageInfo(getMap(cycles, "pageInfo")),
	}
	for _, n := range nodes {
		m := toMap(n)
		out.Cycles = append(out.Cycles, &Cycle{
			Id:       getString(m, "id"),
			Name:     getString(m, "name"),
			Number:   getString(m, "number"),
			StartsAt: getString(m, "startsAt"),
			EndsAt:   getString(m, "endsAt"),
			TeamId:   getString(getMap(m, "team"), "id"),
			Progress: getFloat32(m, "progress"),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 17. ListDocuments
// ---------------------------------------------------------------------------

func (c *Client) ListDocuments(ctx context.Context, req *ListDocumentsRequest) (*ListDocumentsResponse, error) {
	if req == nil {
		req = &ListDocumentsRequest{}
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{"first": first}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	query := `query($first: Int!, $after: String) {
		documents(first: $first, after: $after) {
			nodes {` + documentFields + `}
			pageInfo { hasNextPage endCursor }
		}
	}`

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	docs := getMap(data, "documents")
	nodes := toSlice(getAny(docs, "nodes"))
	out := &ListDocumentsResponse{
		PageInfo: toProtoPageInfo(getMap(docs, "pageInfo")),
	}
	for _, n := range nodes {
		out.Documents = append(out.Documents, toProtoDocument(toMap(n)))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 18. GetDocument
// ---------------------------------------------------------------------------

func (c *Client) GetDocument(ctx context.Context, req *GetDocumentRequest) (*GetDocumentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get document request is required")
	}
	if strings.TrimSpace(req.GetId()) == "" {
		return nil, fmt.Errorf("id is required")
	}

	query := `query($id: String!) {
		document(id: $id) {` + documentFields + `}
	}`
	data, err := c.do(ctx, query, map[string]any{"id": req.GetId()})
	if err != nil {
		return nil, err
	}

	return &GetDocumentResponse{
		Document: toProtoDocument(getMap(data, "document")),
	}, nil
}

// ---------------------------------------------------------------------------
// 19. SearchDocuments
// ---------------------------------------------------------------------------

func (c *Client) SearchDocuments(ctx context.Context, req *SearchDocumentsRequest) (*SearchDocumentsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("search documents request is required")
	}
	if strings.TrimSpace(req.GetQuery()) == "" {
		return nil, fmt.Errorf("query is required")
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{
		"query": req.GetQuery(),
		"first": first,
	}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	gql := `query($query: String!, $first: Int!, $after: String) {
		searchDocuments(term: $query, first: $first, after: $after) {
			nodes {` + documentFields + `}
			pageInfo { hasNextPage endCursor }
		}
	}`

	data, err := c.do(ctx, gql, vars)
	if err != nil {
		return nil, err
	}

	search := getMap(data, "searchDocuments")
	nodes := toSlice(getAny(search, "nodes"))
	out := &SearchDocumentsResponse{
		PageInfo: toProtoPageInfo(getMap(search, "pageInfo")),
	}
	for _, n := range nodes {
		out.Documents = append(out.Documents, toProtoDocument(toMap(n)))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 20. ListWorkflowStates
// ---------------------------------------------------------------------------

func (c *Client) ListWorkflowStates(ctx context.Context, req *ListWorkflowStatesRequest) (*ListWorkflowStatesResponse, error) {
	if req == nil {
		req = &ListWorkflowStatesRequest{}
	}
	first := pageSize(req.GetFirst())

	vars := map[string]any{"first": first}
	if after := strings.TrimSpace(req.GetAfter()); after != "" {
		vars["after"] = after
	}

	filterClause := ""
	if teamID := strings.TrimSpace(req.GetTeamId()); teamID != "" {
		vars["teamId"] = teamID
		filterClause = `, filter: {team: {id: {eq: $teamId}}}`
	}

	var query string
	if filterClause != "" {
		query = `query($first: Int!, $after: String, $teamId: String!) {
			workflowStates(first: $first, after: $after` + filterClause + `) {
				nodes { id name color type position team { id } }
				pageInfo { hasNextPage endCursor }
			}
		}`
	} else {
		query = `query($first: Int!, $after: String) {
			workflowStates(first: $first, after: $after) {
				nodes { id name color type position team { id } }
				pageInfo { hasNextPage endCursor }
			}
		}`
	}

	data, err := c.do(ctx, query, vars)
	if err != nil {
		return nil, err
	}

	states := getMap(data, "workflowStates")
	nodes := toSlice(getAny(states, "nodes"))
	out := &ListWorkflowStatesResponse{
		PageInfo: toProtoPageInfo(getMap(states, "pageInfo")),
	}
	for _, n := range nodes {
		m := toMap(n)
		out.WorkflowStates = append(out.WorkflowStates, &WorkflowState{
			Id:       getString(m, "id"),
			Name:     getString(m, "name"),
			Color:    getString(m, "color"),
			Type:     getString(m, "type"),
			Position: getFloat32(m, "position"),
			TeamId:   getString(getMap(m, "team"), "id"),
		})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 21. GetViewer
// ---------------------------------------------------------------------------

func (c *Client) GetViewer(ctx context.Context, _ *GetViewerRequest) (*GetViewerResponse, error) {
	query := `query { viewer { id name displayName email active admin } }`
	data, err := c.do(ctx, query, nil)
	if err != nil {
		return nil, err
	}

	return &GetViewerResponse{
		Viewer: toProtoMember(getMap(data, "viewer")),
	}, nil
}

// ---------------------------------------------------------------------------
// GraphQL transport
// ---------------------------------------------------------------------------

func (c *Client) do(ctx context.Context, query string, variables map[string]any) (map[string]any, error) {
	body := map[string]any{"query": query}
	if len(variables) > 0 {
		body["variables"] = variables
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal linear request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("build linear request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("linear request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		requestID := strings.TrimSpace(resp.Header.Get("X-Request-Id"))

		// Detect rate limiting and surface retry-after header
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			return nil, fmt.Errorf(
				"linear rate limit exceeded: retry_after=%s request_id=%s",
				retryAfter,
				requestID,
			)
		}

		return nil, fmt.Errorf(
			"linear POST %s failed: status=%d request_id=%s response_bytes=%d",
			c.apiURL,
			resp.StatusCode,
			requestID,
			len(raw),
		)
	}

	// Limit response body to 10MB to prevent memory exhaustion
	const maxResponseSize = 10 * 1024 * 1024
	limitedBody := io.LimitReader(resp.Body, maxResponseSize)

	var result struct {
		Data   map[string]any `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(limitedBody).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode linear response: %w", err)
	}
	if len(result.Errors) > 0 {
		msgs := make([]string, 0, len(result.Errors))
		for _, e := range result.Errors {
			msg := strings.TrimSpace(e.Message)
			if msg == "" {
				msg = "(unknown error)"
			}
			msgs = append(msgs, msg)
		}
		return nil, fmt.Errorf("linear graphql errors: %s", strings.Join(msgs, "; "))
	}
	return result.Data, nil
}

// ---------------------------------------------------------------------------
// Proto conversion helpers
// ---------------------------------------------------------------------------

func toProtoIssue(m map[string]any) *Issue {
	if len(m) == 0 {
		return nil
	}

	stateMap := getMap(m, "state")
	assigneeMap := getMap(m, "assignee")
	teamMap := getMap(m, "team")
	projectMap := getMap(m, "project")
	cycleMap := getMap(m, "cycle")

	labelsMap := getMap(m, "labels")
	labelNodes := toSlice(getAny(labelsMap, "nodes"))
	var labelIDs, labelNames []string
	for _, ln := range labelNodes {
		lm := toMap(ln)
		if id := getString(lm, "id"); id != "" {
			labelIDs = append(labelIDs, id)
		}
		if name := getString(lm, "name"); name != "" {
			labelNames = append(labelNames, name)
		}
	}

	// Linear API returns priority as integer (0-4) and priorityLabel as string
	// Proto schema maps these as: priority (string), priority_label (float)
	// We convert integer priority to string, and keep priorityLabel as-is for proto compatibility
	priorityInt := getFloat32(m, "priority")
	priorityStr := fmt.Sprintf("%.0f", priorityInt)
	if priorityInt == 0 {
		priorityStr = "0" // No priority
	}

	return &Issue{
		Id:            getString(m, "id"),
		Identifier:    getString(m, "identifier"),
		Title:         getString(m, "title"),
		Description:   getString(m, "description"),
		Priority:      priorityStr,
		PriorityLabel: priorityInt, // Store numeric priority in priorityLabel
		StateName:     getString(stateMap, "name"),
		StateId:       getString(stateMap, "id"),
		AssigneeId:    getString(assigneeMap, "id"),
		AssigneeName:  getString(assigneeMap, "name"),
		TeamId:        getString(teamMap, "id"),
		TeamName:      getString(teamMap, "name"),
		ProjectId:     getString(projectMap, "id"),
		ProjectName:   getString(projectMap, "name"),
		LabelIds:      labelIDs,
		LabelNames:    labelNames,
		CycleId:       getString(cycleMap, "id"),
		CreatedAt:     getString(m, "createdAt"),
		UpdatedAt:     getString(m, "updatedAt"),
		Url:           getString(m, "url"),
	}
}

func toProtoProject(m map[string]any) *Project {
	if len(m) == 0 {
		return nil
	}
	return &Project{
		Id:          getString(m, "id"),
		Name:        getString(m, "name"),
		Description: getString(m, "description"),
		State:       getString(m, "state"),
		Progress:    getString(m, "progress"),
		TargetDate:  getString(m, "targetDate"),
		StartDate:   getString(m, "startDate"),
		CreatedAt:   getString(m, "createdAt"),
		UpdatedAt:   getString(m, "updatedAt"),
		Url:         getString(m, "url"),
	}
}

func toProtoProjectUpdate(m map[string]any) *ProjectUpdate {
	if len(m) == 0 {
		return nil
	}
	userMap := getMap(m, "user")
	return &ProjectUpdate{
		Id:        getString(m, "id"),
		Body:      getString(m, "body"),
		Health:    getString(m, "health"),
		UserId:    getString(userMap, "id"),
		UserName:  getString(userMap, "name"),
		CreatedAt: getString(m, "createdAt"),
		Url:       getString(m, "url"),
	}
}

func toProtoMember(m map[string]any) *Member {
	if len(m) == 0 {
		return nil
	}
	return &Member{
		Id:          getString(m, "id"),
		Name:        getString(m, "name"),
		DisplayName: getString(m, "displayName"),
		Email:       getString(m, "email"),
		Active:      getBool(m, "active"),
		Admin:       getBool(m, "admin"),
	}
}

func toProtoComment(m map[string]any) *CommentEntry {
	if len(m) == 0 {
		return nil
	}
	userMap := getMap(m, "user")
	return &CommentEntry{
		Id:        getString(m, "id"),
		Body:      getString(m, "body"),
		UserId:    getString(userMap, "id"),
		UserName:  getString(userMap, "name"),
		CreatedAt: getString(m, "createdAt"),
		UpdatedAt: getString(m, "updatedAt"),
		Url:       getString(m, "url"),
	}
}

func toProtoDocument(m map[string]any) *Document {
	if len(m) == 0 {
		return nil
	}
	return &Document{
		Id:        getString(m, "id"),
		Title:     getString(m, "title"),
		Content:   getString(m, "content"),
		Icon:      getString(m, "icon"),
		Color:     getString(m, "color"),
		ProjectId: getString(getMap(m, "project"), "id"),
		CreatedAt: getString(m, "createdAt"),
		UpdatedAt: getString(m, "updatedAt"),
		Url:       getString(m, "url"),
	}
}

func toProtoPageInfo(m map[string]any) *PageInfo {
	if len(m) == 0 {
		return nil
	}
	return &PageInfo{
		HasNextPage: getBool(m, "hasNextPage"),
		EndCursor:   getString(m, "endCursor"),
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
	case float32:
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

func getFloat32(m map[string]any, key string) float32 {
	return float32(getFloat64(m, key))
}

func toMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func toSlice(v any) []any {
	s, _ := v.([]any)
	return s
}

func toStringSlice(v any) []string {
	items := toSlice(v)
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		str, ok := item.(string)
		if ok {
			out = append(out, str)
		}
	}
	return out
}

// pageSize returns the given value if positive, otherwise defaultPageSize.
func pageSize(first int32) int {
	if first > 0 {
		return int(first)
	}
	return defaultPageSize
}
