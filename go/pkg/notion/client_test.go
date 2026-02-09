package notion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestNewClient_RequiresToken(t *testing.T) {
	t.Parallel()

	if _, err := NewClientWithConfig("", nil, "https://api.notion.com", ""); err == nil {
		t.Fatal("expected error for empty token")
	}
	if _, err := NewClientWithConfig("   ", nil, "https://api.notion.com", ""); err == nil {
		t.Fatal("expected error for whitespace-only token")
	}
}

func TestSearch_ParsesResponse(t *testing.T) {
	t.Parallel()

	var seenAuth, seenVersion, seenContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		seenVersion = r.Header.Get("Notion-Version")
		seenContentType = r.Header.Get("Content-Type")
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":               "page-123",
					"object":           "page",
					"url":              "https://notion.so/page-123",
					"last_edited_time": "2024-01-02T00:00:00.000Z",
					"parent": map[string]any{
						"type":        "database_id",
						"database_id": "db-456",
					},
					"properties": map[string]any{
						"Name": map[string]any{
							"type": "title",
							"title": []map[string]any{
								{"type": "text", "plain_text": "Test Page"},
							},
						},
					},
				},
			},
			"has_more":    false,
			"next_cursor": "",
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	res, err := client.Search(context.Background(), &SearchRequest{Query: "test"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if len(res.GetResults()) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res.GetResults()))
	}
	r := res.GetResults()[0]
	if r.GetId() != "page-123" {
		t.Fatalf("unexpected id: %s", r.GetId())
	}
	if r.GetObject() != "page" {
		t.Fatalf("unexpected object: %s", r.GetObject())
	}
	if r.GetUrl() != "https://notion.so/page-123" {
		t.Fatalf("unexpected url: %s", r.GetUrl())
	}
	if r.GetTitle() != "Test Page" {
		t.Fatalf("unexpected title: %s", r.GetTitle())
	}
	if r.GetParentId() != "db-456" {
		t.Fatalf("unexpected parent_id: %s", r.GetParentId())
	}
	if res.GetHasMore() {
		t.Fatal("expected has_more=false")
	}

	if seenAuth != "Bearer test-token" {
		t.Fatalf("unexpected auth header: %s", seenAuth)
	}
	if seenVersion != defaultNotionVersion {
		t.Fatalf("unexpected version header: %s", seenVersion)
	}
	if seenContentType != "application/json" {
		t.Fatalf("unexpected content-type header: %s", seenContentType)
	}
}

func TestGetPage_ReturnsPage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/pages/page-123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":               "page-123",
			"object":           "page",
			"url":              "https://notion.so/page-123",
			"created_time":     "2024-01-01T00:00:00.000Z",
			"last_edited_time": "2024-01-02T00:00:00.000Z",
			"archived":         false,
			"parent": map[string]any{
				"type":        "database_id",
				"database_id": "db-456",
			},
			"icon": map[string]any{
				"type":  "emoji",
				"emoji": "\xf0\x9f\x93\x9d",
			},
			"cover": map[string]any{
				"type": "external",
				"external": map[string]any{
					"url": "https://example.com/cover.jpg",
				},
			},
			"properties": map[string]any{
				"Name": map[string]any{
					"type": "title",
					"title": []map[string]any{
						{"type": "text", "plain_text": "Test Page"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	res, err := client.GetPage(context.Background(), &GetPageRequest{PageId: "page-123"})
	if err != nil {
		t.Fatalf("GetPage failed: %v", err)
	}
	page := res.GetPage()
	if page == nil {
		t.Fatal("expected non-nil page")
	}
	if page.GetId() != "page-123" {
		t.Fatalf("unexpected id: %s", page.GetId())
	}
	if page.GetUrl() != "https://notion.so/page-123" {
		t.Fatalf("unexpected url: %s", page.GetUrl())
	}
	if page.GetCreatedTime() != "2024-01-01T00:00:00.000Z" {
		t.Fatalf("unexpected created_time: %s", page.GetCreatedTime())
	}
	if page.GetLastEditedTime() != "2024-01-02T00:00:00.000Z" {
		t.Fatalf("unexpected last_edited_time: %s", page.GetLastEditedTime())
	}
	if page.GetArchived() {
		t.Fatal("expected archived=false")
	}

	// Verify parent
	parent := page.GetParent()
	if parent == nil {
		t.Fatal("expected non-nil parent")
	}
	if parent.GetType() != "database_id" {
		t.Fatalf("unexpected parent type: %s", parent.GetType())
	}
	if parent.GetDatabaseId() != "db-456" {
		t.Fatalf("unexpected parent database_id: %s", parent.GetDatabaseId())
	}

	// Verify icon (emoji)
	if page.GetIcon() != "\xf0\x9f\x93\x9d" {
		t.Fatalf("unexpected icon: %s", page.GetIcon())
	}

	// Verify cover
	if page.GetCover() != "https://example.com/cover.jpg" {
		t.Fatalf("unexpected cover: %s", page.GetCover())
	}

	// Verify properties exist
	if page.GetProperties() == nil {
		t.Fatal("expected non-nil properties")
	}
}

func TestCreatePage_SendsParentAndProperties(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/pages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		// Verify parent structure
		parent, ok := payload["parent"].(map[string]any)
		if !ok {
			t.Fatal("expected parent in payload")
		}
		if parent["type"] != "database_id" {
			t.Fatalf("unexpected parent type: %v", parent["type"])
		}
		if parent["database_id"] != "db-456" {
			t.Fatalf("unexpected parent database_id: %v", parent["database_id"])
		}

		// Verify properties exist
		if _, ok := payload["properties"]; !ok {
			t.Fatal("expected properties in payload")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":               "new-page-1",
			"object":           "page",
			"url":              "https://notion.so/new-page-1",
			"created_time":     "2024-01-01T00:00:00.000Z",
			"last_edited_time": "2024-01-01T00:00:00.000Z",
			"archived":         false,
			"parent": map[string]any{
				"type":        "database_id",
				"database_id": "db-456",
			},
			"properties": map[string]any{
				"Name": map[string]any{
					"type": "title",
					"title": []map[string]any{
						{"type": "text", "plain_text": "New Page"},
					},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	props, _ := structpb.NewStruct(map[string]any{
		"Name": map[string]any{
			"title": []any{
				map[string]any{
					"text": map[string]any{
						"content": "New Page",
					},
				},
			},
		},
	})

	res, err := client.CreatePage(context.Background(), &CreatePageRequest{
		Parent: &Parent{
			DatabaseId: "db-456",
		},
		Properties: props,
	})
	if err != nil {
		t.Fatalf("CreatePage failed: %v", err)
	}
	if res.GetPage().GetId() != "new-page-1" {
		t.Fatalf("unexpected page id: %s", res.GetPage().GetId())
	}
}

func TestCreatePage_RequiresParent(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-token", &http.Client{}, "https://api.notion.com", "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.CreatePage(context.Background(), &CreatePageRequest{
		Parent: nil,
	})
	if err == nil {
		t.Fatal("expected error for nil parent")
	}
	if !strings.Contains(err.Error(), "parent is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestUpdatePage_RequiresPageId(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-token", &http.Client{}, "https://api.notion.com", "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	props, _ := structpb.NewStruct(map[string]any{"Status": "Done"})
	_, err = client.UpdatePage(context.Background(), &UpdatePageRequest{
		PageId:     "",
		Properties: props,
	})
	if err == nil {
		t.Fatal("expected error for empty page_id")
	}
	if !strings.Contains(err.Error(), "page_id is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestUpdatePage_RequiresAtLeastOneField(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-token", &http.Client{}, "https://api.notion.com", "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.UpdatePage(context.Background(), &UpdatePageRequest{
		PageId: "page-123",
	})
	if err == nil {
		t.Fatal("expected validation error for empty update fields")
	}
	if !strings.Contains(err.Error(), "at least one field is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestArchivePage_SetsArchivedFlag(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v1/pages/page-123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		archived, ok := payload["archived"]
		if !ok {
			t.Fatal("expected archived field in payload")
		}
		if archived != true {
			t.Fatalf("expected archived=true, got %v", archived)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":               "page-123",
			"object":           "page",
			"url":              "https://notion.so/page-123",
			"created_time":     "2024-01-01T00:00:00.000Z",
			"last_edited_time": "2024-01-03T00:00:00.000Z",
			"archived":         true,
			"parent": map[string]any{
				"type":        "database_id",
				"database_id": "db-456",
			},
			"properties": map[string]any{},
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	res, err := client.ArchivePage(context.Background(), &ArchivePageRequest{
		PageId:   "page-123",
		Archived: true,
	})
	if err != nil {
		t.Fatalf("ArchivePage failed: %v", err)
	}
	if !res.GetPage().GetArchived() {
		t.Fatal("expected archived=true in response")
	}
}

func TestQueryDatabase_RequiresDatabaseId(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-token", &http.Client{}, "https://api.notion.com", "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.QueryDatabase(context.Background(), &QueryDatabaseRequest{
		DatabaseId: "",
	})
	if err == nil {
		t.Fatal("expected error for empty database_id")
	}
	if !strings.Contains(err.Error(), "database_id is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestGetBlockChildren_ParsesBlocks(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/blocks/block-parent/children") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":               "block-1",
					"type":             "paragraph",
					"created_time":     "2024-01-01T00:00:00.000Z",
					"last_edited_time": "2024-01-01T00:00:00.000Z",
					"has_children":     false,
					"archived":         false,
					"paragraph": map[string]any{
						"rich_text": []map[string]any{
							{"type": "text", "plain_text": "Hello"},
						},
					},
				},
			},
			"has_more":    false,
			"next_cursor": nil,
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	res, err := client.GetBlockChildren(context.Background(), &GetBlockChildrenRequest{
		BlockId: "block-parent",
	})
	if err != nil {
		t.Fatalf("GetBlockChildren failed: %v", err)
	}
	if len(res.GetResults()) != 1 {
		t.Fatalf("expected 1 block, got %d", len(res.GetResults()))
	}
	block := res.GetResults()[0]
	if block.GetId() != "block-1" {
		t.Fatalf("unexpected block id: %s", block.GetId())
	}
	if block.GetType() != "paragraph" {
		t.Fatalf("unexpected block type: %s", block.GetType())
	}
	if block.GetCreatedTime() != "2024-01-01T00:00:00.000Z" {
		t.Fatalf("unexpected created_time: %s", block.GetCreatedTime())
	}
	if block.GetHasChildren() {
		t.Fatal("expected has_children=false")
	}
	if block.GetArchived() {
		t.Fatal("expected archived=false")
	}
	// Verify block data parsed (paragraph content)
	if block.GetData() == nil {
		t.Fatal("expected non-nil block data for paragraph")
	}
	if res.GetHasMore() {
		t.Fatal("expected has_more=false")
	}
}

func TestCreateComment_RequiresRichTextAndParent(t *testing.T) {
	t.Parallel()

	client, err := NewClientWithConfig("test-token", &http.Client{}, "https://api.notion.com", "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	// Missing rich_text
	_, err = client.CreateComment(context.Background(), &CreateCommentRequest{
		ParentPageId: "page-123",
		RichText:     "",
	})
	if err == nil {
		t.Fatal("expected error for empty rich_text")
	}
	if !strings.Contains(err.Error(), "rich_text is required") {
		t.Fatalf("unexpected error message: %v", err)
	}

	// Missing parent (both parent_page_id and discussion_id empty)
	_, err = client.CreateComment(context.Background(), &CreateCommentRequest{
		RichText: "Hello",
	})
	if err == nil {
		t.Fatal("expected error for missing parent")
	}
	if !strings.Contains(err.Error(), "parent_page_id or discussion_id is required") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestListUsers_ParsesResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/users") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"id":         "user-1",
					"type":       "person",
					"name":       "Test User",
					"avatar_url": "https://example.com/avatar.jpg",
					"person": map[string]any{
						"email": "test@example.com",
					},
				},
			},
			"has_more": false,
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	res, err := client.ListUsers(context.Background(), &ListUsersRequest{})
	if err != nil {
		t.Fatalf("ListUsers failed: %v", err)
	}
	if len(res.GetUsers()) != 1 {
		t.Fatalf("expected 1 user, got %d", len(res.GetUsers()))
	}
	user := res.GetUsers()[0]
	if user.GetId() != "user-1" {
		t.Fatalf("unexpected user id: %s", user.GetId())
	}
	if user.GetType() != "person" {
		t.Fatalf("unexpected user type: %s", user.GetType())
	}
	if user.GetName() != "Test User" {
		t.Fatalf("unexpected user name: %s", user.GetName())
	}
	if user.GetAvatarUrl() != "https://example.com/avatar.jpg" {
		t.Fatalf("unexpected avatar_url: %s", user.GetAvatarUrl())
	}
	if user.GetEmail() != "test@example.com" {
		t.Fatalf("unexpected email: %s", user.GetEmail())
	}
	if res.GetHasMore() {
		t.Fatal("expected has_more=false")
	}
}

func TestGetSelf_ParsesResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/users/me" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":         "bot-1",
			"type":       "bot",
			"name":       "My Integration",
			"avatar_url": "https://example.com/bot-avatar.jpg",
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	res, err := client.GetSelf(context.Background(), &GetSelfRequest{})
	if err != nil {
		t.Fatalf("GetSelf failed: %v", err)
	}
	user := res.GetUser()
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.GetId() != "bot-1" {
		t.Fatalf("unexpected user id: %s", user.GetId())
	}
	if user.GetType() != "bot" {
		t.Fatalf("unexpected user type: %s", user.GetType())
	}
	if user.GetName() != "My Integration" {
		t.Fatalf("unexpected user name: %s", user.GetName())
	}
	if user.GetAvatarUrl() != "https://example.com/bot-avatar.jpg" {
		t.Fatalf("unexpected avatar_url: %s", user.GetAvatarUrl())
	}
}

func TestNotionErrorDoesNotLeakResponseBody(t *testing.T) {
	t.Parallel()

	secret := "super-secret-notion-detail"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-456")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(secret))
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.GetPage(context.Background(), &GetPageRequest{PageId: "page-123"})
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

func TestNotionRateLimitError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.Header().Set("X-Request-Id", "req-789")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"message":"rate limited"}`))
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.Search(context.Background(), &SearchRequest{Query: "test"})
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("expected rate limit error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "retry_after=30") {
		t.Fatalf("expected retry_after=30 in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "request_id=req-789") {
		t.Fatalf("expected request_id in error, got: %v", err)
	}
}

func TestNotionVersionHeader(t *testing.T) {
	t.Parallel()

	var seenVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenVersion = r.Header.Get("Notion-Version")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "bot-1",
			"type": "bot",
			"name": "Test Bot",
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, "")
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.GetSelf(context.Background(), &GetSelfRequest{})
	if err != nil {
		t.Fatalf("GetSelf failed: %v", err)
	}
	if seenVersion != defaultNotionVersion {
		t.Fatalf("expected Notion-Version=%s, got %s", defaultNotionVersion, seenVersion)
	}
}

func TestNotionCustomApiVersion(t *testing.T) {
	t.Parallel()

	customVersion := "2023-08-01"
	var seenVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenVersion = r.Header.Get("Notion-Version")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "bot-1",
			"type": "bot",
			"name": "Test Bot",
		})
	}))
	defer srv.Close()

	client, err := NewClientWithConfig("test-token", srv.Client(), srv.URL, customVersion)
	if err != nil {
		t.Fatalf("NewClientWithConfig failed: %v", err)
	}

	_, err = client.GetSelf(context.Background(), &GetSelfRequest{})
	if err != nil {
		t.Fatalf("GetSelf failed: %v", err)
	}
	if seenVersion != customVersion {
		t.Fatalf("expected Notion-Version=%s, got %s", customVersion, seenVersion)
	}
}

func TestExtractTitle_VariousFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		properties map[string]any
		want       string
	}{
		{
			name: "standard Name property",
			properties: map[string]any{
				"Name": map[string]any{
					"type": "title",
					"title": []any{
						map[string]any{"type": "text", "plain_text": "My Page"},
					},
				},
			},
			want: "My Page",
		},
		{
			name: "custom title property name",
			properties: map[string]any{
				"Task Name": map[string]any{
					"type": "title",
					"title": []any{
						map[string]any{"type": "text", "plain_text": "Important Task"},
					},
				},
			},
			want: "Important Task",
		},
		{
			name: "multi-segment title",
			properties: map[string]any{
				"Name": map[string]any{
					"type": "title",
					"title": []any{
						map[string]any{"type": "text", "plain_text": "Hello "},
						map[string]any{"type": "text", "plain_text": "World"},
					},
				},
			},
			want: "Hello World",
		},
		{
			name: "empty title array",
			properties: map[string]any{
				"Name": map[string]any{
					"type":  "title",
					"title": []any{},
				},
			},
			want: "",
		},
		{
			name:       "nil properties",
			properties: nil,
			want:       "",
		},
		{
			name:       "empty properties",
			properties: map[string]any{},
			want:       "",
		},
		{
			name: "no title type property",
			properties: map[string]any{
				"Status": map[string]any{
					"type":   "select",
					"select": map[string]any{"name": "Done"},
				},
			},
			want: "",
		},
		{
			name: "title with mixed non-title properties",
			properties: map[string]any{
				"Status": map[string]any{
					"type":   "select",
					"select": map[string]any{"name": "Done"},
				},
				"Title": map[string]any{
					"type": "title",
					"title": []any{
						map[string]any{"type": "text", "plain_text": "Found It"},
					},
				},
				"Priority": map[string]any{
					"type":   "number",
					"number": 5,
				},
			},
			want: "Found It",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := extractTitle(tt.properties)
			if got != tt.want {
				t.Fatalf("extractTitle() = %q, want %q", got, tt.want)
			}
		})
	}
}
