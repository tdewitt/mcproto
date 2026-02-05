package jira

import (
	"bytes"
	"context"
	"encoding/base64"
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
	defaultTimeout         = 15 * time.Second
	defaultSearchMaxResult = 50
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	email      string
	apiToken   string
}

type SearchOptions struct {
	Fields     []string
	StartAt    int
	MaxResults int
	Expand     []string
}

type SearchResult struct {
	Issues []*Issue `json:"issues"`
	Total  int      `json:"total"`
}

type issueWire struct {
	ID     string         `json:"id"`
	Key    string         `json:"key"`
	Self   string         `json:"self"`
	Fields map[string]any `json:"fields"`
}

func NewClient() (*Client, error) {
	baseURL := strings.TrimSpace(os.Getenv("JIRA_URL"))
	email := strings.TrimSpace(os.Getenv("JIRA_EMAIL"))
	token := strings.TrimSpace(os.Getenv("JIRA_API_TOKEN"))
	if baseURL == "" || email == "" || token == "" {
		return nil, fmt.Errorf("JIRA_URL, JIRA_EMAIL, and JIRA_API_TOKEN required")
	}
	return NewClientWithConfig(baseURL, email, token, &http.Client{Timeout: defaultTimeout})
}

func NewClientWithConfig(baseURL, email, token string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("jira base URL is required")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("invalid JIRA_URL: %w", err)
	}
	if strings.TrimSpace(email) == "" || strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("jira email and api token are required")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
		email:      email,
		apiToken:   token,
	}, nil
}

func (c *Client) SearchIssues(ctx context.Context, jql string, opts SearchOptions) (*SearchResult, error) {
	jql = strings.TrimSpace(jql)
	if jql == "" {
		return nil, fmt.Errorf("jql is required")
	}

	values := url.Values{}
	values.Set("jql", jql)
	if len(opts.Fields) > 0 {
		values.Set("fields", strings.Join(opts.Fields, ","))
	}
	if len(opts.Expand) > 0 {
		values.Set("expand", strings.Join(opts.Expand, ","))
	}
	if opts.StartAt > 0 {
		values.Set("startAt", strconv.Itoa(opts.StartAt))
	}
	max := opts.MaxResults
	if max <= 0 {
		max = defaultSearchMaxResult
	}
	values.Set("maxResults", strconv.Itoa(max))

	var raw struct {
		Issues []issueWire `json:"issues"`
		Total  int         `json:"total"`
	}
	if err := c.do(ctx, http.MethodGet, "/rest/api/3/search?"+values.Encode(), nil, &raw); err != nil {
		return nil, err
	}

	issues := make([]*Issue, 0, len(raw.Issues))
	for _, i := range raw.Issues {
		issues = append(issues, toProtoIssue(i))
	}
	return &SearchResult{
		Issues: issues,
		Total:  raw.Total,
	}, nil
}

func (c *Client) GetIssue(ctx context.Context, key string) (*Issue, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("issue key is required")
	}

	var raw issueWire
	if err := c.do(ctx, http.MethodGet, "/rest/api/3/issue/"+url.PathEscape(key), nil, &raw); err != nil {
		return nil, err
	}
	return toProtoIssue(raw), nil
}

func (c *Client) CreateIssue(ctx context.Context, req *CreateIssueRequest) (*CreateIssueResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create issue request is required")
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

	fields := map[string]any{
		"project": map[string]any{
			"key": req.GetProjectKey(),
		},
		"issuetype": map[string]any{
			"name": req.GetIssueType(),
		},
		"summary": req.GetSummary(),
	}
	if desc := strings.TrimSpace(req.GetDescription()); desc != "" {
		fields["description"] = toADF(desc)
	}
	if assignee := strings.TrimSpace(req.GetAssigneeAccountId()); assignee != "" {
		fields["assignee"] = map[string]any{"accountId": assignee}
	}
	if priority := strings.TrimSpace(req.GetPriority()); priority != "" {
		fields["priority"] = map[string]any{"name": priority}
	}
	if len(req.GetLabels()) > 0 {
		fields["labels"] = req.GetLabels()
	}
	for k, v := range req.GetCustomFields() {
		if strings.TrimSpace(k) == "" {
			continue
		}
		fields[k] = v
	}

	payload := map[string]any{"fields": fields}
	var resp CreateIssueResponse
	if err := c.do(ctx, http.MethodPost, "/rest/api/3/issue", payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UpdateIssue(ctx context.Context, req *UpdateIssueRequest) (*UpdateIssueResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("update issue request is required")
	}
	if strings.TrimSpace(req.GetIssueKey()) == "" {
		return nil, fmt.Errorf("issue_key is required")
	}

	fields := map[string]any{}
	if summary := strings.TrimSpace(req.GetSummary()); summary != "" {
		fields["summary"] = summary
	}
	if desc := strings.TrimSpace(req.GetDescription()); desc != "" {
		fields["description"] = toADF(desc)
	}
	if assignee := strings.TrimSpace(req.GetAssigneeAccountId()); assignee != "" {
		fields["assignee"] = map[string]any{"accountId": assignee}
	}
	if priority := strings.TrimSpace(req.GetPriority()); priority != "" {
		fields["priority"] = map[string]any{"name": priority}
	}
	if labels := req.GetLabels(); len(labels) > 0 {
		fields["labels"] = labels
	}
	for k, v := range req.GetCustomFields() {
		if strings.TrimSpace(k) == "" {
			continue
		}
		fields[k] = v
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf("at least one field is required for update")
	}

	notifyUsers := true
	if req.GetNotifyUsers() != nil {
		notifyUsers = req.GetNotifyUsers().GetValue()
	}

	payload := map[string]any{
		"fields":      fields,
		"notifyUsers": notifyUsers,
	}
	path := "/rest/api/3/issue/" + url.PathEscape(req.GetIssueKey())
	if err := c.do(ctx, http.MethodPut, path, payload, nil); err != nil {
		return nil, err
	}
	return &UpdateIssueResponse{Success: true}, nil
}

func (c *Client) TransitionIssue(ctx context.Context, req *TransitionIssueRequest) (*TransitionIssueResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("transition request is required")
	}
	if strings.TrimSpace(req.GetIssueKey()) == "" {
		return nil, fmt.Errorf("issue_key is required")
	}
	if strings.TrimSpace(req.GetTransitionId()) == "" {
		return nil, fmt.Errorf("transition_id is required")
	}

	payload := map[string]any{
		"transition": map[string]any{
			"id": req.GetTransitionId(),
		},
	}

	if len(req.GetFields()) > 0 {
		payload["fields"] = req.GetFields()
	}

	if comment := strings.TrimSpace(req.GetComment()); comment != "" {
		payload["update"] = map[string]any{
			"comment": []map[string]any{
				{
					"add": map[string]any{
						"body": toADF(comment),
					},
				},
			},
		}
	}

	path := "/rest/api/3/issue/" + url.PathEscape(req.GetIssueKey()) + "/transitions"
	if err := c.do(ctx, http.MethodPost, path, payload, nil); err != nil {
		return nil, err
	}
	return &TransitionIssueResponse{Success: true}, nil
}

func (c *Client) AddComment(ctx context.Context, req *AddCommentRequest) (*AddCommentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("add comment request is required")
	}
	if strings.TrimSpace(req.GetIssueKey()) == "" {
		return nil, fmt.Errorf("issue_key is required")
	}
	if strings.TrimSpace(req.GetBody()) == "" {
		return nil, fmt.Errorf("body is required")
	}

	payload := map[string]any{
		"body": toADF(req.GetBody()),
	}
	if vis := req.GetVisibility(); vis != nil {
		typ := strings.TrimSpace(vis.GetType())
		value := strings.TrimSpace(vis.GetValue())
		if typ != "" && value != "" {
			payload["visibility"] = map[string]any{
				"type":  typ,
				"value": value,
			}
		}
	}

	path := "/rest/api/3/issue/" + url.PathEscape(req.GetIssueKey()) + "/comment"
	var resp AddCommentResponse
	if err := c.do(ctx, http.MethodPost, path, payload, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) AssignIssue(ctx context.Context, req *AssignIssueRequest) (*AssignIssueResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("assign issue request is required")
	}
	if strings.TrimSpace(req.GetIssueKey()) == "" {
		return nil, fmt.Errorf("issue_key is required")
	}

	payload := map[string]any{
		"accountId": strings.TrimSpace(req.GetAccountId()),
	}

	path := "/rest/api/3/issue/" + url.PathEscape(req.GetIssueKey()) + "/assignee"
	if err := c.do(ctx, http.MethodPut, path, payload, nil); err != nil {
		return nil, err
	}
	return &AssignIssueResponse{Success: true}, nil
}

func (c *Client) GetTransitions(ctx context.Context, key string) ([]*Transition, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("issue key is required")
	}

	path := "/rest/api/3/issue/" + url.PathEscape(key) + "/transitions"
	var resp struct {
		Transitions []*Transition `json:"transitions"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Transitions, nil
}

func (c *Client) SearchUsers(ctx context.Context, query string, maxResults int32) ([]*User, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	values := url.Values{}
	values.Set("query", query)
	if maxResults > 0 {
		values.Set("maxResults", strconv.Itoa(int(maxResults)))
	}

	path := "/rest/api/3/user/search?" + values.Encode()
	var users []*User
	if err := c.do(ctx, http.MethodGet, path, nil, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (c *Client) GetServiceDesks(ctx context.Context, req *GetServiceDesksRequest) (*GetServiceDesksResponse, error) {
	if req == nil {
		req = &GetServiceDesksRequest{}
	}
	path := "/rest/servicedeskapi/servicedesk" + withPagination(req.GetStart(), req.GetLimit())

	var raw struct {
		Values     []map[string]any `json:"values"`
		Size       int32            `json:"size"`
		Start      int32            `json:"start"`
		Limit      int32            `json:"limit"`
		IsLastPage bool             `json:"isLastPage"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	out := &GetServiceDesksResponse{
		Size:       raw.Size,
		Start:      raw.Start,
		Limit:      raw.Limit,
		IsLastPage: raw.IsLastPage,
	}
	for _, item := range raw.Values {
		project := getMap(item, "project")
		out.ServiceDesks = append(out.ServiceDesks, &ServiceDesk{
			Id:          getString(item, "id"),
			ProjectId:   getString(project, "id"),
			ProjectName: getString(project, "name"),
			ProjectKey:  getString(project, "key"),
		})
	}
	return out, nil
}

func (c *Client) GetRequestTypes(ctx context.Context, req *GetRequestTypesRequest) (*GetRequestTypesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get request types request is required")
	}
	if strings.TrimSpace(req.GetServiceDeskId()) == "" {
		return nil, fmt.Errorf("service_desk_id is required")
	}
	path := "/rest/servicedeskapi/servicedesk/" + url.PathEscape(req.GetServiceDeskId()) + "/requesttype" + withPagination(req.GetStart(), req.GetLimit())

	var raw struct {
		Values     []map[string]any `json:"values"`
		Size       int32            `json:"size"`
		Start      int32            `json:"start"`
		Limit      int32            `json:"limit"`
		IsLastPage bool             `json:"isLastPage"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	out := &GetRequestTypesResponse{
		Size:       raw.Size,
		Start:      raw.Start,
		Limit:      raw.Limit,
		IsLastPage: raw.IsLastPage,
	}
	for _, item := range raw.Values {
		out.RequestTypes = append(out.RequestTypes, &RequestType{
			Id:            getString(item, "id"),
			Name:          getString(item, "name"),
			Description:   getString(item, "description"),
			HelpText:      getString(item, "helpText"),
			ServiceDeskId: getString(item, "serviceDeskId"),
			GroupIds:      toStringSlice(getAny(item, "groupIds")),
		})
	}
	return out, nil
}

func (c *Client) CreateRequest(ctx context.Context, req *CreateRequestRequest) (*CreateRequestResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("create request is required")
	}
	if strings.TrimSpace(req.GetServiceDeskId()) == "" {
		return nil, fmt.Errorf("service_desk_id is required")
	}
	if strings.TrimSpace(req.GetRequestTypeId()) == "" {
		return nil, fmt.Errorf("request_type_id is required")
	}
	if len(req.GetRequestFieldValues()) == 0 {
		return nil, fmt.Errorf("request_field_values is required")
	}

	payload := map[string]any{
		"serviceDeskId":      req.GetServiceDeskId(),
		"requestTypeId":      req.GetRequestTypeId(),
		"requestFieldValues": req.GetRequestFieldValues(),
	}
	if participants := req.GetRequestParticipants(); len(participants) > 0 {
		payload["requestParticipants"] = participants
	}
	if req.GetRaiseOnBehalfOf() && strings.TrimSpace(req.GetRaiseOnBehalfOfAccountId()) != "" {
		payload["raiseOnBehalfOf"] = req.GetRaiseOnBehalfOfAccountId()
	}

	var raw map[string]any
	if err := c.do(ctx, http.MethodPost, "/rest/servicedeskapi/request", payload, &raw); err != nil {
		return nil, err
	}
	return &CreateRequestResponse{
		IssueId:       getString(raw, "issueId"),
		IssueKey:      getString(raw, "issueKey"),
		RequestTypeId: getString(raw, "requestTypeId"),
		ServiceDeskId: getString(raw, "serviceDeskId"),
		CurrentStatus: toCustomerRequestStatus(getMap(raw, "currentStatus")),
	}, nil
}

func (c *Client) GetRequest(ctx context.Context, req *GetRequestRequest) (*GetRequestResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get request request is required")
	}
	if strings.TrimSpace(req.GetIssueIdOrKey()) == "" {
		return nil, fmt.Errorf("issue_id_or_key is required")
	}

	path := "/rest/servicedeskapi/request/" + url.PathEscape(req.GetIssueIdOrKey())
	var raw map[string]any
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	participants := []*User{}
	for _, p := range toSlice(getAny(raw, "requestParticipants")) {
		participants = append(participants, toProtoUser(toMap(p)))
	}
	return &GetRequestResponse{
		Request: &CustomerRequest{
			IssueId:             getString(raw, "issueId"),
			IssueKey:            getString(raw, "issueKey"),
			RequestTypeId:       getString(raw, "requestTypeId"),
			ServiceDeskId:       getString(raw, "serviceDeskId"),
			CurrentStatus:       toCustomerRequestStatus(getMap(raw, "currentStatus")),
			RequestParticipants: participants,
		},
	}, nil
}

func (c *Client) AddRequestComment(ctx context.Context, req *AddRequestCommentRequest) (*AddRequestCommentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("add request comment request is required")
	}
	if strings.TrimSpace(req.GetIssueIdOrKey()) == "" {
		return nil, fmt.Errorf("issue_id_or_key is required")
	}
	if strings.TrimSpace(req.GetBody()) == "" {
		return nil, fmt.Errorf("body is required")
	}

	payload := map[string]any{
		"body":   req.GetBody(),
		"public": req.GetPublic(),
	}
	path := "/rest/servicedeskapi/request/" + url.PathEscape(req.GetIssueIdOrKey()) + "/comment"
	var raw map[string]any
	if err := c.do(ctx, http.MethodPost, path, payload, &raw); err != nil {
		return nil, err
	}
	return &AddRequestCommentResponse{
		Id:      getString(raw, "id"),
		Body:    getString(raw, "body"),
		Public:  getBool(raw, "public"),
		Author:  toProtoUser(getMap(raw, "author")),
		Created: getString(raw, "created"),
	}, nil
}

func (c *Client) GetOrganizations(ctx context.Context, req *GetOrganizationsRequest) (*GetOrganizationsResponse, error) {
	if req == nil {
		req = &GetOrganizationsRequest{}
	}
	path := "/rest/servicedeskapi/organization" + withPagination(req.GetStart(), req.GetLimit())

	var raw struct {
		Values     []map[string]any `json:"values"`
		Size       int32            `json:"size"`
		Start      int32            `json:"start"`
		Limit      int32            `json:"limit"`
		IsLastPage bool             `json:"isLastPage"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	out := &GetOrganizationsResponse{
		Size:       raw.Size,
		Start:      raw.Start,
		Limit:      raw.Limit,
		IsLastPage: raw.IsLastPage,
	}
	for _, item := range raw.Values {
		out.Organizations = append(out.Organizations, &Organization{
			Id:   getString(item, "id"),
			Name: getString(item, "name"),
		})
	}
	return out, nil
}

func (c *Client) GetCustomers(ctx context.Context, req *GetCustomersRequest) (*GetCustomersResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get customers request is required")
	}
	if strings.TrimSpace(req.GetOrganizationId()) == "" {
		return nil, fmt.Errorf("organization_id is required")
	}
	path := "/rest/servicedeskapi/organization/" + url.PathEscape(req.GetOrganizationId()) + "/user" + withPagination(req.GetStart(), req.GetLimit())

	var raw struct {
		Values     []map[string]any `json:"values"`
		Size       int32            `json:"size"`
		Start      int32            `json:"start"`
		Limit      int32            `json:"limit"`
		IsLastPage bool             `json:"isLastPage"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	out := &GetCustomersResponse{
		Size:       raw.Size,
		Start:      raw.Start,
		Limit:      raw.Limit,
		IsLastPage: raw.IsLastPage,
	}
	for _, item := range raw.Values {
		out.Users = append(out.Users, toProtoUser(item))
	}
	return out, nil
}

func (c *Client) GetSlaInfo(ctx context.Context, req *GetSlaInfoRequest) (*GetSlaInfoResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("get sla info request is required")
	}
	if strings.TrimSpace(req.GetIssueIdOrKey()) == "" {
		return nil, fmt.Errorf("issue_id_or_key is required")
	}
	path := "/rest/servicedeskapi/request/" + url.PathEscape(req.GetIssueIdOrKey()) + "/sla"

	var raw struct {
		Values []map[string]any `json:"values"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &raw); err != nil {
		return nil, err
	}

	out := &GetSlaInfoResponse{}
	for _, item := range raw.Values {
		out.Values = append(out.Values, &SlaInfo{
			Id:              getString(item, "id"),
			Name:            getString(item, "name"),
			CompletedCycles: toCompletedCycles(getAny(item, "completedCycles")),
			OngoingCycle:    toOngoingCycle(getMap(item, "ongoingCycle")),
		})
	}
	return out, nil
}

func (c *Client) do(ctx context.Context, method, path string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal jira request: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build jira request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+basicToken(c.email, c.apiToken))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("jira request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8*1024))
		requestID := strings.TrimSpace(resp.Header.Get("X-AREQUESTID"))
		if requestID == "" {
			requestID = strings.TrimSpace(resp.Header.Get("X-Request-Id"))
		}
		return fmt.Errorf(
			"jira %s %s failed: status=%d request_id=%s response_bytes=%d",
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

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decode jira response: %w", err)
	}
	return nil
}

func basicToken(email, token string) string {
	return base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
}

func protoStructFromMap(m map[string]any) *structpb.Struct {
	if len(m) == 0 {
		return nil
	}
	out, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return out
}

func toProtoIssue(raw issueWire) *Issue {
	fields := raw.Fields
	if fields == nil {
		fields = map[string]any{}
	}

	project := getMap(fields, "project")
	issueType := getMap(fields, "issuetype")
	status := getMap(fields, "status")
	priority := getMap(fields, "priority")
	assignee := getMap(fields, "assignee")
	reporter := getMap(fields, "reporter")

	labels := toStringSlice(getAny(fields, "labels"))
	custom := map[string]any{}
	for k, v := range fields {
		switch k {
		case "summary", "description", "issuetype", "status", "priority", "assignee", "reporter", "project", "labels", "created", "updated", "comment":
			continue
		default:
			custom[k] = v
		}
	}

	return &Issue{
		Id:   raw.ID,
		Key:  raw.Key,
		Self: raw.Self,
		Fields: &IssueFields{
			Summary:     getString(fields, "summary"),
			Description: stringify(getAny(fields, "description")),
			IssueType: &IssueType{
				Id:          getString(issueType, "id"),
				Name:        getString(issueType, "name"),
				Description: getString(issueType, "description"),
				Subtask:     getBool(issueType, "subtask"),
			},
			Status: &Status{
				Id:          getString(status, "id"),
				Name:        getString(status, "name"),
				Description: getString(status, "description"),
				Category: &StatusCategory{
					Id:   getString(getMap(status, "statusCategory"), "id"),
					Key:  getString(getMap(status, "statusCategory"), "key"),
					Name: getString(getMap(status, "statusCategory"), "name"),
				},
			},
			Priority: &Priority{
				Id:   getString(priority, "id"),
				Name: getString(priority, "name"),
			},
			Assignee:     toProtoUser(assignee),
			Reporter:     toProtoUser(reporter),
			Project:      toProtoProject(project),
			Labels:       labels,
			Created:      getString(fields, "created"),
			Updated:      getString(fields, "updated"),
			Comments:     toProtoComments(getMap(fields, "comment")),
			CustomFields: protoStructFromMap(custom),
		},
	}
}

func toProtoUser(raw map[string]any) *User {
	if len(raw) == 0 {
		return nil
	}
	return &User{
		AccountId:    getString(raw, "accountId"),
		DisplayName:  getString(raw, "displayName"),
		EmailAddress: getString(raw, "emailAddress"),
		Active:       getBool(raw, "active"),
	}
}

func toProtoProject(raw map[string]any) *Project {
	if len(raw) == 0 {
		return nil
	}
	return &Project{
		Id:   getString(raw, "id"),
		Key:  getString(raw, "key"),
		Name: getString(raw, "name"),
	}
}

func toProtoComments(raw map[string]any) []*Comment {
	comments := toSlice(raw["comments"])
	if len(comments) == 0 {
		return nil
	}
	out := make([]*Comment, 0, len(comments))
	for _, entry := range comments {
		m := toMap(entry)
		out = append(out, &Comment{
			Id:      getString(m, "id"),
			Body:    stringify(getAny(m, "body")),
			Author:  toProtoUser(getMap(m, "author")),
			Created: getString(m, "created"),
			Updated: getString(m, "updated"),
		})
	}
	return out
}

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

func getInt64(m map[string]any, key string) int64 {
	v := getAny(m, key)
	switch t := v.(type) {
	case int64:
		return t
	case int32:
		return int64(t)
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case json.Number:
		out, _ := t.Int64()
		return out
	default:
		return 0
	}
}

func stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
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

func toCustomerRequestStatus(raw map[string]any) *CustomerRequestStatus {
	if len(raw) == 0 {
		return nil
	}
	return &CustomerRequestStatus{
		Status:         getString(raw, "status"),
		StatusCategory: getString(raw, "statusCategory"),
		StatusDate:     getString(raw, "statusDate"),
	}
}

func toCompletedCycles(raw any) []*CompletedCycle {
	items := toSlice(raw)
	if len(items) == 0 {
		if m := toMap(raw); len(m) != 0 {
			items = []any{m}
		}
	}
	if len(items) == 0 {
		return nil
	}

	out := make([]*CompletedCycle, 0, len(items))
	for _, item := range items {
		m := toMap(item)
		if len(m) == 0 {
			continue
		}
		out = append(out, &CompletedCycle{
			StartTime:     getString(m, "startTime"),
			StopTime:      getString(m, "stopTime"),
			Breached:      getBool(m, "breached"),
			GoalDuration:  toDuration(getMap(m, "goalDuration")),
			ElapsedTime:   toDuration(getMap(m, "elapsedTime")),
			RemainingTime: toDuration(getMap(m, "remainingTime")),
		})
	}
	return out
}

func toOngoingCycle(raw map[string]any) *OngoingCycle {
	if len(raw) == 0 {
		return nil
	}
	return &OngoingCycle{
		StartTime:           getString(raw, "startTime"),
		Breached:            getBool(raw, "breached"),
		Paused:              getBool(raw, "paused"),
		WithinCalendarHours: getBool(raw, "withinCalendarHours"),
		GoalDuration:        toDuration(getMap(raw, "goalDuration")),
		ElapsedTime:         toDuration(getMap(raw, "elapsedTime")),
		RemainingTime:       toDuration(getMap(raw, "remainingTime")),
	}
}

func toDuration(raw map[string]any) *Duration {
	if len(raw) == 0 {
		return nil
	}
	return &Duration{
		Millis:   getInt64(raw, "millis"),
		Friendly: getString(raw, "friendly"),
	}
}

func withPagination(start, limit int32) string {
	values := url.Values{}
	if start > 0 {
		values.Set("start", strconv.Itoa(int(start)))
	}
	if limit > 0 {
		values.Set("limit", strconv.Itoa(int(limit)))
	}
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}

func toADF(text string) any {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return trimmed
	}

	var parsed any
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			return parsed
		}
	}

	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []map[string]any{
			{
				"type": "paragraph",
				"content": []map[string]any{
					{
						"type": "text",
						"text": text,
					},
				},
			},
		},
	}
}
