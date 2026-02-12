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

func TestExtractSearchQuery_Fallback(t *testing.T) {
	args := []byte{0x0, 0x0, 'g', 'o'}
	query := extractSearchQuery(args)
	if query != "go" {
		t.Fatalf("Expected fallback query go, got %s", query)
	}
}
