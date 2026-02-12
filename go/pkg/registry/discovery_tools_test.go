package registry

import (
	"context"
	"testing"
)

func TestExtractSearchQuery_JSON(t *testing.T) {
	args := []byte(`{"query":"github"}`)
	query := extractSearchQuery(args)
	if query != "github" {
		t.Fatalf("Expected query github, got %s", query)
	}
}

func TestExtractSearchQuery_Default(t *testing.T) {
	query := extractSearchQuery(nil)
	if query != "" {
		t.Fatalf("Expected empty query for nil input, got %s", query)
	}
}

func TestExtractSearchQuery_PlainText(t *testing.T) {
	args := []byte("golang tools")
	query := extractSearchQuery(args)
	if query != "golang tools" {
		t.Fatalf("Expected plain text query 'golang tools', got %q", query)
	}
}

func TestExtractSearchQuery_EmptyJSON(t *testing.T) {
	args := []byte(`{}`)
	query := extractSearchQuery(args)
	if query != "" {
		t.Fatalf("Expected empty query for empty JSON, got %q", query)
	}
}

func TestDescriptionFromMessageName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"SearchIssuesRequest", "Search issues"},
		{"CreateProjectEvent", "Create project"},
		{"RunBuildTask", "Run build"},
		{"GetUserCall", "Get user"},
		{"ListRepositories", "List repositories"},
		{"Request", "Request"},
		{"X", "X"},
		{"HTMLParser", "H t m l parser"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := descriptionFromMessageName(tt.input)
			if got != tt.want {
				t.Errorf("descriptionFromMessageName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCallByBsrRef_InvalidPrefix(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	_, err := reg.CallByBsrRef(context.Background(), "invalid-ref", nil)
	if err == nil {
		t.Fatal("Expected error for invalid BSR ref prefix, got nil")
	}
	if got := err.Error(); got != `invalid bsr_ref "invalid-ref": must start with "buf.build/"` {
		t.Fatalf("Unexpected error message: %q", got)
	}
}

func TestCallByBsrRef_EmptyRef(t *testing.T) {
	reg := NewUnifiedRegistry(nil)
	_, err := reg.CallByBsrRef(context.Background(), "", nil)
	if err == nil {
		t.Fatal("Expected error for empty BSR ref, got nil")
	}
}

func TestSearchCandidate_CallableField(t *testing.T) {
	// Verify that Callable is correctly derived from LocalToolNames.
	c1 := SearchCandidate{
		Callable:       true,
		LocalToolNames: []string{"my_tool"},
	}
	if !c1.Callable {
		t.Error("Expected Callable=true when LocalToolNames is non-empty")
	}

	c2 := SearchCandidate{
		Callable: false,
	}
	if c2.Callable {
		t.Error("Expected Callable=false when LocalToolNames is empty")
	}
}
