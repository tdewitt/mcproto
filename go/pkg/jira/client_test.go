package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSearchIssues_ParsesIssueFields(t *testing.T) {
	t.Parallel()

	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.String(), "/rest/api/3/search?") {
			t.Fatalf("unexpected path: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"total": 1,
			"issues": []map[string]any{
				{
					"id":   "10001",
					"key":  "PROJ-1",
					"self": "https://jira.example/rest/api/3/issue/10001",
					"fields": map[string]any{
						"summary": "Example",
						"description": map[string]any{
							"type": "doc",
						},
						"labels":            []any{"one", "two"},
						"customfield_11111": "custom",
					},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig(srv.URL, "user@example.com", "token123", srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	res, err := client.SearchIssues(context.Background(), "project = PROJ", SearchOptions{})
	if err != nil {
		t.Fatalf("SearchIssues failed: %v", err)
	}
	if res.Total != 1 || len(res.Issues) != 1 {
		t.Fatalf("unexpected result: %#v", res)
	}
	if res.Issues[0].GetKey() != "PROJ-1" {
		t.Fatalf("unexpected issue key: %s", res.Issues[0].GetKey())
	}
	if res.Issues[0].GetFields().GetSummary() != "Example" {
		t.Fatalf("unexpected summary: %s", res.Issues[0].GetFields().GetSummary())
	}
	if res.Issues[0].GetFields().GetDescription() == "" {
		t.Fatal("expected non-empty description")
	}
	custom := res.Issues[0].GetFields().GetCustomFields()
	if custom == nil || custom.Fields["customfield_11111"].GetStringValue() != "custom" {
		t.Fatalf("expected custom field, got: %#v", custom)
	}

	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:token123"))
	if seenAuth != want {
		t.Fatalf("unexpected auth header: %s", seenAuth)
	}
}

func TestCreateIssue_SendsFieldsPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		fields := payload["fields"].(map[string]any)
		if fields["summary"] != "Need fix" {
			t.Fatalf("unexpected summary: %v", fields["summary"])
		}
		project := fields["project"].(map[string]any)
		if project["key"] != "PROJ" {
			t.Fatalf("unexpected project key: %v", project["key"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issue_key": "PROJ-99",
			"issue_id":  "99",
			"self":      "https://jira.example/rest/api/3/issue/99",
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig(srv.URL, "user@example.com", "token123", srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	resp, err := client.CreateIssue(context.Background(), &CreateIssueRequest{
		ProjectKey: "PROJ",
		IssueType:  "Task",
		Summary:    "Need fix",
	})
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if resp.GetIssueKey() != "PROJ-99" {
		t.Fatalf("unexpected issue key: %s", resp.GetIssueKey())
	}
}

func TestNewClientWithConfig_ValidatesInput(t *testing.T) {
	t.Parallel()

	if _, err := NewClientWithConfig("://bad-url", "user@example.com", "token", nil); err == nil {
		t.Fatal("expected invalid URL error")
	}
	if _, err := NewClientWithConfig("https://jira.example", "", "token", nil); err == nil {
		t.Fatal("expected missing credentials error")
	}
}

func TestUpdateIssue_RequiresAtLeastOneField(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("https://jira.example", "user@example.com", "token123", &http.Client{})
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.UpdateIssue(context.Background(), &UpdateIssueRequest{
		IssueKey: "PROJ-1",
	})
	if err == nil {
		t.Fatal("expected validation error for empty update fields")
	}
}

func TestUpdateIssue_DefaultNotifyUsersTrue(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if payload["notifyUsers"] != true {
			t.Fatalf("expected notifyUsers=true by default, got %v", payload["notifyUsers"])
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClientWithConfig(srv.URL, "user@example.com", "token123", srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.UpdateIssue(context.Background(), &UpdateIssueRequest{
		IssueKey: "PROJ-1",
		Summary:  "updated",
	})
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}
}

func TestUpdateIssue_NotifyUsersFalseWhenSet(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if payload["notifyUsers"] != false {
			t.Fatalf("expected notifyUsers=false when explicitly set, got %v", payload["notifyUsers"])
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClientWithConfig(srv.URL, "user@example.com", "token123", srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.UpdateIssue(context.Background(), &UpdateIssueRequest{
		IssueKey:    "PROJ-1",
		Summary:     "updated",
		NotifyUsers: wrapperspb.Bool(false),
	})
	if err != nil {
		t.Fatalf("UpdateIssue failed: %v", err)
	}
}

func TestGetServiceDesks_ParsesResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.String(), "/rest/servicedeskapi/servicedesk") {
			t.Fatalf("unexpected path: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"size":       1,
			"start":      0,
			"limit":      25,
			"isLastPage": true,
			"values": []map[string]any{
				{
					"id": "3",
					"project": map[string]any{
						"id":   "10003",
						"key":  "HELP",
						"name": "Help Desk",
					},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig(srv.URL, "user@example.com", "token123", srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	resp, err := client.GetServiceDesks(context.Background(), &GetServiceDesksRequest{})
	if err != nil {
		t.Fatalf("GetServiceDesks failed: %v", err)
	}
	if len(resp.GetServiceDesks()) != 1 {
		t.Fatalf("unexpected service desk count: %d", len(resp.GetServiceDesks()))
	}
	if resp.GetServiceDesks()[0].GetProjectKey() != "HELP" {
		t.Fatalf("unexpected project key: %s", resp.GetServiceDesks()[0].GetProjectKey())
	}
}

func TestSearchUsers_UsesMaxResults(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("maxResults"); got != "7" {
			t.Fatalf("expected maxResults=7, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig(srv.URL, "user@example.com", "token123", srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	if _, err := client.SearchUsers(context.Background(), "abc", 7); err != nil {
		t.Fatalf("SearchUsers failed: %v", err)
	}
}

func TestGetSlaInfo_ParsesCompletedCyclesArray(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"id":   "sla-1",
					"name": "Time to resolution",
					"completedCycles": []map[string]any{
						{"startTime": "t1", "stopTime": "t2"},
						{"startTime": "t3", "stopTime": "t4"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig(srv.URL, "user@example.com", "token123", srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	resp, err := client.GetSlaInfo(context.Background(), &GetSlaInfoRequest{IssueIdOrKey: "HELP-1"})
	if err != nil {
		t.Fatalf("GetSlaInfo failed: %v", err)
	}
	if len(resp.GetValues()) != 1 {
		t.Fatalf("unexpected SLA count: %d", len(resp.GetValues()))
	}
	if len(resp.GetValues()[0].GetCompletedCycles()) != 2 {
		t.Fatalf("expected 2 completed cycles, got %d", len(resp.GetValues()[0].GetCompletedCycles()))
	}
}

func TestJiraErrorDoesNotLeakResponseBody(t *testing.T) {
	t.Parallel()

	secret := "super-secret-jira-detail"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-AREQUESTID", "req-123")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(secret))
	}))
	defer srv.Close()

	client, err := NewClientWithConfig(srv.URL, "user@example.com", "token123", srv.Client())
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}
	_, err = client.GetIssue(context.Background(), "PROJ-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("error leaked response body: %v", err)
	}
	if !strings.Contains(err.Error(), "request_id=req-123") {
		t.Fatalf("error missing request id context: %v", err)
	}
}
