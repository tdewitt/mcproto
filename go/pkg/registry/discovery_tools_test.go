package registry

import "testing"

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
