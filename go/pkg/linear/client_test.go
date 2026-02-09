package linear

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// newTestClient creates a Client pointing at the given httptest.Server.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	client, err := NewClientWithConfig("lin_api_test_key", srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	client.SetAPIURL(srv.URL + "/graphql")
	return client
}

// mockIssueNode returns a realistic GraphQL issue node for use in mock
// responses. All fields that toProtoIssue inspects are populated.
func mockIssueNode() map[string]any {
	return map[string]any{
		"id":            "uuid-1",
		"identifier":    "ENG-123",
		"title":         "Test Issue",
		"description":   "A test description",
		"priority":      float64(2),
		"priorityLabel": float64(2),
		"state":         map[string]any{"id": "state-1", "name": "In Progress"},
		"assignee":      map[string]any{"id": "user-1", "name": "Test User"},
		"team":          map[string]any{"id": "team-1", "name": "Engineering"},
		"project":       map[string]any{"id": "proj-1", "name": "Test Project"},
		"labels": map[string]any{
			"nodes": []any{
				map[string]any{"id": "label-1", "name": "Bug"},
			},
		},
		"cycle":     map[string]any{"id": "cycle-1"},
		"createdAt": "2024-01-01T00:00:00Z",
		"updatedAt": "2024-01-02T00:00:00Z",
		"url":       "https://linear.app/team/ENG-123",
	}
}

// ---------------------------------------------------------------------------
// 1. NewClient validation
// ---------------------------------------------------------------------------

func TestNewClient_RequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := NewClientWithConfig("", nil)
	if err == nil {
		t.Fatal("expected error for empty API key")
	}
	if !strings.Contains(err.Error(), "api key is required") {
		t.Fatalf("unexpected error message: %v", err)
	}

	// Whitespace-only should also be rejected.
	_, err = NewClientWithConfig("   ", nil)
	if err == nil {
		t.Fatal("expected error for whitespace-only API key")
	}
}

// ---------------------------------------------------------------------------
// 2. ListIssues
// ---------------------------------------------------------------------------

func TestListIssues_ParsesResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var gqlReq struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"issues": map[string]any{
					"nodes": []any{mockIssueNode()},
					"pageInfo": map[string]any{
						"hasNextPage": true,
						"endCursor":   "cursor-abc",
					},
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.ListIssues(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if len(resp.GetIssues()) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(resp.GetIssues()))
	}
	issue := resp.GetIssues()[0]
	if issue.GetIdentifier() != "ENG-123" {
		t.Fatalf("unexpected identifier: %s", issue.GetIdentifier())
	}
	if issue.GetTitle() != "Test Issue" {
		t.Fatalf("unexpected title: %s", issue.GetTitle())
	}
	if !resp.GetPageInfo().GetHasNextPage() {
		t.Fatal("expected hasNextPage=true")
	}
	if resp.GetPageInfo().GetEndCursor() != "cursor-abc" {
		t.Fatalf("unexpected endCursor: %s", resp.GetPageInfo().GetEndCursor())
	}
}

// ---------------------------------------------------------------------------
// 3. GetIssue
// ---------------------------------------------------------------------------

func TestGetIssue_ReturnsIssue(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var gqlReq struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if gqlReq.Variables["id"] != "ENG-123" {
			t.Fatalf("unexpected id variable: %v", gqlReq.Variables["id"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"issue": mockIssueNode(),
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.GetIssue(context.Background(), &GetIssueRequest{Id: "ENG-123"})
	if err != nil {
		t.Fatalf("GetIssue failed: %v", err)
	}
	issue := resp.GetIssue()
	if issue == nil {
		t.Fatal("expected non-nil issue")
	}
	if issue.GetId() != "uuid-1" {
		t.Fatalf("unexpected id: %s", issue.GetId())
	}
	if issue.GetIdentifier() != "ENG-123" {
		t.Fatalf("unexpected identifier: %s", issue.GetIdentifier())
	}
	if issue.GetTitle() != "Test Issue" {
		t.Fatalf("unexpected title: %s", issue.GetTitle())
	}
	if issue.GetDescription() != "A test description" {
		t.Fatalf("unexpected description: %s", issue.GetDescription())
	}
	if issue.GetPriority() != "2" {
		t.Fatalf("unexpected priority: %s", issue.GetPriority())
	}
	if issue.GetPriorityLabel() != 2.0 {
		t.Fatalf("unexpected priorityLabel: %f", issue.GetPriorityLabel())
	}
	if issue.GetStateName() != "In Progress" {
		t.Fatalf("unexpected state name: %s", issue.GetStateName())
	}
	if issue.GetStateId() != "state-1" {
		t.Fatalf("unexpected state id: %s", issue.GetStateId())
	}
	if issue.GetAssigneeId() != "user-1" {
		t.Fatalf("unexpected assignee id: %s", issue.GetAssigneeId())
	}
	if issue.GetAssigneeName() != "Test User" {
		t.Fatalf("unexpected assignee name: %s", issue.GetAssigneeName())
	}
	if issue.GetTeamId() != "team-1" {
		t.Fatalf("unexpected team id: %s", issue.GetTeamId())
	}
	if issue.GetTeamName() != "Engineering" {
		t.Fatalf("unexpected team name: %s", issue.GetTeamName())
	}
	if issue.GetProjectId() != "proj-1" {
		t.Fatalf("unexpected project id: %s", issue.GetProjectId())
	}
	if issue.GetProjectName() != "Test Project" {
		t.Fatalf("unexpected project name: %s", issue.GetProjectName())
	}
	if len(issue.GetLabelIds()) != 1 || issue.GetLabelIds()[0] != "label-1" {
		t.Fatalf("unexpected label ids: %v", issue.GetLabelIds())
	}
	if len(issue.GetLabelNames()) != 1 || issue.GetLabelNames()[0] != "Bug" {
		t.Fatalf("unexpected label names: %v", issue.GetLabelNames())
	}
	if issue.GetCycleId() != "cycle-1" {
		t.Fatalf("unexpected cycle id: %s", issue.GetCycleId())
	}
	if issue.GetCreatedAt() != "2024-01-01T00:00:00Z" {
		t.Fatalf("unexpected createdAt: %s", issue.GetCreatedAt())
	}
	if issue.GetUpdatedAt() != "2024-01-02T00:00:00Z" {
		t.Fatalf("unexpected updatedAt: %s", issue.GetUpdatedAt())
	}
	if issue.GetUrl() != "https://linear.app/team/ENG-123" {
		t.Fatalf("unexpected url: %s", issue.GetUrl())
	}
}

// ---------------------------------------------------------------------------
// 4. CreateIssue - mutation body
// ---------------------------------------------------------------------------

func TestCreateIssue_SendsMutation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		var gqlReq struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if !strings.Contains(gqlReq.Query, "issueCreate") {
			t.Fatalf("expected issueCreate mutation, got: %s", gqlReq.Query)
		}

		input, ok := gqlReq.Variables["input"].(map[string]any)
		if !ok {
			t.Fatalf("expected input variable to be a map, got: %T", gqlReq.Variables["input"])
		}
		if input["teamId"] != "team-1" {
			t.Fatalf("unexpected teamId: %v", input["teamId"])
		}
		if input["title"] != "New Feature" {
			t.Fatalf("unexpected title: %v", input["title"])
		}
		if input["description"] != "Implement the thing" {
			t.Fatalf("unexpected description: %v", input["description"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"issueCreate": map[string]any{
					"success": true,
					"issue":   mockIssueNode(),
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.CreateIssue(context.Background(), &CreateIssueRequest{
		TeamId:      "team-1",
		Title:       "New Feature",
		Description: "Implement the thing",
	})
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if !resp.GetSuccess() {
		t.Fatal("expected success=true")
	}
	if resp.GetIssue() == nil {
		t.Fatal("expected non-nil issue in response")
	}
}

// ---------------------------------------------------------------------------
// 5. CreateIssue - validation
// ---------------------------------------------------------------------------

func TestCreateIssue_RequiresTeamAndTitle(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-key", &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	// Missing both team_id and title.
	_, err = client.CreateIssue(context.Background(), &CreateIssueRequest{})
	if err == nil {
		t.Fatal("expected error for missing team_id")
	}
	if !strings.Contains(err.Error(), "team_id is required") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Has team_id but missing title.
	_, err = client.CreateIssue(context.Background(), &CreateIssueRequest{TeamId: "team-1"})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nil request.
	_, err = client.CreateIssue(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ---------------------------------------------------------------------------
// 6. UpdateIssue - requires at least one field
// ---------------------------------------------------------------------------

func TestUpdateIssue_RequiresAtLeastOneField(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-key", &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.UpdateIssue(context.Background(), &UpdateIssueRequest{Id: "uuid-1"})
	if err == nil {
		t.Fatal("expected validation error for empty update fields")
	}
	if !strings.Contains(err.Error(), "at least one field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 7. DeleteIssue - requires id
// ---------------------------------------------------------------------------

func TestDeleteIssue_RequiresId(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-key", &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.DeleteIssue(context.Background(), &DeleteIssueRequest{})
	if err == nil {
		t.Fatal("expected error for empty id")
	}
	if !strings.Contains(err.Error(), "id is required") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nil request.
	_, err = client.DeleteIssue(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ---------------------------------------------------------------------------
// 8. SearchIssues - requires query
// ---------------------------------------------------------------------------

func TestSearchIssues_RequiresQuery(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-key", &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.SearchIssues(context.Background(), &SearchIssuesRequest{})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
	if !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nil request.
	_, err = client.SearchIssues(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ---------------------------------------------------------------------------
// 9. ListTeams
// ---------------------------------------------------------------------------

func TestListTeams_ParsesResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"teams": map[string]any{
					"nodes": []any{
						map[string]any{
							"id":          "team-1",
							"name":        "Engineering",
							"key":         "ENG",
							"description": "The engineering team",
						},
						map[string]any{
							"id":          "team-2",
							"name":        "Design",
							"key":         "DES",
							"description": "The design team",
						},
					},
					"pageInfo": map[string]any{
						"hasNextPage": false,
						"endCursor":   "",
					},
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.ListTeams(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTeams failed: %v", err)
	}
	if len(resp.GetTeams()) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(resp.GetTeams()))
	}
	if resp.GetTeams()[0].GetName() != "Engineering" {
		t.Fatalf("unexpected first team name: %s", resp.GetTeams()[0].GetName())
	}
	if resp.GetTeams()[0].GetKey() != "ENG" {
		t.Fatalf("unexpected first team key: %s", resp.GetTeams()[0].GetKey())
	}
	if resp.GetTeams()[1].GetName() != "Design" {
		t.Fatalf("unexpected second team name: %s", resp.GetTeams()[1].GetName())
	}
	if resp.GetPageInfo().GetHasNextPage() {
		t.Fatal("expected hasNextPage=false")
	}
}

// ---------------------------------------------------------------------------
// 10. GetViewer
// ---------------------------------------------------------------------------

func TestGetViewer_ParsesResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"viewer": map[string]any{
					"id":          "viewer-1",
					"name":        "Tucker DeWitt",
					"displayName": "Tucker",
					"email":       "tucker@example.com",
					"active":      true,
					"admin":       false,
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	resp, err := client.GetViewer(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetViewer failed: %v", err)
	}
	viewer := resp.GetViewer()
	if viewer == nil {
		t.Fatal("expected non-nil viewer")
	}
	if viewer.GetId() != "viewer-1" {
		t.Fatalf("unexpected viewer id: %s", viewer.GetId())
	}
	if viewer.GetName() != "Tucker DeWitt" {
		t.Fatalf("unexpected viewer name: %s", viewer.GetName())
	}
	if viewer.GetDisplayName() != "Tucker" {
		t.Fatalf("unexpected viewer displayName: %s", viewer.GetDisplayName())
	}
	if viewer.GetEmail() != "tucker@example.com" {
		t.Fatalf("unexpected viewer email: %s", viewer.GetEmail())
	}
	if !viewer.GetActive() {
		t.Fatal("expected viewer active=true")
	}
	if viewer.GetAdmin() {
		t.Fatal("expected viewer admin=false")
	}
}

// ---------------------------------------------------------------------------
// 11. AddComment - validation
// ---------------------------------------------------------------------------

func TestAddComment_RequiresIssueIdAndBody(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-key", &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	// Missing both.
	_, err = client.AddComment(context.Background(), &AddCommentRequest{})
	if err == nil {
		t.Fatal("expected error for missing issue_id")
	}
	if !strings.Contains(err.Error(), "issue_id is required") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Has issue_id but missing body.
	_, err = client.AddComment(context.Background(), &AddCommentRequest{IssueId: "uuid-1"})
	if err == nil {
		t.Fatal("expected error for missing body")
	}
	if !strings.Contains(err.Error(), "body is required") {
		t.Fatalf("unexpected error: %v", err)
	}

	// Nil request.
	_, err = client.AddComment(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil request")
	}
}

// ---------------------------------------------------------------------------
// 12. Error does not leak response body
// ---------------------------------------------------------------------------

func TestLinearErrorDoesNotLeakResponseBody(t *testing.T) {
	t.Parallel()

	secret := "super-secret-linear-detail"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-456")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(secret))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.GetIssue(context.Background(), &GetIssueRequest{Id: "ENG-123"})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked response body: %v", err)
	}
	if !strings.Contains(err.Error(), "request_id=req-456") {
		t.Fatalf("error missing request id context: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 13. GraphQL errors handled correctly
// ---------------------------------------------------------------------------

func TestLinearGraphQLErrors_HandledCorrectly(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": nil,
			"errors": []map[string]any{
				{"message": "Entity not found"},
				{"message": "Access denied"},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.GetIssue(context.Background(), &GetIssueRequest{Id: "bad-id"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "Entity not found") {
		t.Fatalf("expected 'Entity not found' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Access denied") {
		t.Fatalf("expected 'Access denied' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "linear graphql errors") {
		t.Fatalf("expected 'linear graphql errors' prefix, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 14. GraphQL null/empty error messages
// ---------------------------------------------------------------------------

func TestLinearGraphQLNullErrorMessage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": nil,
			"errors": []map[string]any{
				{"message": ""},
				{"message": "   "},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.ListIssues(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "(unknown error)") {
		t.Fatalf("expected '(unknown error)' for empty messages, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 15. Rate limit error
// ---------------------------------------------------------------------------

func TestLinearRateLimitError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-rate")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
	}))
	defer srv.Close()

	client := newTestClient(t, srv)
	_, err := client.ListIssues(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("expected rate limit error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "retry_after=30") {
		t.Fatalf("expected retry_after=30, got: %v", err)
	}
	if !strings.Contains(err.Error(), "request_id=req-rate") {
		t.Fatalf("expected request_id=req-rate, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 16. Default page size
// ---------------------------------------------------------------------------

func TestListIssues_DefaultPageSize(t *testing.T) {
	t.Parallel()

	var capturedFirst float64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var gqlReq struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&gqlReq); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if v, ok := gqlReq.Variables["first"].(float64); ok {
			capturedFirst = v
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"issues": map[string]any{
					"nodes":    []any{},
					"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
				},
			},
		})
	}))
	defer srv.Close()

	client := newTestClient(t, srv)

	// Call with First=0 (unset) to trigger the default.
	_, err := client.ListIssues(context.Background(), &ListIssuesRequest{First: 0})
	if err != nil {
		t.Fatalf("ListIssues failed: %v", err)
	}
	if capturedFirst != 50 {
		t.Fatalf("expected default page size 50, got %.0f", capturedFirst)
	}
}

// ---------------------------------------------------------------------------
// 17. Authorization header
// ---------------------------------------------------------------------------

func TestLinearAuthHeader(t *testing.T) {
	t.Parallel()

	var seenAuth string
	var seenContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		seenContentType = r.Header.Get("Content-Type")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"viewer": map[string]any{
					"id":          "viewer-1",
					"name":        "Test",
					"displayName": "Test",
					"email":       "test@example.com",
					"active":      true,
					"admin":       false,
				},
			},
		})
	}))
	defer srv.Close()

	apiKey := "lin_api_abc123xyz"
	client, err := NewClientWithConfig(apiKey, srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	client.SetAPIURL(srv.URL + "/graphql")

	_, err = client.GetViewer(context.Background(), nil)
	if err != nil {
		t.Fatalf("GetViewer failed: %v", err)
	}

	// Linear uses bare API key in Authorization header (not Bearer).
	if seenAuth != apiKey {
		t.Fatalf("expected Authorization=%q, got %q", apiKey, seenAuth)
	}
	if seenContentType != "application/json" {
		t.Fatalf("expected Content-Type=application/json, got %q", seenContentType)
	}
}
