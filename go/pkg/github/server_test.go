package github

import (
	"context"
	"strings"
	"testing"
)

func TestNewServerRequiresToken(t *testing.T) {
	t.Setenv("GITHUB_PERSONAL_ACCESS_TOKEN", "")
	_, err := NewServer()
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
	if !strings.Contains(err.Error(), "GITHUB_PERSONAL_ACCESS_TOKEN") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateIssueRejectsEmptyTitleBeforeNetworkCall(t *testing.T) {
	s := &Server{}
	_, err := s.CreateIssue(context.Background(), &CreateIssueRequest{
		Owner: "o",
		Repo:  "r",
		Title: "  ",
	})
	if err == nil {
		t.Fatal("expected validation error for empty title")
	}
	if !strings.Contains(err.Error(), "title is required") {
		t.Fatalf("unexpected error: %v", err)
	}
}
